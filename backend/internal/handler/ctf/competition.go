// competition.go
// 模块05 — CTF竞赛：竞赛、题目、团队与报名 HTTP 处理层。
// 负责竞赛主流程、题目管理、漏洞转化、审核、团队管理、报名、排行榜、公告、配额、监控和统计接口。
// 对照 docs/modules/05-CTF竞赛/03-API接口设计.md。

package ctf

import (
	"github.com/gin-gonic/gin"

	"github.com/lenschain/backend/internal/model/dto"
	"github.com/lenschain/backend/internal/pkg/handlerctx"
	"github.com/lenschain/backend/internal/pkg/response"
	"github.com/lenschain/backend/internal/pkg/validator"
	svc "github.com/lenschain/backend/internal/service/ctf"
)

// CompetitionHandler 竞赛、题目与团队域处理器。
// 统一承接模块05中非攻防赛、非题目环境的 HTTP 接口。
type CompetitionHandler struct {
	competitionService svc.CompetitionService
	challengeService   svc.ChallengeService
	teamService        svc.TeamService
}

// NewCompetitionHandler 创建竞赛、题目与团队域处理器。
func NewCompetitionHandler(
	competitionService svc.CompetitionService,
	challengeService svc.ChallengeService,
	teamService svc.TeamService,
) *CompetitionHandler {
	return &CompetitionHandler{
		competitionService: competitionService,
		challengeService:   challengeService,
		teamService:        teamService,
	}
}

// CreateCompetition 创建竞赛。
// POST /api/v1/ctf/competitions
func (h *CompetitionHandler) CreateCompetition(c *gin.Context) {
	var req dto.CreateCompetitionReq
	if !validator.BindJSON(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	respData, err := h.competitionService.Create(c.Request.Context(), sc, &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "创建成功", respData)
}

// ListCompetitions 获取竞赛列表。
// GET /api/v1/ctf/competitions
func (h *CompetitionHandler) ListCompetitions(c *gin.Context) {
	var req dto.CompetitionListReq
	if !validator.BindQuery(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	respData, err := h.competitionService.List(c.Request.Context(), sc, &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}

// GetCompetition 获取竞赛详情。
// GET /api/v1/ctf/competitions/:id
func (h *CompetitionHandler) GetCompetition(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	respData, err := h.competitionService.Get(c.Request.Context(), sc, id)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}

// UpdateCompetition 编辑竞赛。
// PUT /api/v1/ctf/competitions/:id
func (h *CompetitionHandler) UpdateCompetition(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.UpdateCompetitionReq
	if !validator.BindJSON(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	if err := h.competitionService.Update(c.Request.Context(), sc, id, &req); err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "更新成功", nil)
}

// DeleteCompetition 删除竞赛。
// DELETE /api/v1/ctf/competitions/:id
func (h *CompetitionHandler) DeleteCompetition(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	if err := h.competitionService.Delete(c.Request.Context(), sc, id); err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "删除成功", nil)
}

// PublishCompetition 发布竞赛。
// POST /api/v1/ctf/competitions/:id/publish
func (h *CompetitionHandler) PublishCompetition(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	respData, err := h.competitionService.Publish(c.Request.Context(), sc, id)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "发布成功", respData)
}

// ArchiveCompetition 归档竞赛。
// POST /api/v1/ctf/competitions/:id/archive
func (h *CompetitionHandler) ArchiveCompetition(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	respData, err := h.competitionService.Archive(c.Request.Context(), sc, id)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "归档成功", respData)
}

// TerminateCompetition 强制终止竞赛。
// POST /api/v1/ctf/competitions/:id/terminate
func (h *CompetitionHandler) TerminateCompetition(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.TerminateCompetitionReq
	if !validator.BindJSON(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	respData, err := h.competitionService.Terminate(c.Request.Context(), sc, id, &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "竞赛已强制终止", respData)
}

// ListCompetitionChallenges 获取竞赛题目列表。
// GET /api/v1/ctf/competitions/:id/challenges
func (h *CompetitionHandler) ListCompetitionChallenges(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	respData, err := h.competitionService.ListChallenges(c.Request.Context(), sc, id)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}

// AddCompetitionChallenges 添加竞赛题目。
// POST /api/v1/ctf/competitions/:id/challenges
func (h *CompetitionHandler) AddCompetitionChallenges(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.AddCompetitionChallengeReq
	if !validator.BindJSON(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	respData, err := h.competitionService.AddChallenges(c.Request.Context(), sc, id, &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "添加成功", respData)
}

// SortCompetitionChallenges 调整竞赛题目排序。
// PUT /api/v1/ctf/competitions/:id/challenges/sort
func (h *CompetitionHandler) SortCompetitionChallenges(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.SortCompetitionChallengesReq
	if !validator.BindJSON(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	if err := h.competitionService.SortChallenges(c.Request.Context(), sc, id, &req); err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "排序已更新", nil)
}

// RemoveCompetitionChallenge 移除竞赛题目。
// DELETE /api/v1/ctf/competition-challenges/:id
func (h *CompetitionHandler) RemoveCompetitionChallenge(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	if err := h.competitionService.RemoveChallenge(c.Request.Context(), sc, id); err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "移除成功", nil)
}

// CreateChallenge 创建题目。
// POST /api/v1/ctf/challenges
func (h *CompetitionHandler) CreateChallenge(c *gin.Context) {
	var req dto.CreateChallengeReq
	if !validator.BindJSON(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	respData, err := h.challengeService.Create(c.Request.Context(), sc, &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "创建成功", respData)
}

// ListChallenges 获取题目列表。
// GET /api/v1/ctf/challenges
func (h *CompetitionHandler) ListChallenges(c *gin.Context) {
	var req dto.ChallengeListReq
	if !validator.BindQuery(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	respData, err := h.challengeService.List(c.Request.Context(), sc, &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}

// GetChallenge 获取题目详情。
// GET /api/v1/ctf/challenges/:id
func (h *CompetitionHandler) GetChallenge(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	respData, err := h.challengeService.Get(c.Request.Context(), sc, id)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}

// UpdateChallenge 编辑题目。
// PUT /api/v1/ctf/challenges/:id
func (h *CompetitionHandler) UpdateChallenge(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.UpdateChallengeReq
	if !validator.BindJSON(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	if err := h.challengeService.Update(c.Request.Context(), sc, id, &req); err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "更新成功", nil)
}

// DeleteChallenge 删除题目。
// DELETE /api/v1/ctf/challenges/:id
func (h *CompetitionHandler) DeleteChallenge(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	if err := h.challengeService.Delete(c.Request.Context(), sc, id); err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "删除成功", nil)
}

// SubmitChallengeReview 提交题目审核。
// POST /api/v1/ctf/challenges/:id/submit-review
func (h *CompetitionHandler) SubmitChallengeReview(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	respData, err := h.challengeService.SubmitReview(c.Request.Context(), sc, id)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}

// ListChallengeContracts 获取题目合约列表。
// GET /api/v1/ctf/challenges/:id/contracts
func (h *CompetitionHandler) ListChallengeContracts(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	respData, err := h.challengeService.ListContracts(c.Request.Context(), sc, id)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}

// CreateChallengeContract 创建题目合约。
// POST /api/v1/ctf/challenges/:id/contracts
func (h *CompetitionHandler) CreateChallengeContract(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.CreateChallengeContractReq
	if !validator.BindJSON(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	respData, err := h.challengeService.CreateContract(c.Request.Context(), sc, id, &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "创建成功", respData)
}

// UpdateChallengeContract 编辑题目合约。
// PUT /api/v1/ctf/challenge-contracts/:id
func (h *CompetitionHandler) UpdateChallengeContract(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.UpdateChallengeContractReq
	if !validator.BindJSON(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	if err := h.challengeService.UpdateContract(c.Request.Context(), sc, id, &req); err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "更新成功", nil)
}

// DeleteChallengeContract 删除题目合约。
// DELETE /api/v1/ctf/challenge-contracts/:id
func (h *CompetitionHandler) DeleteChallengeContract(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	if err := h.challengeService.DeleteContract(c.Request.Context(), sc, id); err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "删除成功", nil)
}

// ListChallengeAssertions 获取题目断言列表。
// GET /api/v1/ctf/challenges/:id/assertions
func (h *CompetitionHandler) ListChallengeAssertions(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	respData, err := h.challengeService.ListAssertions(c.Request.Context(), sc, id)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}

// CreateChallengeAssertion 创建题目断言。
// POST /api/v1/ctf/challenges/:id/assertions
func (h *CompetitionHandler) CreateChallengeAssertion(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.CreateChallengeAssertionReq
	if !validator.BindJSON(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	respData, err := h.challengeService.CreateAssertion(c.Request.Context(), sc, id, &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "添加成功", respData)
}

// SortChallengeAssertions 调整题目断言排序。
// PUT /api/v1/ctf/challenges/:id/assertions/sort
func (h *CompetitionHandler) SortChallengeAssertions(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.SortChallengeAssertionReq
	if !validator.BindJSON(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	if err := h.challengeService.SortAssertions(c.Request.Context(), sc, id, &req); err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "排序已更新", nil)
}

// UpdateChallengeAssertion 编辑题目断言。
// PUT /api/v1/ctf/challenge-assertions/:id
func (h *CompetitionHandler) UpdateChallengeAssertion(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.UpdateChallengeAssertionReq
	if !validator.BindJSON(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	if err := h.challengeService.UpdateAssertion(c.Request.Context(), sc, id, &req); err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "更新成功", nil)
}

// DeleteChallengeAssertion 删除题目断言。
// DELETE /api/v1/ctf/challenge-assertions/:id
func (h *CompetitionHandler) DeleteChallengeAssertion(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	if err := h.challengeService.DeleteAssertion(c.Request.Context(), sc, id); err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "删除成功", nil)
}

// VerifyChallenge 发起题目预验证。
// POST /api/v1/ctf/challenges/:id/verify
func (h *CompetitionHandler) VerifyChallenge(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.VerifyChallengeReq
	if !validator.BindOptionalJSON(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	respData, err := h.challengeService.StartVerification(c.Request.Context(), sc, id, &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "预验证已启动", respData)
}

// ListChallengeVerifications 获取题目预验证记录。
// GET /api/v1/ctf/challenges/:id/verifications
func (h *CompetitionHandler) ListChallengeVerifications(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	respData, err := h.challengeService.ListVerifications(c.Request.Context(), sc, id)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}

// GetChallengeVerification 获取预验证详情。
// GET /api/v1/ctf/challenge-verifications/:id
func (h *CompetitionHandler) GetChallengeVerification(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	respData, err := h.challengeService.GetVerification(c.Request.Context(), sc, id)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}

// ReviewChallenge 审核题目。
// POST /api/v1/ctf/challenges/:id/review
func (h *CompetitionHandler) ReviewChallenge(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.ReviewChallengeReq
	if !validator.BindJSON(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	respData, err := h.challengeService.Review(c.Request.Context(), sc, id, &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "审核完成", respData)
}

// ListChallengeReviews 获取题目审核历史。
// GET /api/v1/ctf/challenges/:id/reviews
func (h *CompetitionHandler) ListChallengeReviews(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	respData, err := h.challengeService.ListReviews(c.Request.Context(), sc, id)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}

// ListSWCRegistry 获取 SWC Registry 列表。
// GET /api/v1/ctf/swc-registry
func (h *CompetitionHandler) ListSWCRegistry(c *gin.Context) {
	var req dto.SWCRegistryListReq
	if !validator.BindQuery(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	items, err := h.challengeService.ListSWCRegistry(c.Request.Context(), sc, &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, items)
}

// ImportSWCChallenge 从 SWC 模板导入题目。
// POST /api/v1/ctf/challenges/import-swc
func (h *CompetitionHandler) ImportSWCChallenge(c *gin.Context) {
	var req dto.ImportSWCChallengeReq
	if !validator.BindJSON(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	respData, err := h.challengeService.ImportSWC(c.Request.Context(), sc, &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "导入成功", respData)
}

// ListChallengeTemplates 获取题目模板列表。
// GET /api/v1/ctf/challenge-templates
func (h *CompetitionHandler) ListChallengeTemplates(c *gin.Context) {
	var req dto.ChallengeTemplateListReq
	if !validator.BindQuery(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	respData, err := h.challengeService.ListTemplates(c.Request.Context(), sc, &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}

// GetChallengeTemplate 获取题目模板详情。
// GET /api/v1/ctf/challenge-templates/:id
func (h *CompetitionHandler) GetChallengeTemplate(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	respData, err := h.challengeService.GetTemplate(c.Request.Context(), sc, id)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}

// GenerateChallengeFromTemplate 从模板生成题目。
// POST /api/v1/ctf/challenges/generate-from-template
func (h *CompetitionHandler) GenerateChallengeFromTemplate(c *gin.Context) {
	var req dto.GenerateChallengeFromTemplateReq
	if !validator.BindJSON(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	respData, err := h.challengeService.GenerateFromTemplate(c.Request.Context(), sc, &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "生成成功", respData)
}

// ListPendingChallengeReviews 获取待审核题目列表。
// GET /api/v1/ctf/challenge-reviews/pending
func (h *CompetitionHandler) ListPendingChallengeReviews(c *gin.Context) {
	var req dto.ChallengeListReq
	if !validator.BindQuery(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	respData, err := h.challengeService.ListPendingReviews(c.Request.Context(), sc, &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}
