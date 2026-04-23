// ctf_realtime_publisher.go
// 模块05 — CTF竞赛：实时推送实现。
// 将 service 层发布的业务事件转换为 WebSocket 房间广播消息，
// 避免把协议拼装逻辑塞回 handler 或 cmd 装配层。

package ctf

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"github.com/lenschain/backend/internal/model/dto"
	svcctx "github.com/lenschain/backend/internal/pkg/context"
	wsmanager "github.com/lenschain/backend/internal/pkg/ws"
)

// RealtimePublisherAdapter 负责把模块05业务事件广播到对应 WebSocket 房间。
type RealtimePublisherAdapter struct {
	competitionService CompetitionService
}

// NewRealtimePublisherAdapter 创建模块05实时推送适配器。
func NewRealtimePublisherAdapter() *RealtimePublisherAdapter {
	return &RealtimePublisherAdapter{}
}

// SetCompetitionService 注入排行榜推送所需的竞赛服务。
func (a *RealtimePublisherAdapter) SetCompetitionService(service CompetitionService) {
	if a == nil {
		return
	}
	a.competitionService = service
}

// ctfRealtimeEnvelope 统一封装模块05实时消息格式。
type ctfRealtimeEnvelope struct {
	Type      string      `json:"type"`
	Channel   string      `json:"channel"`
	Data      interface{} `json:"data"`
	Timestamp string      `json:"timestamp"`
}

// PublishLeaderboardUpdate 广播排行榜变更消息。
func (a *RealtimePublisherAdapter) PublishLeaderboardUpdate(ctx context.Context, competitionID int64, groupID *int64, trigger *LeaderboardRealtimeTrigger) error {
	if a == nil || a.competitionService == nil || wsmanager.GetManager() == nil {
		return nil
	}
	req := &dto.LeaderboardReq{Top: 100}
	room := LeaderboardRoom(competitionID)
	if groupID != nil {
		req.GroupID = strconv.FormatInt(*groupID, 10)
	}
	resp, err := a.competitionService.GetLeaderboard(ctx, systemServiceContext(ctx), competitionID, req)
	if err != nil || resp == nil {
		return err
	}
	payload := map[string]interface{}{
		"event":          "rank_update",
		"competition_id": resp.CompetitionID,
		"is_frozen":      resp.IsFrozen,
		"rankings":       resp.Rankings,
	}
	if resp.GroupID != nil {
		payload["group_id"] = *resp.GroupID
	}
	if resp.GroupName != nil {
		payload["group_name"] = *resp.GroupName
	}
	if resp.CurrentRound != nil {
		payload["current_round"] = *resp.CurrentRound
	}
	if resp.TotalRounds != nil {
		payload["total_rounds"] = *resp.TotalRounds
	}
	if trigger != nil {
		payload["trigger"] = trigger
	}
	return broadcastCTFRealtime(room, "leaderboard", payload)
}

// PublishAnnouncement 广播公告消息。
func (a *RealtimePublisherAdapter) PublishAnnouncement(ctx context.Context, competitionID int64, payload *AnnouncementRealtimePayload) error {
	if payload == nil {
		return nil
	}
	return broadcastCTFRealtime(AnnouncementRoom(competitionID), "announcement", payload)
}

// PublishRoundPhaseChange 广播回合阶段切换消息。
func (a *RealtimePublisherAdapter) PublishRoundPhaseChange(ctx context.Context, competitionID, groupID int64, payload *RoundPhaseRealtimePayload) error {
	if payload == nil {
		return nil
	}
	return broadcastCTFRealtime(RoundRoom(competitionID, groupID), "round", payload)
}

// PublishAttackResult 广播攻击事件消息。
func (a *RealtimePublisherAdapter) PublishAttackResult(ctx context.Context, competitionID, groupID int64, payload *AttackRealtimePayload) error {
	if payload == nil {
		return nil
	}
	return broadcastCTFRealtime(AttackRoom(competitionID, groupID), "attacks", payload)
}

// broadcastCTFRealtime 将模块05事件编码为标准消息并广播到房间。
func broadcastCTFRealtime(room, channel string, data interface{}) error {
	manager := wsmanager.GetManager()
	if manager == nil || room == "" {
		return nil
	}
	payload, err := json.Marshal(ctfRealtimeEnvelope{
		Type:      "message",
		Channel:   channel,
		Data:      data,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
	if err != nil {
		return err
	}
	return manager.BroadcastRawToRoom(room, payload)
}

// systemServiceContext 构造模块内部实时推送使用的系统上下文。
// 该上下文只用于读取实时展示数据，不承载外部用户权限。
func systemServiceContext(ctx context.Context) *svcctx.ServiceContext {
	return &svcctx.ServiceContext{
		Ctx:   ctx,
		Roles: []string{"super_admin"},
	}
}
