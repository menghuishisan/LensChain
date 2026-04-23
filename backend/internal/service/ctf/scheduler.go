// scheduler.go
// 模块05 — CTF竞赛：定时任务执行器。
// 负责竞赛状态流转、排行榜快照、环境回收、攻防赛回合推进和归档清理。

package ctf

import (
	"context"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/lenschain/backend/internal/model/dto"
	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/model/enum"
	"github.com/lenschain/backend/internal/pkg/database"
	"github.com/lenschain/backend/internal/pkg/errcode"
	"github.com/lenschain/backend/internal/pkg/logger"
	"github.com/lenschain/backend/internal/pkg/snowflake"
	ctfrepo "github.com/lenschain/backend/internal/repository/ctf"
)

// CTFScheduler 模块05后台定时任务执行器。
type CTFScheduler struct {
	db                 *gorm.DB
	competitionRepo    ctfrepo.CompetitionRepository
	compChallengeRepo  ctfrepo.CompetitionChallengeRepository
	teamRepo           ctfrepo.TeamRepository
	leaderboardRepo    ctfrepo.LeaderboardSnapshotRepository
	environmentRepo    ctfrepo.ChallengeEnvironmentRepository
	verificationRepo   ctfrepo.ChallengeVerificationRepository
	adGroupRepo        ctfrepo.AdGroupRepository
	adRoundRepo        ctfrepo.AdRoundRepository
	adAttackRepo       ctfrepo.AdAttackRepository
	adDefenseRepo      ctfrepo.AdDefenseRepository
	adLedgerRepo       ctfrepo.AdTokenLedgerRepository
	adChainRepo        ctfrepo.AdTeamChainRepository
	competitionService competitionSchedulerService
	battleService      battleSchedulerService
	environmentService environmentSchedulerService
	realtimePublisher  CTFRealtimePublisher
}

// competitionSchedulerService 声明调度器复用的竞赛服务最小能力集合。
type competitionSchedulerService interface {
	CompetitionService
	buildLeaderboardRankings(ctx context.Context, competition *entity.Competition, groupID *int64, top int) ([]dto.LeaderboardRankingItem, *string, error)
}

// environmentSchedulerService 声明调度器复用的环境服务最小能力集合。
type environmentSchedulerService interface {
	EnvironmentService
	destroyEnvironment(ctx context.Context, environment *entity.ChallengeEnvironment) error
}

var _ competitionSchedulerService = (*competitionService)(nil)

// NewCTFScheduler 创建模块05定时任务执行器。
func NewCTFScheduler(
	db *gorm.DB,
	competitionRepo ctfrepo.CompetitionRepository,
	compChallengeRepo ctfrepo.CompetitionChallengeRepository,
	teamRepo ctfrepo.TeamRepository,
	leaderboardRepo ctfrepo.LeaderboardSnapshotRepository,
	environmentRepo ctfrepo.ChallengeEnvironmentRepository,
	verificationRepo ctfrepo.ChallengeVerificationRepository,
	adGroupRepo ctfrepo.AdGroupRepository,
	adRoundRepo ctfrepo.AdRoundRepository,
	adAttackRepo ctfrepo.AdAttackRepository,
	adDefenseRepo ctfrepo.AdDefenseRepository,
	adLedgerRepo ctfrepo.AdTokenLedgerRepository,
	adChainRepo ctfrepo.AdTeamChainRepository,
	competitionService competitionSchedulerService,
	battleService battleSchedulerService,
	environmentService environmentSchedulerService,
	realtimePublisher CTFRealtimePublisher,
) *CTFScheduler {
	return &CTFScheduler{
		db:                 db,
		competitionRepo:    competitionRepo,
		compChallengeRepo:  compChallengeRepo,
		teamRepo:           teamRepo,
		leaderboardRepo:    leaderboardRepo,
		environmentRepo:    environmentRepo,
		verificationRepo:   verificationRepo,
		adGroupRepo:        adGroupRepo,
		adRoundRepo:        adRoundRepo,
		adAttackRepo:       adAttackRepo,
		adDefenseRepo:      adDefenseRepo,
		adLedgerRepo:       adLedgerRepo,
		adChainRepo:        adChainRepo,
		competitionService: competitionService,
		battleService:      battleService,
		environmentService: environmentService,
		realtimePublisher:  realtimePublisher,
	}
}

// RunCompetitionStatusTransition 按时间推进竞赛状态，并补齐报名锁定与攻防赛分组启动。
func (s *CTFScheduler) RunCompetitionStatusTransition() {
	ctx := context.Background()
	now := time.Now().UTC()

	toStart, err := s.competitionRepo.ListRegistrationToStart(ctx, now)
	if err != nil {
		logger.L.Error("查询待开始竞赛失败", zap.Error(err))
		return
	}
	for _, competition := range toStart {
		if err := s.startCompetition(ctx, competition); err != nil {
			logger.L.Error("自动开始竞赛失败",
				zap.Int64("competition_id", competition.ID),
				zap.Error(err),
			)
			continue
		}
		logger.L.Info("竞赛已自动开始", zap.Int64("competition_id", competition.ID))
	}

	toEnd, err := s.competitionRepo.ListRunningToEnd(ctx, now)
	if err != nil {
		logger.L.Error("查询待结束竞赛失败", zap.Error(err))
		return
	}
	for _, competition := range toEnd {
		if err := s.finishCompetition(ctx, competition); err != nil {
			logger.L.Error("自动结束竞赛失败",
				zap.Int64("competition_id", competition.ID),
				zap.Error(err),
			)
			continue
		}
		logger.L.Info("竞赛已自动结束", zap.Int64("competition_id", competition.ID))
	}
}

// RunLeaderboardFreeze 检查进行中竞赛是否进入或退出冻结窗口，并维护冻结标记缓存。
func (s *CTFScheduler) RunLeaderboardFreeze() {
	ctx := context.Background()
	competitions, _, err := s.competitionRepo.List(ctx, &ctfrepo.CompetitionListParams{
		Statuses: []int16{enum.CompetitionStatusRunning},
		Page:     1,
		PageSize: 10000,
	})
	if err != nil {
		logger.L.Error("查询冻结榜候选竞赛失败", zap.Error(err))
		return
	}
	now := time.Now().UTC()
	for _, competition := range competitions {
		if competition == nil {
			continue
		}
		isFrozen := competition.FreezeAt != nil && !competition.FreezeAt.After(now)
		writeCompetitionFrozenCache(ctx, competition.ID, isFrozen)
	}
}

// RunLeaderboardSnapshot 为进行中的竞赛定期落排行榜历史快照。
func (s *CTFScheduler) RunLeaderboardSnapshot() {
	ctx := context.Background()
	competitions, _, err := s.competitionRepo.List(ctx, &ctfrepo.CompetitionListParams{
		Statuses: []int16{enum.CompetitionStatusRunning},
		Page:     1,
		PageSize: 10000,
	})
	if err != nil {
		logger.L.Error("查询待快照竞赛失败", zap.Error(err))
		return
	}
	for _, competition := range competitions {
		if err := s.snapshotCompetitionLeaderboard(ctx, competition); err != nil {
			logger.L.Error("写入排行榜快照失败",
				zap.Int64("competition_id", competition.ID),
				zap.Error(err),
			)
		}
	}
}

// RunEnvironmentRecycle 回收已结束或已归档竞赛下仍未销毁的题目环境。
func (s *CTFScheduler) RunEnvironmentRecycle() {
	ctx := context.Background()
	competitions, _, err := s.competitionRepo.List(ctx, &ctfrepo.CompetitionListParams{
		Statuses: []int16{enum.CompetitionStatusEnded, enum.CompetitionStatusArchived},
		Page:     1,
		PageSize: 10000,
	})
	if err != nil {
		logger.L.Error("查询待回收环境竞赛失败", zap.Error(err))
		return
	}
	for _, competition := range competitions {
		if err := s.recycleCompetitionEnvironments(ctx, competition.ID); err != nil {
			logger.L.Error("回收竞赛环境失败",
				zap.Int64("competition_id", competition.ID),
				zap.Error(err),
			)
		}
	}
}

// RunVerificationCleanup 清理超时未完成的题目预验证记录。
func (s *CTFScheduler) RunVerificationCleanup() {
	ctx := context.Background()
	if err := s.cleanupTimeoutVerifications(ctx); err != nil {
		logger.L.Error("清理超时预验证失败", zap.Error(err))
	}
}

// RunTokenBalanceSync 将攻防赛 Redis Token 余额同步回 teams 表，避免缓存和数据库长期漂移。
func (s *CTFScheduler) RunTokenBalanceSync() {
	ctx := context.Background()
	competitions, _, err := s.competitionRepo.List(ctx, &ctfrepo.CompetitionListParams{
		Statuses:        []int16{enum.CompetitionStatusRunning},
		CompetitionType: enum.CompetitionTypeAttackDefense,
		Page:            1,
		PageSize:        10000,
	})
	if err != nil {
		logger.L.Error("查询 Token 同步候选竞赛失败", zap.Error(err))
		return
	}
	for _, competition := range competitions {
		if competition == nil {
			continue
		}
		teams, teamErr := s.teamRepo.ListRegisteredByCompetitionID(ctx, competition.ID)
		if teamErr != nil {
			logger.L.Error("查询 Token 同步候选团队失败",
				zap.Int64("competition_id", competition.ID),
				zap.Error(teamErr),
			)
			continue
		}
		for _, team := range teams {
			if team == nil {
				continue
			}
			cachedBalance, ok := readAdTokenBalanceCache(ctx, competition.ID, team.ID)
			if !ok || derefInt(team.TokenBalance) == cachedBalance {
				continue
			}
			if updateErr := s.teamRepo.UpdateTokenBalance(ctx, team.ID, cachedBalance); updateErr != nil {
				logger.L.Error("同步攻防赛 Token 余额失败",
					zap.Int64("competition_id", competition.ID),
					zap.Int64("team_id", team.ID),
					zap.Error(updateErr),
				)
			}
		}
	}
}

// RunDynamicScoreSync 将解题赛 Redis 动态分值同步回 competition_challenges 表。
func (s *CTFScheduler) RunDynamicScoreSync() {
	ctx := context.Background()
	competitions, _, err := s.competitionRepo.List(ctx, &ctfrepo.CompetitionListParams{
		Statuses:        []int16{enum.CompetitionStatusRunning},
		CompetitionType: enum.CompetitionTypeJeopardy,
		Page:            1,
		PageSize:        10000,
	})
	if err != nil {
		logger.L.Error("查询动态分值同步候选竞赛失败", zap.Error(err))
		return
	}
	for _, competition := range competitions {
		if competition == nil {
			continue
		}
		items, itemErr := s.compChallengeRepo.ListByCompetitionID(ctx, competition.ID)
		if itemErr != nil {
			logger.L.Error("读取动态分值同步候选题目失败",
				zap.Int64("competition_id", competition.ID),
				zap.Error(itemErr),
			)
			continue
		}
		for _, item := range items {
			if item == nil {
				continue
			}
			cachedScore, ok := readChallengeScoreCache(ctx, competition.ID, item.ChallengeID)
			if !ok {
				continue
			}
			if updateErr := s.compChallengeRepo.UpdateScoreFields(ctx, item.ID, map[string]interface{}{
				"current_score": cachedScore.CurrentScore,
				"solve_count":   cachedScore.SolveCount,
				"updated_at":    time.Now(),
			}); updateErr != nil {
				logger.L.Error("同步题目动态分值失败",
					zap.Int64("competition_id", competition.ID),
					zap.Int64("challenge_id", item.ChallengeID),
					zap.Error(updateErr),
				)
			}
		}
	}
}

// RunArchiveCleanup 将结束超过 30 天的竞赛转归档并清理 Redis 热缓存。
func (s *CTFScheduler) RunArchiveCleanup() {
	ctx := context.Background()
	expireBefore := time.Now().UTC().AddDate(0, 0, -30)

	competitions, err := s.competitionRepo.ListEndedBefore(ctx, expireBefore)
	if err != nil {
		logger.L.Error("查询待归档竞赛失败", zap.Error(err))
		return
	}
	for _, competition := range competitions {
		if err := s.archiveCompetition(ctx, competition); err != nil {
			logger.L.Error("归档竞赛失败",
				zap.Int64("competition_id", competition.ID),
				zap.Error(err),
			)
			continue
		}
		logger.L.Info("竞赛已自动归档", zap.Int64("competition_id", competition.ID))
	}
}

// startCompetition 将报名中的竞赛推进到进行中，并同步锁定参赛团队。
func (s *CTFScheduler) startCompetition(ctx context.Context, competition *entity.Competition) error {
	if competition == nil {
		return nil
	}
	var adGroups []*entity.AdGroup
	if competition.CompetitionType == enum.CompetitionTypeAttackDefense {
		if s.battleService == nil {
			return errcode.ErrInternal.WithMessage("攻防赛启动服务未配置")
		}
		groups, err := s.battleService.ensureGroupsReadyForCompetitionStart(ctx, competition)
		if err != nil {
			return err
		}
		adGroups = groups
	}
	err := database.TransactionWithDB(ctx, s.db, func(tx *gorm.DB) error {
		txCompetitionRepo := ctfrepo.NewCompetitionRepository(tx)
		txTeamRepo := ctfrepo.NewTeamRepository(tx)
		txAdGroupRepo := ctfrepo.NewAdGroupRepository(tx)

		if err := txCompetitionRepo.UpdateStatus(ctx, competition.ID, enum.CompetitionStatusRunning); err != nil {
			return err
		}
		if err := txTeamRepo.LockByCompetitionID(ctx, competition.ID); err != nil {
			return err
		}
		if competition.CompetitionType == enum.CompetitionTypeAttackDefense {
			for _, group := range adGroups {
				if err := txAdGroupRepo.UpdateStatus(ctx, group.ID, enum.AdGroupStatusRunning); err != nil {
					return err
				}
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	writeCompetitionStatusCache(ctx, competition.ID, enum.CompetitionStatusRunning)
	writeCompetitionFrozenCache(ctx, competition.ID, false)
	s.publishCompetitionLeaderboard(ctx, competition.ID, nil)
	if competition.CompetitionType == enum.CompetitionTypeAttackDefense {
		s.publishInitialGroupRounds(ctx, competition.ID)
	}
	return nil
}

// finishCompetition 将进行中的竞赛推进到已结束，并补齐最终快照、排名和攻防资源收口。
func (s *CTFScheduler) finishCompetition(ctx context.Context, competition *entity.Competition) error {
	if competition == nil {
		return nil
	}
	err := database.TransactionWithDB(ctx, s.db, func(tx *gorm.DB) error {
		txCompetitionRepo := ctfrepo.NewCompetitionRepository(tx)
		txAdGroupRepo := ctfrepo.NewAdGroupRepository(tx)
		txAdRoundRepo := ctfrepo.NewAdRoundRepository(tx)
		txAdChainRepo := ctfrepo.NewAdTeamChainRepository(tx)

		if err := txCompetitionRepo.UpdateStatus(ctx, competition.ID, enum.CompetitionStatusEnded); err != nil {
			return err
		}
		if competition.CompetitionType == enum.CompetitionTypeAttackDefense {
			groups, err := txAdGroupRepo.ListByCompetitionID(ctx, competition.ID)
			if err != nil {
				return err
			}
			for _, group := range groups {
				if err := txAdGroupRepo.UpdateStatus(ctx, group.ID, enum.AdGroupStatusFinished); err != nil {
					return err
				}
				rounds, err := txAdRoundRepo.ListByGroupID(ctx, group.ID)
				if err != nil {
					return err
				}
				for _, round := range rounds {
					if round.Phase == enum.RoundPhaseCompleted {
						continue
					}
					fields := map[string]interface{}{
						"phase":      enum.RoundPhaseCompleted,
						"updated_at": time.Now(),
					}
					if err := txAdRoundRepo.UpdateFields(ctx, round.ID, fields); err != nil {
						return err
					}
				}
			}
			if err := txAdChainRepo.StopByCompetitionID(ctx, competition.ID); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	if err := s.cleanupEndedCompetitionResources(ctx, competition); err != nil {
		return err
	}
	writeCompetitionStatusCache(ctx, competition.ID, enum.CompetitionStatusEnded)
	writeCompetitionFrozenCache(ctx, competition.ID, false)
	if err := s.persistFinalArtifacts(ctx, competition); err != nil {
		return err
	}
	s.publishCompetitionLeaderboard(ctx, competition.ID, nil)
	return nil
}

// archiveCompetition 将竞赛标记为已归档，并清理仅用于热数据访问的缓存键。
func (s *CTFScheduler) archiveCompetition(ctx context.Context, competition *entity.Competition) error {
	if competition == nil {
		return nil
	}
	if competition.CompetitionType == enum.CompetitionTypeAttackDefense && s.battleService != nil {
		if err := s.battleService.cleanupGroupRuntimesByCompetition(ctx, competition.ID); err != nil {
			return err
		}
	}
	if err := s.competitionRepo.UpdateStatus(ctx, competition.ID, enum.CompetitionStatusArchived); err != nil {
		return err
	}
	writeCompetitionStatusCache(ctx, competition.ID, enum.CompetitionStatusArchived)
	writeCompetitionFrozenCache(ctx, competition.ID, false)
	clearCompetitionCacheData(ctx, s.adGroupRepo, s.adRoundRepo, competition.ID)
	return nil
}

// recycleCompetitionEnvironments 逐个销毁竞赛下尚未销毁的环境，确保配额也按正确事务路径回收。
func (s *CTFScheduler) recycleCompetitionEnvironments(ctx context.Context, competitionID int64) error {
	environments, err := s.environmentRepo.ListByCompetitionID(ctx, competitionID)
	if err != nil {
		return err
	}
	for _, environment := range environments {
		if environment.Status == enum.ChallengeEnvStatusDestroyed {
			continue
		}
		if err := s.environmentService.destroyEnvironment(ctx, environment); err != nil {
			return err
		}
	}
	if s.battleService != nil {
		if err := s.battleService.cleanupGroupRuntimesByCompetition(ctx, competitionID); err != nil {
			return err
		}
	}
	return nil
}

// cleanupEndedCompetitionResources 在竞赛进入结束态时立即启动环境与运行时资源回收。
func (s *CTFScheduler) cleanupEndedCompetitionResources(ctx context.Context, competition *entity.Competition) error {
	if competition == nil {
		return nil
	}
	return s.recycleCompetitionEnvironments(ctx, competition.ID)
}

// snapshotCompetitionLeaderboard 将当前排行榜写入 leaderboard_snapshots，用于历史榜和冻结榜读取。
func (s *CTFScheduler) snapshotCompetitionLeaderboard(ctx context.Context, competition *entity.Competition) error {
	if competition == nil {
		return nil
	}
	rankings, _, err := s.competitionService.buildLeaderboardRankings(ctx, competition, nil, 0)
	if err != nil {
		return err
	}
	snapshotAt := time.Now().UTC()
	isFrozen := competition.FreezeAt != nil && !competition.FreezeAt.After(snapshotAt)
	snapshots := make([]*entity.LeaderboardSnapshot, 0, len(rankings))
	for _, ranking := range rankings {
		teamID, parseErr := snowflake.ParseString(ranking.TeamID)
		if parseErr != nil {
			return parseErr
		}
		snapshot := &entity.LeaderboardSnapshot{
			ID:            snowflake.Generate(),
			CompetitionID: competition.ID,
			TeamID:        teamID,
			Rank:          ranking.Rank,
			IsFrozen:      isFrozen,
			SnapshotAt:    snapshotAt,
			CreatedAt:     snapshotAt,
		}
		if competition.CompetitionType == enum.CompetitionTypeAttackDefense {
			snapshot.Score = derefInt(ranking.TokenBalance)
		} else {
			snapshot.Score = derefInt(ranking.Score)
		}
		if ranking.SolveCount != nil {
			value := *ranking.SolveCount
			snapshot.SolveCount = &value
		}
		if ranking.LastSolveAt != nil && *ranking.LastSolveAt != "" {
			parsedAt, parseErr := time.Parse(time.RFC3339, *ranking.LastSolveAt)
			if parseErr == nil {
				snapshot.LastSolveAt = &parsedAt
			}
		}
		snapshots = append(snapshots, snapshot)
	}
	return s.leaderboardRepo.BatchCreate(ctx, snapshots)
}

// persistFinalArtifacts 在竞赛结束时补齐最终排行榜快照与 team.final_rank 字段。
func (s *CTFScheduler) persistFinalArtifacts(ctx context.Context, competition *entity.Competition) error {
	if err := s.snapshotCompetitionLeaderboard(ctx, competition); err != nil {
		return err
	}
	return s.updateCompetitionFinalRanks(ctx, competition)
}

// updateCompetitionFinalRanks 按最终排行榜回写团队最终名次。
func (s *CTFScheduler) updateCompetitionFinalRanks(ctx context.Context, competition *entity.Competition) error {
	if competition == nil {
		return nil
	}
	rankings, _, err := s.competitionService.buildLeaderboardRankings(ctx, competition, nil, 0)
	if err != nil {
		return err
	}
	updates := make([]ctfrepo.TeamRankUpdate, 0, len(rankings))
	for _, ranking := range rankings {
		teamID, parseErr := snowflake.ParseString(ranking.TeamID)
		if parseErr != nil {
			return parseErr
		}
		updates = append(updates, ctfrepo.TeamRankUpdate{
			TeamID:    teamID,
			FinalRank: ranking.Rank,
		})
	}
	return s.teamRepo.UpdateFinalRanks(ctx, updates)
}

// cleanupTimeoutVerifications 清理超时未完成的预验证记录，并补写可解释的失败结论。
func (s *CTFScheduler) cleanupTimeoutVerifications(ctx context.Context) error {
	if s.verificationRepo == nil {
		return nil
	}
	timeoutBefore := time.Now().UTC().Add(-1 * time.Hour)
	verifications, err := s.verificationRepo.ListTimeoutRunning(ctx, timeoutBefore)
	if err != nil {
		return err
	}
	for _, verification := range verifications {
		stepResults := buildVerificationTimeoutSteps(verification)
		if err := s.verificationRepo.UpdateFields(ctx, verification.ID, map[string]interface{}{
			"step_results":   mustJSON(stepResults),
			"environment_id": verification.EnvironmentID,
		}); err != nil {
			return err
		}
		errMsg := "预验证执行超时，系统已自动终止临时环境"
		if err := s.verificationRepo.Complete(ctx, verification.ID, enum.VerificationStatusFailed, &errMsg); err != nil {
			return err
		}
	}
	return nil
}

// publishCompetitionLeaderboard 在调度器驱动的状态迁移后广播最新排行榜。
func (s *CTFScheduler) publishCompetitionLeaderboard(ctx context.Context, competitionID int64, groupID *int64) {
	if s.realtimePublisher == nil {
		return
	}
	_ = s.realtimePublisher.PublishLeaderboardUpdate(ctx, competitionID, groupID, nil)
}
