// init_notification.go
// 模块07 — 通知与消息：依赖注入初始化。
// 按 repository → service → handler 顺序组装模块07依赖，并向其他模块暴露最小通知分发接口。

package main

import (
	"go.uber.org/zap"

	notificationhandler "github.com/lenschain/backend/internal/handler/notification"
	cronpkg "github.com/lenschain/backend/internal/pkg/cron"
	"github.com/lenschain/backend/internal/pkg/database"
	"github.com/lenschain/backend/internal/pkg/logger"
	notificationrepo "github.com/lenschain/backend/internal/repository/notification"
	"github.com/lenschain/backend/internal/router"
	notificationsvc "github.com/lenschain/backend/internal/service/notification"
)

// initNotificationModule 初始化模块07。
// 返回模块07路由所需的 Handler 集合以及供其他模块复用的事件分发接口。
func initNotificationModule() (*router.NotificationHandlers, notificationsvc.EventDispatcher) {
	db := database.Get()

	notificationRepo := notificationrepo.NewNotificationRepository(db)
	announcementRepo := notificationrepo.NewSystemAnnouncementRepository(db)
	announcementReadRepo := notificationrepo.NewAnnouncementReadStatusRepository(db)
	templateRepo := notificationrepo.NewNotificationTemplateRepository(db)
	preferenceRepo := notificationrepo.NewUserNotificationPreferenceRepository(db)
	sourceRepo := notificationrepo.NewNotificationSourceRepository(db)

	service := notificationsvc.NewService(
		db,
		notificationRepo,
		announcementRepo,
		announcementReadRepo,
		templateRepo,
		preferenceRepo,
		sourceRepo,
	)
	announcer, _ := service.(notificationsvc.AnnouncementBroadcaster)
	scheduler := notificationsvc.NewScheduler(service, service, announcer, announcementRepo, notificationRepo, sourceRepo)
	cronpkg.AddTask(cronpkg.CronNotificationScan, "通知定时扫描", scheduler.RunScan)
	cronpkg.AddTask(cronpkg.CronNotificationCleanup, "通知历史清理与未读对账", scheduler.RunCleanup)
	if err := notificationsvc.RegisterInternalEventConsumer(service); err != nil {
		logger.L.Error("注册模块07内部通知事件消费者失败", zap.Error(err))
	} else {
		notificationsvc.EnableAsyncDispatch(service, nil)
	}

	return &router.NotificationHandlers{
		NotificationHandler: notificationhandler.NewNotificationHandler(service),
	}, service
}
