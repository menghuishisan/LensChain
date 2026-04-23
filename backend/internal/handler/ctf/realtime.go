// realtime.go
// 模块05 — CTF竞赛：WebSocket 处理层。
// 负责排行榜、公告、攻防回合和攻击事件的实时订阅与状态推送。

package ctf

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/lenschain/backend/internal/model/dto"
	"github.com/lenschain/backend/internal/model/enum"
	svcctx "github.com/lenschain/backend/internal/pkg/context"
	"github.com/lenschain/backend/internal/pkg/errcode"
	"github.com/lenschain/backend/internal/pkg/handlerctx"
	"github.com/lenschain/backend/internal/pkg/response"
	"github.com/lenschain/backend/internal/pkg/validator"
	wsmanager "github.com/lenschain/backend/internal/pkg/ws"
	svc "github.com/lenschain/backend/internal/service/ctf"
)

type realtimeCompetitionReader interface {
	Get(ctx context.Context, sc *svcctx.ServiceContext, id int64) (*dto.CompetitionDetailResp, error)
	GetLeaderboard(ctx context.Context, sc *svcctx.ServiceContext, competitionID int64, req *dto.LeaderboardReq) (*dto.LeaderboardResp, error)
}

type realtimeTeamReader interface {
	GetMyRegistration(ctx context.Context, sc *svcctx.ServiceContext, competitionID int64) (*dto.MyRegistrationResp, error)
}

type realtimeBattleReader interface {
	GetCurrentRound(ctx context.Context, sc *svcctx.ServiceContext, groupID int64) (*dto.CurrentRoundResp, error)
	GetTeamChain(ctx context.Context, sc *svcctx.ServiceContext, teamID int64) (*dto.TeamChainResp, error)
}

// RealtimeHandler CTF 实时推送处理器。
type RealtimeHandler struct {
	competitionService realtimeCompetitionReader
	teamService        realtimeTeamReader
	battleService      realtimeBattleReader
}

// ctfInboundMessage 表示客户端发来的订阅或心跳消息。
type ctfInboundMessage struct {
	Type    string                 `json:"type"`
	Channel string                 `json:"channel"`
	Params  map[string]interface{} `json:"params"`
}

// ctfOutboundMessage 表示服务端推送的标准消息结构。
type ctfOutboundMessage struct {
	Type      string      `json:"type"`
	Channel   string      `json:"channel"`
	Data      interface{} `json:"data"`
	Timestamp string      `json:"timestamp"`
}

// NewRealtimeHandler 创建 CTF 实时推送处理器。
func NewRealtimeHandler(
	competitionService svc.CompetitionService,
	teamService svc.TeamService,
	battleService svc.BattleService,
) *RealtimeHandler {
	return &RealtimeHandler{
		competitionService: competitionService,
		teamService:        teamService,
		battleService:      battleService,
	}
}

// ServeWS 建立 CTF WebSocket 连接。
// GET /api/v1/ctf/ws
func (h *RealtimeHandler) ServeWS(c *gin.Context) {
	competitionID, ok := parseQuerySnowflakeID(c, "competition_id")
	if !ok {
		return
	}
	sc := handlerctx.BuildServiceContext(c)
	if _, err := h.competitionService.Get(c.Request.Context(), sc, competitionID); err != nil {
		handlerctx.HandleError(c, err)
		return
	}

	client, manager, ok := upgradeCTFWS(c)
	if !ok {
		return
	}
	go client.WritePump()
	callCtx := websocketServiceContext(c)
	subscriptions, err := h.resolveAutoSubscriptions(callCtx, sc, competitionID)
	if err != nil {
		manager.Unregister(client)
		_ = client.Conn.Close()
		handlerctx.HandleError(c, err)
		return
	}
	h.readLoop(c, client, manager, competitionID, sc, subscriptions)
}

// readLoop 处理订阅指令、心跳和首次快照推送。
func (h *RealtimeHandler) readLoop(c *gin.Context, client *wsmanager.Client, manager *wsmanager.Manager, competitionID int64, sc *svcctx.ServiceContext, subscriptions map[string]map[string]string) {
	defer func() {
		manager.Unregister(client)
		_ = client.Conn.Close()
	}()

	ctx := websocketServiceContext(c)
	client.Conn.SetReadLimit(4096)
	refreshCTFWSReadDeadline(client)
	client.Conn.SetPongHandler(func(string) error {
		refreshCTFWSReadDeadline(client)
		return nil
	})

	if subscriptions == nil {
		subscriptions = map[string]map[string]string{}
	}
	h.applyInitialSubscriptions(ctx, client, manager, sc, competitionID, subscriptions)

	for {
		_, payload, err := client.Conn.ReadMessage()
		if err != nil {
			return
		}
		// 文档约定客户端发送的是应用层 JSON ping，而不是只依赖 WebSocket 原生 Pong。
		// 因此每次收到任意业务消息后都要刷新读超时，避免客户端按协议保活却被服务端误断开。
		refreshCTFWSReadDeadline(client)
		var inbound ctfInboundMessage
		if err := json.Unmarshal(payload, &inbound); err != nil {
			continue
		}
		switch inbound.Type {
		case "ping":
			h.pushPong(client)
		case "subscribe":
			h.applySubscription(ctx, client, manager, sc, subscriptions, inbound, competitionID)
		}
	}
}

func (h *RealtimeHandler) resolveAutoSubscriptions(ctx context.Context, sc *svcctx.ServiceContext, competitionID int64) (map[string]map[string]string, error) {
	subscriptions := map[string]map[string]string{
		"leaderboard":  {"competition_id": formatID(competitionID)},
		"announcement": {"competition_id": formatID(competitionID)},
	}
	competition, err := h.competitionService.Get(ctx, sc, competitionID)
	if err != nil || competition == nil {
		return subscriptions, err
	}
	if competition.CompetitionType != enum.CompetitionTypeAttackDefense || h.teamService == nil || h.battleService == nil || !sc.IsStudent() {
		return subscriptions, nil
	}
	registration, err := h.teamService.GetMyRegistration(ctx, sc, competitionID)
	if err != nil || registration == nil || !registration.IsRegistered || registration.TeamID == nil {
		return subscriptions, nil
	}
	teamID, err := validator.ParseSnowflakeID(*registration.TeamID)
	if err != nil {
		return subscriptions, nil
	}
	teamChain, err := h.battleService.GetTeamChain(ctx, sc, teamID)
	if err != nil || teamChain == nil || teamChain.GroupID == "" {
		return subscriptions, nil
	}
	subscriptions["round"] = map[string]string{
		"competition_id": formatID(competitionID),
		"group_id":       teamChain.GroupID,
	}
	subscriptions["attacks"] = map[string]string{
		"competition_id": formatID(competitionID),
		"group_id":       teamChain.GroupID,
	}
	return subscriptions, nil
}

func (h *RealtimeHandler) applyInitialSubscriptions(
	ctx context.Context,
	client *wsmanager.Client,
	manager *wsmanager.Manager,
	sc *svcctx.ServiceContext,
	competitionID int64,
	subscriptions map[string]map[string]string,
) {
	if params, ok := subscriptions["leaderboard"]; ok {
		manager.JoinRoom(client, svc.LeaderboardRoom(competitionID))
		h.pushSnapshot(ctx, client, sc, competitionID, "leaderboard", params)
	}
	if params, ok := subscriptions["announcement"]; ok {
		manager.JoinRoom(client, svc.AnnouncementRoom(competitionID))
		h.pushSnapshot(ctx, client, sc, competitionID, "announcement", params)
	}
	if params, ok := subscriptions["round"]; ok && params["group_id"] != "" {
		if groupID, err := validator.ParseSnowflakeID(params["group_id"]); err == nil {
			manager.JoinRoom(client, svc.RoundRoom(competitionID, groupID))
			h.pushSnapshot(ctx, client, sc, competitionID, "round", params)
		}
	}
	if params, ok := subscriptions["attacks"]; ok && params["group_id"] != "" {
		if groupID, err := validator.ParseSnowflakeID(params["group_id"]); err == nil {
			manager.JoinRoom(client, svc.AttackRoom(competitionID, groupID))
			h.pushSnapshot(ctx, client, sc, competitionID, "attacks", params)
		}
	}
}

// applySubscription 根据客户端订阅消息更新订阅集合。
func (h *RealtimeHandler) applySubscription(
	ctx context.Context,
	client *wsmanager.Client,
	manager *wsmanager.Manager,
	sc *svcctx.ServiceContext,
	subscriptions map[string]map[string]string,
	inbound ctfInboundMessage,
	competitionID int64,
) {
	switch inbound.Channel {
	case "leaderboard", "announcement":
		subscriptions[inbound.Channel] = map[string]string{
			"competition_id": formatID(competitionID),
		}
		h.pushSnapshot(ctx, client, sc, competitionID, inbound.Channel, subscriptions[inbound.Channel])
	case "round", "attacks":
		groupID := stringParam(inbound.Params, "group_id")
		if groupID == "" {
			return
		}
		parsedGroupID, err := validator.ParseSnowflakeID(groupID)
		if err != nil {
			return
		}
		if _, readErr := h.battleService.GetCurrentRound(ctx, sc, parsedGroupID); readErr != nil {
			return
		}
		if existing, ok := subscriptions[inbound.Channel]; ok && existing["group_id"] != "" {
			if currentGroupID, parseErr := validator.ParseSnowflakeID(existing["group_id"]); parseErr == nil {
				if inbound.Channel == "round" {
					manager.LeaveRoom(client, svc.RoundRoom(competitionID, currentGroupID))
				} else {
					manager.LeaveRoom(client, svc.AttackRoom(competitionID, currentGroupID))
				}
			}
		}
		subscriptions[inbound.Channel] = map[string]string{
			"competition_id": formatID(competitionID),
			"group_id":       groupID,
		}
		if inbound.Channel == "round" {
			manager.JoinRoom(client, svc.RoundRoom(competitionID, parsedGroupID))
		} else {
			manager.JoinRoom(client, svc.AttackRoom(competitionID, parsedGroupID))
		}
		h.pushSnapshot(ctx, client, sc, competitionID, inbound.Channel, subscriptions[inbound.Channel])
	}
}

// pushSnapshot 在首次连接和订阅切换时推送一次最新快照。
func (h *RealtimeHandler) pushSnapshot(
	ctx context.Context,
	client *wsmanager.Client,
	sc *svcctx.ServiceContext,
	competitionID int64,
	channel string,
	params map[string]string,
) {
	switch channel {
	case "leaderboard":
		req := &dto.LeaderboardReq{Top: 100}
		if params["group_id"] != "" {
			req.GroupID = params["group_id"]
		}
		respData, err := h.competitionService.GetLeaderboard(ctx, sc, competitionID, req)
		if err == nil && respData != nil {
			h.pushMessage(client, "leaderboard", map[string]interface{}{
				"event":          "rank_update",
				"competition_id": respData.CompetitionID,
				"is_frozen":      respData.IsFrozen,
				"rankings":       respData.Rankings,
			})
		}
	case "announcement":
		// 公告频道的文档事件只定义“new_announcement”实时推送；
		// 历史公告由 REST 列表接口获取，这里不再发送额外的自定义快照事件。
	case "round":
		groupID, err := validator.ParseSnowflakeID(params["group_id"])
		if err == nil {
			respData, roundErr := h.battleService.GetCurrentRound(ctx, sc, groupID)
			if roundErr == nil && respData != nil {
				h.pushMessage(client, "round", map[string]interface{}{
					"event":          "phase_change",
					"group_id":       respData.GroupID,
					"round_number":   respData.RoundNumber,
					"total_rounds":   respData.TotalRounds,
					"phase":          respData.Phase,
					"phase_text":     respData.PhaseText,
					"phase_start_at": respData.PhaseStartAt,
					"phase_end_at":   respData.PhaseEndAt,
				})
			}
		}
	case "attacks":
		// 攻击事件频道只推送实时 attack_result，历史记录由 REST 接口查询。
	}
}

// pushMessage 向客户端发送标准 CTF WebSocket 消息。
func (h *RealtimeHandler) pushMessage(client *wsmanager.Client, channel string, data interface{}) {
	payload, err := json.Marshal(ctfOutboundMessage{
		Type:      "message",
		Channel:   channel,
		Data:      data,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
	if err != nil {
		return
	}
	client.Send <- payload
}

// pushPong 响应客户端心跳。
func (h *RealtimeHandler) pushPong(client *wsmanager.Client) {
	payload, err := json.Marshal(map[string]string{"type": "pong"})
	if err != nil {
		return
	}
	client.Send <- payload
}

// upgradeCTFWS 执行 WebSocket 升级并注册连接。
func upgradeCTFWS(c *gin.Context) (*wsmanager.Client, *wsmanager.Manager, bool) {
	conn, err := wsmanager.Upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return nil, nil, false
	}
	manager := wsmanager.GetManager()
	client := wsmanager.NewClient(handlerctx.BuildServiceContext(c).UserID, conn)
	manager.Register(client)
	return client, manager, true
}

// websocketServiceContext 返回 WebSocket 生命周期内可安全复用的上下文。
func websocketServiceContext(c *gin.Context) context.Context {
	return context.WithoutCancel(c.Request.Context())
}

// refreshCTFWSReadDeadline 刷新模块05实时连接的读超时窗口。
// 模块05同时支持原生 WebSocket ping/pong 和文档定义的应用层 JSON 心跳，
// 因此每次收到客户端消息或 Pong 帧后都统一延长 60 秒超时。
func refreshCTFWSReadDeadline(client *wsmanager.Client) {
	if client == nil || client.Conn == nil {
		return
	}
	client.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
}

// stringParam 从通用参数字典中读取字符串字段。
func stringParam(params map[string]interface{}, key string) string {
	if len(params) == 0 {
		return ""
	}
	value, ok := params[key]
	if !ok || value == nil {
		return ""
	}
	if text, ok := value.(string); ok {
		return text
	}
	return ""
}

// parseQuerySnowflakeID 解析查询参数中的雪花 ID。
func parseQuerySnowflakeID(c *gin.Context, key string) (int64, bool) {
	raw := c.Query(key)
	if raw == "" {
		response.Error(c, errcode.ErrInvalidParams.WithMessage(key+" 不能为空"))
		return 0, false
	}
	value, err := validator.ParseSnowflakeID(raw)
	if err != nil {
		response.Error(c, errcode.ErrInvalidParams.WithMessage(key+" 格式错误"))
		return 0, false
	}
	return value, true
}

// formatID 将 int64 ID 格式化为字符串，满足 WebSocket 协议的字符串 ID 约定。
func formatID(id int64) string {
	return strconv.FormatInt(id, 10)
}
