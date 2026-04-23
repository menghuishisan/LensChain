// battle_listing.go
// 模块05 — CTF竞赛：攻防对抗赛列表与详情组装。
// 负责分组、攻击、防守、Token 流水等查询接口的参数构建与响应聚合。

package ctf

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/lenschain/backend/internal/model/dto"
	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/model/enum"
	svcctx "github.com/lenschain/backend/internal/pkg/context"
	"github.com/lenschain/backend/internal/pkg/errcode"
	"github.com/lenschain/backend/internal/pkg/snowflake"
	ctfrepo "github.com/lenschain/backend/internal/repository/ctf"
)

// buildAdGroupResp 构建分组响应。
func (s *battleService) buildAdGroupResp(ctx context.Context, group *entity.AdGroup) (*dto.AdGroupResp, error) {
	teams, err := s.teamRepo.ListByAdGroupID(ctx, group.ID)
	if err != nil {
		return nil, err
	}
	items := make([]dto.CTFTeamBrief, 0, len(teams))
	for _, team := range teams {
		items = append(items, dto.CTFTeamBrief{
			ID:   int64String(team.ID),
			Name: team.Name,
		})
	}
	return &dto.AdGroupResp{
		ID:            int64String(group.ID),
		CompetitionID: int64String(group.CompetitionID),
		GroupName:     group.GroupName,
		Namespace:     group.Namespace,
		Status:        group.Status,
		StatusText:    enum.GetAdGroupStatusText(group.Status),
		Teams:         items,
	}, nil
}

// loadRoundActionContext 加载攻击或防守动作的公共上下文。
func (s *battleService) loadRoundActionContext(ctx context.Context, sc *svcctx.ServiceContext, roundID int64) (*entity.AdRound, *entity.AdGroup, *entity.Competition, *dto.ADCompetitionConfig, *entity.Team, error) {
	round, err := s.adRoundRepo.GetByID(ctx, roundID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, nil, nil, nil, errcode.ErrAdRoundNotFound
		}
		return nil, nil, nil, nil, nil, err
	}
	group, err := s.adGroupRepo.GetByID(ctx, round.GroupID)
	if err != nil {
		return nil, nil, nil, nil, nil, errcode.ErrAdGroupNotFound
	}
	competition, err := getCompetition(ctx, s.competitionRepo, round.CompetitionID)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	cfg, err := s.loadADConfig(competition)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	_, team, err := s.getCurrentMemberTeam(ctx, competition.ID, sc.UserID)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	if team.AdGroupID == nil || *team.AdGroupID != group.ID {
		return nil, nil, nil, nil, nil, errcode.ErrAdCrossGroupForbidden
	}
	return round, group, competition, cfg, team, nil
}

// buildAdAttackListParams 组装攻击记录查询参数。
func (s *battleService) buildAdAttackListParams(competitionID, roundID, groupID int64, req *dto.AdAttackListReq) (*ctfrepo.AdAttackListParams, error) {
	params := &ctfrepo.AdAttackListParams{
		CompetitionID: competitionID,
		RoundID:       roundID,
		GroupID:       groupID,
		Page:          req.Page,
		PageSize:      req.PageSize,
	}
	if req.ChallengeID != "" {
		challengeID, err := snowflake.ParseString(req.ChallengeID)
		if err != nil {
			return nil, errcode.ErrInvalidID
		}
		params.ChallengeID = challengeID
	}
	if req.TeamID != "" {
		teamID, err := snowflake.ParseString(req.TeamID)
		if err != nil {
			return nil, errcode.ErrInvalidID
		}
		params.AttackerTeamID = teamID
	}
	return params, nil
}

// listAttacks 查询并组装攻击记录响应。
func (s *battleService) listAttacks(ctx context.Context, params *ctfrepo.AdAttackListParams, page, pageSize int) (*dto.AdAttackListResp, error) {
	attacks, total, err := s.adAttackRepo.List(ctx, params)
	if err != nil {
		return nil, err
	}
	teams, challenges, err := s.loadAttackRelatedData(ctx, attacks)
	if err != nil {
		return nil, err
	}
	items := make([]dto.AdAttackListItem, 0, len(attacks))
	for _, attack := range attacks {
		items = append(items, dto.AdAttackListItem{
			ID:               int64String(attack.ID),
			AttackerTeamID:   int64String(attack.AttackerTeamID),
			AttackerTeamName: teamNameFromMap(teams, attack.AttackerTeamID),
			TargetTeamID:     int64String(attack.TargetTeamID),
			TargetTeamName:   teamNameFromMap(teams, attack.TargetTeamID),
			ChallengeID:      int64String(attack.ChallengeID),
			ChallengeTitle:   challengeTitleFromMap(challenges, attack.ChallengeID),
			IsSuccessful:     attack.IsSuccessful,
			TokenReward:      attack.TokenReward,
			IsFirstBlood:     attack.IsFirstBlood,
			CreatedAt:        timeString(attack.CreatedAt),
		})
	}
	return &dto.AdAttackListResp{
		List:       items,
		Pagination: paginationResp(page, pageSize, total),
	}, nil
}

// buildAdDefenseListParams 组装防守记录查询参数。
func (s *battleService) buildAdDefenseListParams(competitionID, roundID, groupID int64, req *dto.AdDefenseListReq) (*ctfrepo.AdDefenseListParams, error) {
	params := &ctfrepo.AdDefenseListParams{
		CompetitionID: competitionID,
		RoundID:       roundID,
		GroupID:       groupID,
		Page:          req.Page,
		PageSize:      req.PageSize,
	}
	if req.ChallengeID != "" {
		challengeID, err := snowflake.ParseString(req.ChallengeID)
		if err != nil {
			return nil, errcode.ErrInvalidID
		}
		params.ChallengeID = challengeID
	}
	if req.TeamID != "" {
		teamID, err := snowflake.ParseString(req.TeamID)
		if err != nil {
			return nil, errcode.ErrInvalidID
		}
		params.TeamID = teamID
	}
	return params, nil
}

// listDefenses 查询并组装防守记录响应。
func (s *battleService) listDefenses(ctx context.Context, params *ctfrepo.AdDefenseListParams, page, pageSize int) (*dto.AdDefenseListResp, error) {
	defenses, total, err := s.adDefenseRepo.List(ctx, params)
	if err != nil {
		return nil, err
	}
	teams, challenges, err := s.loadDefenseRelatedData(ctx, defenses)
	if err != nil {
		return nil, err
	}
	items := make([]dto.AdDefenseListItem, 0, len(defenses))
	for _, defense := range defenses {
		items = append(items, dto.AdDefenseListItem{
			ID:                  int64String(defense.ID),
			TeamID:              int64String(defense.TeamID),
			TeamName:            teamNameFromMap(teams, defense.TeamID),
			ChallengeID:         int64String(defense.ChallengeID),
			ChallengeTitle:      challengeTitleFromMap(challenges, defense.ChallengeID),
			IsAccepted:          defense.IsAccepted,
			FunctionalityPassed: defense.FunctionalityPassed,
			VulnerabilityFixed:  defense.VulnerabilityFixed,
			IsFirstPatch:        defense.IsFirstPatch,
			TokenReward:         defense.TokenReward,
			CreatedAt:           timeString(defense.CreatedAt),
		})
	}
	return &dto.AdDefenseListResp{
		List:       items,
		Pagination: paginationResp(page, pageSize, total),
	}, nil
}

// buildLedgerListParams 组装 Token 流水查询参数。
func (s *battleService) buildLedgerListParams(competitionID, teamID int64, req *dto.TokenLedgerListReq) (*ctfrepo.AdTokenLedgerListParams, error) {
	params := &ctfrepo.AdTokenLedgerListParams{
		CompetitionID: competitionID,
		TeamID:        teamID,
		ChangeType:    req.ChangeType,
		Page:          req.Page,
		PageSize:      req.PageSize,
	}
	if req.RoundID != "" {
		roundID, err := snowflake.ParseString(req.RoundID)
		if err != nil {
			return nil, errcode.ErrInvalidID
		}
		params.RoundID = roundID
	}
	return params, nil
}

// listLedgers 查询并组装 Token 流水响应。
func (s *battleService) listLedgers(ctx context.Context, params *ctfrepo.AdTokenLedgerListParams, page, pageSize int) (*dto.TokenLedgerListResp, error) {
	ledgers, total, err := s.adLedgerRepo.List(ctx, params)
	if err != nil {
		return nil, err
	}
	roundIDs := make([]int64, 0, len(ledgers))
	for _, ledger := range ledgers {
		if ledger.RoundID != nil {
			roundIDs = append(roundIDs, *ledger.RoundID)
		}
	}
	roundMap, err := s.loadRoundsMap(ctx, roundIDs)
	if err != nil {
		return nil, err
	}
	items := make([]dto.TokenLedgerListItem, 0, len(ledgers))
	for _, ledger := range ledgers {
		var roundNumber *int
		if ledger.RoundID != nil && roundMap[*ledger.RoundID] != nil {
			roundNumber = intPtr(roundMap[*ledger.RoundID].RoundNumber)
		}
		var relatedAttackID *string
		if ledger.RelatedAttackID != nil {
			value := int64String(*ledger.RelatedAttackID)
			relatedAttackID = &value
		}
		items = append(items, dto.TokenLedgerListItem{
			ID:              int64String(ledger.ID),
			RoundNumber:     roundNumber,
			ChangeType:      ledger.ChangeType,
			ChangeTypeText:  enum.GetTokenChangeTypeText(ledger.ChangeType),
			Amount:          ledger.Amount,
			BalanceAfter:    ledger.BalanceAfter,
			Description:     ledger.Description,
			RelatedAttackID: relatedAttackID,
			CreatedAt:       timeString(ledger.CreatedAt),
		})
	}
	return &dto.TokenLedgerListResp{
		List:       items,
		Pagination: paginationResp(page, pageSize, total),
	}, nil
}

// loadAttackRelatedData 批量加载攻击记录依赖的团队和题目。
func (s *battleService) loadAttackRelatedData(ctx context.Context, attacks []*entity.AdAttack) (map[int64]*entity.Team, map[int64]*entity.Challenge, error) {
	teamIDs := make([]int64, 0, len(attacks)*2)
	challengeIDs := make([]int64, 0, len(attacks))
	for _, attack := range attacks {
		teamIDs = append(teamIDs, attack.AttackerTeamID, attack.TargetTeamID)
		challengeIDs = append(challengeIDs, attack.ChallengeID)
	}
	teams, err := s.teamRepo.ListByIDs(ctx, teamIDs)
	if err != nil {
		return nil, nil, err
	}
	challenges, err := s.challengeRepo.GetByIDs(ctx, challengeIDs)
	if err != nil {
		return nil, nil, err
	}
	return buildTeamMap(teams), buildChallengeMap(challenges), nil
}

// loadDefenseRelatedData 批量加载防守记录依赖的团队和题目。
func (s *battleService) loadDefenseRelatedData(ctx context.Context, defenses []*entity.AdDefense) (map[int64]*entity.Team, map[int64]*entity.Challenge, error) {
	teamIDs := make([]int64, 0, len(defenses))
	challengeIDs := make([]int64, 0, len(defenses))
	for _, defense := range defenses {
		teamIDs = append(teamIDs, defense.TeamID)
		challengeIDs = append(challengeIDs, defense.ChallengeID)
	}
	teams, err := s.teamRepo.ListByIDs(ctx, teamIDs)
	if err != nil {
		return nil, nil, err
	}
	challenges, err := s.challengeRepo.GetByIDs(ctx, challengeIDs)
	if err != nil {
		return nil, nil, err
	}
	return buildTeamMap(teams), buildChallengeMap(challenges), nil
}

// loadRoundsMap 批量加载回合映射。
func (s *battleService) loadRoundsMap(ctx context.Context, roundIDs []int64) (map[int64]*entity.AdRound, error) {
	result := make(map[int64]*entity.AdRound)
	seen := make(map[int64]struct{})
	for _, roundID := range roundIDs {
		if roundID == 0 {
			continue
		}
		if _, ok := seen[roundID]; ok {
			continue
		}
		seen[roundID] = struct{}{}
		round, err := s.adRoundRepo.GetByID(ctx, roundID)
		if err != nil {
			return nil, err
		}
		result[roundID] = round
	}
	return result, nil
}
