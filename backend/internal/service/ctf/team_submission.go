// team_submission.go
// 模块05 — CTF竞赛：解题提交、限流与排行榜广播业务逻辑。

package ctf

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"gorm.io/gorm"

	"github.com/lenschain/backend/internal/model/dto"
	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/model/enum"
	"github.com/lenschain/backend/internal/pkg/cache"
	svcctx "github.com/lenschain/backend/internal/pkg/context"
	"github.com/lenschain/backend/internal/pkg/database"
	"github.com/lenschain/backend/internal/pkg/errcode"
	"github.com/lenschain/backend/internal/pkg/snowflake"
	ctfrepo "github.com/lenschain/backend/internal/repository/ctf"
)

// SubmitChallenge 提交解题赛题目答案或攻击交易。
func (s *teamService) SubmitChallenge(ctx context.Context, sc *svcctx.ServiceContext, competitionID int64, req *dto.SubmitCompetitionChallengeReq) (*dto.CompetitionSubmissionResp, error) {
	if !sc.IsStudent() {
		return nil, errcode.ErrForbidden
	}
	competition, err := getCompetition(ctx, s.competitionRepo, competitionID)
	if err != nil {
		return nil, err
	}
	if competition.Status != enum.CompetitionStatusRunning {
		return nil, errcode.ErrCompetitionStatusInvalid.WithMessage("竞赛不在进行中")
	}
	challengeID, err := snowflake.ParseString(req.ChallengeID)
	if err != nil {
		return nil, errcode.ErrInvalidID
	}
	challengeItem, err := s.compChallengeRepo.GetByCompetitionAndChallenge(ctx, competitionID, challengeID)
	if err != nil {
		return nil, errcode.ErrSubmissionChallengeGone
	}
	challenge, err := getChallenge(ctx, s.challengeRepo, challengeID)
	if err != nil {
		return nil, err
	}
	member, team, err := s.ensureRegisteredCompetitionMember(ctx, competitionID, sc.UserID)
	if err != nil {
		return nil, err
	}
	_ = member
	alreadySolved, err := s.hasSolvedChallenge(ctx, competitionID, team.ID, challengeID)
	if err != nil {
		return nil, err
	}
	if alreadySolved {
		return nil, errcode.ErrAlreadySolved.WithMessage("您的团队已解出此题")
	}
	cooldownUntil, remainingAttempts, err := s.checkSubmissionWindow(ctx, competition, team.ID, challengeID)
	if err != nil {
		return nil, err
	}
	if cooldownUntil != nil {
		waitMinutes := int(time.Until(*cooldownUntil).Minutes())
		if waitMinutes <= 0 {
			waitMinutes = 1
		}
		return nil, errcode.ErrSubmissionCooldown.WithMessage(fmt.Sprintf("提交已冷却，请等待%d分钟", waitMinutes))
	}

	isCorrect, assertionResults, failureReason, err := s.evaluateCompetitionSubmission(ctx, competitionID, team, challenge, req)
	if err != nil {
		return nil, err
	}

	submission := &entity.Submission{
		ID:             snowflake.Generate(),
		CompetitionID:  competitionID,
		ChallengeID:    challengeID,
		TeamID:         team.ID,
		StudentID:      sc.UserID,
		SubmissionType: req.SubmissionType,
		Content:        req.Content,
		IsCorrect:      isCorrect,
		CreatedAt:      time.Now(),
	}
	if assertionResults != nil {
		submission.AssertionResults = mustJSON(assertionResults)
	}
	if failureReason != nil {
		submission.ErrorMessage = failureReason
	}

	resp := &dto.CompetitionSubmissionResp{
		SubmissionID:      int64String(submission.ID),
		IsCorrect:         isCorrect,
		ErrorMessage:      failureReason,
		RemainingAttempts: intPtr(remainingAttempts),
	}
	if !isCorrect {
		if err := s.submissionRepo.Create(ctx, submission); err != nil {
			return nil, err
		}
		s.recordFailedSubmission(ctx, competition, team.ID, challengeID)
		return resp, nil
	}

	isFirstBlood := challengeItem.FirstBloodTeamID == nil
	awardedScore, newChallengeScore := s.calculateSubmissionAward(competition, challenge, challengeItem, isFirstBlood)
	submission.IsFirstBlood = isFirstBlood
	submission.ScoreAwarded = &awardedScore

	err = database.TransactionWithDB(ctx, s.db, func(tx *gorm.DB) error {
		txSubmissionRepo := ctfrepo.NewSubmissionRepository(tx)
		txTeamRepo := ctfrepo.NewTeamRepository(tx)
		txCompChallengeRepo := ctfrepo.NewCompetitionChallengeRepository(tx)
		if err := txSubmissionRepo.Create(ctx, submission); err != nil {
			return err
		}
		if err := txTeamRepo.IncrementTotalScore(ctx, team.ID, awardedScore); err != nil {
			return err
		}
		if err := txCompChallengeRepo.IncrementSolveCount(ctx, challengeItem.ID, newChallengeScore); err != nil {
			return err
		}
		if isFirstBlood {
			if err := txCompChallengeRepo.MarkFirstBlood(ctx, challengeItem.ID, team.ID, submission.CreatedAt); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	s.recordSuccessfulSubmission(ctx, competition, team.ID, challengeID)
	writeChallengeScoreCache(ctx, competition.ID, challengeID, newChallengeScore, itemSolveCountAfterSuccess(challengeItem), challenge.BaseScore)

	updatedTeam, _ := s.teamRepo.GetByID(ctx, team.ID)
	teamRank := s.calculateTeamRank(ctx, competitionID, team.ID)
	resp.ScoreAwarded = &awardedScore
	resp.IsFirstBlood = &isFirstBlood
	resp.ChallengeNewScore = &newChallengeScore
	if updatedTeam != nil {
		resp.TeamTotalScore = updatedTeam.TotalScore
	}
	resp.TeamRank = &teamRank
	resp.AssertionResults = assertionResults
	s.publishLeaderboardUpdate(ctx, competitionID, team, challenge, isFirstBlood)
	return resp, nil
}

// ListSubmissions 查询团队提交记录。
func (s *teamService) ListSubmissions(ctx context.Context, sc *svcctx.ServiceContext, competitionID int64, req *dto.CompetitionSubmissionListReq) (*dto.CompetitionSubmissionListResp, error) {
	if sc == nil || !sc.IsStudent() {
		return nil, errcode.ErrForbidden
	}
	_, team, err := s.ensureRegisteredCompetitionMember(ctx, competitionID, sc.UserID)
	if err != nil {
		return nil, err
	}
	submissions, total, err := s.submissionRepo.List(ctx, &ctfrepo.SubmissionListParams{
		CompetitionID: competitionID,
		TeamID:        team.ID,
		Page:          req.Page,
		PageSize:      req.PageSize,
	})
	if err != nil {
		return nil, err
	}
	challengeIDs := make([]int64, 0, len(submissions))
	for _, item := range submissions {
		challengeIDs = append(challengeIDs, item.ChallengeID)
	}
	challenges, _ := s.challengeRepo.GetByIDs(ctx, challengeIDs)
	challengeMap := make(map[int64]*entity.Challenge, len(challenges))
	for _, challenge := range challenges {
		challengeMap[challenge.ID] = challenge
	}
	items := make([]dto.CompetitionSubmissionListItem, 0, len(submissions))
	for _, submission := range submissions {
		challengeTitle := ""
		if challengeMap[submission.ChallengeID] != nil {
			challengeTitle = challengeMap[submission.ChallengeID].Title
		}
		items = append(items, dto.CompetitionSubmissionListItem{
			ID:                 int64String(submission.ID),
			ChallengeID:        int64String(submission.ChallengeID),
			ChallengeTitle:     challengeTitle,
			SubmissionType:     submission.SubmissionType,
			SubmissionTypeText: enum.GetSubmissionTypeText(submission.SubmissionType),
			IsCorrect:          submission.IsCorrect,
			ScoreAwarded:       submission.ScoreAwarded,
			IsFirstBlood:       submission.IsFirstBlood,
			ErrorMessage:       submission.ErrorMessage,
			CreatedAt:          timeString(submission.CreatedAt),
		})
	}
	return &dto.CompetitionSubmissionListResp{
		List:       items,
		Pagination: paginationResp(req.Page, req.PageSize, total),
	}, nil
}

// GetSubmissionStatistics 获取竞赛提交统计。
// 该接口只开放给竞赛创建者，返回竞赛维度的提交总览，而不是当前团队视角。
func (s *teamService) GetSubmissionStatistics(ctx context.Context, sc *svcctx.ServiceContext, competitionID int64) (*dto.CompetitionSubmissionStatisticsResp, error) {
	competition, err := getCompetition(ctx, s.competitionRepo, competitionID)
	if err != nil {
		return nil, err
	}
	if err := s.ensureCompetitionOwned(sc, competition); err != nil {
		return nil, err
	}
	stats, err := s.submissionRepo.OverviewByCompetition(ctx, competitionID)
	if err != nil {
		return nil, err
	}
	return &dto.CompetitionSubmissionStatisticsResp{
		TotalSubmissions:   int(stats.TotalSubmissions),
		CorrectSubmissions: int(stats.CorrectSubmissions),
		CorrectRate:        buildCorrectRate(stats.CorrectSubmissions, stats.TotalSubmissions),
		FirstBloodCount:    int(stats.FirstBloodCount),
		TeamsParticipated:  int(stats.TeamsParticipated),
	}, nil
}

// calculateSubmissionAward 计算正确提交得分和题目新分值。
func (s *teamService) calculateSubmissionAward(competition *entity.Competition, challenge *entity.Challenge, item *entity.CompetitionChallenge, isFirstBlood bool) (int, int) {
	if competition.CompetitionType != enum.CompetitionTypeJeopardy {
		return 0, derefInt(item.CurrentScore)
	}
	cfg := dto.JeopardyScoringConfig{
		DecayFactor:     0.95,
		MinScoreRatio:   0.2,
		FirstBloodBonus: 0.1,
	}
	if competition != nil && len(competition.JeopardyConfig) > 0 {
		var competitionCfg dto.JeopardyCompetitionConfig
		if err := decodeJSON(competition.JeopardyConfig, &competitionCfg); err == nil {
			if competitionCfg.Scoring.DecayFactor > 0 {
				cfg.DecayFactor = competitionCfg.Scoring.DecayFactor
			}
			if competitionCfg.Scoring.MinScoreRatio > 0 {
				cfg.MinScoreRatio = competitionCfg.Scoring.MinScoreRatio
			}
			if competitionCfg.Scoring.FirstBloodBonus > 0 {
				cfg.FirstBloodBonus = competitionCfg.Scoring.FirstBloodBonus
			}
		}
	}
	// item.SolveCount 表示当前提交入库前、题目已经被解出的次数。
	// 文档里的 solve_count 是“当前解题队伍数”，包含本次成功提交。
	// 因此本次奖励按 solve_count+1 计算当前分值；下一支队伍看到的公开分值再按 solve_count+2 计算。
	currentSolveScore := calculateJeopardyScore(challenge.BaseScore, int64(item.SolveCount+1), cfg, false)
	scoreAwarded := currentSolveScore
	if isFirstBlood && cfg.FirstBloodBonus > 0 {
		scoreAwarded += int(float64(currentSolveScore) * cfg.FirstBloodBonus)
	}
	newChallengeScore := calculateJeopardyScore(challenge.BaseScore, int64(item.SolveCount+2), cfg, false)
	return scoreAwarded, newChallengeScore
}

// calculateTeamRank 计算团队当前排名。
func (s *teamService) calculateTeamRank(ctx context.Context, competitionID, teamID int64) int {
	teams, err := s.teamRepo.ListRegisteredByCompetitionID(ctx, competitionID)
	if err != nil {
		return 0
	}
	rankings := make([]dto.LeaderboardRankingItem, 0, len(teams))
	for _, team := range teams {
		lastSolveAt, _ := s.submissionRepo.LastCorrectSubmissionAt(ctx, competitionID, team.ID)
		rankings = append(rankings, dto.LeaderboardRankingItem{
			TeamID:      int64String(team.ID),
			TeamName:    team.Name,
			Score:       team.TotalScore,
			SolveCount:  intPtr(0),
			LastSolveAt: optionalTimeString(lastSolveAt),
		})
	}
	sortLeaderboard(rankings)
	for idx, item := range rankings {
		if item.TeamID == int64String(teamID) {
			return idx + 1
		}
	}
	return 0
}

// checkSubmissionWindow 检查题目提交是否命中限流或冷却。
// 该方法优先使用 Redis 规范键实现实时限流，Redis 不可用时再回退到数据库兜底。
func (s *teamService) checkSubmissionWindow(ctx context.Context, competition *entity.Competition, teamID, challengeID int64) (*time.Time, int, error) {
	competitionID := competition.ID
	cfg := loadJeopardySubmissionLimitConfig(competition)
	now := time.Now()
	cooldownKey := buildCTFCooldownKey(competitionID, teamID, challengeID)
	rateKey := buildCTFRateLimitKey(competitionID, teamID, challengeID)

	if cache.Get() != nil {
		if exists, err := cache.Exists(ctx, cooldownKey); err == nil && exists {
			ttl, ttlErr := cache.TTL(ctx, cooldownKey)
			if ttlErr == nil && ttl > 0 {
				cooldownUntil := now.Add(ttl)
				return &cooldownUntil, 0, nil
			}
			cooldownUntil := now.Add(time.Duration(cfg.CooldownMinutes) * time.Minute)
			return &cooldownUntil, 0, nil
		}
		attempts, err := cache.IncrWithExpire(ctx, rateKey, time.Minute)
		if err == nil {
			remaining := cfg.MaxPerMinute - int(attempts)
			if remaining < 0 {
				remaining = 0
			}
			if attempts > int64(cfg.MaxPerMinute) {
				return nil, 0, errcode.ErrSubmissionRateLimit
			}
			return nil, remaining, nil
		}
	}

	recentSubmissions, _, err := s.submissionRepo.List(ctx, &ctfrepo.SubmissionListParams{
		CompetitionID: competitionID,
		TeamID:        teamID,
		ChallengeID:   challengeID,
		From:          timePtr(now.Add(-time.Duration(cfg.CooldownMinutes) * time.Minute)),
		Page:          1,
		PageSize:      100,
	})
	if err != nil {
		return nil, 0, err
	}
	lastMinuteAttempts := 0
	consecutiveFailures := 0
	for _, item := range recentSubmissions {
		if item.CreatedAt.After(now.Add(-1 * time.Minute)) {
			lastMinuteAttempts++
		}
		if item.IsCorrect {
			break
		}
		consecutiveFailures++
	}
	if consecutiveFailures >= cfg.CooldownThreshold && len(recentSubmissions) > 0 {
		cooldownUntil := recentSubmissions[0].CreatedAt.Add(time.Duration(cfg.CooldownMinutes) * time.Minute)
		if cooldownUntil.After(now) {
			return &cooldownUntil, 0, nil
		}
	}
	remaining := cfg.MaxPerMinute - lastMinuteAttempts
	if remaining < 0 {
		remaining = 0
	}
	if lastMinuteAttempts >= cfg.MaxPerMinute {
		return nil, 0, errcode.ErrSubmissionRateLimit
	}
	return nil, remaining, nil
}

// hasSolvedChallenge 优先通过缓存判断团队是否已解出题目，缓存失效时回退数据库判定。
func (s *teamService) hasSolvedChallenge(ctx context.Context, competitionID, teamID, challengeID int64) (bool, error) {
	solvedKey := buildCTFSolvedKey(competitionID, teamID, challengeID)
	if cache.Get() != nil {
		if exists, err := cache.Exists(ctx, solvedKey); err == nil && exists {
			return true, nil
		}
	}
	return s.submissionRepo.HasCorrectSubmission(ctx, competitionID, teamID, challengeID)
}

// recordFailedSubmission 记录失败提交次数，并在达到阈值后写入冷却键。
func (s *teamService) recordFailedSubmission(ctx context.Context, competition *entity.Competition, teamID, challengeID int64) {
	if cache.Get() == nil || competition == nil {
		return
	}
	cfg := loadJeopardySubmissionLimitConfig(competition)
	failKey := buildCTFFailCountKey(competition.ID, teamID, challengeID)
	cooldownKey := buildCTFCooldownKey(competition.ID, teamID, challengeID)
	expiration := time.Duration(cfg.CooldownMinutes) * time.Minute
	failCount, err := cache.IncrWithExpire(ctx, failKey, expiration)
	if err != nil {
		return
	}
	if failCount >= int64(cfg.CooldownThreshold) {
		_ = cache.Set(ctx, cooldownKey, "1", expiration)
	}
}

// recordSuccessfulSubmission 清理失败计数并写入已解题缓存，避免重复解题和无效冷却。
func (s *teamService) recordSuccessfulSubmission(ctx context.Context, competition *entity.Competition, teamID, challengeID int64) {
	if cache.Get() == nil || competition == nil {
		return
	}
	_ = cache.Del(
		ctx,
		buildCTFFailCountKey(competition.ID, teamID, challengeID),
		buildCTFCooldownKey(competition.ID, teamID, challengeID),
	)
	_ = cache.Set(ctx, buildCTFSolvedKey(competition.ID, teamID, challengeID), "1", buildSolvedCacheTTL(competition))
}

// itemSolveCountAfterSuccess 计算题目在本次成功提交后的累计解出次数。
func itemSolveCountAfterSuccess(item *entity.CompetitionChallenge) int {
	if item == nil {
		return 0
	}
	return item.SolveCount + 1
}

// loadJeopardySubmissionLimitConfig 解析解题赛提交限流配置，并补齐文档默认值。
func loadJeopardySubmissionLimitConfig(competition *entity.Competition) dto.JeopardySubmissionLimitConfig {
	cfg := dto.JeopardySubmissionLimitConfig{
		MaxPerMinute:      5,
		CooldownThreshold: 10,
		CooldownMinutes:   5,
	}
	if competition == nil || len(competition.JeopardyConfig) == 0 {
		return cfg
	}
	var competitionCfg dto.JeopardyCompetitionConfig
	if err := decodeJSON(competition.JeopardyConfig, &competitionCfg); err != nil {
		return cfg
	}
	if competitionCfg.SubmissionLimit.MaxPerMinute > 0 {
		cfg.MaxPerMinute = competitionCfg.SubmissionLimit.MaxPerMinute
	}
	if competitionCfg.SubmissionLimit.CooldownThreshold > 0 {
		cfg.CooldownThreshold = competitionCfg.SubmissionLimit.CooldownThreshold
	}
	if competitionCfg.SubmissionLimit.CooldownMinutes > 0 {
		cfg.CooldownMinutes = competitionCfg.SubmissionLimit.CooldownMinutes
	}
	return cfg
}

// buildSolvedCacheTTL 计算已解题缓存的有效期，优先对齐竞赛结束时间。
func buildSolvedCacheTTL(competition *entity.Competition) time.Duration {
	if competition != nil && competition.EndAt != nil && competition.EndAt.After(time.Now()) {
		return time.Until(*competition.EndAt) + time.Hour
	}
	return 30 * 24 * time.Hour
}

// buildCTFRateLimitKey 构建题目提交限流键。
func buildCTFRateLimitKey(competitionID, teamID, challengeID int64) string {
	return cache.KeyCTFRateLimit + strconv.FormatInt(competitionID, 10) + ":" + strconv.FormatInt(teamID, 10) + ":" + strconv.FormatInt(challengeID, 10)
}

// buildCTFFailCountKey 构建题目连续失败计数键。
func buildCTFFailCountKey(competitionID, teamID, challengeID int64) string {
	return cache.KeyCTFFailCount + strconv.FormatInt(competitionID, 10) + ":" + strconv.FormatInt(teamID, 10) + ":" + strconv.FormatInt(challengeID, 10)
}

// buildCTFCooldownKey 构建题目冷却状态键。
func buildCTFCooldownKey(competitionID, teamID, challengeID int64) string {
	return cache.KeyCTFCooldown + strconv.FormatInt(competitionID, 10) + ":" + strconv.FormatInt(teamID, 10) + ":" + strconv.FormatInt(challengeID, 10)
}

// buildCTFSolvedKey 构建团队已解题缓存键。
func buildCTFSolvedKey(competitionID, teamID, challengeID int64) string {
	return cache.KeyCTFSolved + strconv.FormatInt(competitionID, 10) + ":" + strconv.FormatInt(teamID, 10) + ":" + strconv.FormatInt(challengeID, 10)
}

// publishLeaderboardUpdate 在成功解题后广播最新排行榜。
func (s *teamService) publishLeaderboardUpdate(ctx context.Context, competitionID int64, team *entity.Team, challenge *entity.Challenge, isFirstBlood bool) {
	if s.realtimePublisher == nil || team == nil || challenge == nil {
		return
	}
	_ = s.realtimePublisher.PublishLeaderboardUpdate(ctx, competitionID, nil, &LeaderboardRealtimeTrigger{
		TeamName:       team.Name,
		ChallengeTitle: challenge.Title,
		IsFirstBlood:   isFirstBlood,
	})
}
