// competition_challenges.go
// 模块05 — CTF竞赛：竞赛题目配置业务逻辑。

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

func (s *competitionService) AddChallenges(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.AddCompetitionChallengeReq) (*dto.AddCompetitionChallengeResp, error) {
	competition, err := getCompetition(ctx, s.competitionRepo, id)
	if err != nil {
		return nil, err
	}
	if err := s.ensureCompetitionOwned(sc, competition); err != nil {
		return nil, err
	}
	if competition.Status != enum.CompetitionStatusDraft {
		return nil, errcode.ErrCompetitionNotDraft
	}
	challengeIDs := make([]int64, 0, len(req.ChallengeIDs))
	for _, challengeIDText := range req.ChallengeIDs {
		challengeID, parseErr := snowflake.ParseString(challengeIDText)
		if parseErr != nil {
			return nil, errcode.ErrInvalidID
		}
		challengeIDs = append(challengeIDs, challengeID)
	}
	existingItems, err := s.compChallengeRepo.ListByCompetitionID(ctx, competition.ID)
	if err != nil {
		return nil, err
	}
	existingChallengeSet := make(map[int64]struct{}, len(existingItems))
	sortOrder := len(existingItems)
	for _, item := range existingItems {
		existingChallengeSet[item.ChallengeID] = struct{}{}
	}

	items := make([]*entity.CompetitionChallenge, 0, len(challengeIDs))
	challengeMap := make(map[int64]*entity.Challenge, len(challengeIDs))
	for _, challengeID := range challengeIDs {
		challenge, getErr := getChallenge(ctx, s.challengeRepo, challengeID)
		if getErr != nil {
			return nil, getErr
		}
		if challenge.Status != enum.ChallengeStatusApproved {
			return nil, errcode.ErrChallengeNotApproved
		}
		if competition.CompetitionType == enum.CompetitionTypeAttackDefense {
			if challenge.FlagType != enum.FlagTypeOnChain || challenge.Category != enum.ChallengeCategoryContract {
				return nil, errcode.ErrCompetitionConfigRequired.WithMessage("攻防对抗赛仅支持链上验证的智能合约题目")
			}
			contracts, contractErr := s.contractRepo.ListByChallengeID(ctx, challengeID)
			if contractErr != nil {
				return nil, contractErr
			}
			if len(contracts) == 0 {
				return nil, errcode.ErrChallengeContractRequired
			}
			assertions, assertionErr := s.assertionRepo.ListByChallengeID(ctx, challengeID)
			if assertionErr != nil {
				return nil, assertionErr
			}
			if len(assertions) == 0 {
				return nil, errcode.ErrChallengeAssertionRequired
			}
		}
		if _, exists := existingChallengeSet[challengeID]; exists {
			return nil, errcode.ErrChallengeAlreadyInContest
		}
		challengeMap[challengeID] = challenge
		sortOrder++
		items = append(items, &entity.CompetitionChallenge{
			ID:            snowflake.Generate(),
			CompetitionID: competition.ID,
			ChallengeID:   challengeID,
			SortOrder:     sortOrder,
			CurrentScore:  intPtr(challenge.BaseScore),
		})
	}
	if len(items) == 0 {
		return &dto.AddCompetitionChallengeResp{
			AddedCount:    0,
			CompetitionID: int64String(competition.ID),
			Challenges:    []dto.AddedCompetitionChallengeItem{},
		}, nil
	}
	if err := s.compChallengeRepo.BatchCreate(ctx, items); err != nil {
		return nil, err
	}
	respItems := make([]dto.AddedCompetitionChallengeItem, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		challenge := challengeMap[item.ChallengeID]
		if challenge == nil {
			continue
		}
		writeChallengeScoreCache(ctx, competition.ID, item.ChallengeID, challenge.BaseScore, 0, challenge.BaseScore)
		respItems = append(respItems, dto.AddedCompetitionChallengeItem{
			ID:        int64String(item.ChallengeID),
			Title:     challenge.Title,
			SortOrder: item.SortOrder,
		})
	}
	return &dto.AddCompetitionChallengeResp{
		AddedCount:    len(respItems),
		CompetitionID: int64String(competition.ID),
		Challenges:    respItems,
	}, nil
}

// ListChallenges 获取竞赛题目列表。
func (s *competitionService) ListChallenges(ctx context.Context, sc *svcctx.ServiceContext, id int64) (*dto.CompetitionChallengeListResp, error) {
	competition, err := getCompetition(ctx, s.competitionRepo, id)
	if err != nil {
		return nil, err
	}
	if err := s.ensureCompetitionChallengeReadable(ctx, sc, competition); err != nil {
		return nil, err
	}
	items, err := s.compChallengeRepo.ListByCompetitionID(ctx, id)
	if err != nil {
		return nil, err
	}
	challengeIDs := make([]int64, 0, len(items))
	for _, item := range items {
		challengeIDs = append(challengeIDs, item.ChallengeID)
	}
	challenges, err := s.challengeRepo.GetByIDs(ctx, challengeIDs)
	if err != nil {
		return nil, err
	}
	challengeMap := make(map[int64]*entity.Challenge, len(challenges))
	for _, challenge := range challenges {
		challengeMap[challenge.ID] = challenge
	}
	var (
		myTeamID      int64
		solvedSet     map[int64]struct{}
		environmentID map[int64]string
	)
	if sc != nil && sc.IsStudent() {
		if _, team, teamErr := getCompetitionMemberTeam(ctx, s.teamMemberRepo, s.teamRepo, competition.ID, sc.UserID); teamErr == nil && team != nil {
			myTeamID = team.ID
			solvedSet, _ = s.loadSolvedChallengeSet(ctx, competition.ID, myTeamID)
			environmentID, _ = s.loadEnvironmentIDMap(ctx, competition.ID, myTeamID)
		}
	}
	fullChallengeView := sc != nil && competition.CreatedBy == sc.UserID
	respItems := make([]dto.CompetitionChallengeListItem, 0, len(items))
	for _, item := range items {
		challenge := challengeMap[item.ChallengeID]
		if challenge == nil {
			continue
		}
		currentScore := item.CurrentScore
		solveCount := item.SolveCount
		if competition.Status == enum.CompetitionStatusRunning && competition.CompetitionType == enum.CompetitionTypeJeopardy {
			if cachedScore, ok := readChallengeScoreCache(ctx, competition.ID, challenge.ID); ok {
				currentScore = intPtr(cachedScore.CurrentScore)
				solveCount = cachedScore.SolveCount
			} else {
				writeChallengeScoreCache(ctx, competition.ID, challenge.ID, derefInt(item.CurrentScore), item.SolveCount, challenge.BaseScore)
			}
		}
		challengeSummary, err := s.buildCompetitionChallengeSummary(ctx, challenge, fullChallengeView)
		if err != nil {
			return nil, err
		}
		var firstBloodTeam *dto.CompetitionChallengeFirstBloodTeam
		if item.FirstBloodTeamID != nil {
			teamName := s.userQuerier.GetUserName(ctx, *item.FirstBloodTeamID)
			if team, teamErr := s.teamRepo.GetByID(ctx, *item.FirstBloodTeamID); teamErr == nil && team != nil {
				teamName = team.Name
			}
			firstBloodTeam = &dto.CompetitionChallengeFirstBloodTeam{
				ID:   int64String(*item.FirstBloodTeamID),
				Name: teamName,
			}
		}
		var myTeamEnvironment *string
		if environmentID != nil {
			if envID, ok := environmentID[challenge.ID]; ok {
				myTeamEnvironment = stringPtr(envID)
			}
		}
		respItems = append(respItems, dto.CompetitionChallengeListItem{
			ID:                int64String(item.ID),
			Challenge:         challengeSummary,
			BaseScore:         challenge.BaseScore,
			CurrentScore:      currentScore,
			SolveCount:        solveCount,
			FirstBloodTeam:    firstBloodTeam,
			FirstBloodAt:      optionalTimeString(item.FirstBloodAt),
			SortOrder:         item.SortOrder,
			MyTeamSolved:      solvedSet != nil && containsChallengeID(solvedSet, challenge.ID),
			MyTeamEnvironment: myTeamEnvironment,
		})
	}
	return &dto.CompetitionChallengeListResp{List: respItems}, nil
}

// ensureCompetitionChallengeReadable 校验竞赛题目列表读取权限。
// 文档约定该入口仅开放给竞赛创建者和参赛选手，不向同校未参赛用户暴露完整题目清单。
func (s *competitionService) ensureCompetitionChallengeReadable(ctx context.Context, sc *svcctx.ServiceContext, competition *entity.Competition) error {
	if err := s.ensureCompetitionReadable(sc, competition); err != nil {
		return err
	}
	if sc == nil {
		return errcode.ErrForbidden
	}
	if sc.IsSuperAdmin() || competition.CreatedBy == sc.UserID {
		return nil
	}
	if !sc.IsStudent() {
		return errcode.ErrForbidden
	}
	_, _, err := getCompetitionMemberTeam(ctx, s.teamMemberRepo, s.teamRepo, competition.ID, sc.UserID)
	if err != nil {
		return errcode.ErrForbidden
	}
	return nil
}

// loadSolvedChallengeSet 加载指定团队在竞赛中的已解题集合。
func (s *competitionService) loadSolvedChallengeSet(ctx context.Context, competitionID, teamID int64) (map[int64]struct{}, error) {
	isCorrect := true
	submissions, _, err := s.submissionRepo.List(ctx, &ctfrepo.SubmissionListParams{
		CompetitionID: competitionID,
		TeamID:        teamID,
		IsCorrect:     &isCorrect,
		Page:          1,
		PageSize:      1000,
	})
	if err != nil {
		return nil, err
	}
	result := make(map[int64]struct{}, len(submissions))
	for _, submission := range submissions {
		if submission == nil {
			continue
		}
		result[submission.ChallengeID] = struct{}{}
	}
	return result, nil
}

// loadEnvironmentIDMap 加载指定团队当前各题目的环境 ID 映射。
func (s *competitionService) loadEnvironmentIDMap(ctx context.Context, competitionID, teamID int64) (map[int64]string, error) {
	environments, err := s.environmentRepo.ListByCompetitionID(ctx, competitionID)
	if err != nil {
		return nil, err
	}
	result := make(map[int64]string)
	for _, environment := range environments {
		if environment == nil || environment.TeamID != teamID || environment.Status == enum.ChallengeEnvStatusDestroyed {
			continue
		}
		result[environment.ChallengeID] = int64String(environment.ID)
	}
	return result, nil
}

// containsChallengeID 判断题目是否存在于已解集合中。
func containsChallengeID(solvedSet map[int64]struct{}, challengeID int64) bool {
	if len(solvedSet) == 0 {
		return false
	}
	_, ok := solvedSet[challengeID]
	return ok
}

// SortChallenges 调整竞赛题目排序。
func (s *competitionService) SortChallenges(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.SortCompetitionChallengesReq) error {
	competition, err := getCompetition(ctx, s.competitionRepo, id)
	if err != nil {
		return err
	}
	if err := s.ensureCompetitionOwned(sc, competition); err != nil {
		return err
	}
	if competition.Status != enum.CompetitionStatusDraft {
		return errcode.ErrCompetitionNotDraft
	}
	sortItems := make([]ctfrepo.CompetitionChallengeSortItem, 0, len(req.Items))
	for _, item := range req.Items {
		targetID, parseErr := snowflake.ParseString(item.ID)
		if parseErr != nil {
			return errcode.ErrInvalidID
		}
		sortItems = append(sortItems, ctfrepo.CompetitionChallengeSortItem{ID: targetID, SortOrder: item.SortOrder})
	}
	return s.compChallengeRepo.BatchUpdateSort(ctx, sortItems)
}

// RemoveChallenge 移除竞赛题目。
func (s *competitionService) RemoveChallenge(ctx context.Context, sc *svcctx.ServiceContext, id int64) error {
	item, err := s.compChallengeRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errcode.ErrSubmissionChallengeGone
		}
		return err
	}
	competition, err := getCompetition(ctx, s.competitionRepo, item.CompetitionID)
	if err != nil {
		return err
	}
	if err := s.ensureCompetitionOwned(sc, competition); err != nil {
		return err
	}
	if competition.Status != enum.CompetitionStatusDraft {
		return errcode.ErrCompetitionNotDraft
	}
	return s.compChallengeRepo.Delete(ctx, id)
}

// GetLeaderboard 获取实时排行榜。
