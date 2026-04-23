// system.go
// 模块08 — 系统管理与监控：HTTP 处理层。
// 该文件负责参数绑定、权限上下文透传和统一响应，不承载系统管理业务规则。

package system

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/lenschain/backend/internal/model/dto"
	"github.com/lenschain/backend/internal/pkg/handlerctx"
	"github.com/lenschain/backend/internal/pkg/response"
	"github.com/lenschain/backend/internal/pkg/validator"
	svc "github.com/lenschain/backend/internal/service/system"
)

// SystemHandler 模块08统一处理器。
type SystemHandler struct {
	service svc.Service
}

// NewSystemHandler 创建模块08处理器。
func NewSystemHandler(service svc.Service) *SystemHandler {
	return &SystemHandler{service: service}
}

// ListAuditLogs 获取聚合审计日志。
func (h *SystemHandler) ListAuditLogs(c *gin.Context) {
	var req dto.AuditLogListReq
	if !validator.BindQuery(c, &req) {
		return
	}
	respData, err := h.service.ListAuditLogs(c.Request.Context(), handlerctx.BuildServiceContext(c), &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}

// ExportAuditLogs 导出聚合审计日志。
func (h *SystemHandler) ExportAuditLogs(c *gin.Context) {
	var req dto.ExportAuditLogReq
	if !validator.BindQuery(c, &req) {
		return
	}
	file, err := h.service.ExportAuditLogs(c.Request.Context(), handlerctx.BuildServiceContext(c), &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	c.Header("Content-Disposition", "attachment; filename=\""+file.FileName+"\"")
	c.Data(http.StatusOK, file.ContentType, file.Content)
}

// GetConfigs 获取配置列表。
func (h *SystemHandler) GetConfigs(c *gin.Context) {
	respData, err := h.service.GetConfigs(c.Request.Context(), handlerctx.BuildServiceContext(c))
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}

// GetConfigGroup 获取指定分组配置。
func (h *SystemHandler) GetConfigGroup(c *gin.Context) {
	respData, err := h.service.GetConfigGroup(c.Request.Context(), handlerctx.BuildServiceContext(c), c.Param("group"))
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}

// UpdateConfig 更新单个配置。
func (h *SystemHandler) UpdateConfig(c *gin.Context) {
	var req dto.UpdateSystemConfigReq
	if !validator.BindJSON(c, &req) {
		return
	}
	if err := h.service.UpdateConfig(c.Request.Context(), handlerctx.BuildServiceContext(c), c.Param("group"), c.Param("key"), &req); err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "更新成功", nil)
}

// BatchUpdateConfigs 批量更新分组配置。
func (h *SystemHandler) BatchUpdateConfigs(c *gin.Context) {
	var req dto.BatchUpdateSystemConfigsReq
	if !validator.BindJSON(c, &req) {
		return
	}
	if err := h.service.BatchUpdateConfigs(c.Request.Context(), handlerctx.BuildServiceContext(c), c.Param("group"), &req); err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "更新成功", nil)
}

// ListConfigChangeLogs 获取配置变更记录。
func (h *SystemHandler) ListConfigChangeLogs(c *gin.Context) {
	var req dto.ConfigChangeLogListReq
	if !validator.BindQuery(c, &req) {
		return
	}
	respData, err := h.service.ListConfigChangeLogs(c.Request.Context(), handlerctx.BuildServiceContext(c), &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}

// CreateAlertRule 创建告警规则。
func (h *SystemHandler) CreateAlertRule(c *gin.Context) {
	var req dto.CreateAlertRuleReq
	if !validator.BindJSON(c, &req) {
		return
	}
	respData, err := h.service.CreateAlertRule(c.Request.Context(), handlerctx.BuildServiceContext(c), &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}

// ListAlertRules 获取告警规则列表。
func (h *SystemHandler) ListAlertRules(c *gin.Context) {
	var req dto.AlertRuleListReq
	if !validator.BindQuery(c, &req) {
		return
	}
	respData, err := h.service.ListAlertRules(c.Request.Context(), handlerctx.BuildServiceContext(c), &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}

// GetAlertRule 获取告警规则详情。
func (h *SystemHandler) GetAlertRule(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	respData, err := h.service.GetAlertRule(c.Request.Context(), handlerctx.BuildServiceContext(c), id)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}

// UpdateAlertRule 更新告警规则。
func (h *SystemHandler) UpdateAlertRule(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.UpdateAlertRuleReq
	if !validator.BindJSON(c, &req) {
		return
	}
	if err := h.service.UpdateAlertRule(c.Request.Context(), handlerctx.BuildServiceContext(c), id, &req); err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "更新成功", nil)
}

// ToggleAlertRule 启用或禁用告警规则。
func (h *SystemHandler) ToggleAlertRule(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.ToggleAlertRuleReq
	if !validator.BindJSON(c, &req) {
		return
	}
	if err := h.service.ToggleAlertRule(c.Request.Context(), handlerctx.BuildServiceContext(c), id, &req); err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "更新成功", nil)
}

// DeleteAlertRule 删除告警规则。
func (h *SystemHandler) DeleteAlertRule(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	if err := h.service.DeleteAlertRule(c.Request.Context(), handlerctx.BuildServiceContext(c), id); err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "删除成功", nil)
}

// ListAlertEvents 获取告警事件列表。
func (h *SystemHandler) ListAlertEvents(c *gin.Context) {
	var req dto.AlertEventListReq
	if !validator.BindQuery(c, &req) {
		return
	}
	respData, err := h.service.ListAlertEvents(c.Request.Context(), handlerctx.BuildServiceContext(c), &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}

// GetAlertEvent 获取告警事件详情。
func (h *SystemHandler) GetAlertEvent(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	respData, err := h.service.GetAlertEvent(c.Request.Context(), handlerctx.BuildServiceContext(c), id)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}

// HandleAlertEvent 处理告警事件。
func (h *SystemHandler) HandleAlertEvent(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.HandleAlertEventReq
	if !validator.BindJSON(c, &req) {
		return
	}
	if err := h.service.HandleAlertEvent(c.Request.Context(), handlerctx.BuildServiceContext(c), id, &req); err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "处理成功", nil)
}

// IgnoreAlertEvent 忽略告警事件。
func (h *SystemHandler) IgnoreAlertEvent(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.HandleAlertEventReq
	if !validator.BindJSON(c, &req) {
		return
	}
	if err := h.service.IgnoreAlertEvent(c.Request.Context(), handlerctx.BuildServiceContext(c), id, &req); err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "忽略成功", nil)
}

// GetDashboardHealth 获取运维仪表盘健康状态。
func (h *SystemHandler) GetDashboardHealth(c *gin.Context) {
	respData, err := h.service.GetDashboardHealth(c.Request.Context(), handlerctx.BuildServiceContext(c))
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}

// GetDashboardResources 获取运维仪表盘资源使用情况。
func (h *SystemHandler) GetDashboardResources(c *gin.Context) {
	respData, err := h.service.GetDashboardResources(c.Request.Context(), handlerctx.BuildServiceContext(c))
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}

// GetDashboardRealtime 获取运维仪表盘实时指标。
func (h *SystemHandler) GetDashboardRealtime(c *gin.Context) {
	respData, err := h.service.GetDashboardRealtime(c.Request.Context(), handlerctx.BuildServiceContext(c))
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}

// GetStatisticsOverview 获取统计总览。
func (h *SystemHandler) GetStatisticsOverview(c *gin.Context) {
	respData, err := h.service.GetStatisticsOverview(c.Request.Context(), handlerctx.BuildServiceContext(c))
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}

// GetStatisticsTrend 获取统计趋势。
func (h *SystemHandler) GetStatisticsTrend(c *gin.Context) {
	var req dto.StatisticsTrendReq
	if !validator.BindQuery(c, &req) {
		return
	}
	respData, err := h.service.GetStatisticsTrend(c.Request.Context(), handlerctx.BuildServiceContext(c), &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}

// GetSchoolStatistics 获取学校活跃度排行统计。
func (h *SystemHandler) GetSchoolStatistics(c *gin.Context) {
	respData, err := h.service.GetSchoolStatistics(c.Request.Context(), handlerctx.BuildServiceContext(c))
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}

// TriggerBackup 手动触发备份。
func (h *SystemHandler) TriggerBackup(c *gin.Context) {
	respData, err := h.service.TriggerBackup(c.Request.Context(), handlerctx.BuildServiceContext(c))
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}

// ListBackups 获取备份列表。
func (h *SystemHandler) ListBackups(c *gin.Context) {
	var req dto.BackupListReq
	if !validator.BindQuery(c, &req) {
		return
	}
	respData, err := h.service.ListBackups(c.Request.Context(), handlerctx.BuildServiceContext(c), &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}

// DownloadBackup 下载备份文件。
func (h *SystemHandler) DownloadBackup(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	file, err := h.service.DownloadBackup(c.Request.Context(), handlerctx.BuildServiceContext(c), id)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	c.Redirect(http.StatusFound, file.RedirectURL)
}

// UpdateBackupConfig 更新备份配置。
func (h *SystemHandler) UpdateBackupConfig(c *gin.Context) {
	var req dto.UpdateBackupConfigReq
	if !validator.BindJSON(c, &req) {
		return
	}
	respData, err := h.service.UpdateBackupConfig(c.Request.Context(), handlerctx.BuildServiceContext(c), &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}

// GetBackupConfig 获取备份配置。
func (h *SystemHandler) GetBackupConfig(c *gin.Context) {
	respData, err := h.service.GetBackupConfig(c.Request.Context(), handlerctx.BuildServiceContext(c))
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}
