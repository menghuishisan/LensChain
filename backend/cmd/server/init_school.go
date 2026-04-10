// init_school.go
// 模块02 — 学校与租户管理：依赖注入初始化
// 按照 repository → service → handler 的顺序组装模块02的依赖
// 同时创建跨模块 adapter（AdminCreator、SessionKicker、SchoolNameQuerier）
// 注册定时任务到 cron 调度器

package main

import (
	"context"
	"strconv"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/model/enum"
	handler "github.com/lenschain/backend/internal/handler/school"
	"github.com/lenschain/backend/internal/pkg/cache"
	cronpkg "github.com/lenschain/backend/internal/pkg/cron"
	"github.com/lenschain/backend/internal/pkg/crypto"
	"github.com/lenschain/backend/internal/pkg/database"
	"github.com/lenschain/backend/internal/pkg/logger"
	"github.com/lenschain/backend/internal/pkg/snowflake"
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

	adminCreator := &adminCreatorAdapter{roleRepo: roleRepo}
	sessionKicker := &sessionKickerAdapter{userRepo: userRepo}

	// ========== Service 层 ==========
	appService := svc.NewApplicationService(db, appRepo, schoolRepo, notifyRepo, adminCreator)
	schoolService := svc.NewSchoolService(db, schoolRepo, adminCreator, sessionKicker)
	ssoService := svc.NewSSOService(ssoRepo, schoolRepo)

	// ========== Handler 层 ==========
	appHandler := handler.NewApplicationHandler(appService)
	schoolHandler := handler.NewSchoolHandler(schoolService)
	ssoHandler := handler.NewSSOHandler(ssoService)

	// ========== 定时任务注册 ==========
	scheduler := svc.NewSchoolScheduler(schoolRepo, notifyRepo, sessionKicker)
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
	roleRepo authrepo.RoleRepository
}

// CreateSchoolAdmin 在事务中创建首个校管账号
// 1. 生成随机密码 2. 创建用户 3. 分配 teacher + school_admin 角色
// 返回 userID 和明文密码（用于短信通知）
func (a *adminCreatorAdapter) CreateSchoolAdmin(ctx context.Context, tx *gorm.DB, schoolID int64, phone, name string, createdBy int64) (int64, string, error) {
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
		role, err := a.roleRepo.GetByCode(ctx, code)
		if err != nil {
			logger.L.Error("查找角色失败", zap.String("code", code), zap.Error(err))
			continue
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
// 1. 查询该校所有用户 ID
// 2. 批量删除 Redis 中的 session:{user_id} 键
// Access Token 在 30 分钟内自然过期，不单独加黑名单
func (a *sessionKickerAdapter) KickSchoolUsers(ctx context.Context, schoolID int64) error {
	userIDs, err := a.userRepo.GetIDsBySchoolID(ctx, schoolID)
	if err != nil {
		logger.L.Error("查询学校用户ID失败", zap.Int64("school_id", schoolID), zap.Error(err))
		return err
	}

	for _, uid := range userIDs {
		sessionKey := cache.KeySession + strconv.FormatInt(uid, 10)
		_ = cache.Del(ctx, sessionKey)
	}

	logger.L.Info("已踢出学校所有用户Session",
		zap.Int64("school_id", schoolID),
		zap.Int("user_count", len(userIDs)),
	)
	return nil
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
