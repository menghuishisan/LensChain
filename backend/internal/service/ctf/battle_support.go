// battle_support.go
// 模块05 — CTF竞赛：攻防对抗赛通用辅助。
// 负责回合构建、账本记录、队伍链响应与基础映射等非入口辅助逻辑。

package ctf

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/lenschain/backend/internal/model/dto"
	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/model/enum"
	"github.com/lenschain/backend/internal/pkg/snowflake"
)

// calculateAdTeamRank 计算分组内团队当前排名。
func (s *battleService) calculateAdTeamRank(ctx context.Context, groupID, teamID int64) int {
	teams, err := s.teamRepo.ListByAdGroupID(ctx, groupID)
	if err != nil {
		return 0
	}
	sort.SliceStable(teams, func(i, j int) bool {
		left := derefInt(teams[i].TokenBalance)
		right := derefInt(teams[j].TokenBalance)
		if left == right {
			return teams[i].ID < teams[j].ID
		}
		return left > right
	})
	for idx, team := range teams {
		if team.ID == teamID {
			return idx + 1
		}
	}
	return 0
}

// isFirstPatch 判断当前成功补丁是否为该漏洞首补丁。
func (s *battleService) isFirstPatch(ctx context.Context, competitionID, groupID, challengeID int64) (bool, error) {
	exists, err := s.adDefenseRepo.HasFirstPatch(ctx, competitionID, groupID, challengeID)
	if err != nil {
		return false, err
	}
	return !exists, nil
}

// buildAdRounds 构建分组全部回合。
func buildAdRounds(competition *entity.Competition, groupID int64, cfg *dto.ADCompetitionConfig) []*entity.AdRound {
	startAt := time.Now()
	if competition.StartAt != nil {
		startAt = *competition.StartAt
	}
	rounds := make([]*entity.AdRound, 0, cfg.TotalRounds)
	for roundNumber := 1; roundNumber <= cfg.TotalRounds; roundNumber++ {
		schedule := buildRoundSchedule(startAt, roundNumber, cfg.AttackDurationMinutes, cfg.DefenseDurationMinutes, 1)
		attackStartAt := schedule.RoundStartAt
		attackEndAt := schedule.AttackEndAt
		defenseStartAt := schedule.AttackEndAt
		defenseEndAt := schedule.DefenseEndAt
		settlementStartAt := schedule.DefenseEndAt
		settlementEndAt := schedule.SettlementEndAt
		phase := int16(enum.RoundPhaseAttacking)
		if roundNumber > 1 && competition.Status != enum.CompetitionStatusRunning {
			phase = int16(enum.RoundPhaseAttacking)
		}
		rounds = append(rounds, &entity.AdRound{
			ID:                snowflake.Generate(),
			CompetitionID:     competition.ID,
			GroupID:           groupID,
			RoundNumber:       roundNumber,
			Phase:             phase,
			AttackStartAt:     &attackStartAt,
			AttackEndAt:       &attackEndAt,
			DefenseStartAt:    &defenseStartAt,
			DefenseEndAt:      &defenseEndAt,
			SettlementStartAt: &settlementStartAt,
			SettlementEndAt:   &settlementEndAt,
			CreatedAt:         time.Now(),
		})
	}
	return rounds
}

// buildAttackLedgers 构建攻击成功后的多条 Token 流水。
func buildAttackLedgers(round *entity.AdRound, groupID int64, attack *entity.AdAttack, attackerBalanceAfter, targetBalanceAfter, stolenAmount, bonusAmount, firstBloodAmount int, targetTeamName, challengeTitle string) []*entity.AdTokenLedger {
	ledgers := []*entity.AdTokenLedger{
		{
			ID:              snowflake.Generate(),
			CompetitionID:   attack.CompetitionID,
			GroupID:         groupID,
			RoundID:         &round.ID,
			TeamID:          attack.AttackerTeamID,
			ChangeType:      enum.TokenChangeAttackSteal,
			Amount:          stolenAmount,
			BalanceAfter:    attackerBalanceAfter - bonusAmount - firstBloodAmount,
			RelatedAttackID: &attack.ID,
			Description:     stringPtr(fmt.Sprintf("攻击 %s 队伍的 %s 漏洞", targetTeamName, challengeTitle)),
			CreatedAt:       attack.CreatedAt,
		},
		{
			ID:              snowflake.Generate(),
			CompetitionID:   attack.CompetitionID,
			GroupID:         groupID,
			RoundID:         &round.ID,
			TeamID:          attack.AttackerTeamID,
			ChangeType:      enum.TokenChangeAttackBonus,
			Amount:          bonusAmount,
			BalanceAfter:    attackerBalanceAfter - firstBloodAmount,
			RelatedAttackID: &attack.ID,
			Description:     stringPtr("攻击奖励（窃取金额的5%）"),
			CreatedAt:       attack.CreatedAt,
		},
		{
			ID:              snowflake.Generate(),
			CompetitionID:   attack.CompetitionID,
			GroupID:         groupID,
			RoundID:         &round.ID,
			TeamID:          attack.TargetTeamID,
			ChangeType:      enum.TokenChangeAttackLoss,
			Amount:          -stolenAmount,
			BalanceAfter:    targetBalanceAfter,
			RelatedAttackID: &attack.ID,
			Description:     stringPtr(fmt.Sprintf("被 %s 漏洞攻击扣除 Token", challengeTitle)),
			CreatedAt:       attack.CreatedAt,
		},
	}
	if firstBloodAmount > 0 {
		ledgers = append(ledgers, &entity.AdTokenLedger{
			ID:              snowflake.Generate(),
			CompetitionID:   attack.CompetitionID,
			GroupID:         groupID,
			RoundID:         &round.ID,
			TeamID:          attack.AttackerTeamID,
			ChangeType:      enum.TokenChangeFirstBlood,
			Amount:          firstBloodAmount,
			BalanceAfter:    attackerBalanceAfter,
			RelatedAttackID: &attack.ID,
			Description:     stringPtr(fmt.Sprintf("首次攻破 %s 漏洞的额外奖励（10%%）", challengeTitle)),
			CreatedAt:       attack.CreatedAt,
		})
	}
	return ledgers
}

// buildDefenseLedger 构建补丁通过后的 Token 流水。
func buildDefenseLedger(round *entity.AdRound, groupID int64, defense *entity.AdDefense, balanceAfter int, challengeTitle string) *entity.AdTokenLedger {
	return &entity.AdTokenLedger{
		ID:               snowflake.Generate(),
		CompetitionID:    defense.CompetitionID,
		GroupID:          groupID,
		RoundID:          &round.ID,
		TeamID:           defense.TeamID,
		ChangeType:       enum.TokenChangeFirstPatch,
		Amount:           derefInt(defense.TokenReward),
		BalanceAfter:     balanceAfter,
		RelatedDefenseID: &defense.ID,
		Description:      stringPtr(fmt.Sprintf("首个修复 %s 漏洞的额外奖励", challengeTitle)),
		CreatedAt:        defense.CreatedAt,
	}
}

// resolvePatchRejectionReason 补齐补丁验证拒绝原因，保证 AC-31 响应可解释。
func resolvePatchRejectionReason(result *ADPatchVerificationResult, functionalityPassed, vulnerabilityFixed bool) *string {
	if functionalityPassed && vulnerabilityFixed {
		return nil
	}
	if result != nil && result.RejectionReason != nil && strings.TrimSpace(*result.RejectionReason) != "" {
		return result.RejectionReason
	}
	if !functionalityPassed {
		return stringPtr("功能完整性检查未通过")
	}
	if !vulnerabilityFixed {
		return stringPtr("漏洞修复验证失败")
	}
	return nil
}

// buildAdGroupNamespace 构建攻防赛分组命名空间。
func buildAdGroupNamespace(competitionID, groupID int64) string {
	return fmt.Sprintf("ctf-ad-%d-%d", competitionID, groupID)
}

// buildAdGroupName 根据顺序生成默认分组名称。
func buildAdGroupName(index int) string {
	return fmt.Sprintf("%c组", rune('A'+index-1))
}

// buildAdRoundDetailResp 构建回合详情响应。
func buildAdRoundDetailResp(round *entity.AdRound) *dto.AdRoundDetailResp {
	return &dto.AdRoundDetailResp{
		ID:                int64String(round.ID),
		CompetitionID:     int64String(round.CompetitionID),
		GroupID:           int64String(round.GroupID),
		RoundNumber:       round.RoundNumber,
		Phase:             round.Phase,
		PhaseText:         enum.GetRoundPhaseText(round.Phase),
		AttackStartAt:     optionalTimeString(round.AttackStartAt),
		AttackEndAt:       optionalTimeString(round.AttackEndAt),
		DefenseStartAt:    optionalTimeString(round.DefenseStartAt),
		DefenseEndAt:      optionalTimeString(round.DefenseEndAt),
		SettlementStartAt: optionalTimeString(round.SettlementStartAt),
		SettlementEndAt:   optionalTimeString(round.SettlementEndAt),
	}
}

// roundPhaseWindow 返回当前阶段的开始和结束时间。
func roundPhaseWindow(round *entity.AdRound) (*time.Time, *time.Time) {
	switch round.Phase {
	case enum.RoundPhaseAttacking:
		return round.AttackStartAt, round.AttackEndAt
	case enum.RoundPhaseDefending:
		return round.DefenseStartAt, round.DefenseEndAt
	case enum.RoundPhaseSettling:
		return round.SettlementStartAt, round.SettlementEndAt
	default:
		return round.SettlementStartAt, round.SettlementEndAt
	}
}

// buildTeamMap 将团队切片转换为按 ID 索引的映射。
func buildTeamMap(teams []*entity.Team) map[int64]*entity.Team {
	result := make(map[int64]*entity.Team, len(teams))
	for _, team := range teams {
		result[team.ID] = team
	}
	return result
}

// buildChallengeMap 将题目切片转换为按 ID 索引的映射。
func buildChallengeMap(challenges []*entity.Challenge) map[int64]*entity.Challenge {
	result := make(map[int64]*entity.Challenge, len(challenges))
	for _, challenge := range challenges {
		result[challenge.ID] = challenge
	}
	return result
}

// teamNameFromMap 从团队映射中读取名称。
func teamNameFromMap(teams map[int64]*entity.Team, teamID int64) string {
	if teams[teamID] == nil {
		return ""
	}
	return teams[teamID].Name
}

// challengeTitleFromMap 从题目映射中读取标题。
func challengeTitleFromMap(challenges map[int64]*entity.Challenge, challengeID int64) string {
	if challenges[challengeID] == nil {
		return ""
	}
	return challenges[challengeID].Title
}

// buildTeamChainResp 构建队伍链响应。
func buildTeamChainResp(chain *entity.AdTeamChain) (*dto.TeamChainResp, error) {
	contracts, err := decodeTeamChainContracts(chain.DeployedContracts)
	if err != nil {
		return nil, err
	}
	return &dto.TeamChainResp{
		ID:                  int64String(chain.ID),
		CompetitionID:       int64String(chain.CompetitionID),
		GroupID:             int64String(chain.GroupID),
		TeamID:              int64String(chain.TeamID),
		ChainRPCURL:         chain.ChainRPCURL,
		ChainWSURL:          chain.ChainWSURL,
		DeployedContracts:   contracts,
		CurrentPatchVersion: chain.CurrentPatchVersion,
		Status:              chain.Status,
		StatusText:          enum.GetAdTeamChainStatusText(chain.Status),
	}, nil
}

// decodeTeamChainContracts 解析队伍链部署合约列表。
func decodeTeamChainContracts(raw []byte) ([]dto.TeamChainContractItem, error) {
	contracts := []dto.TeamChainContractItem{}
	if len(raw) == 0 {
		return contracts, nil
	}
	if err := decodeJSON(raw, &contracts); err != nil {
		return nil, err
	}
	return contracts, nil
}

// markPatchedContract 标记队伍链中指定漏洞已修复，并递增补丁版本。
func markPatchedContract(items []dto.TeamChainContractItem, challengeID int64) []dto.TeamChainContractItem {
	targetID := int64String(challengeID)
	for idx := range items {
		if items[idx].ChallengeID == targetID {
			items[idx].IsPatched = true
			items[idx].PatchVersion++
		}
	}
	return items
}

// parseSnowflakeList 解析字符串 ID 列表。
func parseSnowflakeList(values []string) ([]int64, error) {
	result := make([]int64, 0, len(values))
	for _, value := range values {
		id, err := snowflake.ParseString(value)
		if err != nil {
			return nil, err
		}
		result = append(result, id)
	}
	return result, nil
}
