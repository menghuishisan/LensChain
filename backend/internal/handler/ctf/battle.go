// battle.go
// 模块05 — CTF竞赛：攻防赛 HTTP 处理层。
// 负责攻防赛分组、回合、攻击、防守、Token 流水与队伍链接口。

package ctf

import (
	"github.com/gin-gonic/gin"

	"github.com/lenschain/backend/internal/model/dto"
	"github.com/lenschain/backend/internal/pkg/handlerctx"
	"github.com/lenschain/backend/internal/pkg/response"
	"github.com/lenschain/backend/internal/pkg/validator"
	svc "github.com/lenschain/backend/internal/service/ctf"
)

// BattleHandler 攻防赛域处理器。
type BattleHandler struct {
	battleService svc.BattleService
}

// NewBattleHandler 创建攻防赛域处理器。
func NewBattleHandler(battleService svc.BattleService) *BattleHandler {
	return &BattleHandler{battleService: battleService}
}

// CreateAdGroup 创建攻防赛分组。
// POST /api/v1/ctf/competitions/:id/ad-groups
func (h *BattleHandler) CreateAdGroup(c *gin.Context) {
	competitionID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.CreateAdGroupReq
	if !validator.BindJSON(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	respData, err := h.battleService.CreateAdGroup(c.Request.Context(), sc, competitionID, &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "创建成功", respData)
}

// ListAdGroups 获取攻防赛分组列表。
// GET /api/v1/ctf/competitions/:id/ad-groups
func (h *BattleHandler) ListAdGroups(c *gin.Context) {
	competitionID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	respData, err := h.battleService.ListAdGroups(c.Request.Context(), sc, competitionID)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}

// AutoAssignAdGroups 自动分组。
// POST /api/v1/ctf/competitions/:id/ad-groups/auto-assign
func (h *BattleHandler) AutoAssignAdGroups(c *gin.Context) {
	competitionID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.AutoAssignAdGroupsReq
	if !validator.BindJSON(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	respData, err := h.battleService.AutoAssignAdGroups(c.Request.Context(), sc, competitionID, &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.SuccessWithMsg(c, "自动分组完成", respData)
}

// GetAdGroup 获取攻防赛分组详情。
// GET /api/v1/ctf/ad-groups/:id
func (h *BattleHandler) GetAdGroup(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	respData, err := h.battleService.GetAdGroup(c.Request.Context(), sc, id)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}

// ListRounds 获取回合列表。
// GET /api/v1/ctf/ad-groups/:id/rounds
func (h *BattleHandler) ListRounds(c *gin.Context) {
	groupID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	respData, err := h.battleService.ListRounds(c.Request.Context(), sc, groupID)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}

// GetCurrentRound 获取当前回合状态。
// GET /api/v1/ctf/ad-groups/:id/current-round
func (h *BattleHandler) GetCurrentRound(c *gin.Context) {
	groupID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	respData, err := h.battleService.GetCurrentRound(c.Request.Context(), sc, groupID)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}

// ListGroupAttacks 获取分组全部攻击记录。
// GET /api/v1/ctf/ad-groups/:id/attacks
func (h *BattleHandler) ListGroupAttacks(c *gin.Context) {
	groupID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.AdAttackListReq
	if !validator.BindQuery(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	respData, err := h.battleService.ListGroupAttacks(c.Request.Context(), sc, groupID, &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}

// ListGroupChains 获取分组所有队伍链。
// GET /api/v1/ctf/ad-groups/:id/chains
func (h *BattleHandler) ListGroupChains(c *gin.Context) {
	groupID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	respData, err := h.battleService.ListGroupChains(c.Request.Context(), sc, groupID)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}

// GetRound 获取回合详情。
// GET /api/v1/ctf/ad-rounds/:id
func (h *BattleHandler) GetRound(c *gin.Context) {
	id, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	respData, err := h.battleService.GetRound(c.Request.Context(), sc, id)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}

// SubmitAttack 提交攻击交易。
// POST /api/v1/ctf/ad-rounds/:id/attacks
func (h *BattleHandler) SubmitAttack(c *gin.Context) {
	roundID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.SubmitAdAttackReq
	if !validator.BindJSON(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	respData, err := h.battleService.SubmitAttack(c.Request.Context(), sc, roundID, &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	if respData != nil && respData.IsSuccessful {
		response.SuccessWithMsg(c, "攻击成功", respData)
		return
	}
	response.SuccessWithMsg(c, "攻击失败", respData)
}

// ListRoundAttacks 获取本回合攻击记录。
// GET /api/v1/ctf/ad-rounds/:id/attacks
func (h *BattleHandler) ListRoundAttacks(c *gin.Context) {
	roundID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.AdAttackListReq
	if !validator.BindQuery(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	respData, err := h.battleService.ListRoundAttacks(c.Request.Context(), sc, roundID, &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}

// SubmitDefense 提交补丁合约。
// POST /api/v1/ctf/ad-rounds/:id/defenses
func (h *BattleHandler) SubmitDefense(c *gin.Context) {
	roundID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.SubmitAdDefenseReq
	if !validator.BindJSON(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	respData, err := h.battleService.SubmitDefense(c.Request.Context(), sc, roundID, &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	if respData != nil && respData.IsAccepted {
		response.SuccessWithMsg(c, "补丁验证通过", respData)
		return
	}
	response.SuccessWithMsg(c, "补丁验证未通过", respData)
}

// ListRoundDefenses 获取本回合防守记录。
// GET /api/v1/ctf/ad-rounds/:id/defenses
func (h *BattleHandler) ListRoundDefenses(c *gin.Context) {
	roundID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.AdDefenseListReq
	if !validator.BindQuery(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	respData, err := h.battleService.ListRoundDefenses(c.Request.Context(), sc, roundID, &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}

// ListCompetitionTokenLedger 获取竞赛 Token 流水。
// GET /api/v1/ctf/competitions/:id/token-ledger
func (h *BattleHandler) ListCompetitionTokenLedger(c *gin.Context) {
	competitionID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.TokenLedgerListReq
	if !validator.BindQuery(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	respData, err := h.battleService.ListTokenLedgerByCompetition(c.Request.Context(), sc, competitionID, &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}

// ListTeamTokenLedger 获取团队 Token 流水。
// GET /api/v1/ctf/teams/:id/token-ledger
func (h *BattleHandler) ListTeamTokenLedger(c *gin.Context) {
	teamID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	var req dto.TokenLedgerListReq
	if !validator.BindQuery(c, &req) {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	respData, err := h.battleService.ListTokenLedgerByTeam(c.Request.Context(), sc, teamID, &req)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}

// GetTeamChain 获取队伍链信息。
// GET /api/v1/ctf/teams/:id/chain
func (h *BattleHandler) GetTeamChain(c *gin.Context) {
	teamID, ok := validator.ParsePathID(c, "id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	respData, err := h.battleService.GetTeamChain(c.Request.Context(), sc, teamID)
	if err != nil {
		handlerctx.HandleError(c, err)
		return
	}
	response.Success(c, respData)
}
