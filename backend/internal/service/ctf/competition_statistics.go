// competition_statistics.go
// 模块05 — CTF竞赛：竞赛监控、统计与资源展示辅助。
// 负责把仓储查询结果转换为监控面板、统计报表和资源配额响应，
// 避免竞赛主流程文件同时承担展示聚合职责。

package ctf

import (
	"context"
	"time"

	"github.com/lenschain/backend/internal/model/dto"
	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/model/enum"
	ctfrepo "github.com/lenschain/backend/internal/repository/ctf"
)

// buildMonitorChallengeStats 构建竞赛监控页的题目统计项。
// 运行中的解题赛优先读取 Redis 中的动态分值缓存，保证监控页与选手视角的当前分值一致。
func (s *competitionService) buildMonitorChallengeStats(ctx context.Context, competitionID int64, stats []*ctfrepo.ChallengeSubmissionStats, environments []*entity.ChallengeEnvironment) []dto.CompetitionMonitorChallengeStat {
	challengeItems, _ := s.compChallengeRepo.ListByCompetitionID(ctx, competitionID)
	challengeIDs := make([]int64, 0, len(challengeItems))
	itemMap := make(map[int64]*entity.CompetitionChallenge, len(challengeItems))
	for _, item := range challengeItems {
		challengeIDs = append(challengeIDs, item.ChallengeID)
		itemMap[item.ChallengeID] = item
	}
	challenges, _ := s.challengeRepo.GetByIDs(ctx, challengeIDs)
	challengeMap := make(map[int64]*entity.Challenge, len(challenges))
	for _, challenge := range challenges {
		challengeMap[challenge.ID] = challenge
	}
	statMap := make(map[int64]*ctfrepo.ChallengeSubmissionStats, len(stats))
	for _, stat := range stats {
		statMap[stat.ChallengeID] = stat
	}
	envCountMap := make(map[int64]int)
	for _, env := range environments {
		if env.Status == enum.ChallengeEnvStatusRunning {
			envCountMap[env.ChallengeID]++
		}
	}
	result := make([]dto.CompetitionMonitorChallengeStat, 0, len(challengeItems))
	for _, item := range challengeItems {
		challenge := challengeMap[item.ChallengeID]
		if challenge == nil {
			continue
		}
		submissionStat := statMap[item.ChallengeID]
		attemptCount := int64(0)
		correctCount := int64(0)
		if submissionStat != nil {
			attemptCount = submissionStat.AttemptCount
			correctCount = submissionStat.CorrectCount
		}
		currentScore := item.CurrentScore
		if cachedScore, ok := readChallengeScoreCache(ctx, competitionID, challenge.ID); ok {
			currentScore = intPtr(cachedScore.CurrentScore)
		}
		result = append(result, dto.CompetitionMonitorChallengeStat{
			ChallengeID:         int64String(challenge.ID),
			Title:               challenge.Title,
			Category:            challenge.Category,
			SolveCount:          int(correctCount),
			AttemptCount:        int(attemptCount),
			SolveRate:           buildCorrectRate(correctCount, attemptCount),
			CurrentScore:        currentScore,
			EnvironmentsRunning: envCountMap[challenge.ID],
		})
	}
	return result
}

// buildCompetitionStatisticsResp 构建竞赛统计响应，供统计页和结果页共享。
func (s *competitionService) buildCompetitionStatisticsResp(ctx context.Context, competitionID int64, startAt, endAt *time.Time) (*dto.CompetitionStatisticsResp, error) {
	teams, err := s.teamRepo.ListRegisteredByCompetitionID(ctx, competitionID)
	if err != nil {
		return nil, err
	}
	teamMembers, _ := s.teamMemberRepo.CountByTeamIDs(ctx, collectTeamIDs(teams))
	submissionStats, _ := s.submissionRepo.CountByCompetition(ctx, competitionID)
	challengeSubmissionStats, _ := s.submissionRepo.CountByChallenge(ctx, competitionID)
	hourlySubmissionStats, _ := s.submissionRepo.CountByCompetitionPerHour(ctx, competitionID, startAt, endAt)
	resp := &dto.CompetitionStatisticsResp{
		CompetitionID: int64String(competitionID),
	}
	scores := make([]int, 0, len(teams))
	totalParticipants := 0
	for _, team := range teams {
		score := derefInt(team.TotalScore)
		scores = append(scores, score)
		totalParticipants += int(teamMembers[team.ID])
	}
	resp.Summary = dto.CompetitionStatisticsSummary{
		TotalTeams:        len(teams),
		TotalParticipants: totalParticipants,
		TotalSubmissions:  int(valueOrZeroSubmissionStat(submissionStats, true)),
		TotalCorrect:      int(valueOrZeroSubmissionStat(submissionStats, false)),
		OverallSolveRate:  buildCorrectRate(valueOrZeroSubmissionStat(submissionStats, false), valueOrZeroSubmissionStat(submissionStats, true)),
		AverageScore:      averageInt(scores),
		HighestScore:      maxInt(scores),
		LowestScore:       minInt(scores),
	}
	resp.ScoreDistribution = buildScoreDistribution(scores)
	resp.Timeline = buildCompetitionTimeline(hourlySubmissionStats, startAt, endAt)
	resp.ChallengeStatistics = s.buildCompetitionStatisticsItems(ctx, competitionID, startAt, challengeSubmissionStats)
	return resp, nil
}

// buildCompetitionStatisticsItems 构建竞赛统计页的题目维度指标。
func (s *competitionService) buildCompetitionStatisticsItems(ctx context.Context, competitionID int64, startAt *time.Time, stats []*ctfrepo.ChallengeSubmissionStats) []dto.CompetitionStatisticsChallengeItem {
	challengeItems, _ := s.compChallengeRepo.ListByCompetitionID(ctx, competitionID)
	challengeIDs := make([]int64, 0, len(challengeItems))
	itemMap := make(map[int64]*entity.CompetitionChallenge, len(challengeItems))
	for _, item := range challengeItems {
		challengeIDs = append(challengeIDs, item.ChallengeID)
		itemMap[item.ChallengeID] = item
	}
	challenges, _ := s.challengeRepo.GetByIDs(ctx, challengeIDs)
	challengeMap := make(map[int64]*entity.Challenge, len(challenges))
	for _, challenge := range challenges {
		challengeMap[challenge.ID] = challenge
	}
	statMap := make(map[int64]*ctfrepo.ChallengeSubmissionStats, len(stats))
	for _, stat := range stats {
		statMap[stat.ChallengeID] = stat
	}
	averageSolveMinutesMap := map[int64]int{}
	if startAt != nil {
		if avgMap, err := s.submissionRepo.AverageSolveMinutesByChallenge(ctx, competitionID, *startAt); err == nil {
			averageSolveMinutesMap = avgMap
		}
	}
	result := make([]dto.CompetitionStatisticsChallengeItem, 0, len(challengeItems))
	for _, item := range challengeItems {
		challenge := challengeMap[item.ChallengeID]
		if challenge == nil {
			continue
		}
		stat := statMap[item.ChallengeID]
		attemptCount := int64(0)
		correctCount := int64(0)
		if stat != nil {
			attemptCount = stat.AttemptCount
			correctCount = stat.CorrectCount
		}
		var firstBloodTeam *string
		if item.FirstBloodTeamID != nil {
			if team, err := s.teamRepo.GetByID(ctx, *item.FirstBloodTeamID); err == nil {
				firstBloodTeam = &team.Name
			}
		}
		firstBloodTimeMinutes := calculateElapsedMinutes(startAt, item.FirstBloodAt)
		averageSolveTimeMinutes := intPtrFromMap(averageSolveMinutesMap, challenge.ID)
		result = append(result, dto.CompetitionStatisticsChallengeItem{
			ChallengeID:             int64String(challenge.ID),
			Title:                   challenge.Title,
			Category:                challenge.Category,
			Difficulty:              challenge.Difficulty,
			DifficultyText:          enum.GetCtfDifficultyText(challenge.Difficulty),
			SolveCount:              int(correctCount),
			AttemptCount:            int(attemptCount),
			SolveRate:               buildCorrectRate(correctCount, attemptCount),
			FirstBloodTeam:          firstBloodTeam,
			FirstBloodTimeMinutes:   firstBloodTimeMinutes,
			AverageSolveTimeMinutes: averageSolveTimeMinutes,
		})
	}
	return result
}

// buildQuotaResp 将资源配额实体转换为接口响应。
func buildQuotaResp(quota *entity.CtfResourceQuota) *dto.ResourceQuotaResp {
	return &dto.ResourceQuotaResp{
		CompetitionID:     int64String(quota.CompetitionID),
		MaxCPU:            derefString(quota.MaxCPU),
		MaxMemory:         derefString(quota.MaxMemory),
		MaxStorage:        derefString(quota.MaxStorage),
		MaxNamespaces:     derefInt(quota.MaxNamespaces),
		UsedCPU:           quota.UsedCPU,
		UsedMemory:        quota.UsedMemory,
		UsedStorage:       quota.UsedStorage,
		CurrentNamespaces: quota.CurrentNamespaces,
	}
}

// buildCorrectRate 计算正确率。
func buildCorrectRate(correct int64, total int64) float64 {
	if total == 0 {
		return 0
	}
	return float64(correct) / float64(total)
}

// calculateElapsedMinutes 计算两个时间点之间的分钟差，任一为空时返回 nil。
func calculateElapsedMinutes(startAt, endAt *time.Time) *int {
	if startAt == nil || endAt == nil {
		return nil
	}
	minutes := int(endAt.Sub(*startAt).Minutes())
	if minutes < 0 {
		minutes = 0
	}
	return &minutes
}

// intPtrFromMap 从整型映射中提取指针值，不存在时返回 nil。
func intPtrFromMap(values map[int64]int, key int64) *int {
	value, ok := values[key]
	if !ok {
		return nil
	}
	return &value
}

// valueOrZeroSubmissionStat 从提交统计对象中读取指定值，不存在时返回 0。
func valueOrZeroSubmissionStat(stats *ctfrepo.SubmissionCountStats, total bool) int64 {
	if stats == nil {
		return 0
	}
	if total {
		return stats.TotalSubmissions
	}
	return stats.CorrectSubmissions
}

// buildCompetitionTimeline 基于仓储层的小时聚合结果构建统计时间线。
func buildCompetitionTimeline(stats []*ctfrepo.SubmissionHourlyStat, startAt, endAt *time.Time) dto.CompetitionTimeline {
	if len(stats) == 0 {
		return dto.CompetitionTimeline{SubmissionsPerHour: []dto.CompetitionTimelinePoint{}}
	}

	countMap := make(map[time.Time]int64, len(stats))
	firstBucket := stats[0].HourBucket.UTC().Truncate(time.Hour)
	lastBucket := stats[len(stats)-1].HourBucket.UTC().Truncate(time.Hour)
	for _, item := range stats {
		bucket := item.HourBucket.UTC().Truncate(time.Hour)
		countMap[bucket] = item.Count
		if bucket.Before(firstBucket) {
			firstBucket = bucket
		}
		if bucket.After(lastBucket) {
			lastBucket = bucket
		}
	}

	if startAt != nil {
		startBucket := startAt.UTC().Truncate(time.Hour)
		if startBucket.Before(firstBucket) {
			firstBucket = startBucket
		}
	}
	if endAt != nil {
		endBucket := endAt.UTC().Truncate(time.Hour)
		if endBucket.After(lastBucket) {
			lastBucket = endBucket
		}
	}

	points := make([]dto.CompetitionTimelinePoint, 0, int(lastBucket.Sub(firstBucket).Hours())+1)
	for bucket := firstBucket; !bucket.After(lastBucket); bucket = bucket.Add(time.Hour) {
		points = append(points, dto.CompetitionTimelinePoint{
			Hour:  bucket.Format("15:04"),
			Count: int(countMap[bucket]),
		})
	}
	return dto.CompetitionTimeline{SubmissionsPerHour: points}
}
