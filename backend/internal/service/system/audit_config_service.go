// audit_config_service.go
// 模块08 — 系统管理与监控：统一审计与全局配置业务。
// 该文件集中处理聚合审计查询/导出、系统配置读写和配置变更记录，保持运维配置域内聚。

package system

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/lenschain/backend/internal/model/dto"
	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/pkg/cache"
	svcctx "github.com/lenschain/backend/internal/pkg/context"
	"github.com/lenschain/backend/internal/pkg/errcode"
	"github.com/lenschain/backend/internal/pkg/excel"
	"github.com/lenschain/backend/internal/pkg/pagination"
	"github.com/lenschain/backend/internal/pkg/timeutil"
	systemrepo "github.com/lenschain/backend/internal/repository/system"
)

// ListAuditLogs 获取统一审计日志列表。
func (s *service) ListAuditLogs(ctx context.Context, sc *svcctx.ServiceContext, req *dto.AuditLogListReq) (*dto.AuditLogListResp, error) {
	if err := ensureSuperAdmin(sc); err != nil {
		return nil, err
	}
	params, err := s.buildAuditParams(req)
	if err != nil {
		return nil, err
	}
	items, total, counts, err := s.auditRepo.List(ctx, params)
	if err != nil {
		return nil, err
	}
	list := make([]dto.AuditLogItem, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		list = append(list, mapAuditItem(item))
	}
	return &dto.AuditLogListResp{
		List:       list,
		Pagination: buildPaginationResp(params.Page, params.PageSize, total),
		SourceCounts: dto.AuditLogSourceCounts{
			Login:      int(counts.Login),
			Operation:  int(counts.Operation),
			Experiment: int(counts.Experiment),
		},
	}, nil
}

// ExportAuditLogs 导出统一审计日志。
func (s *service) ExportAuditLogs(ctx context.Context, sc *svcctx.ServiceContext, req *dto.ExportAuditLogReq) (*ServiceFile, error) {
	if err := ensureSuperAdmin(sc); err != nil {
		return nil, err
	}
	params, err := s.buildAuditParams(&req.AuditLogListReq)
	if err != nil {
		return nil, err
	}
	params.Page = 1
	params.PageSize = maxAuditExportRows
	items, total, _, err := s.auditRepo.List(ctx, params)
	if err != nil {
		return nil, err
	}
	if total > maxAuditExportRows {
		return nil, errcode.ErrInvalidParams.WithMessage("数据量过大，请缩小筛选范围")
	}

	rows := make([][]interface{}, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		rows = append(rows, []interface{}{
			int64String(item.ID),
			describeAuditSource(item.Source),
			auditOperatorNameOrEmpty(item),
			describeAuditAction(item.Source, item.Action),
			optionalString(item.IP),
			optionalString(item.UserAgent),
			string(item.Detail),
			item.CreatedAt.UTC().Format(time.RFC3339),
		})
	}

	headers := []string{"ID", "来源", "操作人", "操作", "IP", "UserAgent", "详情", "时间"}
	format := strings.TrimSpace(req.Format)
	if format == "" {
		format = "excel"
	}
	fileNamePrefix := "audit_logs_" + time.Now().UTC().Format("20060102_150405")
	if format == "csv" {
		buf, exportErr := excel.ExportCSV(headers, rows)
		if exportErr != nil {
			return nil, exportErr
		}
		return &ServiceFile{
			FileName:    fileNamePrefix + ".csv",
			ContentType: "text/csv; charset=utf-8",
			Content:     buf.Bytes(),
		}, nil
	}

	buf, exportErr := excel.Export(&excel.ExportConfig{
		SheetName: "审计日志",
		Headers:   headers,
	}, rows)
	if exportErr != nil {
		return nil, exportErr
	}
	return &ServiceFile{
		FileName:    fileNamePrefix + ".xlsx",
		ContentType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		Content:     buf.Bytes(),
	}, nil
}

// GetConfigs 获取系统配置分组列表。
func (s *service) GetConfigs(ctx context.Context, sc *svcctx.ServiceContext) (*dto.SystemConfigListResp, error) {
	if err := ensureSuperAdmin(sc); err != nil {
		return nil, err
	}
	groupOrder := []string{"platform", "storage", "security", "backup"}
	groups := make([]dto.SystemConfigGroupItem, 0, len(groupOrder))
	for _, group := range groupOrder {
		configs, err := s.loadConfigGroupConfigs(ctx, group)
		if err != nil {
			return nil, err
		}
		if len(configs) == 0 {
			continue
		}
		groups = append(groups, buildConfigGroupItem(group, configs))
	}
	return &dto.SystemConfigListResp{Groups: groups}, nil
}

// GetConfigGroup 获取指定配置分组。
func (s *service) GetConfigGroup(ctx context.Context, sc *svcctx.ServiceContext, group string) (*dto.SystemConfigGroupResp, error) {
	if err := ensureSuperAdmin(sc); err != nil {
		return nil, err
	}
	configs, err := s.loadConfigGroupConfigs(ctx, group)
	if err != nil {
		return nil, err
	}
	return &dto.SystemConfigGroupResp{Group: buildConfigGroupItem(group, configs)}, nil
}

// UpdateConfig 更新单个配置项。
func (s *service) UpdateConfig(ctx context.Context, sc *svcctx.ServiceContext, group, key string, req *dto.UpdateSystemConfigReq) error {
	if err := ensureSuperAdmin(sc); err != nil {
		return err
	}
	updates, cacheUpdates, err := s.buildConfigValueUpdates(ctx, sc, group, []dto.BatchUpdateSystemConfigItem{{
		Key:       key,
		Value:     req.Value,
		UpdatedAt: req.UpdatedAt,
	}})
	if err != nil {
		return err
	}
	updated, err := s.configRepo.UpdateValuesWithChangeLogs(ctx, updates)
	if err != nil {
		return err
	}
	if !updated {
		return errcode.ErrConflict.WithMessage("配置已被修改，请刷新后重试")
	}
	for _, item := range cacheUpdates {
		s.writeConfigCache(ctx, item.Group, item.Key)
	}
	if group == "security" {
		if err := s.syncRuntimeSecurityConfig(ctx); err != nil {
			return err
		}
	}
	if group == "backup" {
		if err := s.syncAutoBackupSchedule(ctx); err != nil {
			return err
		}
	}
	return nil
}

// BatchUpdateConfigs 批量更新分组配置。
func (s *service) BatchUpdateConfigs(ctx context.Context, sc *svcctx.ServiceContext, group string, req *dto.BatchUpdateSystemConfigsReq) error {
	if err := ensureSuperAdmin(sc); err != nil {
		return err
	}
	updates, cacheUpdates, err := s.buildConfigValueUpdates(ctx, sc, group, req.Configs)
	if err != nil {
		return err
	}
	updated, err := s.configRepo.UpdateValuesWithChangeLogs(ctx, updates)
	if err != nil {
		return err
	}
	if !updated {
		return errcode.ErrConflict.WithMessage("配置已被修改，请刷新后重试")
	}
	for _, item := range cacheUpdates {
		s.writeConfigCache(ctx, item.Group, item.Key)
	}
	if group == "security" {
		if err := s.syncRuntimeSecurityConfig(ctx); err != nil {
			return err
		}
	}
	if group == "backup" {
		if err := s.syncAutoBackupSchedule(ctx); err != nil {
			return err
		}
	}
	return nil
}

// ListConfigChangeLogs 获取配置变更记录。
func (s *service) ListConfigChangeLogs(ctx context.Context, sc *svcctx.ServiceContext, req *dto.ConfigChangeLogListReq) (*dto.ConfigChangeLogListResp, error) {
	if err := ensureSuperAdmin(sc); err != nil {
		return nil, err
	}
	page, pageSize := normalizePage(req.Page, req.PageSize)
	dateFrom, dateTo, err := normalizeQueryDateRange(req.DateFrom, req.DateTo)
	if err != nil {
		return nil, err
	}
	items, total, err := s.configChangeLogRepo.List(ctx, &systemrepo.ConfigChangeLogListParams{
		ConfigGroup: req.ConfigGroup,
		ConfigKey:   req.ConfigKey,
		DateFrom:    dateFrom,
		DateTo:      dateTo,
		Page:        page,
		PageSize:    pageSize,
	})
	if err != nil {
		return nil, err
	}
	list := make([]dto.ConfigChangeLogItem, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		changedByName := ""
		if item.ChangedByName != nil {
			changedByName = *item.ChangedByName
		}
		list = append(list, dto.ConfigChangeLogItem{
			ID:            int64String(item.ID),
			ConfigGroup:   item.ConfigGroup,
			ConfigKey:     item.ConfigKey,
			OldValue:      item.OldValue,
			NewValue:      item.NewValue,
			ChangedBy:     int64String(item.ChangedBy),
			ChangedByName: changedByName,
			ChangedAt:     item.ChangedAt.UTC().Format(time.RFC3339),
			IP:            item.IP,
		})
	}
	return &dto.ConfigChangeLogListResp{
		List:       list,
		Pagination: buildPaginationResp(page, pageSize, total),
	}, nil
}

// buildAuditParams 统一构建审计查询参数。
func (s *service) buildAuditParams(req *dto.AuditLogListReq) (*systemrepo.AuditLogListParams, error) {
	page, pageSize := normalizePage(req.Page, req.PageSize)
	dateFrom, dateTo, err := validateQueryDateRange(req.DateFrom, req.DateTo)
	if err != nil {
		return nil, err
	}
	source := strings.TrimSpace(req.Source)
	if err := validateAuditQueryScope(source, dateFrom, dateTo); err != nil {
		return nil, err
	}
	params := &systemrepo.AuditLogListParams{
		Source:   source,
		Keyword:  strings.TrimSpace(req.Keyword),
		Action:   strings.TrimSpace(req.Action),
		IP:       strings.TrimSpace(req.IP),
		DateFrom: strings.TrimSpace(req.DateFrom),
		DateTo:   strings.TrimSpace(req.DateTo),
		Page:     page,
		PageSize: pageSize,
	}
	if req.OperatorID != "" {
		operatorID, err := parseSnowflakeID(req.OperatorID)
		if err != nil {
			return nil, err
		}
		params.OperatorID = operatorID
	}
	return params, nil
}

// validateQueryDateRange 校验查询时间范围必须使用 RFC3339 格式且先后顺序正确。
func validateQueryDateRange(dateFrom, dateTo string) (*time.Time, *time.Time, error) {
	var startAt *time.Time
	if strings.TrimSpace(dateFrom) == "" || strings.TrimSpace(dateTo) == "" {
		if strings.TrimSpace(dateFrom) != "" {
			parsedFrom, err := parseAuditDateTime(dateFrom)
			if err != nil {
				return nil, nil, errcode.ErrInvalidParams.WithMessage("date_from 格式错误")
			}
			startAt = &parsedFrom
		}
		if strings.TrimSpace(dateTo) != "" {
			parsedTo, err := parseAuditDateTime(dateTo)
			if err != nil {
				return nil, nil, errcode.ErrInvalidParams.WithMessage("date_to 格式错误")
			}
			return startAt, &parsedTo, nil
		}
		return startAt, nil, nil
	}
	parsedFrom, err := parseAuditDateTime(dateFrom)
	if err != nil {
		return nil, nil, errcode.ErrInvalidParams.WithMessage("date_from 格式错误")
	}
	parsedTo, err := parseAuditDateTime(dateTo)
	if err != nil {
		return nil, nil, errcode.ErrInvalidParams.WithMessage("date_to 格式错误")
	}
	if parsedFrom.After(parsedTo) {
		return nil, nil, errcode.ErrInvalidParams.WithMessage("开始时间不能晚于结束时间")
	}
	return &parsedFrom, &parsedTo, nil
}

// parseAuditDateTime 解析统一审计查询中的 RFC3339 时间。
func parseAuditDateTime(value string) (time.Time, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return time.Time{}, nil
	}
	parsedAt, err := time.Parse(time.RFC3339, trimmed)
	if err != nil {
		return time.Time{}, err
	}
	return parsedAt.UTC(), nil
}

// validateAuditQueryScope 校验统一审计查询的范围边界。
func validateAuditQueryScope(source string, dateFrom, dateTo *time.Time) error {
	if source != "" {
		return nil
	}
	if dateFrom == nil || dateTo == nil {
		return errcode.ErrInvalidParams.WithMessage("查询全部日志来源时必须指定开始和结束时间，且范围不能超过30天")
	}
	if dateTo.Sub(*dateFrom) > 30*24*time.Hour {
		return errcode.ErrInvalidParams.WithMessage("查询全部日志来源时范围不能超过30天")
	}
	return nil
}

// writeConfigCache 将配置项写入 Redis 缓存。
func (s *service) writeConfigCache(ctx context.Context, group, key string) {
	if cache.Get() == nil {
		return
	}
	item, err := s.configRepo.GetByGroupAndKey(ctx, group, key)
	if err != nil || item == nil {
		return
	}
	payload, marshalErr := json.Marshal(item)
	if marshalErr != nil {
		return
	}
	_ = cache.Set(ctx, systemConfigCachePrefix+group+":"+key, string(payload), systemConfigCacheTTL)
}

// buildConfigGroupItem 将配置分组映射为响应结构。
func buildConfigGroupItem(group string, configs []*entity.SystemConfig) dto.SystemConfigGroupItem {
	items := make([]dto.SystemConfigItem, 0, len(configs))
	for _, item := range configs {
		if item == nil {
			continue
		}
		description := ""
		if item.Description != nil {
			description = *item.Description
		}
		items = append(items, dto.SystemConfigItem{
			Key:         item.ConfigKey,
			Value:       maskSensitiveValue(item),
			ValueType:   item.ValueType,
			Description: description,
			IsSensitive: item.IsSensitive,
			UpdatedAt:   item.UpdatedAt.UTC().Format(optimisticLockTimeLayout),
		})
	}
	return dto.SystemConfigGroupItem{
		Group:     group,
		GroupText: getConfigGroupText(group),
		Configs:   items,
	}
}

// normalizePage 统一模块08分页默认值。
func normalizePage(page, pageSize int) (int, int) {
	return pagination.NormalizeValues(page, pageSize)
}

// optionalString 返回可选字符串的安全值。
func optionalString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

// loadConfigGroupConfigs 按 cache-aside 方式读取指定分组配置，Redis 不可用时降级数据库。
func (s *service) loadConfigGroupConfigs(ctx context.Context, group string) ([]*entity.SystemConfig, error) {
	keys, ok := systemConfigGroupKeys[group]
	if !ok {
		return nil, errcode.ErrConfigNotFound
	}
	if cache.Get() == nil {
		return s.configRepo.ListByGroup(ctx, group)
	}
	items := make([]*entity.SystemConfig, 0, len(keys))
	for _, key := range keys {
		item, hit, err := s.readConfigCache(ctx, group, key)
		if err != nil {
			return s.configRepo.ListByGroup(ctx, group)
		}
		if !hit {
			dbItem, dbErr := s.configRepo.GetByGroupAndKey(ctx, group, key)
			if dbErr != nil {
				return nil, dbErr
			}
			s.writeConfigCache(ctx, group, key)
			item = dbItem
		}
		if item != nil {
			items = append(items, item)
		}
	}
	return items, nil
}

// readConfigCache 读取单个系统配置缓存。
func (s *service) readConfigCache(ctx context.Context, group, key string) (*entity.SystemConfig, bool, error) {
	raw, err := cache.GetString(ctx, systemConfigCachePrefix+group+":"+key)
	if err != nil || strings.TrimSpace(raw) == "" {
		return nil, false, err
	}
	var item entity.SystemConfig
	if err := json.Unmarshal([]byte(raw), &item); err != nil {
		return nil, false, err
	}
	return &item, true, nil
}

// normalizeQueryDateRange 将查询时间范围规范化为 RFC3339 字符串。
func normalizeQueryDateRange(dateFrom, dateTo string) (string, string, error) {
	startAt, endAt, err := validateQueryDateRange(dateFrom, dateTo)
	if err != nil {
		return "", "", err
	}
	var startText string
	if startAt != nil {
		startText = startAt.UTC().Format(time.RFC3339)
	}
	var endText string
	if endAt != nil {
		endText = endAt.UTC().Format(time.RFC3339)
	}
	return startText, endText, nil
}

type configCacheUpdate struct {
	Group string
	Key   string
}

// buildConfigValueUpdates 构建配置批量更新所需的仓储事务请求和缓存刷新项。
func (s *service) buildConfigValueUpdates(ctx context.Context, sc *svcctx.ServiceContext, group string, items []dto.BatchUpdateSystemConfigItem) ([]systemrepo.SystemConfigValueUpdate, []configCacheUpdate, error) {
	updates := make([]systemrepo.SystemConfigValueUpdate, 0, len(items))
	cacheUpdates := make([]configCacheUpdate, 0, len(items))
	for _, item := range items {
		current, err := s.configRepo.GetByGroupAndKey(ctx, group, item.Key)
		if err != nil {
			return nil, nil, errcode.ErrConfigNotFound
		}
		if err := validateConfigValueType(item.Value, current.ValueType); err != nil {
			return nil, nil, err
		}
		if current.ConfigGroup == "backup" && current.ConfigKey == "auto_backup_cron" {
			if err := validateBackupCronSpec(item.Value); err != nil {
				return nil, nil, err
			}
		}
		expectedUpdatedAt, err := timeutil.ParseRFC3339(item.UpdatedAt)
		if err != nil || expectedUpdatedAt == nil {
			return nil, nil, errcode.ErrInvalidParams.WithMessage("updated_at 格式错误")
		}
		oldValue := current.ConfigValue
		updates = append(updates, systemrepo.SystemConfigValueUpdate{
			Group:             current.ConfigGroup,
			Key:               current.ConfigKey,
			Value:             item.Value,
			UpdatedBy:         int64Ptr(sc.UserID),
			ExpectedUpdatedAt: expectedUpdatedAt.UTC(),
			ChangeLog: &entity.ConfigChangeLog{
				ConfigGroup: current.ConfigGroup,
				ConfigKey:   current.ConfigKey,
				OldValue:    &oldValue,
				NewValue:    item.Value,
				ChangedBy:   sc.UserID,
				ChangedAt:   time.Now().UTC(),
				IP:          sc.ClientIP,
			},
		})
		cacheUpdates = append(cacheUpdates, configCacheUpdate{
			Group: current.ConfigGroup,
			Key:   current.ConfigKey,
		})
	}
	return updates, cacheUpdates, nil
}

// syncRuntimeSecurityConfig 将模块08安全配置同步到模块01运行时策略。
func (s *service) syncRuntimeSecurityConfig(ctx context.Context) error {
	if s == nil || s.securitySyncer == nil {
		return nil
	}
	configs, err := s.configRepo.ListByGroup(ctx, "security")
	if err != nil {
		return err
	}

	configMap := make(map[string]string, len(configs))
	for _, item := range configs {
		if item == nil {
			continue
		}
		configMap[item.ConfigKey] = item.ConfigValue
	}

	timeoutHours, _ := strconv.Atoi(configMap["session_timeout_hours"])
	maxFailCount, _ := strconv.Atoi(configMap["max_login_fail_count"])
	lockMinutes, _ := strconv.Atoi(configMap["lock_duration_minutes"])
	passwordMinLength, _ := strconv.Atoi(configMap["password_min_length"])
	passwordRequireUppercase, _ := strconv.ParseBool(configMap["password_require_uppercase"])
	passwordRequireLowercase, _ := strconv.ParseBool(configMap["password_require_lowercase"])
	passwordRequireDigit, _ := strconv.ParseBool(configMap["password_require_digit"])
	passwordRequireSpecialChar, _ := strconv.ParseBool(configMap["password_require_special_char"])
	if timeoutHours <= 0 {
		timeoutHours = 24
	}
	if maxFailCount <= 0 {
		maxFailCount = 5
	}
	if lockMinutes <= 0 {
		lockMinutes = 30
	}
	if passwordMinLength <= 0 {
		passwordMinLength = 8
	}

	return s.securitySyncer.SyncRuntimeSecurityConfig(ctx, RuntimeSecurityConfig{
		SessionTimeoutHours:        timeoutHours,
		MaxLoginFailCount:          maxFailCount,
		LockDurationMinutes:        lockMinutes,
		PasswordMinLength:          passwordMinLength,
		PasswordRequireUppercase:   passwordRequireUppercase,
		PasswordRequireLowercase:   passwordRequireLowercase,
		PasswordRequireDigit:       passwordRequireDigit,
		PasswordRequireSpecialChar: passwordRequireSpecialChar,
	})
}

// auditOperatorNameOrEmpty 返回审计记录中的操作人名称。
func auditOperatorNameOrEmpty(item *systemrepo.AuditLogItem) string {
	if item == nil || item.OperatorName == nil {
		return ""
	}
	return *item.OperatorName
}
