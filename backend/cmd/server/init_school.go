// init_school.go
// 模块02 — 学校与租户管理：依赖注入初始化
// 按照 repository → service → handler 的顺序组装模块02的依赖
// 同时创建跨模块 adapter（AdminCreator、SessionKicker、SchoolNameQuerier）
// 注册定时任务到 cron 调度器

package main

import (
	"context"
	"errors"
	"strconv"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	handler "github.com/lenschain/backend/internal/handler/school"
	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/model/enum"
	"github.com/lenschain/backend/internal/pkg/cache"
	cronpkg "github.com/lenschain/backend/internal/pkg/cron"
	"github.com/lenschain/backend/internal/pkg/crypto"
	"github.com/lenschain/backend/internal/pkg/database"
	"github.com/lenschain/backend/internal/pkg/errcode"
	"github.com/lenschain/backend/internal/pkg/logger"
	"github.com/lenschain/backend/internal/pkg/snowflake"
	"github.com/lenschain/backend/internal/pkg/tokenstate"
	authrepo "github.com/lenschain/backend/internal/repository/auth"
	schoolrepo "github.com/lenschain/backend/internal/repository/school"
	"github.com/lenschain/backend/internal/router"
	svc "github.com/lenschain/backend/internal/service/school"
)

// initSchoolModule 初始化模块02（学校与租户管理）的 Handler
// 按照 repository → adapter → service → handler → cron 的顺序组装依赖
func initSchoolModule() *router.SchoolHandlers {
	db := database.Get()

	// ========== Repository 层 ==========
	schoolRepo := schoolrepo.NewSchoolRepository(db)
	appRepo := schoolrepo.NewApplicationRepository(db)
	ssoRepo := schoolrepo.NewSSOConfigRepository(db)
	notifyRepo := schoolrepo.NewNotificationRepository(db)

	// ========== 跨模块 Adapter ==========
	// 模块01 的 repo（用于 adapter 内部调用）
	userRepo := authrepo.NewUserRepository(db)
	roleRepo := authrepo.NewRoleRepository(db)

	adminCreator := &adminCreatorAdapter{userRepo: userRepo, roleRepo: roleRepo}
	sessionKicker := &sessionKickerAdapter{userRepo: userRepo}
	userLifecycle := &schoolUserLifecycleAdapter{}
	adminContactProvider := &schoolAdminContactProviderAdapter{userRepo: userRepo}

	// ========== Service 层 ==========
	appService := svc.NewApplicationService(db, appRepo, schoolRepo, notifyRepo, adminCreator)
	schoolService := svc.NewSchoolService(db, schoolRepo, adminCreator, sessionKicker, userLifecycle)
	ssoService := svc.NewSSOService(ssoRepo)

	// ========== Handler 层 ==========
	appHandler := handler.NewApplicationHandler(appService)
	schoolHandler := handler.NewSchoolHandler(schoolService)
	ssoHandler := handler.NewSSOHandler(ssoService)

	// ========== 定时任务注册 ==========
	scheduler := svc.NewSchoolScheduler(schoolRepo, notifyRepo, sessionKicker, adminContactProvider)
	cronpkg.AddTask(cronpkg.CronSchoolExpireBuffer, "学校到期转缓冲期", scheduler.RunExpireToBuffering)
	cronpkg.AddTask(cronpkg.CronSchoolExpiryCheck, "学校到期提醒检查", scheduler.RunExpiryReminder)
	cronpkg.AddTask(cronpkg.CronSchoolBufferFreeze, "学校缓冲期转冻结", scheduler.RunBufferingToFrozen)

	return &router.SchoolHandlers{
		ApplicationHandler: appHandler,
		SchoolHandler:      schoolHandler,
		SSOHandler:         ssoHandler,
	}
}

// ========== AdminCreator Adapter ==========

// adminCreatorAdapter 跨模块 adapter：创建首个校管账号
// 实现 school.AdminCreator 接口，内部调用模块01的 repo 层
type adminCreatorAdapter struct {
	userRepo authrepo.UserRepository
	roleRepo authrepo.RoleRepository
}

// CreateSchoolAdmin 在事务中创建首个校管账号
// 1. 生成随机密码 2. 创建用户 3. 分配 teacher + school_admin 角色
// 返回 userID 和明文密码（用于短信通知）
func (a *adminCreatorAdapter) CreateSchoolAdmin(ctx context.Context, schoolID int64, phone, name string, createdBy int64) (int64, string, error) {
	tx, ok := database.TxFromContext(ctx)
	if !ok {
		return 0, "", errors.New("创建校管账号缺少事务上下文")
	}

	txUserRepo := authrepo.NewUserRepository(tx)
	txRoleRepo := authrepo.NewRoleRepository(tx)

	// 联系人手机号若已被现有账号占用，需要按文档提示超管人工处理。
	if existing, err := txUserRepo.GetByPhone(ctx, phone); err == nil && existing != nil {
		return 0, "", errcode.ErrDuplicatePhone.WithMessage("联系人手机号已存在，请超管手动处理")
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, "", err
	}

	// 1. 生成随机密码
	password, err := crypto.GenerateRandomPassword(12)
	if err != nil {
		return 0, "", err
	}

	// 2. 哈希密码
	passwordHash, err := crypto.HashPassword(password)
	if err != nil {
		return 0, "", err
	}

	// 3. 创建用户
	userID := snowflake.Generate()
	user := &entity.User{
		ID:            userID,
		Phone:         phone,
		PasswordHash:  passwordHash,
		Name:          name,
		SchoolID:      schoolID,
		Status:        enum.UserStatusActive,
		IsFirstLogin:  true,
		IsSchoolAdmin: true,
		CreatedBy:     &createdBy,
	}
	if err := tx.WithContext(ctx).Create(user).Error; err != nil {
		return 0, "", err
	}

	// 4. 创建用户扩展信息
	profile := &entity.UserProfile{
		ID:     snowflake.Generate(),
		UserID: userID,
	}
	if err := tx.WithContext(ctx).Create(profile).Error; err != nil {
		return 0, "", err
	}

	// 5. 分配角色：teacher + school_admin
	roleCodes := []string{enum.RoleTeacher, enum.RoleSchoolAdmin}
	for _, code := range roleCodes {
		role, err := txRoleRepo.GetByCode(ctx, code)
		if err != nil {
			logger.L.Error("查找角色失败", zap.String("code", code), zap.Error(err))
			return 0, "", err
		}
		userRole := &entity.UserRole{
			ID:     snowflake.Generate(),
			UserID: userID,
			RoleID: role.ID,
		}
		if err := tx.WithContext(ctx).Create(userRole).Error; err != nil {
			return 0, "", err
		}
	}

	return userID, password, nil
}

// ========== SessionKicker Adapter ==========

// sessionKickerAdapter 跨模块 adapter：踢出学校所有用户的 Session
// 实现 school.SessionKicker 接口
type sessionKickerAdapter struct {
	userRepo authrepo.UserRepository
}

// KickSchoolUsers 踢出指定学校的所有用户 Session
// 1. 批量刷新该校用户的 token_valid_after，使历史 Access/Refresh Token 立即失效
// 2. 删除 Redis 中的 session:{user_id} 键，清空在线会话
func (a *sessionKickerAdapter) KickSchoolUsers(ctx context.Context, schoolID int64) error {
	now := time.Now()
	if err := a.userRepo.BatchUpdateTokenValidAfterBySchool(ctx, schoolID, now); err != nil {
		logger.L.Error("批量更新学校用户Token生效时间失败", zap.Int64("school_id", schoolID), zap.Error(err))
		return err
	}

	userIDs, err := a.userRepo.GetIDsBySchoolID(ctx, schoolID)
	if err != nil {
		logger.L.Error("查询学校用户ID失败", zap.Int64("school_id", schoolID), zap.Error(err))
		return err
	}

	for _, uid := range userIDs {
		_ = tokenstate.SetTokenValidAfter(ctx, uid, now)
		sessionKey := cache.KeySession + strconv.FormatInt(uid, 10)
		_ = cache.Del(ctx, sessionKey)
	}

	logger.L.Info("已踢出学校所有用户Session",
		zap.Int64("school_id", schoolID),
		zap.Int("user_count", len(userIDs)),
	)
	return nil
}

// schoolUserLifecycleAdapter 跨模块 adapter：学校维度用户软删与恢复。
// 具体写库逻辑由模块01 repository 执行，模块02 service 只通过接口协调。
type schoolUserLifecycleAdapter struct{}

// SoftDeleteSchoolUsers 软删除学校下全部用户
func (a *schoolUserLifecycleAdapter) SoftDeleteSchoolUsers(ctx context.Context, schoolID int64) error {
	tx, ok := database.TxFromContext(ctx)
	if !ok {
		return errors.New("软删除学校用户缺少事务上下文")
	}
	return authrepo.NewUserRepository(tx).SoftDeleteBySchoolID(ctx, schoolID)
}

// RestoreSchoolUsers 恢复学校下全部用户
func (a *schoolUserLifecycleAdapter) RestoreSchoolUsers(ctx context.Context, schoolID int64) error {
	tx, ok := database.TxFromContext(ctx)
	if !ok {
		return errors.New("恢复学校用户缺少事务上下文")
	}
	return authrepo.NewUserRepository(tx).RestoreBySchoolID(ctx, schoolID)
}

// schoolAdminContactProviderAdapter 跨模块 adapter：提供学校管理员手机号列表。
// 学校模块的到期提醒只依赖接口，不直接跨模块读 users 表。
type schoolAdminContactProviderAdapter struct {
	userRepo authrepo.UserRepository
}

// ListAdminPhones 返回学校管理员手机号列表。
func (a *schoolAdminContactProviderAdapter) ListAdminPhones(ctx context.Context, schoolID int64) ([]string, error) {
	return a.userRepo.ListAdminPhonesBySchoolID(ctx, schoolID)
}

// ========== SchoolNameQuerier Adapter ==========

// SchoolNameQuerier 跨模块接口：查询学校名称
// 由模块01的 service 层使用，注入模块02的 repo
type SchoolNameQuerier interface {
	GetSchoolName(ctx context.Context, schoolID int64) string
}

// schoolNameQuerierAdapter 学校名称查询 adapter
type schoolNameQuerierAdapter struct {
	schoolRepo schoolrepo.SchoolRepository
}

// newSchoolNameQuerier 创建学校名称查询 adapter
func newSchoolNameQuerier(schoolRepo schoolrepo.SchoolRepository) SchoolNameQuerier {
	return &schoolNameQuerierAdapter{schoolRepo: schoolRepo}
}

// GetSchoolName 根据学校ID查询学校名称
// 查询失败或学校不存在时返回空字符串，不影响主流程
func (a *schoolNameQuerierAdapter) GetSchoolName(ctx context.Context, schoolID int64) string {
	if schoolID == 0 {
		return ""
	}
	school, err := a.schoolRepo.GetByID(ctx, schoolID)
	if err != nil {
		return ""
	}
	return school.Name
}
