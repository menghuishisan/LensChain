// scheduler.go
// 模块08 — 系统管理与监控：后台调度器。
// 负责告警检测、健康巡检、统计聚合、自动备份与保留策略清理，统一接入平台 cron 调度器。

package system

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"sync"
	"time"

	"gorm.io/datatypes"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/lenschain/backend/internal/model/dto"
	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/model/enum"
	"github.com/lenschain/backend/internal/pkg/cache"
	cronpkg "github.com/lenschain/backend/internal/pkg/cron"
	"github.com/lenschain/backend/internal/pkg/healthcheck"
	"github.com/lenschain/backend/internal/pkg/logger"
	"github.com/lenschain/backend/internal/pkg/storage"
	authrepo "github.com/lenschain/backend/internal/repository/auth"
	cronlib "github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

// Scheduler 模块08定时任务执行器。
type Scheduler struct {
	service            *service
	autoBackupTaskMu   sync.Mutex
	autoBackupTaskID   cronlib.EntryID
	autoBackupTaskSpec string
}

// NewScheduler 创建模块08调度器。
func NewScheduler(svc Service) *Scheduler {
	impl, ok := svc.(*service)
	if !ok || impl == nil {
		return nil
	}
	return &Scheduler{service: impl}
}

// SyncAutoBackupTask 按当前备份配置刷新自动备份任务注册。
func (s *Scheduler) SyncAutoBackupTask(ctx context.Context) error {
	if s == nil || s.service == nil {
		return nil
	}
	configResp, err := s.service.getBackupConfigNoAuth(ctx)
	if err != nil {
		return err
	}
	spec, err := normalizeAutoBackupCronSpec(configResp)
	if err != nil {
		return err
	}

	s.autoBackupTaskMu.Lock()
	defer s.autoBackupTaskMu.Unlock()

	if s.autoBackupTaskID != 0 {
		cronpkg.RemoveTask(s.autoBackupTaskID)
		s.autoBackupTaskID = 0
		s.autoBackupTaskSpec = ""
	}
	if configResp == nil || !configResp.AutoEnabled {
		return nil
	}
	id, err := cronpkg.AddTask(spec, "模块08自动备份", s.RunAutoBackup)
	if err != nil {
		return err
	}
	s.autoBackupTaskID = id
	s.autoBackupTaskSpec = spec
	return nil
}

// RunThresholdChecks 执行阈值告警检测。
func (s *Scheduler) RunThresholdChecks() {
	if s == nil || s.service == nil {
		return
	}
	ctx := context.Background()
	rules, err := s.service.alertRuleRepo.ListEnabled(ctx)
	if err != nil {
		logger.L.Error("加载阈值告警规则失败", zap.Error(err))
		return
	}
	resources, err := s.service.buildDashboardResources(ctx)
	if err != nil {
		logger.L.Error("计算仪表盘资源快照失败", zap.Error(err))
		return
	}
	for _, rule := range rules {
		if rule == nil || rule.AlertType != enum.AlertTypeThreshold {
			continue
		}
		s.checkThresholdRule(ctx, rule, resources)
	}
}

// RunEventChecks 执行事件告警检测。
func (s *Scheduler) RunEventChecks() {
	if s == nil || s.service == nil {
		return
	}
	ctx := context.Background()
	rules, err := s.service.alertRuleRepo.ListEnabled(ctx)
	if err != nil {
		logger.L.Error("加载事件告警规则失败", zap.Error(err))
		return
	}
	for _, rule := range rules {
		if rule == nil || rule.AlertType != enum.AlertTypeEvent {
			continue
		}
		s.checkEventRule(ctx, rule)
	}
}

// RunHealthChecks 执行服务健康巡检。
func (s *Scheduler) RunHealthChecks() {
	if s == nil || s.service == nil {
		return
	}
	ctx := context.Background()
	results := s.service.collectHealthResults(ctx)
	for _, result := range results {
		payload, _ := json.Marshal(result)
		_ = cache.Set(ctx, cache.KeyServiceHealth+strings.ToLower(result.Name), string(payload), time.Minute)
	}

	rules, err := s.service.alertRuleRepo.ListEnabled(ctx)
	if err != nil {
		logger.L.Error("加载服务状态告警规则失败", zap.Error(err))
		return
	}
	for _, rule := range rules {
		if rule == nil || rule.AlertType != enum.AlertTypeService {
			continue
		}
		s.checkServiceRule(ctx, rule, results)
	}
}

// RunStatsAggregation 执行平台统计聚合。
func (s *Scheduler) RunStatsAggregation() {
	if s == nil || s.service == nil {
		return
	}
	ctx := context.Background()
	statDate := time.Now().UTC().AddDate(0, 0, -1)
	storageUsedGB, err := s.service.calculateStorageUsedGB(ctx)
	if err != nil {
		logger.L.Warn("计算平台存储使用量失败，统计聚合降级为0", zap.Error(err))
		storageUsedGB = 0
	}
	statistic, err := s.service.statRepo.BuildDailyStatistic(ctx, statDate, storageUsedGB)
	if err != nil {
		logger.L.Error("构建平台统计失败", zap.Error(err))
		return
	}
	if err := s.service.statRepo.Upsert(ctx, statistic); err != nil {
		logger.L.Error("写入平台统计失败", zap.Error(err))
	}
}

// RunAutoBackup 执行自动备份。
func (s *Scheduler) RunAutoBackup() {
	if s == nil || s.service == nil {
		return
	}
	ctx := context.Background()
	configResp, err := s.service.getBackupConfigNoAuth(ctx)
	if err != nil {
		logger.L.Error("读取自动备份配置失败", zap.Error(err))
		return
	}
	if configResp == nil || !configResp.AutoEnabled {
		return
	}
	running, _, err := s.service.backupsRunning(ctx)
	if err != nil {
		logger.L.Error("检查备份执行状态失败", zap.Error(err))
		return
	}
	if running {
		return
	}
	now := time.Now().UTC()
	record := &entity.BackupRecord{
		BackupType:   enum.BackupTypeAuto,
		Status:       enum.BackupStatusRunning,
		DatabaseName: s.service.cfg.Database.DBName,
		StartedAt:    now,
	}
	if err := s.service.backupRepo.Create(ctx, record); err != nil {
		logger.L.Error("创建自动备份记录失败", zap.Error(err))
		return
	}
	cronpkg.RunAsync("模块08自动备份执行", func(asyncCtx context.Context) {
		s.service.executeBackup(record.ID, record.BackupType, nil)
	})
}

// RunBackupCleanup 执行备份保留清理。
func (s *Scheduler) RunBackupCleanup() {
	if s == nil || s.service == nil {
		return
	}
	ctx := context.Background()
	configResp, err := s.service.getBackupConfigNoAuth(ctx)
	if err != nil {
		logger.L.Error("读取备份保留配置失败", zap.Error(err))
		return
	}
	records, err := s.service.backupRepo.ListOldSuccessful(ctx, configResp.RetentionCount)
	if err != nil {
		logger.L.Error("查询待清理备份记录失败", zap.Error(err))
		return
	}
	if len(records) == 0 {
		return
	}
	deletedIDs := make([]int64, 0, len(records))
	for _, record := range records {
		if record == nil {
			continue
		}
		if record.FilePath != nil && *record.FilePath != "" {
			if err := storage.DeleteFile(ctx, *record.FilePath); err != nil {
				logger.L.Error("删除备份对象失败", zap.Int64("backup_id", record.ID), zap.Error(err))
				continue
			}
		}
		deletedIDs = append(deletedIDs, record.ID)
	}
	if err := s.service.backupRepo.DeleteByIDs(ctx, deletedIDs); err != nil {
		logger.L.Error("删除备份记录失败", zap.Error(err))
	}
}

// RunStatsDataCleanup 清理过期统计数据。
func (s *Scheduler) RunStatsDataCleanup() {
	if s == nil || s.service == nil {
		return
	}
	ctx := context.Background()
	before := time.Now().UTC().AddDate(-1, 0, 0)
	if err := s.service.statRepo.DeleteBeforeDate(ctx, before); err != nil {
		logger.L.Error("清理过期平台统计失败", zap.Error(err))
	}
}

// checkThresholdRule 检查单条阈值告警规则。
func (s *Scheduler) checkThresholdRule(ctx context.Context, rule *entity.AlertRule, resources *dto.DashboardResourcesResp) {
	condition := decodeAlertRuleCondition(rule.Condition)
	if condition.Metric == nil || condition.Operator == nil || condition.Value == nil || condition.Duration == nil {
		return
	}
	currentValue, ok := resolveMetricValue(resources, *condition.Metric)
	if !ok {
		return
	}
	if !compareMetricValue(currentValue, *condition.Operator, *condition.Value) {
		clearAlertWindowState(ctx, rule.ID)
		return
	}
	startedAt := loadOrInitAlertWindowState(ctx, rule.ID)
	if time.Since(startedAt) < time.Duration(*condition.Duration)*time.Second {
		return
	}

	title := buildThresholdAlertTitle(*condition.Metric, currentValue, *condition.Value)
	detailJSON, _ := json.Marshal(dto.AlertEventDetail{
		Metric:          condition.Metric,
		CurrentValue:    float64Ptr(currentValue),
		Threshold:       condition.Value,
		DurationSeconds: condition.Duration,
	})
	s.service.triggerAlertEvent(ctx, rule, title, datatypes.JSON(detailJSON))
}

// checkEventRule 检查单条事件告警规则。
func (s *Scheduler) checkEventRule(ctx context.Context, rule *entity.AlertRule) {
	condition := decodeAlertRuleCondition(rule.Condition)
	if condition.EventSource == nil || condition.CountThreshold == nil || condition.TimeWindow == nil {
		return
	}
	if *condition.EventSource != "login_logs" {
		return
	}
	action := int16(enum.LoginActionFail)
	if raw, ok := condition.EventFilter["action"]; ok {
		switch v := raw.(type) {
		case float64:
			action = int16(v)
		case int:
			action = int16(v)
		}
	}
	groups, err := s.service.alertEventRepo.ListLoginFailGroups(ctx, time.Now().UTC().Add(-time.Duration(*condition.TimeWindow)*time.Second), action, *condition.CountThreshold)
	if err != nil {
		logger.L.Error("查询事件告警聚合数据失败", zap.Int64("rule_id", rule.ID), zap.Error(err))
		return
	}
	for _, group := range groups {
		if group == nil {
			continue
		}
		samples := make([]dto.AlertEventSampleEvent, 0)
		if len(group.SamplesRaw) > 0 {
			var rawSamples []struct {
				UserID     *int64    `json:"user_id"`
				FailReason *string   `json:"fail_reason"`
				CreatedAt  time.Time `json:"created_at"`
			}
			if err := json.Unmarshal(group.SamplesRaw, &rawSamples); err == nil {
				for _, sample := range rawSamples {
					samples = append(samples, dto.AlertEventSampleEvent{
						UserID:     optionalInt64String(sample.UserID),
						FailReason: sample.FailReason,
						CreatedAt:  sample.CreatedAt.UTC().Format(time.RFC3339),
					})
				}
			}
		}
		title := "同一IP连续登录失败异常：" + group.GroupValue
		detailJSON, _ := json.Marshal(dto.AlertEventDetail{
			EventSource:  condition.EventSource,
			GroupValue:   stringPtr(group.GroupValue),
			EventCount:   intPtr(int(group.EventCount)),
			TimeWindow:   condition.TimeWindow,
			SampleEvents: samples,
		})
		s.service.triggerAlertEvent(ctx, rule, title, datatypes.JSON(detailJSON))
	}
}

// checkServiceRule 检查单条服务状态告警规则。
func (s *Scheduler) checkServiceRule(ctx context.Context, rule *entity.AlertRule, results []serviceHealthResult) {
	condition := decodeAlertRuleCondition(rule.Condition)
	if condition.ServiceName == nil || condition.FailThreshold == nil {
		return
	}
	serviceName := strings.TrimSpace(*condition.ServiceName)
	result, found := findHealthResult(results, serviceName)
	if !found {
		return
	}
	failKey := serviceFailCountPrefix + serviceName
	if result.Status == "healthy" {
		_ = cache.Del(ctx, failKey)
		return
	}
	failCount, err := cache.IncrWithExpire(ctx, failKey, 24*time.Hour)
	if err != nil {
		logger.L.Warn("写入服务连续失败计数失败", zap.String("service", serviceName), zap.Error(err))
		return
	}
	if failCount < int64(*condition.FailThreshold) {
		return
	}
	title := result.Name + " 服务健康检查异常"
	detailJSON, _ := json.Marshal(dto.AlertEventDetail{
		EventSource: stringPtr("service_health"),
		GroupValue:  stringPtr(result.Name),
	})
	s.service.triggerAlertEvent(ctx, rule, title, datatypes.JSON(detailJSON))
}

// resolveMetricValue 从资源快照中解析阈值告警使用的指标值。
func resolveMetricValue(resources *dto.DashboardResourcesResp, metric string) (float64, bool) {
	if resources == nil {
		return 0, false
	}
	switch metric {
	case "cpu_usage":
		return resources.CPU.UsagePercent, true
	case "memory_usage":
		return resources.Memory.UsagePercent, true
	case "storage_usage":
		return resources.Storage.UsagePercent, true
	default:
		return 0, false
	}
}

// compareMetricValue 根据运算符比较当前值与阈值。
func compareMetricValue(current float64, operator string, expected float64) bool {
	switch operator {
	case ">":
		return current > expected
	case ">=":
		return current >= expected
	case "<":
		return current < expected
	case "<=":
		return current <= expected
	case "==":
		return current == expected
	default:
		return false
	}
}

// buildThresholdAlertTitle 生成阈值告警标题。
func buildThresholdAlertTitle(metric string, current, threshold float64) string {
	name := metric
	switch metric {
	case "cpu_usage":
		name = "CPU使用率"
	case "memory_usage":
		name = "内存使用率"
	case "storage_usage":
		name = "存储使用率"
	}
	return name + "过高：" + strconv.FormatFloat(current, 'f', 1, 64) + "% / 阈值 " + strconv.FormatFloat(threshold, 'f', 1, 64) + "%"
}

// loadOrInitAlertWindowState 加载或初始化阈值连续触发起点。
func loadOrInitAlertWindowState(ctx context.Context, ruleID int64) time.Time {
	key := alertWindowStatePrefix + int64String(ruleID)
	if raw, err := cache.GetString(ctx, key); err == nil && raw != "" {
		if unix, parseErr := strconv.ParseInt(raw, 10, 64); parseErr == nil && unix > 0 {
			return time.Unix(unix, 0).UTC()
		}
	}
	now := time.Now().UTC()
	_ = cache.Set(ctx, key, strconv.FormatInt(now.Unix(), 10), 24*time.Hour)
	return now
}

// clearAlertWindowState 清理阈值连续触发状态。
func clearAlertWindowState(ctx context.Context, ruleID int64) {
	_ = cache.Del(ctx, alertWindowStatePrefix+int64String(ruleID))
}

// serviceHealthResult 表示模块08健康检查归一化结果。
type serviceHealthResult struct {
	Name      string `json:"name"`
	Status    string `json:"status"`
	LatencyMS int64  `json:"latency_ms"`
}

// collectHealthResults 获取仪表盘与告警共用的健康检查结果。
func (s *service) collectHealthResults(ctx context.Context) []serviceHealthResult {
	coreResults := healthcheck.CheckCoreServices(ctx)
	results := make([]serviceHealthResult, 0, len(coreResults)+2)
	for _, item := range coreResults {
		results = append(results, serviceHealthResult{
			Name:      item.Name,
			Status:    string(item.Status),
			LatencyMS: item.LatencyMS,
		})
	}
	results = append(results, serviceHealthResult{
		Name:      "API Server",
		Status:    "healthy",
		LatencyMS: 1,
	})
	cluster, err := s.loadClusterStatus(ctx)
	if err != nil {
		results = append(results, serviceHealthResult{Name: "K8s Cluster", Status: "unknown"})
		return results
	}
	status := "healthy"
	if cluster.ReadyNodes < cluster.TotalNodes {
		status = "unhealthy"
	}
	results = append(results, serviceHealthResult{
		Name:      "K8s Cluster",
		Status:    status,
		LatencyMS: 1,
	})
	return results
}

// loadClusterStatus 读取当前集群状态。
func (s *service) loadClusterStatus(ctx context.Context) (*ClusterStatusSnapshot, error) {
	if s.clusterProvider == nil {
		return nil, nil
	}
	return s.clusterProvider.GetClusterStatus(ctx)
}

// buildDashboardResources 构建真实资源快照。
func (s *service) buildDashboardResources(ctx context.Context) (*dto.DashboardResourcesResp, error) {
	cluster, err := s.loadClusterStatus(ctx)
	if err != nil {
		return nil, err
	}
	storageUsedGB, storageErr := s.calculateStorageUsedGB(ctx)
	if storageErr != nil {
		storageUsedGB = 0
	}

	resp := &dto.DashboardResourcesResp{}
	if cluster != nil {
		totalCPU := parseCPUToCores(cluster.TotalCPU)
		usedCPU := parseCPUToCores(cluster.UsedCPU)
		totalMemGB := parseBytesToGB(cluster.TotalMemory)
		usedMemGB := parseBytesToGB(cluster.UsedMemory)
		totalStorageGB := parseBytesToGB(cluster.TotalStorage)
		if totalStorageGB < storageUsedGB {
			totalStorageGB = storageUsedGB
		}

		resp.CPU.CoresTotal = int(totalCPU)
		resp.CPU.CoresUsed = roundFloat(usedCPU, 2)
		resp.CPU.UsagePercent = calculatePercent(usedCPU, totalCPU)
		resp.Memory.TotalGB = int(totalMemGB)
		resp.Memory.UsedGB = int(usedMemGB)
		resp.Memory.UsagePercent = calculatePercent(usedMemGB, totalMemGB)
		resp.Storage.TotalGB = int(totalStorageGB)
		resp.Storage.UsedGB = int(storageUsedGB)
		resp.Storage.UsagePercent = calculatePercent(storageUsedGB, totalStorageGB)
		resp.K8s = dto.DashboardK8sResource{
			Nodes:       cluster.TotalNodes,
			PodsTotal:   cluster.TotalPods,
			PodsRunning: cluster.RunningPods,
			PodsPending: cluster.PendingPods,
			Namespaces:  cluster.Namespaces,
		}
		return resp, nil
	}

	resp.Storage.UsedGB = int(storageUsedGB)
	return resp, nil
}

// calculateStorageUsedGB 统计对象存储使用量。
func (s *service) calculateStorageUsedGB(ctx context.Context) (float64, error) {
	objects, err := storage.ListObjects(ctx, "", true)
	if err != nil {
		return 0, err
	}
	var total int64
	for _, object := range objects {
		total += object.Size
	}
	return roundFloat(float64(total)/(1024*1024*1024), 2), nil
}

// triggerAlertEvent 创建告警事件并通知超级管理员。
func (s *service) triggerAlertEvent(ctx context.Context, rule *entity.AlertRule, title string, detail datatypes.JSON) {
	if s == nil || rule == nil {
		return
	}
	if latest, err := s.alertEventRepo.GetLatestPendingByRuleAndTitle(ctx, rule.ID, title); err == nil && latest != nil {
		return
	}
	triggerKey := cache.KeyAlertLastTrigger + int64String(rule.ID)
	if raw, err := cache.GetString(ctx, triggerKey); err == nil && raw != "" {
		if unix, parseErr := strconv.ParseInt(raw, 10, 64); parseErr == nil && unix > 0 {
			if time.Since(time.Unix(unix, 0).UTC()) < time.Duration(rule.SilencePeriod)*time.Second {
				return
			}
		}
	}

	event := &entity.AlertEvent{
		RuleID:      rule.ID,
		Level:       rule.Level,
		Title:       title,
		Detail:      detail,
		Status:      enum.AlertEventStatusPending,
		TriggeredAt: time.Now().UTC(),
	}
	if err := s.alertEventRepo.Create(ctx, event); err != nil {
		logger.L.Error("创建告警事件失败", zap.Int64("rule_id", rule.ID), zap.Error(err))
		return
	}
	_ = cache.Set(ctx, triggerKey, strconv.FormatInt(event.TriggeredAt.Unix(), 10), 0)
	s.dispatchAlertNotification(ctx, rule, event)
}

// dispatchAlertNotification 通过模块07发送告警通知。
func (s *service) dispatchAlertNotification(ctx context.Context, rule *entity.AlertRule, event *entity.AlertEvent) {
	if s == nil || s.notification == nil || rule == nil || event == nil {
		return
	}
	users, _, err := s.userRepo.List(ctx, &authrepo.UserListParams{
		Role:     enum.RoleSuperAdmin,
		Page:     1,
		PageSize: 1000,
	})
	if err != nil {
		logger.L.Warn("查询超级管理员列表失败", zap.Error(err))
		return
	}
	receiverIDs := make([]string, 0, len(users))
	for _, user := range users {
		if user == nil || user.ID == 0 {
			continue
		}
		receiverIDs = append(receiverIDs, int64String(user.ID))
	}
	if len(receiverIDs) == 0 {
		return
	}
	_ = s.notification.DispatchEvent(ctx, &dto.InternalSendNotificationEventReq{
		EventType:   "system.alert.triggered",
		ReceiverIDs: receiverIDs,
		Params: map[string]interface{}{
			"level_text": enum.GetAlertLevelText(event.Level),
			"rule_name":  rule.Name,
			"title":      event.Title,
			"detail":     string(event.Detail),
		},
		SourceModule: "module_08",
		SourceType:   "alert_event",
		SourceID:     int64String(event.ID),
	})
}

// findHealthResult 按名称查找健康检查结果。
func findHealthResult(results []serviceHealthResult, serviceName string) (serviceHealthResult, bool) {
	for _, result := range results {
		if serviceKeyFromHealthResultName(result.Name) == serviceName {
			return result, true
		}
	}
	return serviceHealthResult{}, false
}

// serviceKeyFromHealthResultName 将健康检查展示名转换为模块08内部标准服务键。
func serviceKeyFromHealthResultName(name string) string {
	switch strings.TrimSpace(name) {
	case "PostgreSQL":
		return "postgresql"
	case "Redis":
		return "redis"
	case "NATS":
		return "nats"
	case "MinIO":
		return "minio"
	case "API Server":
		return "api_server"
	case "K8s Cluster":
		return "k8s_cluster"
	default:
		return ""
	}
}

// parseCPUToCores 将 K8s CPU 字符串解析为核数。
func parseCPUToCores(raw string) float64 {
	q, err := resource.ParseQuantity(strings.TrimSpace(raw))
	if err != nil {
		return 0
	}
	return float64(q.MilliValue()) / 1000
}

// parseBytesToGB 将 K8s 资源字符串解析为 GB。
func parseBytesToGB(raw string) float64 {
	q, err := resource.ParseQuantity(strings.TrimSpace(raw))
	if err != nil {
		return 0
	}
	return float64(q.Value()) / (1024 * 1024 * 1024)
}

// calculatePercent 计算百分比。
func calculatePercent(used, total float64) float64 {
	if total <= 0 {
		return 0
	}
	return roundFloat(used*100/total, 2)
}

// roundFloat 对浮点值做小数位收敛。
func roundFloat(value float64, scale int) float64 {
	if scale <= 0 {
		return value
	}
	pow := 1.0
	for i := 0; i < scale; i++ {
		pow *= 10
	}
	return float64(int(value*pow+0.5)) / pow
}

// stringPtr 返回字符串指针。
func stringPtr(value string) *string {
	return &value
}

// intPtr 返回 int 指针。
func intPtr(value int) *int {
	return &value
}

// float64Ptr 返回 float64 指针。
func float64Ptr(value float64) *float64 {
	return &value
}

// normalizeAutoBackupCronSpec 校验并返回平台统一的 6 段秒级自动备份 cron 表达式。
func normalizeAutoBackupCronSpec(configResp *dto.BackupConfigResp) (string, error) {
	spec := defaultBackupCron
	if configResp != nil && strings.TrimSpace(configResp.Cron) != "" {
		spec = strings.TrimSpace(configResp.Cron)
	}
	if err := validateBackupCronSpec(spec); err != nil {
		return "", err
	}
	return strings.Join(strings.Fields(spec), " "), nil
}
