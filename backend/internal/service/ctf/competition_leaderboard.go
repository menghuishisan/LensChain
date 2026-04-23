// competition_leaderboard.go
// 模块05 — CTF竞赛：竞赛排行榜与结果构建辅助。
// 负责实时榜、快照榜、最终榜题目清单和分数分布等只读聚合逻辑，
// 避免把统计展示相关代码继续堆积在竞赛主流程文件中。

package ctf

import (
	"context"
	"sort"
	"time"

	"github.com/lenschain/backend/internal/model/dto"
	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/model/enum"
	"github.com/lenschain/backend/internal/pkg/cache"
	svcctx "github.com/lenschain/backend/internal/pkg/context"
	"github.com/lenschain/backend/internal/pkg/errcode"
	"github.com/lenschain/backend/internal/pkg/snowflake"
	ctfrepo "github.com/lenschain/backend/internal/repository/ctf"
)

// buildLeaderboardRankings 构建实时排行榜排名结果。
func (s *competitionService) buildLeaderboardRankings(ctx context.Context, competition *entity.Competition, groupID *int64, top int) ([]dto.LeaderboardRankingItem, *string, error) {
	var teams []*entity.Team
	var err error
	if groupID != nil {
		teams, err = s.teamRepo.ListByAdGroupID(ctx, *groupID)
	} else {
		teams, err = s.teamRepo.ListRegisteredByCompetitionID(ctx, competition.ID)
	}
	if err != nil {
		return nil, nil, err
	}
	rankings := make([]dto.LeaderboardRankingItem, 0, len(teams))
	updatedAt := ""
	for _, team := range teams {
		lastSolveAt, _ := s.submissionRepo.LastCorrectSubmissionAt(ctx, competition.ID, team.ID)
		score := derefInt(team.TotalScore)
		if competition.CompetitionType == enum.CompetitionTypeAttackDefense {
			if cachedBalance, ok := readAdTokenBalanceCache(ctx, competition.ID, team.ID); ok {
				team.TokenBalance = intPtr(cachedBalance)
			} else {
				writeAdTokenBalanceCache(ctx, competition.ID, team.ID, derefInt(team.TokenBalance))
			}
		}
		ranking := dto.LeaderboardRankingItem{
			TeamID:       int64String(team.ID),
			TeamName:     team.Name,
			Score:        intPtr(score),
			SolveCount:   intPtr(s.correctSolveCount(ctx, competition.ID, team.ID)),
			LastSolveAt:  optionalTimeString(lastSolveAt),
			TokenBalance: team.TokenBalance,
		}
		if competition.CompetitionType == enum.CompetitionTypeAttackDefense {
			ranking.Score = nil
			ranking.SolveCount = nil
			ranking.AttacksSuccessful = intPtr(s.successfulAttackCount(ctx, competition.ID, team.ID))
			ranking.DefensesSuccessful = intPtr(s.defenseRewardCount(ctx, competition.ID, team.ID))
			ranking.PatchesAccepted = intPtr(s.acceptedPatchCount(ctx, competition.ID, team.ID))
		}
		rankings = append(rankings, ranking)
		if team.UpdatedAt.After(parseOptionalTimeOrZero(updatedAt)) {
			updatedAt = timeString(team.UpdatedAt)
		}
	}
	sortLeaderboard(rankings)
	for i := range rankings {
		rankings[i].Rank = i + 1
	}
	if top > 0 && top < len(rankings) {
		rankings = rankings[:top]
	}
	var updatedAtPtr *string
	if updatedAt != "" {
		updatedAtPtr = &updatedAt
	}
	return rankings, updatedAtPtr, nil
}

// buildSnapshotRankings 将排行榜快照恢复成接口排名结构。
// 对攻防赛快照，score 字段存储的是快照时刻的 Token 余额；解题赛则存储题目总分。
func (s *competitionService) buildSnapshotRankings(ctx context.Context, competition *entity.Competition, snapshots []*entity.LeaderboardSnapshot, allowedTeamIDs map[int64]struct{}) []dto.LeaderboardRankingItem {
	if len(snapshots) == 0 {
		return []dto.LeaderboardRankingItem{}
	}
	teamIDs := make([]int64, 0, len(snapshots))
	snapshotAt := time.Time{}
	for _, snapshot := range snapshots {
		if snapshot == nil {
			continue
		}
		if snapshotAt.IsZero() {
			snapshotAt = snapshot.SnapshotAt
		}
		if len(allowedTeamIDs) > 0 {
			if _, ok := allowedTeamIDs[snapshot.TeamID]; !ok {
				continue
			}
		}
		teamIDs = append(teamIDs, snapshot.TeamID)
	}
	if len(teamIDs) == 0 {
		return []dto.LeaderboardRankingItem{}
	}
	teams, _ := s.teamRepo.ListByIDs(ctx, teamIDs)
	teamMap := make(map[int64]*entity.Team, len(teams))
	for _, team := range teams {
		teamMap[team.ID] = team
	}
	var (
		attackCounts  map[int64]int64
		defenseCounts map[int64]int64
		patchCounts   map[int64]int64
	)
	if competition != nil && competition.CompetitionType == enum.CompetitionTypeAttackDefense && !snapshotAt.IsZero() {
		attackCounts, _ = s.adAttackRepo.CountSuccessfulByTeamsUntil(ctx, competition.ID, teamIDs, snapshotAt)
		defenseCounts, _ = s.adLedgerRepo.CountByTeamsAndChangeTypeUntil(ctx, competition.ID, teamIDs, enum.TokenChangeDefenseReward, snapshotAt)
		patchCounts, _ = s.adDefenseRepo.CountAcceptedByTeamsUntil(ctx, competition.ID, teamIDs, snapshotAt)
	}
	rankings := make([]dto.LeaderboardRankingItem, 0, len(snapshots))
	for _, snapshot := range snapshots {
		if snapshot == nil {
			continue
		}
		if len(allowedTeamIDs) > 0 {
			if _, ok := allowedTeamIDs[snapshot.TeamID]; !ok {
				continue
			}
		}
		teamName := ""
		if teamMap[snapshot.TeamID] != nil {
			teamName = teamMap[snapshot.TeamID].Name
		}
		item := dto.LeaderboardRankingItem{
			Rank:        snapshot.Rank,
			TeamID:      int64String(snapshot.TeamID),
			TeamName:    teamName,
			SolveCount:  snapshot.SolveCount,
			LastSolveAt: optionalTimeString(snapshot.LastSolveAt),
		}
		if competition != nil && competition.CompetitionType == enum.CompetitionTypeAttackDefense {
			item.TokenBalance = intPtr(snapshot.Score)
			item.SolveCount = nil
			item.AttacksSuccessful = intPtr(int(attackCounts[snapshot.TeamID]))
			item.DefensesSuccessful = intPtr(int(defenseCounts[snapshot.TeamID]))
			item.PatchesAccepted = intPtr(int(patchCounts[snapshot.TeamID]))
		} else {
			item.Score = intPtr(snapshot.Score)
		}
		rankings = append(rankings, item)
	}
	sort.SliceStable(rankings, func(i, j int) bool {
		return rankings[i].Rank < rankings[j].Rank
	})
	if len(allowedTeamIDs) > 0 {
		for i := range rankings {
			rankings[i].Rank = i + 1
		}
	}
	return rankings
}

// buildAllowedGroupTeamSet 为分组榜构建可见团队白名单。
func (s *competitionService) buildAllowedGroupTeamSet(ctx context.Context, groupID *int64) (map[int64]struct{}, *string, error) {
	if groupID == nil {
		return nil, nil, nil
	}
	group, err := s.adGroupRepo.GetByID(ctx, *groupID)
	if err != nil {
		return nil, nil, err
	}
	teams, err := s.teamRepo.ListByAdGroupID(ctx, *groupID)
	if err != nil {
		return nil, nil, err
	}
	allowed := make(map[int64]struct{}, len(teams))
	for _, team := range teams {
		allowed[team.ID] = struct{}{}
	}
	return allowed, &group.GroupName, nil
}

// buildSolvedChallengeMap 构建最终榜中每支队伍的已解题目清单。
func (s *competitionService) buildSolvedChallengeMap(ctx context.Context, competitionID int64, teams []*entity.Team) (map[int64][]dto.FinalLeaderboardSolvedChallenge, error) {
	challengeItems, err := s.compChallengeRepo.ListByCompetitionID(ctx, competitionID)
	if err != nil {
		return nil, err
	}
	challengeIDs := make([]int64, 0, len(challengeItems))
	for _, item := range challengeItems {
		challengeIDs = append(challengeIDs, item.ChallengeID)
	}
	challenges, _ := s.challengeRepo.GetByIDs(ctx, challengeIDs)
	challengeMap := make(map[int64]*entity.Challenge, len(challenges))
	for _, challenge := range challenges {
		challengeMap[challenge.ID] = challenge
	}
	result := make(map[int64][]dto.FinalLeaderboardSolvedChallenge, len(teams))
	for _, team := range teams {
		submissions, _ := s.submissionRepo.CorrectSubmissionsByTeam(ctx, competitionID, team.ID)
		for _, submission := range submissions {
			challenge := challengeMap[submission.ChallengeID]
			title := ""
			if challenge != nil {
				title = challenge.Title
			}
			score := 0
			if submission.ScoreAwarded != nil {
				score = *submission.ScoreAwarded
			}
			result[team.ID] = append(result[team.ID], dto.FinalLeaderboardSolvedChallenge{
				ChallengeID:  int64String(submission.ChallengeID),
				Title:        title,
				Score:        score,
				SolvedAt:     timeString(submission.CreatedAt),
				IsFirstBlood: submission.IsFirstBlood,
			})
		}
	}
	return result, nil
}

// buildScoreDistribution 按接口文档约定的固定分段构建竞赛分数分布。
// 验收标准要求统计页返回 0-500、500-1000、1000-1500、1500-2000、2000+ 五个分段，
// 这里统一在 service 层收口，避免不同统计出口出现区间口径不一致。
func buildScoreDistribution(scores []int) dto.CompetitionScoreDistribution {
	ranges := []dto.CompetitionScoreDistributionRange{
		{Label: "0-500", Count: 0},
		{Label: "500-1000", Count: 0},
		{Label: "1000-1500", Count: 0},
		{Label: "1500-2000", Count: 0},
		{Label: "2000+", Count: 0},
	}
	for _, score := range scores {
		switch {
		case score < 500:
			ranges[0].Count++
		case score < 1000:
			ranges[1].Count++
		case score < 1500:
			ranges[2].Count++
		case score < 2000:
			ranges[3].Count++
		default:
			ranges[4].Count++
		}
	}
	return dto.CompetitionScoreDistribution{Ranges: ranges}
}

// correctSolveCount 统计团队在竞赛中的正确解题数。
func (s *competitionService) correctSolveCount(ctx context.Context, competitionID, teamID int64) int {
	submissions, err := s.submissionRepo.CorrectSubmissionsByTeam(ctx, competitionID, teamID)
	if err != nil {
		return 0
	}
	return len(submissions)
}

// successfulAttackCount 统计队伍在攻防赛中的成功攻击次数。
func (s *competitionService) successfulAttackCount(ctx context.Context, competitionID, teamID int64) int {
	success := true
	_, total, err := s.adAttackRepo.List(ctx, &ctfrepo.AdAttackListParams{
		CompetitionID:  competitionID,
		AttackerTeamID: teamID,
		IsSuccessful:   &success,
		Page:           1,
		PageSize:       1,
	})
	if err != nil {
		return 0
	}
	return int(total)
}

// defenseRewardCount 统计队伍在攻防赛中获得“本回合未被攻破”防守奖励的次数。
func (s *competitionService) defenseRewardCount(ctx context.Context, competitionID, teamID int64) int {
	_, total, err := s.adLedgerRepo.List(ctx, &ctfrepo.AdTokenLedgerListParams{
		CompetitionID: competitionID,
		TeamID:        teamID,
		ChangeType:    enum.TokenChangeDefenseReward,
		Page:          1,
		PageSize:      1,
	})
	if err != nil {
		return 0
	}
	return int(total)
}

// acceptedPatchCount 统计队伍在攻防赛中通过验证并被接受的补丁数量。
func (s *competitionService) acceptedPatchCount(ctx context.Context, competitionID, teamID int64) int {
	isAccepted := true
	_, total, err := s.adDefenseRepo.List(ctx, &ctfrepo.AdDefenseListParams{
		CompetitionID: competitionID,
		TeamID:        teamID,
		IsAccepted:    &isAccepted,
		Page:          1,
		PageSize:      1,
	})
	if err != nil {
		return 0
	}
	return int(total)
}

// parseOptionalTimeOrZero 解析可选时间字符串，失败时返回零值时间。
func parseOptionalTimeOrZero(value string) time.Time {
	if value == "" {
		return time.Time{}
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}
	}
	return parsed
}

func (s *competitionService) GetLeaderboard(ctx context.Context, sc *svcctx.ServiceContext, competitionID int64, req *dto.LeaderboardReq) (*dto.LeaderboardResp, error) {
	competition, err := getCompetition(ctx, s.competitionRepo, competitionID)
	if err != nil {
		return nil, err
	}
	if err := s.ensureCompetitionReadable(sc, competition); err != nil {
		return nil, err
	}
	var groupID *int64
	if req.GroupID != "" {
		parsedGroupID, parseErr := snowflake.ParseString(req.GroupID)
		if parseErr != nil {
			return nil, errcode.ErrInvalidID
		}
		groupID = &parsedGroupID
	}
	isFrozen := competition.FreezeAt != nil && !competition.FreezeAt.After(time.Now())
	if exists, cacheErr := cache.Exists(ctx, buildCompetitionFrozenCacheKey(competition.ID)); cacheErr == nil {
		isFrozen = exists
	}
	var (
		rankings  []dto.LeaderboardRankingItem
		updatedAt *string
	)
	if competition.Status == enum.CompetitionStatusRunning && isFrozen {
		rankings, updatedAt, err = s.loadFrozenPublicLeaderboard(ctx, competition, groupID, req.Top)
	} else {
		rankings, updatedAt, err = s.buildLeaderboardRankings(ctx, competition, groupID, req.Top)
	}
	if err != nil {
		return nil, err
	}
	resp := &dto.LeaderboardResp{
		CompetitionID:   int64String(competition.ID),
		CompetitionType: competition.CompetitionType,
		IsFrozen:        isFrozen,
		UpdatedAt:       updatedAt,
		Rankings:        rankings,
	}
	if groupID != nil {
		value := int64String(*groupID)
		resp.GroupID = &value
		_, groupName, groupErr := s.buildAllowedGroupTeamSet(ctx, groupID)
		if groupErr == nil {
			resp.GroupName = groupName
		}
		if competition.CompetitionType == enum.CompetitionTypeAttackDefense {
			rounds, _ := s.adRoundRepo.ListByGroupID(ctx, *groupID)
			totalRounds := len(rounds)
			currentRound := 0
			for _, round := range rounds {
				if round.Phase != enum.RoundPhaseCompleted {
					currentRound = round.RoundNumber
					break
				}
			}
			if totalRounds > 0 {
				resp.TotalRounds = &totalRounds
			}
			if currentRound > 0 {
				resp.CurrentRound = &currentRound
			}
		}
	}
	if resp.IsFrozen {
		resp.FrozenAt = optionalTimeString(competition.FreezeAt)
	}
	return resp, nil
}

// GetFinalLeaderboard 获取最终排行榜。
func (s *competitionService) GetFinalLeaderboard(ctx context.Context, sc *svcctx.ServiceContext, competitionID int64) (*dto.FinalLeaderboardResp, error) {
	competition, err := getCompetition(ctx, s.competitionRepo, competitionID)
	if err != nil {
		return nil, err
	}
	if err := s.ensureCompetitionReadable(sc, competition); err != nil {
		return nil, err
	}
	if competition.Status != enum.CompetitionStatusEnded && competition.Status != enum.CompetitionStatusArchived {
		return nil, errcode.ErrCompetitionResultNotReady
	}
	// 最终榜必须从最终一次排行榜快照恢复，不能在结束后重新实时计算，
	// 否则会破坏冻结揭晓和归档后的结果口径。
	snapshots, err := s.leaderboardRepo.ListLatestByCompetition(ctx, competition.ID, nil, 0)
	if err != nil {
		return nil, err
	}
	if len(snapshots) == 0 {
		return nil, errcode.ErrCompetitionResultNotReady
	}
	rankings := s.buildSnapshotRankings(ctx, competition, snapshots, nil)
	teams, err := s.teamRepo.ListRegisteredByCompetitionID(ctx, competition.ID)
	if err != nil {
		return nil, err
	}
	teamMembers, _ := s.teamMemberRepo.ListByTeamIDs(ctx, collectTeamIDs(teams))
	memberMap := buildTeamMembersMap(teamMembers)
	submissionMap, _ := s.buildSolvedChallengeMap(ctx, competition.ID, teams)
	finalItems := make([]dto.FinalLeaderboardItem, 0, len(rankings))
	for _, ranking := range rankings {
		finalItem := dto.FinalLeaderboardItem{
			Rank:               ranking.Rank,
			TeamID:             ranking.TeamID,
			TeamName:           ranking.TeamName,
			Score:              ranking.Score,
			SolveCount:         ranking.SolveCount,
			LastSolveAt:        ranking.LastSolveAt,
			TokenBalance:       ranking.TokenBalance,
			AttacksSuccessful:  ranking.AttacksSuccessful,
			DefensesSuccessful: ranking.DefensesSuccessful,
			PatchesAccepted:    ranking.PatchesAccepted,
		}
		teamID, _ := snowflake.ParseString(ranking.TeamID)
		for _, member := range memberMap[teamID] {
			finalItem.Members = append(finalItem.Members, dto.FinalLeaderboardMember{
				Name:     s.userQuerier.GetUserName(ctx, member.StudentID),
				RoleText: enum.GetTeamMemberRoleText(member.Role),
			})
		}
		finalItem.SolvedChallenges = submissionMap[teamID]
		finalItems = append(finalItems, finalItem)
	}
	totalSolves := 0
	for _, item := range finalItems {
		totalSolves += len(item.SolvedChallenges)
	}
	// 最终榜的结束时间优先反映真实结束时刻。
	// 自然结束时使用配置的 end_at；若是强制终止导致状态提前进入“已结束”，
	// 则使用状态更新时刻，确保最终结果符合“以终止时刻为最终结果”的文档要求。
	endedAt := competition.UpdatedAt
	if competition.EndAt != nil && !competition.UpdatedAt.Before(*competition.EndAt) {
		endedAt = *competition.EndAt
	}
	return &dto.FinalLeaderboardResp{
		CompetitionID:   int64String(competition.ID),
		CompetitionType: competition.CompetitionType,
		EndedAt:         timeString(endedAt),
		Rankings:        finalItems,
		TotalTeams:      len(finalItems),
		TotalSolves:     totalSolves,
	}, nil
}

// GetLeaderboardHistory 获取排行榜历史快照。
func (s *competitionService) GetLeaderboardHistory(ctx context.Context, sc *svcctx.ServiceContext, competitionID int64, req *dto.LeaderboardHistoryReq) (*dto.LeaderboardHistoryResp, error) {
	competition, err := getCompetition(ctx, s.competitionRepo, competitionID)
	if err != nil {
		return nil, err
	}
	if err := s.ensureCompetitionReadable(sc, competition); err != nil {
		return nil, err
	}
	var (
		groupID      *int64
		allowedTeams map[int64]struct{}
	)
	if req.GroupID != "" {
		parsedGroupID, parseErr := snowflake.ParseString(req.GroupID)
		if parseErr != nil {
			return nil, errcode.ErrInvalidID
		}
		groupID = &parsedGroupID
		allowedTeams, _, err = s.buildAllowedGroupTeamSet(ctx, groupID)
		if err != nil {
			return nil, err
		}
	}
	times, total, err := s.leaderboardRepo.ListSnapshotTimes(ctx, competitionID, req.Page, req.PageSize)
	if err != nil {
		return nil, err
	}
	resp := &dto.LeaderboardHistoryResp{
		List:       make([]dto.LeaderboardHistorySnapshotItem, 0, len(times)),
		Pagination: paginationResp(req.Page, req.PageSize, total),
	}
	for _, snapshotAt := range times {
		snapshots, _, listErr := s.leaderboardRepo.ListByCompetition(ctx, &ctfrepo.LeaderboardSnapshotListParams{
			CompetitionID: competitionID,
			SnapshotAt:    &snapshotAt,
			Page:          1,
			PageSize:      200,
		})
		if listErr != nil {
			return nil, listErr
		}
		rankings := s.buildSnapshotRankings(ctx, competition, snapshots, allowedTeams)
		resp.List = append(resp.List, dto.LeaderboardHistorySnapshotItem{
			SnapshotAt: timeString(snapshotAt),
			Rankings:   rankings,
		})
	}
	return resp, nil
}

// loadFrozenPublicLeaderboard 读取冻结后对外公开的最后一次排行榜快照。
// 冻结期间不能继续暴露实时排名，因此优先返回最后一次非冻结快照；若不存在，再退回最近一次快照。
func (s *competitionService) loadFrozenPublicLeaderboard(ctx context.Context, competition *entity.Competition, groupID *int64, top int) ([]dto.LeaderboardRankingItem, *string, error) {
	allowedTeams, _, err := s.buildAllowedGroupTeamSet(ctx, groupID)
	if err != nil {
		return nil, nil, err
	}
	notFrozen := false
	snapshots, err := s.leaderboardRepo.ListLatestByCompetition(ctx, competition.ID, &notFrozen, 500)
	if err != nil {
		return nil, nil, err
	}
	if len(snapshots) == 0 {
		snapshots, err = s.leaderboardRepo.ListLatestByCompetition(ctx, competition.ID, nil, 500)
		if err != nil {
			return nil, nil, err
		}
	}
	if len(snapshots) == 0 {
		return s.buildLeaderboardRankings(ctx, competition, groupID, top)
	}
	rankings := s.buildSnapshotRankings(ctx, competition, snapshots, allowedTeams)
	if top > 0 && top < len(rankings) {
		rankings = rankings[:top]
	}
	updatedAt := timeString(snapshots[0].SnapshotAt)
	return rankings, &updatedAt, nil
}
