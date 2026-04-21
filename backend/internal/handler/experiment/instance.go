// instance.go
// 模块04 — 实验环境：实例、分组、监控与配额 HTTP 处理层
// 负责实例生命周期、检查点、快照、报告、分组协作、监控统计、资源配额和管理员接口
// 对照 docs/modules/04-实验环境/03-API接口设计.md

package experiment

import (
	"github.com/gin-gonic/gin"

	"github.com/lenschain/backend/internal/model/dto"
	"github.com/lenschain/backend/internal/model/enum"
	"github.com/lenschain/backend/internal/pkg/handlerctx"
	"github.com/lenschain/backend/internal/pkg/pagination"
	"github.com/lenschain/backend/internal/pkg/response"
	"github.com/lenschain/backend/internal/pkg/validator"
	svc "github.com/lenschain/backend/internal/service/experiment"
)

// InstanceHandler 实例与监控域处理器。
// 统一处理实验实例、分组协作、教师监控、资源配额和管理员监控接口。
type InstanceHandler struct {
	instanceService svc.InstanceService
	groupService    svc.GroupService
	monitorService  svc.MonitorService
	quotaService    svc.QuotaService
	imageService    svc.ImageService
}

// NewInstanceHandler 创建实例与监控域处理器。
func NewInstanceHandler(
	instanceService svc.InstanceService,
	groupService svc.GroupService,
	monitorService svc.MonitorService,
	quotaService svc.QuotaService,
	imageService svc.ImageService,
) *InstanceHandler {
	return &InstanceHandler{
		instanceService: instanceService,
		groupService:    groupService,
		monitorService:  monitorService,
		quotaService:    quotaService,
		imageService:    imageService,
	}
}

// CreateInstance 启动实验环境。
// POST /api/v1/experiment-instances
func (h *InstanceHandler) CreateInstance(c *gin.Context) {
	var req dto.CreateInstanceReq
	if !validator.BindJSON(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	respData, err := h.instanceService.Create(c.Request.Context(), sc, &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	if respData != nil && respData.Status == enum.InstanceStatusQueued {
		response.SuccessWithMsg(c, "资源不足，已加入排队", respData)
		return
	}
	response.SuccessWithMsg(c, "实验环境创建中", respData)
}

// ListInstances 获取我的实验实例列表。
// GET /api/v1/experiment-instances
func (h *InstanceHandler) ListInstances(c *gin.Context) {
	var req dto.InstanceListReq
	if !validator.BindQuery(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	items, total, err := h.instanceService.List(c.Request.Context(), sc, &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	page, pageSize := pagination.NormalizeValues(req.Page, req.PageSize)
	response.Paginated(c, items, total, page, pageSize)
}

// GetInstance 获取实验实例详情。
// GET /api/v1/experiment-instances/:id
func (h *InstanceHandler) GetInstance(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	detail, err := h.instanceService.GetByID(c.Request.Context(), sc, id)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, detail)
}

// PauseInstance 暂停实验实例。
// POST /api/v1/experiment-instances/:id/pause
func (h *InstanceHandler) PauseInstance(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	respData, err := h.instanceService.Pause(c.Request.Context(), sc, id)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}

// ResumeInstance 恢复实验实例。
// POST /api/v1/experiment-instances/:id/resume
func (h *InstanceHandler) ResumeInstance(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.ResumeInstanceReq
	if !validator.BindOptionalJSON(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	respData, err := h.instanceService.Resume(c.Request.Context(), sc, id, &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}

// RestartInstance 重新开始实验实例。
// POST /api/v1/experiment-instances/:id/restart
func (h *InstanceHandler) RestartInstance(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	respData, err := h.instanceService.Restart(c.Request.Context(), sc, id)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}

// SubmitInstance 提交实验实例。
// POST /api/v1/experiment-instances/:id/submit
func (h *InstanceHandler) SubmitInstance(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	respData, err := h.instanceService.Submit(c.Request.Context(), sc, id)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}

// DestroyInstance 销毁实验实例。
// POST /api/v1/experiment-instances/:id/destroy
func (h *InstanceHandler) DestroyInstance(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	if err := h.instanceService.Destroy(c.Request.Context(), sc, id); err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "销毁成功", nil)
}

// Heartbeat 上报实验实例心跳。
// POST /api/v1/experiment-instances/:id/heartbeat
func (h *InstanceHandler) Heartbeat(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.HeartbeatReq
	if !validator.BindJSON(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	respData, err := h.instanceService.Heartbeat(c.Request.Context(), sc, id, &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}

// VerifyCheckpoints 触发检查点验证。
// POST /api/v1/experiment-instances/:id/checkpoints/verify
func (h *InstanceHandler) VerifyCheckpoints(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.VerifyCheckpointReq
	if !validator.BindOptionalJSON(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	respData, err := h.instanceService.VerifyCheckpoints(c.Request.Context(), sc, id, &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}

// ListCheckpointResults 获取实例检查点结果列表。
// GET /api/v1/experiment-instances/:id/checkpoints
func (h *InstanceHandler) ListCheckpointResults(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	items, err := h.instanceService.ListCheckpointResults(c.Request.Context(), sc, id)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, items)
}

// ListSnapshots 获取实例快照列表。
// GET /api/v1/experiment-instances/:id/snapshots
func (h *InstanceHandler) ListSnapshots(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	items, err := h.instanceService.ListSnapshots(c.Request.Context(), sc, id)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, items)
}

// CreateSnapshot 创建实例快照。
// POST /api/v1/experiment-instances/:id/snapshots
func (h *InstanceHandler) CreateSnapshot(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.CreateSnapshotReq
	if !validator.BindOptionalJSON(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	respData, err := h.instanceService.CreateSnapshot(c.Request.Context(), sc, id, &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Created(c, respData)
}

// RestoreSnapshot 从快照恢复实例。
// POST /api/v1/experiment-instances/:id/snapshots/:snapshot_id/restore
func (h *InstanceHandler) RestoreSnapshot(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	snapshotID, ok := validator.ParsePathID(c, "snapshot_id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	if err := h.instanceService.RestoreSnapshot(c.Request.Context(), sc, id, snapshotID); err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "恢复成功", nil)
}

// ListOperationLogs 获取实例操作日志列表。
// GET /api/v1/experiment-instances/:id/operation-logs
func (h *InstanceHandler) ListOperationLogs(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.InstanceOpLogListReq
	if !validator.BindQuery(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	items, total, err := h.instanceService.ListOperationLogs(c.Request.Context(), sc, id, &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	page, pageSize := pagination.NormalizeValues(req.Page, req.PageSize)
	response.Paginated(c, items, total, page, pageSize)
}

// CreateReport 提交实验报告。
// POST /api/v1/experiment-instances/:id/report
func (h *InstanceHandler) CreateReport(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.CreateReportReq
	if !validator.BindJSON(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	respData, err := h.instanceService.CreateReport(c.Request.Context(), sc, id, &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "报告提交成功", respData)
}

// GetReport 获取实验报告。
// GET /api/v1/experiment-instances/:id/report
func (h *InstanceHandler) GetReport(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	respData, err := h.instanceService.GetReport(c.Request.Context(), sc, id)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}

// UpdateReport 更新实验报告。
// PUT /api/v1/experiment-instances/:id/report
func (h *InstanceHandler) UpdateReport(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.UpdateReportReq
	if !validator.BindJSON(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	respData, err := h.instanceService.UpdateReport(c.Request.Context(), sc, id, &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}

// SendGuidance 向学生发送指导消息。
// POST /api/v1/experiment-instances/:id/message
func (h *InstanceHandler) SendGuidance(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.SendGuidanceReq
	if !validator.BindJSON(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	if err := h.instanceService.SendGuidance(c.Request.Context(), sc, id, &req); err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "发送成功", nil)
}

// ForceDestroyInstance 强制销毁实验实例。
// POST /api/v1/experiment-instances/:id/force-destroy
func (h *InstanceHandler) ForceDestroyInstance(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	if err := h.instanceService.ForceDestroy(c.Request.Context(), sc, id); err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "回收成功", nil)
}

// ManualGradeInstance 教师手动评分。
// POST /api/v1/experiment-instances/:id/manual-grade
func (h *InstanceHandler) ManualGradeInstance(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.ManualGradeReq
	if !validator.BindJSON(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	respData, err := h.instanceService.ManualGrade(c.Request.Context(), sc, id, &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}

// GradeCheckpoint 手动评分检查点结果。
// POST /api/v1/checkpoint-results/:id/grade
func (h *InstanceHandler) GradeCheckpoint(c *gin.Context) {
	resultID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.GradeCheckpointReq
	if !validator.BindJSON(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	if err := h.instanceService.GradeCheckpoint(c.Request.Context(), sc, resultID, &req); err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "评分成功", nil)
}

// CreateGroup 创建实验分组。
// POST /api/v1/experiment-groups
func (h *InstanceHandler) CreateGroup(c *gin.Context) {
	var req dto.CreateGroupReq
	if !validator.BindJSON(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	items, err := h.groupService.Create(c.Request.Context(), sc, &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "分组创建成功", dto.CreateGroupResp{Groups: derefGroupListItems(items)})
}

// ListGroups 获取实验分组列表。
// GET /api/v1/experiment-groups
func (h *InstanceHandler) ListGroups(c *gin.Context) {
	var req dto.GroupListReq
	if !validator.BindQuery(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	items, total, err := h.groupService.List(c.Request.Context(), sc, &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	page, pageSize := pagination.NormalizeValues(req.Page, req.PageSize)
	response.Paginated(c, items, total, page, pageSize)
}

// GetGroup 获取实验分组详情。
// GET /api/v1/experiment-groups/:id
func (h *InstanceHandler) GetGroup(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	detail, err := h.groupService.GetByID(c.Request.Context(), sc, id)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, detail)
}

// UpdateGroup 编辑实验分组。
// PUT /api/v1/experiment-groups/:id
func (h *InstanceHandler) UpdateGroup(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.UpdateGroupReq
	if !validator.BindJSON(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	if err := h.groupService.Update(c.Request.Context(), sc, id, &req); err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "更新成功", nil)
}

// DeleteGroup 删除实验分组。
// DELETE /api/v1/experiment-groups/:id
func (h *InstanceHandler) DeleteGroup(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	if err := h.groupService.Delete(c.Request.Context(), sc, id); err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "删除成功", nil)
}

// AutoAssignGroups 系统随机分组。
// POST /api/v1/experiment-groups/auto-assign
func (h *InstanceHandler) AutoAssignGroups(c *gin.Context) {
	var req dto.AutoAssignReq
	if !validator.BindJSON(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	respData, err := h.groupService.AutoAssign(c.Request.Context(), sc, &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}

// JoinGroup 学生加入分组。
// POST /api/v1/experiment-groups/:id/join
func (h *InstanceHandler) JoinGroup(c *gin.Context) {
	groupID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.JoinGroupReq
	if !validator.BindJSON(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	if err := h.groupService.Join(c.Request.Context(), sc, groupID, &req); err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "加入成功", nil)
}

// RemoveGroupMember 移除组员。
// DELETE /api/v1/experiment-groups/:id/members/:student_id
func (h *InstanceHandler) RemoveGroupMember(c *gin.Context) {
	groupID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	studentID, ok := validator.ParsePathID(c, "student_id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	if err := h.groupService.RemoveMember(c.Request.Context(), sc, groupID, studentID); err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "移除成功", nil)
}

// ListGroupMembers 获取组员列表。
// GET /api/v1/experiment-groups/:id/members
func (h *InstanceHandler) ListGroupMembers(c *gin.Context) {
	groupID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	items, err := h.groupService.ListMembers(c.Request.Context(), sc, groupID)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, items)
}

// GetGroupProgress 获取组内进度同步数据。
// GET /api/v1/experiment-groups/:id/progress
func (h *InstanceHandler) GetGroupProgress(c *gin.Context) {
	groupID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	respData, err := h.groupService.GetProgress(c.Request.Context(), sc, groupID)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}

// SendGroupMessage 发送组内消息。
// POST /api/v1/experiment-groups/:id/messages
func (h *InstanceHandler) SendGroupMessage(c *gin.Context) {
	groupID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.SendGroupMessageReq
	if !validator.BindJSON(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	if err := h.groupService.SendMessage(c.Request.Context(), sc, groupID, &req); err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "发送成功", nil)
}

// ListGroupMessages 获取组内消息历史。
// GET /api/v1/experiment-groups/:id/messages
func (h *InstanceHandler) ListGroupMessages(c *gin.Context) {
	groupID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.GroupMessageListReq
	if !validator.BindQuery(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	items, total, err := h.groupService.ListMessages(c.Request.Context(), sc, groupID, &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	page, pageSize := pagination.NormalizeValues(req.Page, req.PageSize)
	response.Paginated(c, items, total, page, pageSize)
}

// GetCourseMonitor 获取课程实验监控面板。
// GET /api/v1/courses/:id/experiment-monitor
func (h *InstanceHandler) GetCourseMonitor(c *gin.Context) {
	courseID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.MonitorPanelReq
	if !validator.BindQuery(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	respData, err := h.monitorService.GetCourseMonitor(c.Request.Context(), sc, courseID, &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}

// GetCourseStatistics 获取课程实验统计数据。
// GET /api/v1/courses/:id/experiment-statistics
func (h *InstanceHandler) GetCourseStatistics(c *gin.Context) {
	courseID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.ExperimentStatisticsReq
	if !validator.BindQuery(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	respData, err := h.monitorService.GetCourseStatistics(c.Request.Context(), sc, courseID, &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}

// ListQuotas 获取资源配额列表。
// GET /api/v1/resource-quotas
func (h *InstanceHandler) ListQuotas(c *gin.Context) {
	var req dto.QuotaListReq
	if !validator.BindQuery(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	items, total, err := h.quotaService.List(c.Request.Context(), sc, &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	page, pageSize := pagination.NormalizeValues(req.Page, req.PageSize)
	response.Paginated(c, items, total, page, pageSize)
}

// CreateQuota 创建资源配额。
// POST /api/v1/resource-quotas
func (h *InstanceHandler) CreateQuota(c *gin.Context) {
	var req dto.CreateQuotaReq
	if !validator.BindJSON(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	respData, err := h.quotaService.Create(c.Request.Context(), sc, &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "创建成功", respData)
}

// GetQuota 获取资源配额详情。
// GET /api/v1/resource-quotas/:id
func (h *InstanceHandler) GetQuota(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	respData, err := h.quotaService.GetByID(c.Request.Context(), sc, id)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}

// UpdateQuota 编辑资源配额。
// PUT /api/v1/resource-quotas/:id
func (h *InstanceHandler) UpdateQuota(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.UpdateQuotaReq
	if !validator.BindJSON(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	respData, err := h.quotaService.Update(c.Request.Context(), sc, id, &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}

// GetSchoolUsage 获取学校资源使用情况。
// GET /api/v1/schools/:id/resource-usage
func (h *InstanceHandler) GetSchoolUsage(c *gin.Context) {
	schoolID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	respData, err := h.quotaService.GetSchoolUsage(c.Request.Context(), sc, schoolID)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}

// GetExperimentOverview 获取全平台实验概览。
// GET /api/v1/admin/experiment-overview
func (h *InstanceHandler) GetExperimentOverview(c *gin.Context) {
	sc := handlerctx.BuildServiceContext(c)
	respData, err := h.monitorService.GetExperimentOverview(c.Request.Context(), sc)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}

// GetContainerResources 获取全平台容器资源监控。
// GET /api/v1/admin/container-resources
func (h *InstanceHandler) GetContainerResources(c *gin.Context) {
	sc := handlerctx.BuildServiceContext(c)
	respData, err := h.monitorService.GetContainerResources(c.Request.Context(), sc)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}

// GetK8sClusterStatus 获取 K8s 集群状态。
// GET /api/v1/admin/k8s-cluster-status
func (h *InstanceHandler) GetK8sClusterStatus(c *gin.Context) {
	sc := handlerctx.BuildServiceContext(c)
	respData, err := h.monitorService.GetK8sClusterStatus(c.Request.Context(), sc)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}

// ListAdminInstances 获取全平台实验实例列表。
// GET /api/v1/admin/experiment-instances
func (h *InstanceHandler) ListAdminInstances(c *gin.Context) {
	var req dto.AdminInstanceListReq
	if !validator.BindQuery(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	items, total, err := h.instanceService.ListAdmin(c.Request.Context(), sc, &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	page, pageSize := pagination.NormalizeValues(req.Page, req.PageSize)
	response.Paginated(c, items, total, page, pageSize)
}

// ForceDestroyAdminInstance 超管强制回收任意实验环境。
// POST /api/v1/admin/experiment-instances/:id/force-destroy
func (h *InstanceHandler) ForceDestroyAdminInstance(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	if err := h.instanceService.ForceDestroy(c.Request.Context(), sc, id); err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "回收成功", nil)
}

// GetImagePullStatus 获取镜像预拉取状态。
// GET /api/v1/admin/image-pull-status
func (h *InstanceHandler) GetImagePullStatus(c *gin.Context) {
	var req dto.ImagePullStatusListReq
	if !validator.BindQuery(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	respData, total, err := h.imageService.GetImagePullStatus(c.Request.Context(), sc, &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	page, pageSize := pagination.NormalizeValues(req.Page, req.PageSize)
	respData.Pagination = dto.ListPagination{
		Page:     page,
		PageSize: pageSize,
		Total:    int(total),
	}
	response.Success(c, respData)
}

// TriggerImagePull 触发镜像预拉取。
// POST /api/v1/admin/image-pull
func (h *InstanceHandler) TriggerImagePull(c *gin.Context) {
	var req dto.TriggerImagePullReq
	if !validator.BindOptionalJSON(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	respData, err := h.imageService.TriggerImagePull(c.Request.Context(), sc, &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "预拉取任务已创建", respData)
}

// GetSchoolMonitor 获取学校管理员视角的实验监控。
// GET /api/v1/school/experiment-monitor
func (h *InstanceHandler) GetSchoolMonitor(c *gin.Context) {
	sc := handlerctx.BuildServiceContext(c)
	respData, err := h.monitorService.GetSchoolMonitor(c.Request.Context(), sc)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}

// AssignCourseQuota 为课程分配资源配额。
// PUT /api/v1/school/course-quotas/:course_id
func (h *InstanceHandler) AssignCourseQuota(c *gin.Context) {
	courseID, ok := validator.ParsePathID(c, "course_id")
	if !ok {
		return
	}
	var req dto.CourseQuotaReq
	if !validator.BindJSON(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	respData, err := h.quotaService.AssignCourseQuota(c.Request.Context(), sc, courseID, &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}

// derefGroupListItems 将分组列表指针切片转换为值切片，保持响应结构与 API 文档一致。
func derefGroupListItems(items []*dto.GroupListItem) []dto.GroupListItem {
	result := make([]dto.GroupListItem, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		result = append(result, *item)
	}
	return result
}
