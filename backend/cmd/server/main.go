// main.go
// 链镜平台后端服务入口
// 负责初始化所有基础设施组件、组装业务模块依赖、启动 HTTP 服务、处理优雅退出
// 初始化顺序：配置 → 日志 → 雪花ID → 数据库 → Redis → MinIO → NATS → WebSocket → 定时任务 → 业务模块 → 路由 → HTTP服务

package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"

	"github.com/lenschain/backend/internal/config"
	"github.com/lenschain/backend/internal/pkg/cache"
	"github.com/lenschain/backend/internal/pkg/cron"
	"github.com/lenschain/backend/internal/pkg/database"
	"github.com/lenschain/backend/internal/pkg/logger"
	"github.com/lenschain/backend/internal/pkg/mq"
	"github.com/lenschain/backend/internal/pkg/pdf"
	"github.com/lenschain/backend/internal/pkg/sms"
	"github.com/lenschain/backend/internal/pkg/snowflake"
	"github.com/lenschain/backend/internal/pkg/storage"
	"github.com/lenschain/backend/internal/pkg/ws"
	"github.com/lenschain/backend/internal/router"
)

func main() {
	// 1. 加载配置
	configPath := os.Getenv("LENSCHAIN_CONFIG")
	cfg, err := config.Load(configPath)
	if err != nil {
		fmt.Printf("加载配置失败: %v\n", err)
		os.Exit(1)
	}

	// 2. 初始化日志
	if err := logger.Init(&cfg.Log); err != nil {
		fmt.Printf("初始化日志失败: %v\n", err)
		os.Exit(1)
	}
	defer logger.L.Sync()

	logger.L.Info("链镜平台后端服务启动中...")

	// 3. 初始化雪花ID生成器
	if err := snowflake.Init(cfg.Snowflake.NodeID); err != nil {
		logger.L.Fatal("初始化雪花ID生成器失败", zap.Error(err))
	}

	// 4. 初始化数据库
	if err := database.Init(&cfg.Database); err != nil {
		logger.L.Fatal("初始化数据库失败", zap.Error(err))
	}
	defer func() {
		if err := database.Close(); err != nil {
			logger.L.Error("关闭数据库连接失败", zap.Error(err))
		}
	}()

	// 5. 初始化 Redis
	if err := cache.Init(&cfg.Redis); err != nil {
		logger.L.Fatal("初始化Redis失败", zap.Error(err))
	}
	defer func() {
		if err := cache.Close(); err != nil {
			logger.L.Error("关闭Redis连接失败", zap.Error(err))
		}
	}()

	// 6. 初始化 MinIO
	if err := storage.Init(&cfg.MinIO); err != nil {
		logger.L.Fatal("初始化MinIO失败", zap.Error(err))
	}

	// 7. 初始化 NATS
	if err := mq.Init(&cfg.NATS); err != nil {
		logger.L.Fatal("初始化NATS失败", zap.Error(err))
	}
	defer mq.Close()

	// 8. 初始化 SMS
	if err := sms.Init(&cfg.SMS); err != nil {
		logger.L.Fatal("初始化SMS失败", zap.Error(err))
	}

	// 9. 初始化 WebSocket 管理器
	ws.Init()

	// 10. 初始化 PDF 生成器
	pdf.Init()

	// 11. 初始化定时任务调度器
	cron.Init()
	cron.Start()
	defer cron.Stop()

	// ========== 12. 组装业务模块依赖（DI） ==========
	// 每个模块独立一个 init_xxx.go 文件，避免 main.go 膨胀
	handlers := &router.Handlers{
		Auth:   initAuthModule(),   // 模块01 — 用户与认证（init_auth.go）
		School: initSchoolModule(), // 模块02 — 学校与租户管理（init_school.go）
		Course: initCourseModule(), // 模块03 — 课程与教学（init_course.go）
		// Experiment:   initExperimentModule(),   // 模块04（待实现）
		// CTF:          initCTFModule(),           // 模块05（待实现）
		// Grade:        initGradeModule(),         // 模块06（待实现）
		// Notification: initNotificationModule(), // 模块07（待实现）
		// System:       initSystemModule(),       // 模块08（待实现）
	}

	// 13. 初始化路由
	r := router.Setup(cfg.Server.Mode, handlers)

	// 14. 创建 HTTP 服务器
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	srv := &http.Server{
		Addr:           addr,
		Handler:        r,
		ReadTimeout:    cfg.Server.ReadTimeout,
		WriteTimeout:   cfg.Server.WriteTimeout,
		MaxHeaderBytes: cfg.Server.MaxHeaderBytes,
	}

	// 15. 启动 HTTP 服务（非阻塞）
	go func() {
		logger.L.Info("HTTP 服务已启动",
			zap.String("addr", addr),
			zap.String("mode", cfg.Server.Mode),
		)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.L.Fatal("HTTP 服务启动失败", zap.Error(err))
		}
	}()

	// 16. 优雅退出
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	logger.L.Info("收到退出信号，开始优雅关闭...", zap.String("signal", sig.String()))

	// 给予 10 秒超时时间完成正在处理的请求
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.L.Error("HTTP 服务关闭失败", zap.Error(err))
	}

	logger.L.Info("链镜平台后端服务已停止")
}
