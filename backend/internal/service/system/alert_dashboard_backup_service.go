// alert_dashboard_backup_service.go
// 模块08 — 系统管理与监控：告警、仪表盘、统计和备份业务。
// 该文件聚合运维域能力，统一处理告警规则/事件、健康面板、平台统计和备份流程。

package system

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"

	"github.com/lenschain/backend/internal/model/dto"
	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/model/enum"
	"github.com/lenschain/backend/internal/pkg/backup"
	svcctx "github.com/lenschain/backend/internal/pkg/context"
	"github.com/lenschain/backend/internal/pkg/errcode"
	systemrepo "github.com/lenschain/backend/internal/repository/system"
)

// CreateAlertRule 创建告警规则。
func (s *service) CreateAlertRule(ctx context.Context, sc *svcctx.ServiceContext, req *dto.CreateAlertRuleReq) (*dto.AlertRuleDetailResp, error) {
	if err := ensureSuperAdmin(sc); err != nil {
		return nil, err
	}
	condition, err := validateAndMarshalAlertCondition(req.AlertType, req.Condition)
	if err != nil {
		return nil, err
	}
	rule := &entity.AlertRule{
		Name:          req.Name,
		Description:   req.Description,
		AlertType:     req.AlertType,
		Level:         req.Level,
		Condition:     condition,
		SilencePeriod: normalizeSilencePeriod(req.SilencePeriod),
		IsEnabled:     true,
		CreatedBy:     sc.UserID,
	}
	if err := s.alertRuleRepo.Create(ctx, rule); err != nil {
		return nil, err
	}
	return buildAlertRuleDetail(rule), nil
}

// ListAlertRules 获取告警规则列表。
func (s *service) ListAlertRules(ctx context.Context, sc *svcctx.ServiceContext, req *dto.AlertRuleListReq) (*dto.AlertRuleListResp, error) {
	if err := ensureSuperAdmin(sc); err != nil {
		return nil, err
	}
	page, pageSize := normalizePage(req.Page, req.PageSize)
	items, total, err := s.alertRuleRepo.List(ctx, &systemrepo.AlertRuleListParams{
		AlertType: req.AlertType,
		Level:     req.Level,
		IsEnabled: req.IsEnabled,
		Page:      page,
		PageSize:  pageSize,
	})
	if err != nil {
		return nil, err
	}
	list := make([]dto.AlertRuleItem, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		list = append(list, buildAlertRuleItem(item))
	}
	return &dto.AlertRuleListResp{
		List:       list,
		Pagination: buildPaginationResp(page, pageSize, total),
	}, nil
}

// GetAlertRule 获取告警规则详情。
func (s *service) GetAlertRule(ctx context.Context, sc *svcctx.ServiceContext, id int64) (*dto.AlertRuleDetailResp, error) {
	if err := ensureSuperAdmin(sc); err != nil {
		return nil, err
	}
	rule, err := s.alertRuleRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errcode.ErrAlertRuleNotFound
		}
		return nil, err
	}
	return buildAlertRuleDetail(rule), nil
}

// UpdateAlertRule 更新告警规则。
func (s *service) UpdateAlertRule(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.UpdateAlertRuleReq) error {
	if err := ensureSuperAdmin(sc); err != nil {
		return err
	}
	rule, err := s.alertRuleRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errcode.ErrAlertRuleNotFound
		}
		return err
	}

	values := map[string]interface{}{}
	alertType := rule.AlertType
	if req.AlertType != nil {
		alertType = *req.AlertType
		values["alert_type"] = alertType
	}
	if req.Name != nil {
		values["name"] = *req.Name
	}
	if req.Description != nil {
		values["description"] = req.Description
	}
	if req.Level != nil {
		values["level"] = *req.Level
	}
	if req.SilencePeriod != nil {
		values["silence_period"] = normalizeSilencePeriod(*req.SilencePeriod)
	}
	if req.Condition != nil {
		condition, conditionErr := validateAndMarshalAlertCondition(alertType, *req.Condition)
		if conditionErr != nil {
			return conditionErr
		}
		values["condition"] = condition
	} else if req.AlertType != nil {
		condition, conditionErr := validateAndMarshalAlertCondition(alertType, decodeAlertRuleCondition(rule.Condition))
		if conditionErr != nil {
			return conditionErr
		}
		values["condition"] = condition
	}
	if len(values) == 0 {
		return nil
	}
	values["updated_at"] = time.Now().UTC()
	return s.alertRuleRepo.Update(ctx, id, values)
}

// ToggleAlertRule 启用或禁用告警规则。
func (s *service) ToggleAlertRule(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.ToggleAlertRuleReq) error {
	if err := ensureSuperAdmin(sc); err != nil {
		return err
	}
	if _, err := s.alertRuleRepo.GetByID(ctx, id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errcode.ErrAlertRuleNotFound
		}
		return err
	}
	return s.alertRuleRepo.Toggle(ctx, id, req.IsEnabled)
}

// DeleteAlertRule 删除告警规则。
func (s *service) DeleteAlertRule(ctx context.Context, sc *svcctx.ServiceContext, id int64) error {
	if err := ensureSuperAdmin(sc); err != nil {
		return err
	}
	if _, err := s.alertRuleRepo.GetByID(ctx, id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errcode.ErrAlertRuleNotFound
		}
		return err
	}
	return s.alertRuleRepo.Delete(ctx, id)
}

// ListAlertEvents 获取告警事件列表。
func (s *service) ListAlertEvents(ctx context.Context, sc *svcctx.ServiceContext, req *dto.AlertEventListReq) (*dto.AlertEventListResp, error) {
	if err := ensureSuperAdmin(sc); err != nil {
		return nil, err
	}
	page, pageSize := normalizePage(req.Page, req.PageSize)
	dateFrom, dateTo, err := normalizeQueryDateRange(req.DateFrom, req.DateTo)
	if err != nil {
		return nil, err
	}
	params := &systemrepo.AlertEventListParams{
		Level:    req.Level,
		Status:   req.Status,
		DateFrom: dateFrom,
		DateTo:   dateTo,
		Page:     page,
		PageSize: pageSize,
	}
	if strings.TrimSpace(req.RuleID) != "" {
		ruleID, err := parseSnowflakeID(req.RuleID)
		if err != nil {
			return nil, err
		}
		params.RuleID = ruleID
	}
	items, total, err := s.alertEventRepo.List(ctx, params)
	if err != nil {
		return nil, err
	}
	statusCounts, err := s.alertEventRepo.StatusCounts(ctx, params)
	if err != nil {
		return nil, err
	}
	rules, err := s.loadAlertRuleMap(ctx, items)
	if err != nil {
		return nil, err
	}
	list := make([]dto.AlertEventItem, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		list = append(list, buildAlertEventItem(item, rules[item.RuleID]))
	}
	return &dto.AlertEventListResp{
		List:       list,
		Pagination: buildPaginationResp(page, pageSize, total),
		StatusCounts: dto.AlertEventStatusCounts{
			Pending: int(statusCounts.Pending),
			Handled: int(statusCounts.Handled),
			Ignored: int(statusCounts.Ignored),
		},
	}, nil
}

// GetAlertEvent 获取告警事件详情。
func (s *service) GetAlertEvent(ctx context.Context, sc *svcctx.ServiceContext, id int64) (*dto.AlertEventDetailResp, error) {
	if err := ensureSuperAdmin(sc); err != nil {
		return nil, err
	}
	item, err := s.alertEventRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errcode.ErrAlertEventNotFound
		}
		return nil, err
	}
	rule, err := s.alertRuleRepo.GetByID(ctx, item.RuleID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	userMap, err := s.loadHandledByUserMap(ctx, item)
	if err != nil {
		return nil, err
	}
	return buildAlertEventDetail(item, rule, userMap), nil
}

// HandleAlertEvent 处理告警事件。
func (s *service) HandleAlertEvent(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.HandleAlertEventReq) error {
	if err := ensureSuperAdmin(sc); err != nil {
		return err
	}
	if _, err := s.alertEventRepo.GetByID(ctx, id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errcode.ErrAlertEventNotFound
		}
		return err
	}
	return s.alertEventRepo.Handle(ctx, id, sc.UserID, &req.HandleNote, time.Now().UTC())
}

// IgnoreAlertEvent 忽略告警事件。
func (s *service) IgnoreAlertEvent(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.HandleAlertEventReq) error {
	if err := ensureSuperAdmin(sc); err != nil {
		return err
	}
	if _, err := s.alertEventRepo.GetByID(ctx, id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errcode.ErrAlertEventNotFound
		}
		return err
	}
	return s.alertEventRepo.Ignore(ctx, id, sc.UserID, &req.HandleNote, time.Now().UTC())
}

// validateAndMarshalAlertCondition 校验并序列化告警条件。
func validateAndMarshalAlertCondition(alertType int16, condition dto.AlertRuleCondition) (datatypes.JSON, error) {
	switch alertType {
	case enum.AlertTypeThreshold:
		if condition.Metric == nil || condition.Operator == nil || condition.Value == nil || condition.Duration == nil {
			return nil, errcode.ErrInvalidParams.WithMessage("阈值告警条件不完整")
		}
		if !isSupportedThresholdMetric(*condition.Metric) {
			return nil, errcode.ErrInvalidParams.WithMessage("阈值告警指标不合法")
		}
		if !isSupportedThresholdOperator(*condition.Operator) {
			return nil, errcode.ErrInvalidParams.WithMessage("阈值告警运算符不合法")
		}
	case enum.AlertTypeEvent:
		if condition.EventSource == nil || condition.CountThreshold == nil || condition.TimeWindow == nil {
			return nil, errcode.ErrInvalidParams.WithMessage("事件告警条件不完整")
		}
		if strings.TrimSpace(*condition.EventSource) != "login_logs" {
			return nil, errcode.ErrInvalidParams.WithMessage("事件告警来源不合法")
		}
		if condition.GroupBy != nil && strings.TrimSpace(*condition.GroupBy) != "ip" {
			return nil, errcode.ErrInvalidParams.WithMessage("事件告警分组字段不合法")
		}
	case enum.AlertTypeService:
		if condition.ServiceName == nil || condition.FailThreshold == nil {
			return nil, errcode.ErrInvalidParams.WithMessage("服务状态告警条件不完整")
		}
		if !isSupportedServiceRuleName(*condition.ServiceName) {
			return nil, errcode.ErrInvalidParams.WithMessage("服务状态告警服务名不合法")
		}
	default:
		return nil, errcode.ErrInvalidParams.WithMessage("告警类型不合法")
	}
	payload, err := json.Marshal(condition)
	if err != nil {
		return nil, errcode.ErrInvalidParams.WithMessage("告警规则条件配置错误")
	}
	return datatypes.JSON(payload), nil
}

// normalizeSilencePeriod 统一静默期默认值。
func normalizeSilencePeriod(value int) int {
	if value <= 0 {
		return 1800
	}
	return value
}

// isSupportedThresholdMetric 判断阈值告警是否使用了标准指标名。
func isSupportedThresholdMetric(metric string) bool {
	switch strings.TrimSpace(metric) {
	case "cpu_usage", "memory_usage", "storage_usage":
		return true
	default:
		return false
	}
}

// isSupportedThresholdOperator 判断阈值告警是否使用了标准运算符。
func isSupportedThresholdOperator(operator string) bool {
	switch strings.TrimSpace(operator) {
	case ">", ">=", "<", "<=", "==":
		return true
	default:
		return false
	}
}

// isSupportedServiceRuleName 判断服务状态告警是否使用了标准服务名。
func isSupportedServiceRuleName(serviceName string) bool {
	switch strings.TrimSpace(serviceName) {
	case "postgresql", "redis", "nats", "minio", "api_server", "k8s_cluster":
		return true
	default:
		return false
	}
}

// resolveTrendPeriod 解析趋势周期并返回起始时间。
func resolveTrendPeriod(period string) (time.Time, string) {
	now := time.Now().UTC()
	switch period {
	case "7d":
		return now.AddDate(0, 0, -7), "7d"
	case "90d":
		return now.AddDate(0, 0, -90), "90d"
	case "365d":
		return now.AddDate(0, 0, -365), "365d"
	default:
		return now.AddDate(0, 0, -30), "30d"
	}
}

// loadAlertRuleMap 批量加载列表中的规则名称。
func (s *service) loadAlertRuleMap(ctx context.Context, events []*entity.AlertEvent) (map[int64]*entity.AlertRule, error) {
	result := make(map[int64]*entity.AlertRule)
	for _, item := range events {
		if item == nil {
			continue
		}
		if _, exists := result[item.RuleID]; exists {
			continue
		}
		rule, err := s.alertRuleRepo.GetByID(ctx, item.RuleID)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		if rule != nil {
			result[item.RuleID] = rule
		}
	}
	return result, nil
}

// loadHandledByUserMap 读取单条告警详情需要的处理人名称。
func (s *service) loadHandledByUserMap(ctx context.Context, event *entity.AlertEvent) (map[int64]*entity.User, error) {
	result := make(map[int64]*entity.User)
	if event == nil || event.HandledBy == nil || *event.HandledBy == 0 {
		return result, nil
	}
	user, err := s.userRepo.GetByID(ctx, *event.HandledBy)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if user != nil {
		result[user.ID] = user
	}
	return result, nil
}

// backupsRunning 检查是否存在运行中的备份。
func (s *service) backupsRunning(ctx context.Context) (bool, int64, error) {
	items, total, err := s.backupRepo.List(ctx, &systemrepo.BackupRecordListParams{
		Status:   enum.BackupStatusRunning,
		Page:     1,
		PageSize: 1,
	})
	if err != nil {
		return false, 0, err
	}
	if total > 0 && len(items) > 0 && items[0] != nil {
		return true, items[0].ID, nil
	}
	return false, 0, nil
}

// executeBackup 执行实际备份任务并回写记录状态。
func (s *service) executeBackup(recordID int64, backupType int16, triggeredBy *int64) {
	ctx := context.Background()
	startedAt := time.Now().UTC()
	fileName := buildDefaultBackupFileName(startedAt, backupType)
	result, err := backup.RunPostgresBackup(ctx, backup.Options{
		Database:       s.cfg.Database,
		ObjectPrefix:   defaultBackupObjectPrefix,
		FileName:       fileName,
		TempDir:        "",
		Encrypt:        true,
		Compress:       true,
		CommandTimeout: 30 * time.Minute,
	})
	if err != nil {
		message := err.Error()
		_ = s.backupRepo.Update(ctx, recordID, map[string]interface{}{
			"status":        enum.BackupStatusFailed,
			"error_message": message,
			"completed_at":  time.Now().UTC(),
		})
		return
	}
	_ = s.backupRepo.Update(ctx, recordID, map[string]interface{}{
		"status":        enum.BackupStatusSuccess,
		"file_path":     result.ObjectName,
		"file_size":     result.FileSize,
		"completed_at":  result.FinishedAt,
		"error_message": nil,
		"triggered_by":  triggeredBy,
	})
}

// buildAlertRuleItem 构建告警规则列表项。
func buildAlertRuleItem(item *entity.AlertRule) dto.AlertRuleItem {
	createdAt := item.CreatedAt.UTC().Format(time.RFC3339)
	return dto.AlertRuleItem{
		ID:            int64String(item.ID),
		Name:          item.Name,
		Description:   item.Description,
		AlertType:     item.AlertType,
		AlertTypeText: enum.GetAlertTypeText(item.AlertType),
		Level:         item.Level,
		LevelText:     enum.GetAlertLevelText(item.Level),
		Condition:     decodeAlertRuleCondition(item.Condition),
		SilencePeriod: item.SilencePeriod,
		IsEnabled:     item.IsEnabled,
		CreatedAt:     &createdAt,
	}
}

// buildAlertRuleDetail 构建告警规则详情响应。
func buildAlertRuleDetail(item *entity.AlertRule) *dto.AlertRuleDetailResp {
	if item == nil {
		return nil
	}
	createdAt := item.CreatedAt.UTC().Format(time.RFC3339)
	return &dto.AlertRuleDetailResp{
		ID:            int64String(item.ID),
		Name:          item.Name,
		Description:   item.Description,
		AlertType:     item.AlertType,
		AlertTypeText: enum.GetAlertTypeText(item.AlertType),
		Level:         item.Level,
		LevelText:     enum.GetAlertLevelText(item.Level),
		Condition:     decodeAlertRuleCondition(item.Condition),
		SilencePeriod: item.SilencePeriod,
		IsEnabled:     item.IsEnabled,
		CreatedAt:     &createdAt,
	}
}

// buildAlertEventItem 构建告警事件列表项。
func buildAlertEventItem(item *entity.AlertEvent, rule *entity.AlertRule) dto.AlertEventItem {
	ruleID := ""
	ruleName := ""
	if rule != nil {
		ruleID = int64String(rule.ID)
		ruleName = rule.Name
	}
	return dto.AlertEventItem{
		ID:          int64String(item.ID),
		RuleID:      ruleID,
		RuleName:    ruleName,
		Level:       item.Level,
		LevelText:   enum.GetAlertLevelText(item.Level),
		Title:       item.Title,
		Detail:      decodeAlertEventDetail(item.Detail),
		Status:      item.Status,
		StatusText:  enum.GetAlertEventStatusText(item.Status),
		TriggeredAt: item.TriggeredAt.UTC().Format(time.RFC3339),
	}
}

// buildAlertEventDetail 构建告警事件详情响应。
func buildAlertEventDetail(item *entity.AlertEvent, rule *entity.AlertRule, userMap map[int64]*entity.User) *dto.AlertEventDetailResp {
	if item == nil {
		return nil
	}
	resp := &dto.AlertEventDetailResp{
		ID:          int64String(item.ID),
		Level:       item.Level,
		LevelText:   enum.GetAlertLevelText(item.Level),
		Title:       item.Title,
		Detail:      decodeAlertEventDetail(item.Detail),
		Status:      item.Status,
		StatusText:  enum.GetAlertEventStatusText(item.Status),
		HandledBy:   optionalInt64String(item.HandledBy),
		HandledAt:   formatTime(item.HandledAt),
		HandleNote:  item.HandleNote,
		TriggeredAt: item.TriggeredAt.UTC().Format(time.RFC3339),
	}
	if rule != nil {
		resp.RuleID = int64String(rule.ID)
		resp.RuleName = rule.Name
	}
	if item.HandledBy != nil {
		if user, ok := userMap[*item.HandledBy]; ok && user != nil {
			name := user.Name
			resp.HandledByName = &name
		}
	}
	return resp
}
