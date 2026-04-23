// application_service.go
// 模块02 — 学校与租户管理：入驻申请业务逻辑
// 负责提交申请、查询申请状态、重新申请、申请审核（通过/拒绝）
// 对照 docs/modules/02-学校与租户管理/03-API接口设计.md

package school

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/lenschain/backend/internal/model/dto"
	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/model/enum"
	"github.com/lenschain/backend/internal/pkg/audit"
	"github.com/lenschain/backend/internal/pkg/cache"
	svcctx "github.com/lenschain/backend/internal/pkg/context"
	cronpkg "github.com/lenschain/backend/internal/pkg/cron"
	"github.com/lenschain/backend/internal/pkg/database"
	"github.com/lenschain/backend/internal/pkg/errcode"
	"github.com/lenschain/backend/internal/pkg/logger"
	"github.com/lenschain/backend/internal/pkg/mask"
	"github.com/lenschain/backend/internal/pkg/sms"
	schoolrepo "github.com/lenschain/backend/internal/repository/school"
)

// ApplicationService 入驻申请服务接口
type ApplicationService interface {
	Submit(ctx context.Context, req *dto.SubmitApplicationReq) (*dto.SubmitApplicationResp, error)
	SendSMSCode(ctx context.Context, phone string) error
	Query(ctx context.Context, phone string) (*dto.QueryApplicationResp, error)
	GetReapplyDetail(ctx context.Context, previousID int64, phone string) (*dto.ReapplyDetailResp, error)
	Reapply(ctx context.Context, previousID int64, req *dto.ReapplyReq) (*dto.SubmitApplicationResp, error)
	List(ctx context.Context, sc *svcctx.ServiceContext, req *dto.ApplicationListReq) ([]*dto.ApplicationListItem, int64, error)
	GetByID(ctx context.Context, sc *svcctx.ServiceContext, id int64) (*dto.ApplicationDetailResp, error)
	Approve(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.ApproveApplicationReq) (*dto.ApproveApplicationResp, error)
	Reject(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.RejectApplicationReq) error
}

// applicationService 入驻申请服务实现
type applicationService struct {
	db           *gorm.DB
	appRepo      schoolrepo.ApplicationRepository
	schoolRepo   schoolrepo.SchoolRepository
	notifyRepo   schoolrepo.NotificationRepository
	adminCreator AdminCreator
}

// AdminCreator 跨模块接口：创建首个校管账号
// 由模块01的实现注入，解耦跨模块依赖
type AdminCreator interface {
	CreateSchoolAdmin(ctx context.Context, schoolID int64, phone, name string, createdBy int64) (userID int64, password string, err error)
}

// NewApplicationService 创建入驻申请服务实例
func NewApplicationService(
	db *gorm.DB,
	appRepo schoolrepo.ApplicationRepository,
	schoolRepo schoolrepo.SchoolRepository,
	notifyRepo schoolrepo.NotificationRepository,
	adminCreator AdminCreator,
) ApplicationService {
	return &applicationService{
		db:           db,
		appRepo:      appRepo,
		schoolRepo:   schoolRepo,
		notifyRepo:   notifyRepo,
		adminCreator: adminCreator,
	}
}

// Submit 提交入驻申请
func (s *applicationService) Submit(ctx context.Context, req *dto.SubmitApplicationReq) (*dto.SubmitApplicationResp, error) {
	// 检查是否有待审核的申请（同一手机号）
	existing, err := s.appRepo.GetPendingByPhone(ctx, req.ContactPhone)
	if err == nil && existing != nil {
		return nil, errcode.ErrDuplicateApplication
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errcode.ErrInternal.WithMessage("校验待审核申请失败")
	}

	// 检查学校名称唯一性
	existingSchool, err := s.schoolRepo.GetByName(ctx, req.SchoolName)
	if err == nil && existingSchool != nil {
		return nil, errcode.ErrDuplicateSchoolName
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errcode.ErrInternal.WithMessage("校验学校名称失败")
	}

	// 检查学校编码唯一性
	existingByCode, err := s.schoolRepo.GetByCode(ctx, req.SchoolCode)
	if err == nil && existingByCode != nil {
		return nil, errcode.ErrDuplicateSchoolCode
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errcode.ErrInternal.WithMessage("校验学校编码失败")
	}

	// 创建申请记录
	app := &entity.SchoolApplication{
		SchoolName:    req.SchoolName,
		SchoolCode:    req.SchoolCode,
		SchoolAddress: req.SchoolAddress,
		SchoolWebsite: req.SchoolWebsite,
		SchoolLogoURL: req.SchoolLogoURL,
		ContactName:   req.ContactName,
		ContactPhone:  req.ContactPhone,
		ContactEmail:  req.ContactEmail,
		ContactTitle:  req.ContactTitle,
		Status:        enum.ApplicationStatusPending,
	}
	if err := s.appRepo.Create(ctx, app); err != nil {
		return nil, errcode.ErrInternal.WithMessage("创建申请记录失败")
	}

	return &dto.SubmitApplicationResp{
		ApplicationID: strconv.FormatInt(app.ID, 10),
		Status:        enum.ApplicationStatusPending,
		StatusText:    enum.GetApplicationStatusText(enum.ApplicationStatusPending),
		Tip:           "请使用联系人手机号查询审核进度",
	}, nil
}

// SendSMSCode 发送申请查询/重申验证码
// 为避免泄露申请状态，无论手机号是否存在申请记录都返回统一成功响应。
func (s *applicationService) SendSMSCode(ctx context.Context, phone string) error {
	err := sms.SendVerificationCode(ctx, phone)
	if err == nil {
		return nil
	}
	if err.Error() == "短信发送过于频繁" {
		return errcode.ErrSMSCodeSendTooFrequent
	}
	logger.L.Error("发送申请查询验证码失败", zap.String("phone", phone), zap.Error(err))
	return errcode.ErrInternal.WithMessage("发送验证码失败")
}

// Query 查询申请状态（通过手机号，SMS验证在handler层完成）
func (s *applicationService) Query(ctx context.Context, phone string) (*dto.QueryApplicationResp, error) {
	apps, err := s.appRepo.ListByPhone(ctx, phone)
	if err != nil {
		return nil, errcode.ErrInternal.WithMessage("查询申请记录失败")
	}

	items := make([]dto.ApplicationStatusItem, 0, len(apps))
	for _, app := range apps {
		item := dto.ApplicationStatusItem{
			ApplicationID: strconv.FormatInt(app.ID, 10),
			SchoolName:    app.SchoolName,
			Status:        app.Status,
			StatusText:    enum.GetApplicationStatusText(app.Status),
			CreatedAt:     app.CreatedAt.Format(time.RFC3339),
		}
		if app.ReviewedAt != nil {
			t := app.ReviewedAt.Format(time.RFC3339)
			item.ReviewedAt = &t
		}
		if app.RejectReason != nil {
			item.RejectReason = app.RejectReason
		}
		items = append(items, item)
	}

	return &dto.QueryApplicationResp{Applications: items}, nil
}

// GetReapplyDetail 获取拒绝申请的完整信息，用于重新申请页面预填。
func (s *applicationService) GetReapplyDetail(ctx context.Context, previousID int64, phone string) (*dto.ReapplyDetailResp, error) {
	app, err := s.appRepo.GetByID(ctx, previousID)
	if err != nil {
		return nil, errcode.ErrApplicationNotFound
	}
	if app.Status != enum.ApplicationStatusRejected {
		return nil, errcode.ErrApplicationNotRejected
	}
	if app.ContactPhone != phone {
		return nil, errcode.ErrForbidden.WithMessage("手机号与原申请不一致")
	}

	return &dto.ReapplyDetailResp{
		ApplicationID: strconv.FormatInt(app.ID, 10),
		SchoolName:    app.SchoolName,
		SchoolCode:    app.SchoolCode,
		SchoolAddress: app.SchoolAddress,
		SchoolWebsite: app.SchoolWebsite,
		SchoolLogoURL: app.SchoolLogoURL,
		ContactName:   app.ContactName,
		ContactPhone:  app.ContactPhone,
		ContactEmail:  app.ContactEmail,
		ContactTitle:  app.ContactTitle,
		Status:        app.Status,
		StatusText:    enum.GetApplicationStatusText(app.Status),
		RejectReason:  app.RejectReason,
	}, nil
}

// Reapply 重新申请（仅已拒绝的申请可重新提交）
func (s *applicationService) Reapply(ctx context.Context, previousID int64, req *dto.ReapplyReq) (*dto.SubmitApplicationResp, error) {
	// 获取原申请
	previous, err := s.appRepo.GetByID(ctx, previousID)
	if err != nil {
		return nil, errcode.ErrApplicationNotFound
	}

	// 校验状态：仅已拒绝的申请可重新提交
	if previous.Status != enum.ApplicationStatusRejected {
		return nil, errcode.ErrApplicationNotRejected
	}

	// 校验手机号一致
	if previous.ContactPhone != req.ContactPhone {
		return nil, errcode.ErrForbidden.WithMessage("手机号与原申请不一致")
	}

	// 与首次提交保持一致：同一手机号不能同时存在待审核申请。
	if existingPending, err := s.appRepo.GetPendingByPhone(ctx, req.ContactPhone); err == nil && existingPending != nil {
		return nil, errcode.ErrDuplicateApplication
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errcode.ErrInternal.WithMessage("校验待审核申请失败")
	}

	// 检查学校名称唯一性（与 Submit 保持一致）
	if existingSchool, err := s.schoolRepo.GetByName(ctx, req.SchoolName); err == nil && existingSchool != nil {
		return nil, errcode.ErrDuplicateSchoolName
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errcode.ErrInternal.WithMessage("校验学校名称失败")
	}

	// 检查学校编码唯一性（与 Submit 保持一致）
	if existingByCode, err := s.schoolRepo.GetByCode(ctx, req.SchoolCode); err == nil && existingByCode != nil {
		return nil, errcode.ErrDuplicateSchoolCode
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errcode.ErrInternal.WithMessage("校验学校编码失败")
	}

	// 创建新申请记录，关联上一次申请
	app := &entity.SchoolApplication{
		SchoolName:            req.SchoolName,
		SchoolCode:            req.SchoolCode,
		SchoolAddress:         req.SchoolAddress,
		SchoolWebsite:         req.SchoolWebsite,
		SchoolLogoURL:         req.SchoolLogoURL,
		ContactName:           req.ContactName,
		ContactPhone:          req.ContactPhone,
		ContactEmail:          req.ContactEmail,
		ContactTitle:          req.ContactTitle,
		Status:                enum.ApplicationStatusPending,
		PreviousApplicationID: &previousID,
	}
	if err := s.appRepo.Create(ctx, app); err != nil {
		return nil, errcode.ErrInternal.WithMessage("创建申请记录失败")
	}

	return &dto.SubmitApplicationResp{
		ApplicationID: strconv.FormatInt(app.ID, 10),
		Status:        enum.ApplicationStatusPending,
		StatusText:    enum.GetApplicationStatusText(enum.ApplicationStatusPending),
		Tip:           "重新申请已提交，请等待审核",
	}, nil
}

// List 申请列表
func (s *applicationService) List(ctx context.Context, sc *svcctx.ServiceContext, req *dto.ApplicationListReq) ([]*dto.ApplicationListItem, int64, error) {
	params := &schoolrepo.ApplicationListParams{
		Status:    req.Status,
		Keyword:   req.Keyword,
		SortBy:    req.SortBy,
		SortOrder: req.SortOrder,
		Page:      req.Page,
		PageSize:  req.PageSize,
	}

	apps, total, err := s.appRepo.List(ctx, params)
	if err != nil {
		return nil, 0, errcode.ErrInternal.WithMessage("查询申请列表失败")
	}

	items := make([]*dto.ApplicationListItem, 0, len(apps))
	for _, app := range apps {
		item := &dto.ApplicationListItem{
			ID:           strconv.FormatInt(app.ID, 10),
			SchoolName:   app.SchoolName,
			SchoolCode:   app.SchoolCode,
			ContactName:  app.ContactName,
			ContactPhone: mask.Phone(app.ContactPhone),
			Status:       app.Status,
			StatusText:   enum.GetApplicationStatusText(app.Status),
			CreatedAt:    app.CreatedAt.Format(time.RFC3339),
		}
		if app.ReviewedAt != nil {
			t := app.ReviewedAt.Format(time.RFC3339)
			item.ReviewedAt = &t
		}
		items = append(items, item)
	}

	return items, total, nil
}

// GetByID 申请详情
func (s *applicationService) GetByID(ctx context.Context, sc *svcctx.ServiceContext, id int64) (*dto.ApplicationDetailResp, error) {
	app, err := s.appRepo.GetByID(ctx, id)
	if err != nil {
		return nil, errcode.ErrApplicationNotFound
	}
	return applicationToDetailResp(app), nil
}

// Approve 审核通过
// 1. 更新申请状态 2. 创建学校 3. 创建首个校管 4. 发送短信
func (s *applicationService) Approve(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.ApproveApplicationReq) (*dto.ApproveApplicationResp, error) {
	app, err := s.appRepo.GetByID(ctx, id)
	if err != nil {
		return nil, errcode.ErrApplicationNotFound
	}
	if app.Status != enum.ApplicationStatusPending {
		return nil, errcode.ErrApplicationNotPending
	}

	// 解析授权结束时间
	licenseEnd, err := time.Parse(time.RFC3339, req.LicenseEndAt)
	if err != nil {
		return nil, errcode.ErrInvalidParams.WithMessage("授权结束时间格式不正确")
	}

	// 再次检查学校名称/编码唯一性
	if existing, err := s.schoolRepo.GetByName(ctx, app.SchoolName); err == nil && existing != nil {
		return nil, errcode.ErrDuplicateSchoolName
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errcode.ErrInternal.WithMessage("校验学校名称失败")
	}
	if existing, err := s.schoolRepo.GetByCode(ctx, app.SchoolCode); err == nil && existing != nil {
		return nil, errcode.ErrDuplicateSchoolCode
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errcode.ErrInternal.WithMessage("校验学校编码失败")
	}

	var schoolID, adminUserID int64
	var adminPassword string
	now := time.Now()

	// 事务：创建学校 + 更新申请 + 创建校管
	err = database.TransactionWithDB(ctx, s.db, func(tx *gorm.DB) error {
		txCtx := database.WithTxContext(ctx, tx)
		txSchoolRepo := schoolrepo.NewSchoolRepository(tx)
		txAppRepo := schoolrepo.NewApplicationRepository(tx)

		// 1. 创建学校
		schoolEntity := &entity.School{
			Name:           app.SchoolName,
			Code:           app.SchoolCode,
			LogoURL:        app.SchoolLogoURL,
			Address:        app.SchoolAddress,
			Website:        app.SchoolWebsite,
			Status:         enum.SchoolStatusActive,
			LicenseStartAt: &now,
			LicenseEndAt:   &licenseEnd,
			ContactName:    app.ContactName,
			ContactPhone:   app.ContactPhone,
			ContactEmail:   app.ContactEmail,
			ContactTitle:   app.ContactTitle,
			CreatedBy:      &sc.UserID,
		}
		if err := txSchoolRepo.Create(ctx, schoolEntity); err != nil {
			return err
		}
		schoolID = schoolEntity.ID

		// 2. 更新申请状态
		reviewedAt := time.Now()
		if err := txAppRepo.UpdateFields(ctx, id, map[string]interface{}{
			"status":      enum.ApplicationStatusApproved,
			"reviewer_id": sc.UserID,
			"reviewed_at": reviewedAt,
			"school_id":   schoolID,
		}); err != nil {
			return err
		}

		// 3. 创建首个校管账号（跨模块调用）
		var createErr error
		adminUserID, adminPassword, createErr = s.adminCreator.CreateSchoolAdmin(txCtx, schoolID, app.ContactPhone, app.ContactName, sc.UserID)
		return createErr
	})
	if err != nil {
		if appErr, ok := errcode.IsAppError(err); ok {
			return nil, appErr
		}
		logger.L.Error("审核通过处理失败", zap.Error(err))
		return nil, errcode.ErrInternal.WithMessage("审核通过处理失败")
	}

	// 刷新学校状态缓存
	refreshSchoolStatusCache(ctx, schoolID, enum.SchoolStatusActive, &licenseEnd)

	// 记录学校通知，保留审核通过的发送流水。
	if err := s.notifyRepo.Create(ctx, &entity.SchoolNotification{
		SchoolID:    schoolID,
		Type:        enum.SchoolNotifyApproved,
		Title:       enum.GetSchoolNotifyText(enum.SchoolNotifyApproved),
		Content:     "学校入驻申请已审核通过，首个校管账号已创建",
		IsSent:      true,
		SentAt:      &now,
		TargetPhone: &app.ContactPhone,
	}); err != nil {
		logger.L.Error("创建审核通过通知记录失败", zap.Int64("school_id", schoolID), zap.Error(err))
	}

	// 异步发送短信通知，统一走公共后台任务入口，避免业务层散落裸 goroutine。
	smsSent := true
	cronpkg.RunAsync("学校入驻审核通过短信", func(context.Context) {
		err := sms.Send(app.ContactPhone, sms.TemplateSchoolApproved, map[string]string{
			"school_name": app.SchoolName,
			"phone":       app.ContactPhone,
			"password":    adminPassword,
		})
		if err != nil {
			logger.L.Error("发送审核通过短信失败", zap.String("phone", app.ContactPhone), zap.Error(err))
		}
	})

	// 记录操作日志
	audit.RecordFromContext(s.db, sc.UserID, sc.ClientIP, "approve_application", "school_application", id, map[string]interface{}{
		"school_id":     schoolID,
		"admin_user_id": adminUserID,
	})

	return &dto.ApproveApplicationResp{
		SchoolID:    strconv.FormatInt(schoolID, 10),
		AdminUserID: strconv.FormatInt(adminUserID, 10),
		AdminPhone:  app.ContactPhone,
		SMSSent:     smsSent,
	}, nil
}

// Reject 审核拒绝
func (s *applicationService) Reject(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.RejectApplicationReq) error {
	app, err := s.appRepo.GetByID(ctx, id)
	if err != nil {
		return errcode.ErrApplicationNotFound
	}
	if app.Status != enum.ApplicationStatusPending {
		return errcode.ErrApplicationNotPending
	}

	now := time.Now()
	if err := s.appRepo.UpdateFields(ctx, id, map[string]interface{}{
		"status":        enum.ApplicationStatusRejected,
		"reviewer_id":   sc.UserID,
		"reviewed_at":   now,
		"reject_reason": req.RejectReason,
	}); err != nil {
		return errcode.ErrInternal.WithMessage("更新申请状态失败")
	}

	// 异步发送拒绝短信，统一走公共后台任务入口。
	cronpkg.RunAsync("学校入驻审核拒绝短信", func(context.Context) {
		err := sms.Send(app.ContactPhone, sms.TemplateSchoolRejected, map[string]string{
			"school_name":   app.SchoolName,
			"reject_reason": req.RejectReason,
		})
		if err != nil {
			logger.L.Error("发送审核拒绝短信失败", zap.String("phone", app.ContactPhone), zap.Error(err))
		}
	})

	// 记录操作日志
	audit.RecordFromContext(s.db, sc.UserID, sc.ClientIP, "reject_application", "school_application", id, map[string]interface{}{
		"reject_reason": req.RejectReason,
	})

	return nil
}

// ========== 内部辅助方法 ==========

// refreshSchoolStatusCache 刷新学校状态缓存
// 缓存 JSON 结构与 tenant.go 中间件读取的结构一致
func refreshSchoolStatusCache(ctx context.Context, schoolID int64, status int16, licenseEndAt *time.Time) {
	data := struct {
		Status       int16  `json:"status"`
		LicenseEndAt string `json:"license_end_at,omitempty"`
	}{
		Status: status,
	}
	if licenseEndAt != nil {
		data.LicenseEndAt = licenseEndAt.Format(time.RFC3339)
	}
	value, err := json.Marshal(data)
	if err != nil {
		logger.L.Error("序列化学校状态缓存失败", zap.Error(err))
		return
	}
	_ = cache.Set(ctx, cache.KeySchoolStatus+strconv.FormatInt(schoolID, 10), string(value), 1*time.Hour)
}

// deleteSchoolStatusCache 删除学校状态缓存
func deleteSchoolStatusCache(ctx context.Context, schoolID int64) {
	_ = cache.Del(ctx, cache.KeySchoolStatus+strconv.FormatInt(schoolID, 10))
}

// applicationToDetailResp 申请实体转详情响应
func applicationToDetailResp(app *entity.SchoolApplication) *dto.ApplicationDetailResp {
	resp := &dto.ApplicationDetailResp{
		ID:            strconv.FormatInt(app.ID, 10),
		SchoolName:    app.SchoolName,
		SchoolCode:    app.SchoolCode,
		SchoolAddress: app.SchoolAddress,
		SchoolWebsite: app.SchoolWebsite,
		SchoolLogoURL: app.SchoolLogoURL,
		ContactName:   app.ContactName,
		ContactPhone:  app.ContactPhone,
		ContactEmail:  app.ContactEmail,
		ContactTitle:  app.ContactTitle,
		Status:        app.Status,
		StatusText:    enum.GetApplicationStatusText(app.Status),
		RejectReason:  app.RejectReason,
		CreatedAt:     app.CreatedAt.Format(time.RFC3339),
	}
	if app.ReviewerID != nil {
		rid := strconv.FormatInt(*app.ReviewerID, 10)
		resp.ReviewerID = &rid
	}
	if app.ReviewedAt != nil {
		t := app.ReviewedAt.Format(time.RFC3339)
		resp.ReviewedAt = &t
	}
	if app.SchoolID != nil {
		sid := strconv.FormatInt(*app.SchoolID, 10)
		resp.SchoolID = &sid
	}
	if app.PreviousApplicationID != nil {
		pid := strconv.FormatInt(*app.PreviousApplicationID, 10)
		resp.PreviousApplicationID = &pid
	}
	return resp
}
