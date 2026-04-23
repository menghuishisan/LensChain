// environment.go
// 模块05 — CTF竞赛：题目环境 HTTP 处理层。
// 负责题目环境启动、详情、重置、销毁、竞赛环境列表等接口。

package ctf

import (
	"github.com/gin-gonic/gin"

	"github.com/lenschain/backend/internal/model/dto"
	"github.com/lenschain/backend/internal/pkg/handlerctx"
	"github.com/lenschain/backend/internal/pkg/response"
	"github.com/lenschain/backend/internal/pkg/validator"
	svc "github.com/lenschain/backend/internal/service/ctf"
)

// EnvironmentHandler 题目环境域处理器。
type EnvironmentHandler struct {
	environmentService svc.EnvironmentService
}

// NewEnvironmentHandler 创建题目环境域处理器。
func NewEnvironmentHandler(environmentService svc.EnvironmentService) *EnvironmentHandler {
	return &EnvironmentHandler{environmentService: environmentService}
}

// StartChallengeEnvironment 启动题目环境。
// POST /api/v1/ctf/competitions/:comp_id/challenges/:challenge_id/environment
func (h *EnvironmentHandler) StartChallengeEnvironment(c *gin.Context) {
	competitionID, ok := validator.ParsePathID(c, "comp_id")
	if !ok {
		return
	}
	challengeID, ok := validator.ParsePathID(c, "challenge_id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	respData, err := h.environmentService.Start(c.Request.Context(), sc, competitionID, challengeID)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "环境创建中", respData)
}

// ListMyEnvironments 获取我的题目环境列表。
// GET /api/v1/ctf/competitions/:id/my-environments
func (h *EnvironmentHandler) ListMyEnvironments(c *gin.Context) {
	competitionID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	respData, err := h.environmentService.ListMyEnvironments(c.Request.Context(), sc, competitionID)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}

// ListCompetitionEnvironments 获取竞赛环境资源列表。
// GET /api/v1/ctf/competitions/:id/environments
func (h *EnvironmentHandler) ListCompetitionEnvironments(c *gin.Context) {
	competitionID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.CompetitionEnvironmentListReq
	if !validator.BindQuery(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	respData, err := h.environmentService.ListCompetitionEnvironments(c.Request.Context(), sc, competitionID, &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}

// GetChallengeEnvironment 获取题目环境详情。
// GET /api/v1/ctf/challenge-environments/:id
func (h *EnvironmentHandler) GetChallengeEnvironment(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	respData, err := h.environmentService.Get(c.Request.Context(), sc, id)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}

// ResetChallengeEnvironment 重置题目环境。
// POST /api/v1/ctf/challenge-environments/:id/reset
func (h *EnvironmentHandler) ResetChallengeEnvironment(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	respData, err := h.environmentService.Reset(c.Request.Context(), sc, id)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "环境重置中", respData)
}

// DestroyChallengeEnvironment 销毁题目环境。
// POST /api/v1/ctf/challenge-environments/:id/destroy
func (h *EnvironmentHandler) DestroyChallengeEnvironment(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	if err := h.environmentService.Destroy(c.Request.Context(), sc, id); err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "销毁成功", nil)
}

// ForceDestroyChallengeEnvironment 强制回收题目环境。
// POST /api/v1/ctf/challenge-environments/:id/force-destroy
func (h *EnvironmentHandler) ForceDestroyChallengeEnvironment(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.ForceDestroyChallengeEnvironmentReq
	if !validator.BindJSON(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	respData, err := h.environmentService.ForceDestroy(c.Request.Context(), sc, id, &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "强制回收成功", respData)
}
