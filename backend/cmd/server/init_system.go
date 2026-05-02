// init_system.go
// 模块08 — 系统管理与监控：依赖注入初始化。
// 按 repository → service → handler 顺序组装模块08依赖，并复用模块07通知分发能力。

package main

import (
	"context"

	"github.com/lenschain/backend/internal/config"
	systemhandler "github.com/lenschain/backend/internal/handler/system"
	"github.com/lenschain/backend/internal/model/dto"
	cronpkg "github.com/lenschain/backend/internal/pkg/cron"
	"github.com/lenschain/backend/internal/pkg/database"
	"github.com/lenschain/backend/internal/pkg/logger"
	authrepo "github.com/lenschain/backend/internal/repository/auth"
	schoolrepo "github.com/lenschain/backend/internal/repository/school"
	systemrepo "github.com/lenschain/backend/internal/repository/system"
	"github.com/lenschain/backend/internal/router"
	authsvc "github.com/lenschain/backend/internal/service/auth"
	experimentsvc "github.com/lenschain/backend/internal/service/experiment"
	notificationsvc "github.com/lenschain/backend/internal/service/notification"
	systemsvc "github.com/lenschain/backend/internal/service/system"
	"go.uber.org/zap"
)

// initSystemModule 初始化模块08。
func initSystemModule(cfg *config.Config, notificationDispatcher notificationsvc.EventDispatcher) *router.SystemHandlers {
	db := database.Get()

	auditRepo := systemrepo.NewAuditRepository(db)
	configRepo := systemrepo.NewSystemConfigRepository(db)
	configChangeLogRepo := systemrepo.NewConfigChangeLogRepository(db)
	alertRuleRepo := systemrepo.NewAlertRuleRepository(db)
	alertEventRepo := systemrepo.NewAlertEventRepository(db)
	statRepo := systemrepo.NewPlatformStatisticRepository(db)
	backupRepo := systemrepo.NewBackupRecordRepository(db)
	userRepo := authrepo.NewUserRepository(db)
	loginLogRepo := authrepo.NewLoginLogRepository(db)
	opLogRepo := authrepo.NewOperationLogRepository(db)
	schoolRepo := schoolrepo.NewSchoolRepository(db)

	securitySvc := authsvc.NewSecurityService(loginLogRepo, opLogRepo, userRepo, schoolRepo)
	securitySyncer := &systemSecuritySyncAdapter{securityService: securitySvc}

	var clusterProvider systemsvc.ClusterStatusProvider
	if k8sSvc, err := experimentsvc.NewK8sService(cfg.K8s); err == nil {
		clusterProvider = &systemClusterProviderAdapter{k8sSvc: k8sSvc}
	}

	service := systemsvc.NewService(
		cfg,
		auditRepo,
		configRepo,
		configChangeLogRepo,
		alertRuleRepo,
		alertEventRepo,
		statRepo,
		backupRepo,
		userRepo,
		notificationDispatcher,
		securitySyncer,
		clusterProvider,
	)
	scheduler := systemsvc.NewScheduler(service)
	if scheduler != nil {
		systemsvc.BindBackupScheduleSyncer(service, scheduler)
		cronpkg.AddTask(cronpkg.CronAlertThreshold, "模块08阈值告警检测", scheduler.RunThresholdChecks)
		cronpkg.AddTask(cronpkg.CronAlertEvent, "模块08事件告警检测", scheduler.RunEventChecks)
		cronpkg.AddTask(cronpkg.CronHealthCheck, "模块08服务健康巡检", scheduler.RunHealthChecks)
		cronpkg.AddTask(cronpkg.CronStatsAggregation, "模块08平台统计聚合", scheduler.RunStatsAggregation)
		cronpkg.AddTask(cronpkg.CronBackupCleanup, "模块08备份清理", scheduler.RunBackupCleanup)
		cronpkg.AddTask(cronpkg.CronStatsDataCleanup, "模块08统计清理", scheduler.RunStatsDataCleanup)
		if err := scheduler.SyncAutoBackupTask(context.Background()); err != nil {
			logger.L.Error("初始化模块08自动备份任务失败", zap.Error(err))
		}
	}

	return &router.SystemHandlers{
		SystemHandler: systemhandler.NewSystemHandler(service),
	}
}

// systemSecuritySyncAdapter 将模块08安全配置同步到模块01运行时安全策略。
type systemSecuritySyncAdapter struct {
	securityService authsvc.SecurityService
}

// SyncRuntimeSecurityConfig 同步模块08安全配置到模块01。
func (a *systemSecuritySyncAdapter) SyncRuntimeSecurityConfig(ctx context.Context, config systemsvc.RuntimeSecurityConfig) error {
	if a == nil || a.securityService == nil {
		return nil
	}
	accessMinutes := config.SessionTimeoutHours * 60
	return a.securityService.UpdateSecurityPolicy(ctx, &dto.UpdateSecurityPolicyReq{
		LoginFailMaxCount:          intPtrValue(config.MaxLoginFailCount),
		LoginLockDurationMinutes:   intPtrValue(config.LockDurationMinutes),
		AccessTokenExpireMinutes:   intPtrValue(accessMinutes),
		PasswordMinLength:          intPtrValue(config.PasswordMinLength),
		PasswordRequireUppercase:   boolPtrValue(config.PasswordRequireUppercase),
		PasswordRequireLowercase:   boolPtrValue(config.PasswordRequireLowercase),
		PasswordRequireDigit:       boolPtrValue(config.PasswordRequireDigit),
		PasswordRequireSpecialChar: boolPtrValue(config.PasswordRequireSpecialChar),
	})
}

// systemClusterProviderAdapter 适配模块04 K8s 服务给模块08仪表盘与告警使用。
type systemClusterProviderAdapter struct {
	k8sSvc experimentsvc.K8sService
}

// GetClusterStatus 获取当前 K8s 集群摘要。
func (a *systemClusterProviderAdapter) GetClusterStatus(ctx context.Context) (*systemsvc.ClusterStatusSnapshot, error) {
	if a == nil || a.k8sSvc == nil {
		return nil, nil
	}
	status, err := a.k8sSvc.GetClusterStatus(ctx)
	if err != nil {
		return nil, err
	}
	return &systemsvc.ClusterStatusSnapshot{
		TotalNodes:   status.TotalNodes,
		ReadyNodes:   status.ReadyNodes,
		TotalCPU:     status.TotalCPU,
		UsedCPU:      status.UsedCPU,
		TotalMemory:  status.TotalMemory,
		UsedMemory:   status.UsedMemory,
		TotalStorage: status.TotalStorage,
		UsedStorage:  status.UsedStorage,
		TotalPods:    status.TotalPods,
		RunningPods:  status.RunningPods,
		PendingPods:  status.PendingPods,
		FailedPods:   status.FailedPods,
		Namespaces:   status.Namespaces,
	}, nil
}

// intPtrValue 返回 int 指针，便于构造模块01安全策略更新请求。
func intPtrValue(value int) *int {
	return &value
}

// boolPtrValue 返回 bool 指针，便于构造模块01安全策略更新请求。
func boolPtrValue(value bool) *bool {
	return &value
}
