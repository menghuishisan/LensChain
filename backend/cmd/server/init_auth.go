// init_auth.go
// 模块01 — 用户与认证：依赖注入初始化
// 按照 repository → service → handler 的顺序组装模块01的依赖
// 每个模块独立一个 init 文件，避免 main.go 膨胀为单体

package main

import (
	"context"
	"encoding/json"
	"time"

	"gorm.io/gorm"

	handler "github.com/lenschain/backend/internal/handler/auth"
	"github.com/lenschain/backend/internal/model/dto"
	"github.com/lenschain/backend/internal/model/enum"
	"github.com/lenschain/backend/internal/pkg/database"
	"github.com/lenschain/backend/internal/pkg/errcode"
	authrepo "github.com/lenschain/backend/internal/repository/auth"
	courserepo "github.com/lenschain/backend/internal/repository/course"
	schoolrepo "github.com/lenschain/backend/internal/repository/school"
	"github.com/lenschain/backend/internal/router"
	svc "github.com/lenschain/backend/internal/service/auth"
)

// initAuthModule 初始化模块01（用户与认证）的 Handler
// 按照 repository → service → handler 的顺序组装依赖
func initAuthModule() *router.AuthHandlers {
	db := database.Get()

	// Repository 层
	userRepo := authrepo.NewUserRepository(db)
	profileRepo := authrepo.NewProfileRepository(db)
	roleRepo := authrepo.NewRoleRepository(db)
	loginLogRepo := authrepo.NewLoginLogRepository(db)
	opLogRepo := authrepo.NewOperationLogRepository(db)
	ssoBindingRepo := authrepo.NewSSOBindingRepository(db)

	// 跨模块依赖：学校名称查询（模块02 → 模块01）
	schoolRepo := schoolrepo.NewSchoolRepository(db)
	ssoConfigRepo := schoolrepo.NewSSOConfigRepository(db)
	courseRepo := courserepo.NewCourseRepository(db)
	progressRepo := courserepo.NewProgressRepository(db)
	schoolNameQuerier := &profileContextAdapter{
		schoolNameQuerierAdapter: schoolNameQuerierAdapter{schoolRepo: schoolRepo},
		courseRepo:               courseRepo,
		progressRepo:             progressRepo,
	}
	schoolStatusChecker := &schoolStatusCheckerAdapter{schoolRepo: schoolRepo}
	schoolSSOQuerier := &schoolSSOQuerierAdapter{ssoConfigRepo: ssoConfigRepo}

	// Service 层
	// authService: 认证流程（登录/登出/Token/改密），需要 loginLogRepo 记录登录日志
	authService := svc.NewAuthService(
		db, userRepo, roleRepo, loginLogRepo, ssoBindingRepo,
		schoolNameQuerier, schoolStatusChecker, schoolSSOQuerier,
	)
	// userService: 用户管理 CRUD，操作日志通过 pkg/audit 直接写入，不再需要 opLogRepo
	userService := svc.NewUserService(db, userRepo, profileRepo, roleRepo)
	// profileService: 个人中心
	profileService := svc.NewProfileService(db, userRepo, profileRepo, roleRepo, schoolNameQuerier, schoolNameQuerier)
	// importService: 用户导入，操作日志通过 pkg/audit 直接写入
	importService := svc.NewImportService(db, userRepo, profileRepo, roleRepo)
	// securityService: 安全策略与日志查询，需要 loginLogRepo/opLogRepo 做查询
	securityService := svc.NewSecurityService(loginLogRepo, opLogRepo, userRepo)

	// Handler 层
	authHandler := handler.NewAuthHandler(authService)
	userHandler := handler.NewUserHandler(userService, profileService, importService)
	securityHandler := handler.NewSecurityHandler(securityService)

	return &router.AuthHandlers{
		AuthHandler:     authHandler,
		UserHandler:     userHandler,
		SecurityHandler: securityHandler,
	}
}

// profileContextAdapter 个人中心跨模块 adapter
// 同时提供学校名称与学习概览聚合能力，避免模块01直接依赖模块03实现
type profileContextAdapter struct {
	schoolNameQuerierAdapter
	courseRepo   courserepo.CourseRepository
	progressRepo courserepo.ProgressRepository
}

// GetLearningOverview 获取学习概览
func (a *profileContextAdapter) GetLearningOverview(ctx context.Context, userID int64) (*dto.LearningOverview, error) {
	courses, total, err := a.courseRepo.ListByStudentID(ctx, userID, &courserepo.StudentCourseListParams{
		Page: 1, PageSize: 100,
	})
	if err != nil {
		return nil, err
	}

	var totalStudySeconds int
	for _, course := range courses {
		duration, err := a.progressRepo.SumStudyDurationByStudent(ctx, userID, course.ID)
		if err != nil {
			return nil, err
		}
		totalStudySeconds += duration
	}

	return &dto.LearningOverview{
		CourseCount:      int(total),
		ExperimentCount:  0,
		CompetitionCount: 0,
		TotalStudyHours:  float64(totalStudySeconds) / 3600,
	}, nil
}

// schoolStatusCheckerAdapter 跨模块 adapter：校验学校登录状态
type schoolStatusCheckerAdapter struct {
	schoolRepo schoolrepo.SchoolRepository
}

// CheckLoginAllowed 校验学校是否允许登录
func (a *schoolStatusCheckerAdapter) CheckLoginAllowed(ctx context.Context, schoolID int64) error {
	if schoolID == 0 {
		return nil
	}

	school, err := a.schoolRepo.GetByID(ctx, schoolID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return errcode.ErrForbidden.WithMessage("学校不存在或已不可用")
		}
		return errcode.ErrInternal.WithMessage("查询学校状态失败")
	}

	// 登录准入校验与模块02的学校生命周期规则保持一致：
	// 激活学校允许登录，缓冲期学校仍可登录，冻结/注销学校禁止登录。
	switch school.Status {
	case enum.SchoolStatusFrozen:
		return errcode.ErrSchoolExpired.WithMessage("学校授权已过期，请联系管理员")
	case enum.SchoolStatusCancelled:
		return errcode.ErrForbidden.WithMessage("学校已注销，无法登录")
	case enum.SchoolStatusPending, enum.SchoolStatusRejected:
		return errcode.ErrForbidden.WithMessage("学校状态异常，暂不可登录")
	case enum.SchoolStatusActive:
		if school.LicenseEndAt != nil && time.Now().After(*school.LicenseEndAt) {
			return errcode.ErrSchoolExpired.WithMessage("学校授权已过期，请联系管理员")
		}
	}

	return nil
}

// schoolSSOQuerierAdapter 跨模块 adapter：查询学校SSO配置
type schoolSSOQuerierAdapter struct {
	ssoConfigRepo schoolrepo.SSOConfigRepository
}

// GetSchoolSSOConfig 获取学校SSO配置
func (a *schoolSSOQuerierAdapter) GetSchoolSSOConfig(ctx context.Context, schoolID int64) (*svc.SchoolSSOConfig, error) {
	config, err := a.ssoConfigRepo.GetBySchoolID(ctx, schoolID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, err
		}
		return nil, errcode.ErrInternal.WithMessage("查询学校SSO配置失败")
	}

	// 这里统一解析模块02保存的 JSON 配置，认证层只消费结构化配置。
	configMap := make(map[string]interface{})
	if err := json.Unmarshal([]byte(config.Config), &configMap); err != nil {
		return nil, errcode.ErrInternal.WithMessage("解析学校SSO配置失败")
	}

	return &svc.SchoolSSOConfig{
		SchoolID:  schoolID,
		Provider:  config.Provider,
		IsEnabled: config.IsEnabled,
		IsTested:  config.IsTested,
		Config:    configMap,
	}, nil
}
