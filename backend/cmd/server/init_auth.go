// init_auth.go
// 模块01 — 用户与认证：依赖注入初始化
// 按照 repository → service → handler 的顺序组装模块01的依赖
// 每个模块独立一个 init 文件，避免 main.go 膨胀为单体

package main

import (
	handler "github.com/lenschain/backend/internal/handler/auth"
	"github.com/lenschain/backend/internal/pkg/database"
	authrepo "github.com/lenschain/backend/internal/repository/auth"
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

	// 跨模块依赖：学校名称查询（模块02 → 模块01）
	schoolRepo := schoolrepo.NewSchoolRepository(db)
	schoolNameQuerier := newSchoolNameQuerier(schoolRepo)

	// Service 层
	// authService: 认证流程（登录/登出/Token/改密），需要 loginLogRepo 记录登录日志
	authService := svc.NewAuthService(db, userRepo, roleRepo, loginLogRepo, schoolNameQuerier)
	// userService: 用户管理 CRUD，操作日志通过 pkg/audit 直接写入，不再需要 opLogRepo
	userService := svc.NewUserService(db, userRepo, profileRepo, roleRepo)
	// profileService: 个人中心
	profileService := svc.NewProfileService(db, userRepo, profileRepo, roleRepo, schoolNameQuerier)
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
