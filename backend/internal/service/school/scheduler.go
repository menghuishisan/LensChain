// scheduler.go
// 模块02 — 学校与租户管理：定时任务业务逻辑
// 负责授权到期检查、缓冲期转冻结、到期提醒三个定时任务
// 对照 docs/modules/02-学校与租户管理/02-数据库设计.md §四 定时任务

package school

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"go.uber.org/zap"

	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/model/enum"
	cronpkg "github.com/lenschain/backend/internal/pkg/cron"
	"github.com/lenschain/backend/internal/pkg/logger"
	"github.com/lenschain/backend/internal/pkg/sms"
	schoolrepo "github.com/lenschain/backend/internal/repository/school"
)

// SchoolScheduler 学校定时任务服务
// 包含三个定时任务：
// 1. 到期转缓冲期（每天 00:30）
// 2. 到期提醒检查（每天 01:00）
// 3. 缓冲期转冻结（每天 02:00）
type SchoolScheduler struct {
	schoolRepo           schoolrepo.SchoolRepository
	notifyRepo           schoolrepo.NotificationRepository
	sessionKicker        SessionKicker
	adminContactProvider SchoolAdminContactProvider
}

// SchoolAdminContactProvider 跨模块接口：获取学校管理员手机号列表。
// 到期提醒要求通知本校所有管理员，模块02通过接口读取模块01用户数据，不直接访问跨模块表。
type SchoolAdminContactProvider interface {
	ListAdminPhones(ctx context.Context, schoolID int64) ([]string, error)
}

// NewSchoolScheduler 创建学校定时任务服务实例
func NewSchoolScheduler(
	schoolRepo schoolrepo.SchoolRepository,
	notifyRepo schoolrepo.NotificationRepository,
	sessionKicker SessionKicker,
	adminContactProvider SchoolAdminContactProvider,
) *SchoolScheduler {
	return &SchoolScheduler{
		schoolRepo:           schoolRepo,
		notifyRepo:           notifyRepo,
		sessionKicker:        sessionKicker,
		adminContactProvider: adminContactProvider,
	}
}

// RunExpireToBuffering 到期转缓冲期
// 每天凌晨 00:30 执行
// 检查已激活且授权已过期的学校，将状态从 Active 转为 Buffering
func (s *SchoolScheduler) RunExpireToBuffering() {
	ctx := context.Background()
	now := time.Now()

	// 查询已激活且授权已过期的学校
	schools, err := s.schoolRepo.ListExpiredActive(ctx, now)
	if err != nil {
		logger.L.Error("定时任务[到期转缓冲期]查询学校失败", zap.Error(err))
		return
	}

	count := 0
	for _, sch := range schools {
		// 更新状态为缓冲期
		if err := s.schoolRepo.UpdateFields(ctx, sch.ID, map[string]interface{}{
			"status":     enum.SchoolStatusBuffering,
			"updated_at": now,
		}); err != nil {
			logger.L.Error("定时任务[到期转缓冲期]更新学校状态失败",
				zap.Int64("school_id", sch.ID),
				zap.Error(err),
			)
			continue
		}

		// 刷新缓存
		refreshSchoolStatusCache(ctx, sch.ID, enum.SchoolStatusBuffering, sch.LicenseEndAt)

		// 创建学校通知记录
		expireDate := ""
		if sch.LicenseEndAt != nil {
			expireDate = sch.LicenseEndAt.Format("2006-01-02")
		}
		s.createNotification(ctx, sch.ID, enum.SchoolNotifyBuffering, sch.ContactPhone,
			fmt.Sprintf("学校「%s」授权已于 %s 到期，已进入7天缓冲期，请尽快续期", sch.Name, expireDate))

		// 发送短信通知，统一走公共后台任务入口。
		school := sch
		cronpkg.RunAsync("学校到期转缓冲期短信", func(context.Context) {
			if school.ContactPhone != "" {
				_ = sms.Send(school.ContactPhone, sms.TemplateLicenseExpired, map[string]string{
					"school_name": school.Name,
				})
			}
		})

		count++
		logger.L.Info("学校授权到期，已转为缓冲期",
			zap.Int64("school_id", sch.ID),
			zap.String("school_name", sch.Name),
		)
	}

	if count > 0 {
		logger.L.Info("定时任务[到期转缓冲期]执行完成", zap.Int("count", count))
	}
}

// RunExpiryReminder 到期提醒检查
// 每天凌晨 01:00 执行
// 检查7天内即将到期的学校，发送提醒通知（每校只发一次）
func (s *SchoolScheduler) RunExpiryReminder() {
	ctx := context.Background()
	now := time.Now()
	sevenDaysLater := now.AddDate(0, 0, 7)

	// 查询7天内即将到期的已激活学校
	schools, err := s.schoolRepo.ListExpiringSoon(ctx, sevenDaysLater)
	if err != nil {
		logger.L.Error("定时任务[到期提醒]查询学校失败", zap.Error(err))
		return
	}

	count := 0
	for _, sch := range schools {
		// 文档要求到期提醒只发送一次，不按天重复发送。
		exists, err := s.notifyRepo.ExistsBySchoolAndType(ctx, sch.ID, enum.SchoolNotifyExpiring)
		if err != nil {
			logger.L.Error("定时任务[到期提醒]检查通知记录失败",
				zap.Int64("school_id", sch.ID),
				zap.Error(err),
			)
			continue
		}
		if exists {
			continue // 已发送过提醒，跳过
		}

		// 计算剩余天数
		remaining := int(time.Until(*sch.LicenseEndAt).Hours() / 24)

		// 创建学校通知记录，并向该校所有管理员发送通知流水
		expireDate := ""
		if sch.LicenseEndAt != nil {
			expireDate = sch.LicenseEndAt.Format("2006-01-02")
		}
		detail := fmt.Sprintf("学校「%s」授权将于 %s 到期（剩余%d天），请及时续期", sch.Name, expireDate, remaining)
		adminPhones := s.listAdminPhones(ctx, sch)
		for _, phone := range adminPhones {
			s.createNotification(ctx, sch.ID, enum.SchoolNotifyExpiring, phone, detail)
		}

		// 发送短信提醒给该校所有管理员；无管理员手机号时退回联系人手机号。
		school := sch
		days := remaining
		phones := append([]string(nil), adminPhones...)
		cronpkg.RunAsync("学校授权到期提醒短信", func(context.Context) {
			for _, phone := range phones {
				if phone == "" {
					continue
				}
				_ = sms.Send(phone, sms.TemplateLicenseExpiring, map[string]string{
					"school_name":    school.Name,
					"remaining_days": strconv.Itoa(days),
				})
			}
		})

		count++
		logger.L.Info("发送学校到期提醒",
			zap.Int64("school_id", sch.ID),
			zap.String("school_name", sch.Name),
			zap.Int("remaining_days", remaining),
		)
	}

	if count > 0 {
		logger.L.Info("定时任务[到期提醒]执行完成", zap.Int("count", count))
	}
}

// RunBufferingToFrozen 缓冲期转冻结
// 每天凌晨 02:00 执行
// 检查缓冲期已满7天的学校，自动转为冻结状态，踢出所有用户
func (s *SchoolScheduler) RunBufferingToFrozen() {
	ctx := context.Background()
	now := time.Now()
	sevenDaysAgo := now.AddDate(0, 0, -7)

	// 查询缓冲期已满7天的学校（license_end_at <= 7天前）
	schools, err := s.schoolRepo.ListBufferingExpired(ctx, sevenDaysAgo)
	if err != nil {
		logger.L.Error("定时任务[缓冲期转冻结]查询学校失败", zap.Error(err))
		return
	}

	count := 0
	for _, sch := range schools {
		// 更新状态为冻结
		if err := s.schoolRepo.UpdateFields(ctx, sch.ID, map[string]interface{}{
			"status":        enum.SchoolStatusFrozen,
			"frozen_at":     now,
			"frozen_reason": "授权到期，缓冲期结束自动冻结",
			"updated_at":    now,
		}); err != nil {
			logger.L.Error("定时任务[缓冲期转冻结]更新学校状态失败",
				zap.Int64("school_id", sch.ID),
				zap.Error(err),
			)
			continue
		}

		// 清除缓存
		deleteSchoolStatusCache(ctx, sch.ID)

		// 踢出该校所有用户的Session
		if s.sessionKicker != nil {
			if err := s.sessionKicker.KickSchoolUsers(ctx, sch.ID); err != nil {
				logger.L.Error("定时任务[缓冲期转冻结]踢出用户Session失败",
					zap.Int64("school_id", sch.ID),
					zap.Error(err),
				)
			}
		}

		// 创建通知记录
		s.createNotification(ctx, sch.ID, enum.SchoolNotifyFrozen, sch.ContactPhone,
			fmt.Sprintf("学校「%s」缓冲期已满7天，已自动冻结，如需恢复请联系管理员", sch.Name))

		// 发送冻结通知短信，统一走公共后台任务入口。
		school := sch
		cronpkg.RunAsync("学校缓冲期冻结短信", func(context.Context) {
			if school.ContactPhone != "" {
				_ = sms.Send(school.ContactPhone, sms.TemplateSchoolFrozen, map[string]string{
					"school_name": school.Name,
				})
			}
		})

		count++
		logger.L.Info("学校缓冲期结束，已自动冻结",
			zap.Int64("school_id", sch.ID),
			zap.String("school_name", sch.Name),
		)
	}

	if count > 0 {
		logger.L.Info("定时任务[缓冲期转冻结]执行完成", zap.Int("count", count))
	}
}

// createNotification 创建学校通知记录。
// 该方法只负责通知落库，不参与发送逻辑判断。
func (s *SchoolScheduler) createNotification(ctx context.Context, schoolID int64, notifyType int16, contactPhone, detail string) {
	title := enum.SchoolNotifyText[notifyType]
	content := title
	if detail != "" {
		content = detail
	}

	notification := &entity.SchoolNotification{
		SchoolID: schoolID,
		Type:     notifyType,
		Title:    title,
		Content:  content,
		IsSent:   true,
	}
	now := time.Now()
	notification.SentAt = &now
	if contactPhone != "" {
		notification.TargetPhone = &contactPhone
	}

	if err := s.notifyRepo.Create(ctx, notification); err != nil {
		logger.L.Error("创建学校通知记录失败",
			zap.Int64("school_id", schoolID),
			zap.Int16("type", notifyType),
			zap.Error(err),
		)
	}
}

// listAdminPhones 获取学校管理员手机号列表。
// 文档要求到期提醒发送给所有管理员；若暂时未查询到管理员，则回退到学校联系人手机号，避免提醒完全丢失。
func (s *SchoolScheduler) listAdminPhones(ctx context.Context, school *entity.School) []string {
	phones := make([]string, 0, 4)
	seen := make(map[string]struct{})

	if s.adminContactProvider != nil {
		adminPhones, err := s.adminContactProvider.ListAdminPhones(ctx, school.ID)
		if err != nil {
			logger.L.Error("查询学校管理员手机号失败",
				zap.Int64("school_id", school.ID),
				zap.Error(err),
			)
		} else {
			for _, phone := range adminPhones {
				if phone == "" {
					continue
				}
				if _, ok := seen[phone]; ok {
					continue
				}
				seen[phone] = struct{}{}
				phones = append(phones, phone)
			}
		}
	}

	if len(phones) == 0 && school.ContactPhone != "" {
		phones = append(phones, school.ContactPhone)
	}

	return phones
}
