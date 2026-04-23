// competition_participation.go
// 模块05 — CTF竞赛：参赛、团队、提交、排行榜与公告资源 HTTP 处理层。

package ctf

import (
	"github.com/gin-gonic/gin"

	"github.com/lenschain/backend/internal/model/dto"
	"github.com/lenschain/backend/internal/pkg/handlerctx"
	"github.com/lenschain/backend/internal/pkg/response"
	"github.com/lenschain/backend/internal/pkg/validator"
)

// CreateTeam 创建竞赛团队。
// POST /api/v1/ctf/competitions/:id/teams
func (h *CompetitionHandler) CreateTeam(c *gin.Context) {
	competitionID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.CreateTeamReq
	if !validator.BindJSON(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	respData, err := h.teamService.CreateTeam(c.Request.Context(), sc, competitionID, &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "创建成功", respData)
}

// ListTeams 获取竞赛团队列表。
// GET /api/v1/ctf/competitions/:id/teams
func (h *CompetitionHandler) ListTeams(c *gin.Context) {
	competitionID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	respData, err := h.teamService.ListTeams(c.Request.Context(), sc, competitionID)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}

// GetTeam 获取团队详情。
// GET /api/v1/ctf/teams/:id
func (h *CompetitionHandler) GetTeam(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	respData, err := h.teamService.GetTeam(c.Request.Context(), sc, id)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}

// UpdateTeam 编辑团队信息。
// PUT /api/v1/ctf/teams/:id
func (h *CompetitionHandler) UpdateTeam(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.UpdateTeamReq
	if !validator.BindJSON(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	if err := h.teamService.UpdateTeam(c.Request.Context(), sc, id, &req); err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "更新成功", nil)
}

// DisbandTeam 解散团队。
// POST /api/v1/ctf/teams/:id/disband
func (h *CompetitionHandler) DisbandTeam(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	if err := h.teamService.DisbandTeam(c.Request.Context(), sc, id); err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "解散成功", nil)
}

// JoinTeam 通过邀请码加入团队。
// POST /api/v1/ctf/teams/join
func (h *CompetitionHandler) JoinTeam(c *gin.Context) {
	var req dto.JoinTeamReq
	if !validator.BindJSON(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	respData, err := h.teamService.JoinTeam(c.Request.Context(), sc, &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "加入成功", respData)
}

// RemoveTeamMember 移除队员。
// DELETE /api/v1/ctf/teams/:id/members/:student_id
func (h *CompetitionHandler) RemoveTeamMember(c *gin.Context) {
	teamID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	studentID, ok := validator.ParsePathID(c, "student_id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	if err := h.teamService.RemoveMember(c.Request.Context(), sc, teamID, studentID); err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "移除成功", nil)
}

// LeaveTeam 退出团队。
// POST /api/v1/ctf/teams/:id/leave
func (h *CompetitionHandler) LeaveTeam(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	if err := h.teamService.LeaveTeam(c.Request.Context(), sc, id); err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "退出成功", nil)
}

// RegisterCompetition 报名竞赛。
// POST /api/v1/ctf/competitions/:id/register
func (h *CompetitionHandler) RegisterCompetition(c *gin.Context) {
	competitionID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.RegisterCompetitionReq
	if !validator.BindOptionalJSON(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	respData, err := h.teamService.RegisterCompetition(c.Request.Context(), sc, competitionID, &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "报名成功", respData)
}

// CancelRegistration 取消报名。
// DELETE /api/v1/ctf/competitions/:id/register
func (h *CompetitionHandler) CancelRegistration(c *gin.Context) {
	competitionID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	if err := h.teamService.CancelRegistration(c.Request.Context(), sc, competitionID); err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "取消报名成功", nil)
}

// ListRegistrations 获取报名列表。
// GET /api/v1/ctf/competitions/:id/registrations
func (h *CompetitionHandler) ListRegistrations(c *gin.Context) {
	competitionID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.RegistrationListReq
	if !validator.BindQuery(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	respData, err := h.teamService.ListRegistrations(c.Request.Context(), sc, competitionID, &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}

// GetMyRegistration 获取我的报名状态。
// GET /api/v1/ctf/competitions/:id/my-registration
func (h *CompetitionHandler) GetMyRegistration(c *gin.Context) {
	competitionID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	respData, err := h.teamService.GetMyRegistration(c.Request.Context(), sc, competitionID)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}

// SubmitCompetitionChallenge 提交 Flag 或攻击交易。
// POST /api/v1/ctf/competitions/:id/submissions
func (h *CompetitionHandler) SubmitCompetitionChallenge(c *gin.Context) {
	competitionID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.SubmitCompetitionChallengeReq
	if !validator.BindJSON(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	respData, err := h.teamService.SubmitChallenge(c.Request.Context(), sc, competitionID, &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	if respData != nil && respData.IsCorrect {
		response.SuccessWithMsg(c, "提交正确", respData)
		return
	}
	response.SuccessWithMsg(c, "提交错误", respData)
}

// ListCompetitionSubmissions 获取团队提交记录。
// GET /api/v1/ctf/competitions/:id/submissions
func (h *CompetitionHandler) ListCompetitionSubmissions(c *gin.Context) {
	competitionID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.CompetitionSubmissionListReq
	if !validator.BindQuery(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	respData, err := h.teamService.ListSubmissions(c.Request.Context(), sc, competitionID, &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}

// GetCompetitionSubmissionStatistics 获取提交统计。
// GET /api/v1/ctf/competitions/:id/submissions/statistics
func (h *CompetitionHandler) GetCompetitionSubmissionStatistics(c *gin.Context) {
	competitionID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	respData, err := h.teamService.GetSubmissionStatistics(c.Request.Context(), sc, competitionID)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}

// GetLeaderboard 获取实时排行榜。
// GET /api/v1/ctf/competitions/:id/leaderboard
func (h *CompetitionHandler) GetLeaderboard(c *gin.Context) {
	competitionID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.LeaderboardReq
	if !validator.BindQuery(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	respData, err := h.competitionService.GetLeaderboard(c.Request.Context(), sc, competitionID, &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}

// GetLeaderboardHistory 获取排行榜历史快照。
// GET /api/v1/ctf/competitions/:id/leaderboard/history
func (h *CompetitionHandler) GetLeaderboardHistory(c *gin.Context) {
	competitionID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.LeaderboardHistoryReq
	if !validator.BindQuery(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	respData, err := h.competitionService.GetLeaderboardHistory(c.Request.Context(), sc, competitionID, &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}

// GetFinalLeaderboard 获取最终排行榜。
// GET /api/v1/ctf/competitions/:id/leaderboard/final
func (h *CompetitionHandler) GetFinalLeaderboard(c *gin.Context) {
	competitionID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	respData, err := h.competitionService.GetFinalLeaderboard(c.Request.Context(), sc, competitionID)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}

// CreateAnnouncement 发布竞赛公告。
// POST /api/v1/ctf/competitions/:id/announcements
func (h *CompetitionHandler) CreateAnnouncement(c *gin.Context) {
	competitionID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.CreateCtfAnnouncementReq
	if !validator.BindJSON(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	respData, err := h.competitionService.CreateAnnouncement(c.Request.Context(), sc, competitionID, &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "发布成功", respData)
}

// ListAnnouncements 获取竞赛公告列表。
// GET /api/v1/ctf/competitions/:id/announcements
func (h *CompetitionHandler) ListAnnouncements(c *gin.Context) {
	competitionID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	respData, err := h.competitionService.ListAnnouncements(c.Request.Context(), sc, competitionID)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}

// GetAnnouncement 获取公告详情。
// GET /api/v1/ctf/announcements/:id
func (h *CompetitionHandler) GetAnnouncement(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	respData, err := h.competitionService.GetAnnouncement(c.Request.Context(), sc, id)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}

// DeleteAnnouncement 删除公告。
// DELETE /api/v1/ctf/announcements/:id
func (h *CompetitionHandler) DeleteAnnouncement(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	if err := h.competitionService.DeleteAnnouncement(c.Request.Context(), sc, id); err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "删除成功", nil)
}

// GetResourceQuota 获取竞赛资源配额详情。
// GET /api/v1/ctf/competitions/:id/resource-quota
func (h *CompetitionHandler) GetResourceQuota(c *gin.Context) {
	competitionID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	respData, err := h.competitionService.GetResourceQuota(c.Request.Context(), sc, competitionID)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}

// UpdateResourceQuota 设置竞赛资源配额。
// PUT /api/v1/ctf/competitions/:id/resource-quota
func (h *CompetitionHandler) UpdateResourceQuota(c *gin.Context) {
	competitionID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.UpdateResourceQuotaReq
	if !validator.BindJSON(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	respData, err := h.competitionService.UpdateResourceQuota(c.Request.Context(), sc, competitionID, &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "设置成功", respData)
}

// GetCompetitionMonitor 获取竞赛运行监控。
// GET /api/v1/ctf/competitions/:id/monitor
func (h *CompetitionHandler) GetCompetitionMonitor(c *gin.Context) {
	competitionID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	respData, err := h.competitionService.GetMonitor(c.Request.Context(), sc, competitionID)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}

// GetCompetitionStatistics 获取竞赛统计数据。
// GET /api/v1/ctf/competitions/:id/statistics
func (h *CompetitionHandler) GetCompetitionStatistics(c *gin.Context) {
	competitionID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	respData, err := h.competitionService.GetStatistics(c.Request.Context(), sc, competitionID)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}

// GetCompetitionResults 获取竞赛最终结果。
// GET /api/v1/ctf/competitions/:id/results
func (h *CompetitionHandler) GetCompetitionResults(c *gin.Context) {
	competitionID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	respData, err := h.competitionService.GetResults(c.Request.Context(), sc, competitionID)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}

// GetAdminOverview 获取全平台竞赛概览。
// GET /api/v1/ctf/admin/competitions/overview
func (h *CompetitionHandler) GetAdminOverview(c *gin.Context) {
	sc := handlerctx.BuildServiceContext(c)
	respData, err := h.competitionService.GetAdminOverview(c.Request.Context(), sc)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}
