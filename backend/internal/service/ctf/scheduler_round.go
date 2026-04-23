// scheduler_round.go
// 模块05 — CTF竞赛：攻防赛回合推进与结算调度。

package ctf

import (
	"context"
	"sort"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/lenschain/backend/internal/model/dto"
	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/model/enum"
	"github.com/lenschain/backend/internal/pkg/database"
	"github.com/lenschain/backend/internal/pkg/logger"
	"github.com/lenschain/backend/internal/pkg/snowflake"
	ctfrepo "github.com/lenschain/backend/internal/repository/ctf"
)

// RunADRoundAdvance 推进攻防赛回合的攻击、防守、结算三个阶段。
func (s *CTFScheduler) RunADRoundAdvance() {
	ctx := context.Background()
	now := time.Now().UTC()

	rounds, err := s.adRoundRepo.ListActivePhases(ctx, now)
	if err != nil {
		logger.L.Error("查询待推进攻防赛回合失败", zap.Error(err))
		return
	}
	for _, round := range rounds {
		if err := s.advanceRoundPhase(ctx, round); err != nil {
			logger.L.Error("推进攻防赛回合失败",
				zap.Int64("round_id", round.ID),
				zap.Int64("competition_id", round.CompetitionID),
				zap.Error(err),
			)
		}
	}
}

// advanceRoundPhase 根据当前回合阶段推进到下一阶段或完成结算。
func (s *CTFScheduler) advanceRoundPhase(ctx context.Context, round *entity.AdRound) error {
	if round == nil {
		return nil
	}
	switch round.Phase {
	case enum.RoundPhaseAttacking:
		if err := s.adRoundRepo.UpdateFields(ctx, round.ID, map[string]interface{}{
			"phase":      enum.RoundPhaseDefending,
			"updated_at": time.Now(),
		}); err != nil {
			return err
		}
		round.Phase = enum.RoundPhaseDefending
		s.publishRoundPhaseChange(ctx, round, nil)
		return nil
	case enum.RoundPhaseDefending:
		result, err := s.buildRoundSettlementResult(ctx, round)
		if err != nil {
			return err
		}
		if err := s.adRoundRepo.UpdateFields(ctx, round.ID, map[string]interface{}{
			"phase":             enum.RoundPhaseSettling,
			"settlement_result": mustJSON(result),
			"updated_at":        time.Now(),
		}); err != nil {
			return err
		}
		round.Phase = enum.RoundPhaseSettling
		s.publishRoundPhaseChange(ctx, round, result)
		return nil
	case enum.RoundPhaseSettling:
		return s.completeRoundSettlement(ctx, round)
	default:
		return nil
	}
}

// buildRoundSettlementResult 汇总当前回合的成功攻击、防守通过和防守奖励候选结果。
func (s *CTFScheduler) buildRoundSettlementResult(ctx context.Context, round *entity.AdRound) (map[string]interface{}, error) {
	group, err := s.adGroupRepo.GetByID(ctx, round.GroupID)
	if err != nil {
		return nil, err
	}
	competition, err := s.competitionRepo.GetByID(ctx, round.CompetitionID)
	if err != nil {
		return nil, err
	}
	cfg, err := loadADCompetitionConfig(competition)
	if err != nil {
		return nil, err
	}
	teams, err := s.teamRepo.ListByAdGroupID(ctx, group.ID)
	if err != nil {
		return nil, err
	}
	success := true
	attacks, _, err := s.adAttackRepo.List(ctx, &ctfrepo.AdAttackListParams{
		CompetitionID: round.CompetitionID,
		RoundID:       round.ID,
		GroupID:       group.ID,
		IsSuccessful:  &success,
		Page:          1,
		PageSize:      10000,
	})
	if err != nil {
		return nil, err
	}
	defenses, _, err := s.adDefenseRepo.List(ctx, &ctfrepo.AdDefenseListParams{
		CompetitionID: round.CompetitionID,
		RoundID:       round.ID,
		GroupID:       group.ID,
		Page:          1,
		PageSize:      10000,
	})
	if err != nil {
		return nil, err
	}

	attackedTeams := make(map[int64]struct{}, len(attacks))
	for _, attack := range attacks {
		attackedTeams[attack.TargetTeamID] = struct{}{}
	}
	rewardTeamIDs := make([]string, 0, len(teams))
	for _, team := range teams {
		if _, ok := attackedTeams[team.ID]; ok {
			continue
		}
		if cfg.DefenseRewardPerRound <= 0 {
			continue
		}
		rewardTeamIDs = append(rewardTeamIDs, int64String(team.ID))
	}
	sort.Strings(rewardTeamIDs)
	return map[string]interface{}{
		"successful_attack_count": len(attacks),
		"accepted_defense_count":  countAcceptedDefenses(defenses),
		"defense_reward":          cfg.DefenseRewardPerRound,
		"reward_team_ids":         rewardTeamIDs,
		"generated_at":            time.Now().UTC().Format(time.RFC3339),
	}, nil
}

// completeRoundSettlement 执行结算奖励发放，并推进到下一回合或完成整个分组。
func (s *CTFScheduler) completeRoundSettlement(ctx context.Context, round *entity.AdRound) error {
	group, err := s.adGroupRepo.GetByID(ctx, round.GroupID)
	if err != nil {
		return err
	}
	competition, err := s.competitionRepo.GetByID(ctx, round.CompetitionID)
	if err != nil {
		return err
	}
	cfg, err := loadADCompetitionConfig(competition)
	if err != nil {
		return err
	}
	teams, err := s.teamRepo.ListByAdGroupID(ctx, group.ID)
	if err != nil {
		return err
	}
	success := true
	attacks, _, err := s.adAttackRepo.List(ctx, &ctfrepo.AdAttackListParams{
		CompetitionID: round.CompetitionID,
		RoundID:       round.ID,
		GroupID:       group.ID,
		IsSuccessful:  &success,
		Page:          1,
		PageSize:      10000,
	})
	if err != nil {
		return err
	}
	attackedTeams := make(map[int64]struct{}, len(attacks))
	for _, attack := range attacks {
		attackedTeams[attack.TargetTeamID] = struct{}{}
	}

	nextRound, nextErr := s.adRoundRepo.GetByGroupAndNumber(ctx, group.ID, round.RoundNumber+1)
	hasNextRound := nextErr == nil && nextRound != nil
	updatedAt := time.Now()
	competitionFinished := false

	err = database.TransactionWithDB(ctx, s.db, func(tx *gorm.DB) error {
		txRoundRepo := ctfrepo.NewAdRoundRepository(tx)
		txGroupRepo := ctfrepo.NewAdGroupRepository(tx)
		txTeamRepo := ctfrepo.NewTeamRepository(tx)
		txLedgerRepo := ctfrepo.NewAdTokenLedgerRepository(tx)
		txCompetitionRepo := ctfrepo.NewCompetitionRepository(tx)
		txChainRepo := ctfrepo.NewAdTeamChainRepository(tx)

		if cfg.DefenseRewardPerRound > 0 {
			ledgers := make([]*entity.AdTokenLedger, 0, len(teams))
			for _, team := range teams {
				if _, ok := attackedTeams[team.ID]; ok {
					continue
				}
				newBalance := derefInt(team.TokenBalance) + cfg.DefenseRewardPerRound
				if err := txTeamRepo.UpdateTokenBalance(ctx, team.ID, newBalance); err != nil {
					return err
				}
				ledgers = append(ledgers, buildRoundDefenseRewardLedger(round, group.ID, team.ID, newBalance, cfg.DefenseRewardPerRound))
			}
			if err := txLedgerRepo.BatchCreate(ctx, ledgers); err != nil {
				return err
			}
		}

		if err := txRoundRepo.UpdateFields(ctx, round.ID, map[string]interface{}{
			"phase":      enum.RoundPhaseCompleted,
			"updated_at": updatedAt,
		}); err != nil {
			return err
		}

		if hasNextRound {
			return txRoundRepo.UpdateFields(ctx, nextRound.ID, map[string]interface{}{
				"phase":      enum.RoundPhaseAttacking,
				"updated_at": updatedAt,
			})
		}

		if err := txGroupRepo.UpdateStatus(ctx, group.ID, enum.AdGroupStatusFinished); err != nil {
			return err
		}
		groups, err := txGroupRepo.ListByCompetitionID(ctx, competition.ID)
		if err != nil {
			return err
		}
		competitionFinished = len(groups) > 0
		for _, item := range groups {
			if item.Status != enum.AdGroupStatusFinished {
				competitionFinished = false
				break
			}
		}
		if competitionFinished {
			if err := txCompetitionRepo.UpdateStatus(ctx, competition.ID, enum.CompetitionStatusEnded); err != nil {
				return err
			}
			if err := txChainRepo.StopByCompetitionID(ctx, competition.ID); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	if competitionFinished {
		if s.battleService != nil {
			if err := s.battleService.cleanupGroupRuntimesByCompetition(ctx, competition.ID); err != nil {
				return err
			}
		}
		writeCompetitionStatusCache(ctx, competition.ID, enum.CompetitionStatusEnded)
		if err := s.persistFinalArtifacts(ctx, competition); err != nil {
			return err
		}
		s.publishCompetitionLeaderboard(ctx, competition.ID, nil)
		return nil
	}
	if hasNextRound {
		nextRound.Phase = enum.RoundPhaseAttacking
		s.publishRoundPhaseChange(ctx, nextRound, nil)
	}
	for _, team := range teams {
		if team == nil {
			continue
		}
		balance := derefInt(team.TokenBalance)
		if _, ok := attackedTeams[team.ID]; !ok && cfg.DefenseRewardPerRound > 0 {
			balance += cfg.DefenseRewardPerRound
		}
		writeAdTokenBalanceCache(ctx, competition.ID, team.ID, balance)
	}
	s.publishCompetitionLeaderboard(ctx, competition.ID, &group.ID)
	return nil
}

// loadADCompetitionConfig 解析攻防赛配置，并补齐文档默认值。
func loadADCompetitionConfig(competition *entity.Competition) (*dto.ADCompetitionConfig, error) {
	cfg := &dto.ADCompetitionConfig{
		TotalRounds:              10,
		AttackDurationMinutes:    10,
		DefenseDurationMinutes:   5,
		InitialToken:             10000,
		AttackBonusRatio:         0.05,
		DefenseRewardPerRound:    50,
		FirstPatchBonus:          200,
		FirstBloodBonusRatio:     0.10,
		VulnerabilityDecayFactor: 0.80,
		MaxTeamsPerGroup:         4,
	}
	if competition == nil {
		return cfg, nil
	}
	if len(competition.AdConfig) > 0 {
		if err := decodeJSON(competition.AdConfig, cfg); err != nil {
			return nil, err
		}
	}
	if cfg.TotalRounds <= 0 {
		cfg.TotalRounds = 10
	}
	if cfg.AttackDurationMinutes <= 0 {
		cfg.AttackDurationMinutes = 10
	}
	if cfg.DefenseDurationMinutes <= 0 {
		cfg.DefenseDurationMinutes = 5
	}
	if cfg.InitialToken <= 0 {
		cfg.InitialToken = 10000
	}
	if cfg.AttackBonusRatio <= 0 {
		cfg.AttackBonusRatio = 0.05
	}
	if cfg.DefenseRewardPerRound <= 0 {
		cfg.DefenseRewardPerRound = 50
	}
	if cfg.FirstPatchBonus <= 0 {
		cfg.FirstPatchBonus = 200
	}
	if cfg.FirstBloodBonusRatio <= 0 {
		cfg.FirstBloodBonusRatio = 0.10
	}
	if cfg.VulnerabilityDecayFactor <= 0 {
		cfg.VulnerabilityDecayFactor = 0.80
	}
	if cfg.MaxTeamsPerGroup <= 0 {
		cfg.MaxTeamsPerGroup = 4
	}
	return cfg, nil
}

// buildRoundDefenseRewardLedger 构建“本回合未被攻破”团队的防守奖励流水。
func buildRoundDefenseRewardLedger(round *entity.AdRound, groupID, teamID int64, balanceAfter, reward int) *entity.AdTokenLedger {
	return &entity.AdTokenLedger{
		ID:            snowflake.Generate(),
		CompetitionID: round.CompetitionID,
		GroupID:       groupID,
		RoundID:       &round.ID,
		TeamID:        teamID,
		ChangeType:    enum.TokenChangeDefenseReward,
		Amount:        reward,
		BalanceAfter:  balanceAfter,
		Description:   stringPtr("本回合未被成功攻击，获得防守奖励"),
		CreatedAt:     time.Now(),
	}
}

// countAcceptedDefenses 统计回合中通过验证的补丁数量。
func countAcceptedDefenses(defenses []*entity.AdDefense) int {
	total := 0
	for _, defense := range defenses {
		if defense.IsAccepted {
			total++
		}
	}
	return total
}

// publishRoundPhaseChange 在回合阶段切换后广播阶段更新。
func (s *CTFScheduler) publishRoundPhaseChange(ctx context.Context, round *entity.AdRound, previousSummary map[string]interface{}) {
	if s.realtimePublisher == nil || round == nil {
		return
	}
	group, err := s.adGroupRepo.GetByID(ctx, round.GroupID)
	if err != nil || group == nil {
		return
	}
	competition, err := s.competitionRepo.GetByID(ctx, round.CompetitionID)
	if err != nil || competition == nil {
		return
	}
	cfg, err := loadADCompetitionConfig(competition)
	if err != nil {
		return
	}
	phaseStart, phaseEnd := roundPhaseWindow(round)
	if phaseStart == nil || phaseEnd == nil {
		return
	}
	writeAdRoundCache(ctx, round.CompetitionID, round.GroupID, round.RoundNumber, round.Phase, *phaseEnd)
	_ = s.realtimePublisher.PublishRoundPhaseChange(ctx, round.CompetitionID, round.GroupID, &RoundPhaseRealtimePayload{
		Event:                "phase_change",
		GroupID:              int64String(group.ID),
		RoundNumber:          round.RoundNumber,
		TotalRounds:          cfg.TotalRounds,
		Phase:                round.Phase,
		PhaseText:            enum.GetRoundPhaseText(round.Phase),
		PhaseStartAt:         timeString(*phaseStart),
		PhaseEndAt:           timeString(*phaseEnd),
		PreviousPhaseSummary: previousSummary,
	})
}

// publishInitialGroupRounds 在攻防赛开始时推送各分组首轮状态，避免在线选手只能等待下一次阶段切换。
func (s *CTFScheduler) publishInitialGroupRounds(ctx context.Context, competitionID int64) {
	if s.realtimePublisher == nil {
		return
	}
	groups, err := s.adGroupRepo.ListByCompetitionID(ctx, competitionID)
	if err != nil {
		return
	}
	for _, group := range groups {
		round, roundErr := s.adRoundRepo.GetCurrentByGroupID(ctx, group.ID)
		if roundErr != nil || round == nil {
			continue
		}
		s.publishRoundPhaseChange(ctx, round, nil)
	}
}
