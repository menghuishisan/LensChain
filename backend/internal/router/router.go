// router.go
// 总路由入口
// 初始化 Gin Engine，挂载全局中间件，注册各模块路由组
// 路由层只负责路径注册和中间件绑定，不含任何处理逻辑

package router

import (
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	govalidator "github.com/go-playground/validator/v10"

	authhandler "github.com/lenschain/backend/internal/handler/auth"
	ctfhandler "github.com/lenschain/backend/internal/handler/ctf"
	experimenthandler "github.com/lenschain/backend/internal/handler/experiment"
	gradehandler "github.com/lenschain/backend/internal/handler/grade"
	notificationhandler "github.com/lenschain/backend/internal/handler/notification"
	schoolhandler "github.com/lenschain/backend/internal/handler/school"
	systemhandler "github.com/lenschain/backend/internal/handler/system"
	"github.com/lenschain/backend/internal/middleware"
	appvalidator "github.com/lenschain/backend/internal/pkg/validator"
)

// AuthHandlers 模块01（用户与认证）的 Handler 集合
type AuthHandlers struct {
	AuthHandler     *authhandler.AuthHandler
	UserHandler     *authhandler.UserHandler
	SecurityHandler *authhandler.SecurityHandler
}

// SchoolHandlers 模块02（学校与租户管理）的 Handler 集合
type SchoolHandlers struct {
	ApplicationHandler *schoolhandler.ApplicationHandler
	SchoolHandler      *schoolhandler.SchoolHandler
	SSOHandler         *schoolhandler.SSOHandler
}

// ExperimentHandlers 模块04（实验环境）的 Handler 集合
type ExperimentHandlers struct {
	TemplateHandler *experimenthandler.TemplateHandler
	InstanceHandler *experimenthandler.InstanceHandler
}

// CTFHandlers 模块05（CTF竞赛）的 Handler 集合。
type CTFHandlers struct {
	CompetitionHandler *ctfhandler.CompetitionHandler
	BattleHandler      *ctfhandler.BattleHandler
	EnvironmentHandler *ctfhandler.EnvironmentHandler
	RealtimeHandler    *ctfhandler.RealtimeHandler
}

// GradeHandlers 模块06（评测与成绩）的 Handler 集合。
type GradeHandlers struct {
	GradeHandler *gradehandler.GradeHandler
}

// NotificationHandlers 模块07（通知与消息）的 Handler 集合。
type NotificationHandlers struct {
	NotificationHandler *notificationhandler.NotificationHandler
}

// SystemHandlers 模块08（系统管理与监控）的 Handler 集合。
type SystemHandlers struct {
	SystemHandler *systemhandler.SystemHandler
}

// Handlers 所有模块的 Handler 实例集合
// 由 main.go 初始化后传入路由注册
// 按模块嵌套，每个模块独立结构体，避免扁平化膨胀
type Handlers struct {
	Auth         *AuthHandlers
	School       *SchoolHandlers
	Course       *CourseHandlers       // 模块03 — 课程与教学
	Experiment   *ExperimentHandlers   // 模块04 — 实验环境
	CTF          *CTFHandlers          // 模块05 — CTF竞赛
	Grade        *GradeHandlers        // 模块06 — 评测与成绩
	Notification *NotificationHandlers // 模块07 — 通知与消息
	System       *SystemHandlers       // 模块08 — 系统管理与监控
}

// Setup 初始化路由
// 返回配置好的 Gin Engine 实例
func Setup(mode string, h *Handlers) *gin.Engine {
	gin.SetMode(mode)
	registerCustomValidators()

	r := gin.New()

	// 全局中间件
	r.Use(middleware.Recovery())      // 异常恢复（必须最先注册）
	r.Use(middleware.CORS())          // 跨域处理
	r.Use(middleware.RequestLogger()) // 请求日志
	r.Use(middleware.RateLimit())     // 全局限流

	// API v1 路由组
	v1 := r.Group("/api/v1")

	// 注册各模块路由
	if h != nil && h.Auth != nil {
		RegisterAuthRoutes(v1, h.Auth.AuthHandler, h.Auth.UserHandler, h.Auth.SecurityHandler) // 模块01 — 用户与认证
	}
	if h != nil && h.School != nil {
		RegisterSchoolRoutes(v1, h.School.ApplicationHandler, h.School.SchoolHandler, h.School.SSOHandler) // 模块02 — 学校与租户管理
	}
	if h != nil && h.Course != nil {
		RegisterCourseRoutes(v1, h.Course) // 模块03 — 课程与教学
	}
	if h != nil && h.Experiment != nil {
		RegisterExperimentRoutes(v1, h.Experiment) // 模块04 — 实验环境
	}
	if h != nil && h.CTF != nil {
		RegisterCTFRoutes(v1, h.CTF) // 模块05 — CTF竞赛
	}
	if h != nil && h.Grade != nil {
		RegisterGradeRoutes(v1, h.Grade) // 模块06 — 评测与成绩
	}
	if h != nil && h.Notification != nil {
		RegisterNotificationRoutes(v1, h.Notification) // 模块07 — 通知与消息
	}
	if h != nil && h.System != nil {
		RegisterSystemRoutes(v1, h.System) // 模块08 — 系统管理与监控
	}

	// 内部接口路由组（模块间调用，不经过JWT鉴权）
	internal := r.Group("/internal")
	internal.Use(middleware.InternalOnly())
	RegisterInternalRoutes(internal, h)
	internalV1 := r.Group("/api/v1/notifications/internal")
	internalV1.Use(middleware.InternalOnly())
	RegisterInternalRoutes(internalV1, h)

	// WebSocket 路由
	RegisterWebSocketRoutes(r, h)

	// 健康检查（不经过鉴权）
	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})
	r.GET("/readyz", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ready"})
	})
	r.GET("/startupz", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "started"})
	})

	return r
}

// registerCustomValidators 在 Gin 默认校验器上注册项目级自定义规则。
func registerCustomValidators() {
	engine, ok := binding.Validator.Engine().(*govalidator.Validate)
	if !ok || engine == nil {
		return
	}
	_ = appvalidator.RegisterCustomValidators(engine)
}

// RegisterInternalRoutes 注册内部接口路由
// 模块07 提供 POST /internal/send-event 供其他模块发送通知
func RegisterInternalRoutes(rg *gin.RouterGroup, h *Handlers) {
	if h != nil && h.Notification != nil && h.Notification.NotificationHandler != nil {
		rg.POST("/send-event", h.Notification.NotificationHandler.SendInternalEvent) // 内部通知事件接口
	}
}

// RegisterWebSocketRoutes 注册 WebSocket 路由
// WebSocket 连接需要 JWT 鉴权，但不走标准中间件链
func RegisterWebSocketRoutes(r *gin.Engine, h *Handlers) {
	ws := r.Group("/api/v1/ws")
	ws.Use(middleware.JWTAuth())
	{
		// 模块04 — 实验环境 WebSocket
		if h != nil && h.Experiment != nil && h.Experiment.InstanceHandler != nil {
			ws.GET("/experiment-instances/:id", h.Experiment.InstanceHandler.ServeInstanceWS)            // 实验实例状态推送
			ws.GET("/experiment-groups/:id/chat", h.Experiment.InstanceHandler.ServeGroupChatWS)         // 组内实时消息
			ws.GET("/courses/:id/experiment-monitor", h.Experiment.InstanceHandler.ServeCourseMonitorWS) // 教师监控面板实时推送
			ws.GET("/sim-engine/:session_id", h.Experiment.InstanceHandler.ServeSimEngineWS)             // SimEngine 仿真数据通道
		}

		// 模块05 — CTF竞赛 WebSocket
		if h != nil && h.CTF != nil && h.CTF.RealtimeHandler != nil {
			ws.GET("/ctf", h.CTF.RealtimeHandler.ServeWS) // CTF实时通信（排行榜、公告、回合、攻击）
		}

		// 模块07 — 通知与消息 WebSocket
		if h != nil && h.Notification != nil && h.Notification.NotificationHandler != nil {
			ws.GET("/notifications", h.Notification.NotificationHandler.ServeWS) // 通知推送通道
		}
	}
}
