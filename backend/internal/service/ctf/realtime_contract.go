// realtime_contract.go
// 模块05 — CTF竞赛：实时推送契约与房间命名。
// 统一定义模块05在 service 层使用的实时事件发布接口和频道房间名，
// 避免 handler、service 和 cmd 装配层各自拼接协议细节。

package ctf

import (
	"context"
	"fmt"

	"github.com/lenschain/backend/internal/model/dto"
)

// CTFRealtimePublisher 定义模块05业务事件的实时发布能力。
// service 层只声明“何时发布什么事件”，具体 WebSocket 广播细节由装配层注入实现。
type CTFRealtimePublisher interface {
	PublishLeaderboardUpdate(ctx context.Context, competitionID int64, groupID *int64, trigger *LeaderboardRealtimeTrigger) error
	PublishAnnouncement(ctx context.Context, competitionID int64, payload *AnnouncementRealtimePayload) error
	PublishRoundPhaseChange(ctx context.Context, competitionID, groupID int64, payload *RoundPhaseRealtimePayload) error
	PublishAttackResult(ctx context.Context, competitionID, groupID int64, payload *AttackRealtimePayload) error
}

// LeaderboardRealtimeTrigger 描述触发排行榜刷新的业务事件来源。
type LeaderboardRealtimeTrigger struct {
	TeamName       string `json:"team_name"`
	ChallengeTitle string `json:"challenge_title"`
	IsFirstBlood   bool   `json:"is_first_blood"`
}

// AnnouncementRealtimePayload 描述公告频道的实时消息体。
type AnnouncementRealtimePayload struct {
	Event        string                  `json:"event"`
	Announcement dto.CtfAnnouncementItem `json:"announcement"`
}

// RoundPhaseRealtimePayload 描述回合阶段切换消息体。
type RoundPhaseRealtimePayload struct {
	Event                string                 `json:"event"`
	GroupID              string                 `json:"group_id"`
	RoundNumber          int                    `json:"round_number"`
	TotalRounds          int                    `json:"total_rounds"`
	Phase                int16                  `json:"phase"`
	PhaseText            string                 `json:"phase_text"`
	PhaseStartAt         string                 `json:"phase_start_at"`
	PhaseEndAt           string                 `json:"phase_end_at"`
	PreviousPhaseSummary map[string]interface{} `json:"previous_phase_summary,omitempty"`
}

// AttackRealtimeTeam 表示攻击事件中的团队摘要。
type AttackRealtimeTeam struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// AttackRealtimePayload 描述攻防赛攻击结果实时消息体。
type AttackRealtimePayload struct {
	Event            string                            `json:"event"`
	RoundNumber      int                               `json:"round_number"`
	AttackerTeam     AttackRealtimeTeam                `json:"attacker_team"`
	TargetTeam       AttackRealtimeTeam                `json:"target_team"`
	ChallengeTitle   string                            `json:"challenge_title"`
	IsSuccessful     bool                              `json:"is_successful"`
	IsFirstBlood     bool                              `json:"is_first_blood"`
	TokenReward      *int                              `json:"token_reward,omitempty"`
	AttackerBalance  *int                              `json:"attacker_balance,omitempty"`
	TargetBalance    *int                              `json:"target_balance,omitempty"`
	ErrorMessage     *string                           `json:"error_message,omitempty"`
	AssertionResults *dto.VerificationAssertionResults `json:"assertion_results,omitempty"`
}

// LeaderboardRoom 返回竞赛排行榜频道房间名。
func LeaderboardRoom(competitionID int64) string {
	return fmt.Sprintf("ctf:ws:leaderboard:%d", competitionID)
}

// AnnouncementRoom 返回竞赛公告频道房间名。
func AnnouncementRoom(competitionID int64) string {
	return fmt.Sprintf("ctf:ws:announcement:%d", competitionID)
}

// RoundRoom 返回攻防赛回合频道房间名。
func RoundRoom(competitionID, groupID int64) string {
	return fmt.Sprintf("ctf:ws:round:%d:%d", competitionID, groupID)
}

// AttackRoom 返回攻防赛攻击事件频道房间名。
func AttackRoom(competitionID, groupID int64) string {
	return fmt.Sprintf("ctf:ws:attacks:%d:%d", competitionID, groupID)
}
