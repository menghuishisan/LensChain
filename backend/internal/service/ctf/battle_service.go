// battle_service.go
// 模块05 — CTF竞赛：攻防对抗赛业务逻辑。
// 负责分组初始化、回合查询、攻击与防守验证、Token 流水、队伍链信息和分组视角数据聚合。

package ctf

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/lenschain/backend/internal/model/dto"
	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/model/enum"
	svcctx "github.com/lenschain/backend/internal/pkg/context"
	"github.com/lenschain/backend/internal/pkg/database"
	"github.com/lenschain/backend/internal/pkg/errcode"
	"github.com/lenschain/backend/internal/pkg/snowflake"
	ctfrepo "github.com/lenschain/backend/internal/repository/ctf"
)

// battleService 攻防对抗赛服务实现。
type battleService struct {
	db                 *gorm.DB
	competitionRepo    ctfrepo.CompetitionRepository
	challengeRepo      ctfrepo.ChallengeRepository
	assertionRepo      ctfrepo.ChallengeAssertionRepository
	contractRepo       ctfrepo.ChallengeContractRepository
	verificationRepo   ctfrepo.ChallengeVerificationRepository
	compChallengeRepo  ctfrepo.CompetitionChallengeRepository
	teamRepo           ctfrepo.TeamRepository
	teamMemberRepo     ctfrepo.TeamMemberRepository
	adGroupRepo        ctfrepo.AdGroupRepository
	adRoundRepo        ctfrepo.AdRoundRepository
	adAttackRepo       ctfrepo.AdAttackRepository
	adDefenseRepo      ctfrepo.AdDefenseRepository
	adLedgerRepo       ctfrepo.AdTokenLedgerRepository
	adChainRepo        ctfrepo.AdTeamChainRepository
	quotaRepo          ctfrepo.CtfResourceQuotaRepository
	runtimeProvisioner ADRuntimeProvisioner
	attackExecutor     ADAttackExecutor
	patchVerifier      ADPatchVerifier
	realtimePublisher  CTFRealtimePublisher
}

// battleSchedulerService 声明调度器复用的攻防赛最小能力集合。
// 调度器只负责编排竞赛状态流转，不直接拼接分组与运行时初始化细节。
type battleSchedulerService interface {
	ensureGroupsReadyForCompetitionStart(ctx context.Context, competition *entity.Competition) ([]*entity.AdGroup, error)
	cleanupGroupRuntimesByCompetition(ctx context.Context, competitionID int64) error
}

var _ BattleService = (*battleService)(nil)
var _ battleSchedulerService = (*battleService)(nil)

// NewBattleService 创建攻防对抗赛服务实例。
func NewBattleService(
	db *gorm.DB,
	competitionRepo ctfrepo.CompetitionRepository,
	challengeRepo ctfrepo.ChallengeRepository,
	assertionRepo ctfrepo.ChallengeAssertionRepository,
	contractRepo ctfrepo.ChallengeContractRepository,
	verificationRepo ctfrepo.ChallengeVerificationRepository,
	compChallengeRepo ctfrepo.CompetitionChallengeRepository,
	teamRepo ctfrepo.TeamRepository,
	teamMemberRepo ctfrepo.TeamMemberRepository,
	adGroupRepo ctfrepo.AdGroupRepository,
	adRoundRepo ctfrepo.AdRoundRepository,
	adAttackRepo ctfrepo.AdAttackRepository,
	adDefenseRepo ctfrepo.AdDefenseRepository,
	adLedgerRepo ctfrepo.AdTokenLedgerRepository,
	adChainRepo ctfrepo.AdTeamChainRepository,
	quotaRepo ctfrepo.CtfResourceQuotaRepository,
	runtimeProvisioner ADRuntimeProvisioner,
	attackExecutor ADAttackExecutor,
	patchVerifier ADPatchVerifier,
	realtimePublisher CTFRealtimePublisher,
) *battleService {
	return &battleService{
		db:                 db,
		competitionRepo:    competitionRepo,
		challengeRepo:      challengeRepo,
		assertionRepo:      assertionRepo,
		contractRepo:       contractRepo,
		verificationRepo:   verificationRepo,
		compChallengeRepo:  compChallengeRepo,
		teamRepo:           teamRepo,
		teamMemberRepo:     teamMemberRepo,
		adGroupRepo:        adGroupRepo,
		adRoundRepo:        adRoundRepo,
		adAttackRepo:       adAttackRepo,
		adDefenseRepo:      adDefenseRepo,
		adLedgerRepo:       adLedgerRepo,
		adChainRepo:        adChainRepo,
		quotaRepo:          quotaRepo,
		runtimeProvisioner: runtimeProvisioner,
		attackExecutor:     attackExecutor,
		patchVerifier:      patchVerifier,
		realtimePublisher:  realtimePublisher,
	}
}

// CreateAdGroup 创建攻防赛分组并初始化分组资源。
func (s *battleService) CreateAdGroup(ctx context.Context, sc *svcctx.ServiceContext, competitionID int64, req *dto.CreateAdGroupReq) (*dto.AdGroupResp, error) {
	competition, err := getCompetition(ctx, s.competitionRepo, competitionID)
	if err != nil {
		return nil, err
	}
	if err := s.ensureCompetitionManageable(sc, competition); err != nil {
		return nil, err
	}
	cfg, err := s.loadADConfig(competition)
	if err != nil {
		return nil, err
	}
	teamIDs, err := parseSnowflakeList(req.TeamIDs)
	if err != nil {
		return nil, errcode.ErrInvalidID
	}
	if cfg.MaxTeamsPerGroup > 0 && len(teamIDs) > cfg.MaxTeamsPerGroup {
		return nil, errcode.ErrCompetitionConfigRequired.WithMessage("分组队伍数超过上限")
	}
	teams, err := s.loadAssignableTeams(ctx, competitionID, teamIDs)
	if err != nil {
		return nil, err
	}
	group, err := s.createAdGroupWithTeams(ctx, competition, cfg, strings.TrimSpace(req.GroupName), teams)
	if err != nil {
		return nil, err
	}
	return s.buildAdGroupResp(ctx, group)
}

// ListAdGroups 查询竞赛分组列表。
func (s *battleService) ListAdGroups(ctx context.Context, sc *svcctx.ServiceContext, competitionID int64) (*dto.AdGroupListResp, error) {
	competition, err := getCompetition(ctx, s.competitionRepo, competitionID)
	if err != nil {
		return nil, err
	}
	if err := s.ensureAdCompetitionReadable(ctx, sc, competition); err != nil {
		return nil, err
	}
	groups, err := s.adGroupRepo.ListByCompetitionID(ctx, competitionID)
	if err != nil {
		return nil, err
	}
	items := make([]dto.AdGroupResp, 0, len(groups))
	for _, group := range groups {
		if !s.canReadGroup(ctx, sc, group) {
			continue
		}
		resp, respErr := s.buildAdGroupResp(ctx, group)
		if respErr != nil {
			return nil, respErr
		}
		items = append(items, *resp)
	}
	return &dto.AdGroupListResp{List: items}, nil
}

// GetAdGroup 获取分组详情。
func (s *battleService) GetAdGroup(ctx context.Context, sc *svcctx.ServiceContext, id int64) (*dto.AdGroupResp, error) {
	group, err := s.adGroupRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errcode.ErrAdGroupNotFound
		}
		return nil, err
	}
	if !s.canReadGroup(ctx, sc, group) {
		return nil, errcode.ErrForbidden
	}
	return s.buildAdGroupResp(ctx, group)
}

// AutoAssignAdGroups 按固定人数自动分组并初始化所有分组资源。
func (s *battleService) AutoAssignAdGroups(ctx context.Context, sc *svcctx.ServiceContext, competitionID int64, req *dto.AutoAssignAdGroupsReq) (*dto.AutoAssignAdGroupsResp, error) {
	competition, err := getCompetition(ctx, s.competitionRepo, competitionID)
	if err != nil {
		return nil, err
	}
	if err := s.ensureCompetitionManageable(sc, competition); err != nil {
		return nil, err
	}
	cfg, err := s.loadADConfig(competition)
	if err != nil {
		return nil, err
	}
	if cfg.MaxTeamsPerGroup > 0 && req.TeamsPerGroup > cfg.MaxTeamsPerGroup {
		return nil, errcode.ErrCompetitionConfigRequired.WithMessage("分组队伍数超过上限")
	}
	teams, err := s.teamRepo.ListRegisteredByCompetitionID(ctx, competitionID)
	if err != nil {
		return nil, err
	}
	assignableTeams := make([]*entity.Team, 0, len(teams))
	for _, team := range teams {
		if team == nil {
			continue
		}
		if team.AdGroupID != nil && *team.AdGroupID > 0 {
			continue
		}
		assignableTeams = append(assignableTeams, team)
	}
	if len(assignableTeams) == 0 {
		return nil, errcode.ErrCompetitionConfigRequired.WithMessage("当前无可自动分组的队伍")
	}
	teamIDs := collectTeamIDs(assignableTeams)
	groupBuckets := autoAssignTeamsToGroups(teamIDs, req.TeamsPerGroup)
	teamMap := make(map[int64]*entity.Team, len(assignableTeams))
	for _, team := range assignableTeams {
		teamMap[team.ID] = team
	}
	items := make([]dto.AutoAssignedGroupItem, 0, len(groupBuckets))
	for idx, bucket := range groupBuckets {
		groupTeams := make([]*entity.Team, 0, len(bucket))
		for _, teamID := range bucket {
			if team := teamMap[teamID]; team != nil {
				groupTeams = append(groupTeams, team)
			}
		}
		groupName := buildAdGroupName(idx + 1)
		group, createErr := s.createAdGroupWithTeams(ctx, competition, cfg, groupName, groupTeams)
		if createErr != nil {
			return nil, createErr
		}
		items = append(items, dto.AutoAssignedGroupItem{
			ID:        int64String(group.ID),
			GroupName: group.GroupName,
			TeamCount: len(groupTeams),
		})
	}
	return &dto.AutoAssignAdGroupsResp{
		Groups:      items,
		TotalTeams:  len(assignableTeams),
		TotalGroups: len(items),
	}, nil
}

// ListRounds 查询分组回合列表。
// createAdGroupWithTeams 创建单个分组，并同步初始化回合、队伍链与初始 Token。
func (s *battleService) createAdGroupWithTeams(ctx context.Context, competition *entity.Competition, cfg *dto.ADCompetitionConfig, groupName string, teams []*entity.Team) (*entity.AdGroup, error) {
	if competition.Status != enum.CompetitionStatusDraft && competition.Status != enum.CompetitionStatusRegistration {
		return nil, errcode.ErrCompetitionStatusInvalid.WithMessage("竞赛必须处于草稿或报名状态")
	}
	group := &entity.AdGroup{
		ID:            snowflake.Generate(),
		CompetitionID: competition.ID,
		GroupName:     groupName,
		Namespace:     stringPtr(buildAdGroupNamespace(competition.ID, snowflake.Generate())),
		Status:        enum.AdGroupStatusPreparing,
		CreatedAt:     time.Now(),
	}
	groupNamespace := buildAdGroupNamespace(competition.ID, group.ID)
	group.Namespace = &groupNamespace
	if err := s.reserveAdGroupNamespaceQuota(ctx, competition.ID); err != nil {
		return nil, err
	}
	challengeItems, err := s.compChallengeRepo.ListByCompetitionID(ctx, competition.ID)
	if err != nil {
		s.rollbackFailedAdGroupProvision(ctx, competition.ID, nil)
		return nil, err
	}
	challengeIDs := make([]int64, 0, len(challengeItems))
	for _, item := range challengeItems {
		challengeIDs = append(challengeIDs, item.ChallengeID)
	}
	challenges, err := s.challengeRepo.GetByIDs(ctx, challengeIDs)
	if err != nil {
		s.rollbackFailedAdGroupProvision(ctx, competition.ID, nil)
		return nil, err
	}
	contracts, err := s.contractRepo.ListByChallengeIDs(ctx, challengeIDs)
	if err != nil {
		s.rollbackFailedAdGroupProvision(ctx, competition.ID, nil)
		return nil, err
	}
	challengeMap := make(map[int64]*entity.Challenge, len(challenges))
	for _, challenge := range challenges {
		challengeMap[challenge.ID] = challenge
	}
	contractMap := buildChallengeContractMap(contracts)
	runtimeResult, err := s.createAdGroupRuntime(ctx, competition, groupNamespace, group.ID, teams, challengeItems, challengeMap, contractMap)
	if err != nil {
		s.rollbackFailedAdGroupProvision(ctx, competition.ID, &groupNamespace)
		return nil, err
	}
	teamRuntimeMap := make(map[int64]ADRuntimeTeamResult, len(runtimeResult.Teams))
	for _, item := range runtimeResult.Teams {
		teamRuntimeMap[item.TeamID] = item
	}

	err = database.TransactionWithDB(ctx, s.db, func(tx *gorm.DB) error {
		txGroupRepo := ctfrepo.NewAdGroupRepository(tx)
		txRoundRepo := ctfrepo.NewAdRoundRepository(tx)
		txTeamRepo := ctfrepo.NewTeamRepository(tx)
		txChainRepo := ctfrepo.NewAdTeamChainRepository(tx)
		txLedgerRepo := ctfrepo.NewAdTokenLedgerRepository(tx)
		group.JudgeChainURL = runtimeResult.JudgeChainURL
		group.JudgeContractAddress = runtimeResult.JudgeContractAddress
		if err := txGroupRepo.Create(ctx, group); err != nil {
			return err
		}
		for _, round := range buildAdRounds(competition, group.ID, cfg) {
			if err := txRoundRepo.Create(ctx, round); err != nil {
				return err
			}
		}
		if err := txTeamRepo.AssignAdGroup(ctx, collectTeamIDs(teams), group.ID); err != nil {
			return err
		}
		for _, team := range teams {
			if err := txTeamRepo.UpdateTokenBalance(ctx, team.ID, cfg.InitialToken); err != nil {
				return err
			}
			teamRuntime, ok := teamRuntimeMap[team.ID]
			if !ok {
				return errcode.ErrInternal.WithMessage("队伍链运行时初始化结果缺失")
			}
			chain := &entity.AdTeamChain{
				ID:                  snowflake.Generate(),
				CompetitionID:       competition.ID,
				GroupID:             group.ID,
				TeamID:              team.ID,
				ChainRPCURL:         teamRuntime.ChainRPCURL,
				ChainWSURL:          teamRuntime.ChainWSURL,
				DeployedContracts:   mustJSON(teamRuntime.DeployedContracts),
				CurrentPatchVersion: teamRuntime.CurrentPatchVersion,
				Status:              teamRuntime.Status,
				CreatedAt:           time.Now(),
			}
			if err := txChainRepo.Create(ctx, chain); err != nil {
				return err
			}
			ledger := &entity.AdTokenLedger{
				ID:            snowflake.Generate(),
				CompetitionID: competition.ID,
				GroupID:       group.ID,
				TeamID:        team.ID,
				ChangeType:    enum.TokenChangeInit,
				Amount:        cfg.InitialToken,
				BalanceAfter:  cfg.InitialToken,
				Description:   stringPtr("攻防赛分组初始化 Token"),
				CreatedAt:     time.Now(),
			}
			if err := txLedgerRepo.Create(ctx, ledger); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		s.rollbackFailedAdGroupProvision(ctx, competition.ID, &groupNamespace)
		return nil, err
	}
	for _, team := range teams {
		if team == nil {
			continue
		}
		writeAdTokenBalanceCache(ctx, competition.ID, team.ID, cfg.InitialToken)
	}
	return group, nil
}

// ensureGroupsReadyForCompetitionStart 确保攻防赛在开始前拥有可运行的分组。
// 若管理员未手动分组，则按文档要求在竞赛启动前执行随机自动分组。
func (s *battleService) ensureGroupsReadyForCompetitionStart(ctx context.Context, competition *entity.Competition) ([]*entity.AdGroup, error) {
	if competition == nil || competition.CompetitionType != enum.CompetitionTypeAttackDefense {
		return nil, nil
	}
	groups, err := s.adGroupRepo.ListByCompetitionID(ctx, competition.ID)
	if err != nil {
		return nil, err
	}
	if len(groups) > 0 {
		return groups, nil
	}

	cfg, err := s.loadADConfig(competition)
	if err != nil {
		return nil, err
	}
	teams, err := s.teamRepo.ListRegisteredByCompetitionID(ctx, competition.ID)
	if err != nil {
		return nil, err
	}
	if len(teams) == 0 {
		return nil, errcode.ErrCompetitionStatusInvalid.WithMessage("攻防赛开始前至少需要1支已报名队伍")
	}

	groupBuckets := autoAssignTeamsToGroups(collectTeamIDs(teams), cfg.MaxTeamsPerGroup)
	teamMap := make(map[int64]*entity.Team, len(teams))
	for _, team := range teams {
		teamMap[team.ID] = team
	}

	createdGroups := make([]*entity.AdGroup, 0, len(groupBuckets))
	for idx, bucket := range groupBuckets {
		groupTeams := make([]*entity.Team, 0, len(bucket))
		for _, teamID := range bucket {
			if team := teamMap[teamID]; team != nil {
				groupTeams = append(groupTeams, team)
			}
		}
		if len(groupTeams) == 0 {
			continue
		}
		group, createErr := s.createAdGroupWithTeams(ctx, competition, cfg, buildAdGroupName(idx+1), groupTeams)
		if createErr != nil {
			return nil, createErr
		}
		createdGroups = append(createdGroups, group)
	}
	return createdGroups, nil
}

// cleanupGroupRuntimesByCompetition 清理竞赛下全部攻防赛分组运行时命名空间。
// 该方法在回收真实运行时后，同时释放资源配额占用并清空运行时访问字段，
// 但不修改分组状态字段，分组状态收口仍由调度器事务统一处理。
func (s *battleService) cleanupGroupRuntimesByCompetition(ctx context.Context, competitionID int64) error {
	if competitionID == 0 || s.runtimeProvisioner == nil {
		return nil
	}
	groups, err := s.adGroupRepo.ListByCompetitionID(ctx, competitionID)
	if err != nil {
		return err
	}
	for _, group := range groups {
		if group == nil || group.Namespace == nil || strings.TrimSpace(*group.Namespace) == "" {
			continue
		}
		namespace := strings.TrimSpace(*group.Namespace)
		if err := s.runtimeProvisioner.DeleteADGroupRuntime(ctx, namespace); err != nil {
			return err
		}
		if err := database.TransactionWithDB(ctx, s.db, func(tx *gorm.DB) error {
			txGroupRepo := ctfrepo.NewAdGroupRepository(tx)
			txQuotaRepo := ctfrepo.NewCtfResourceQuotaRepository(tx)
			if err := txGroupRepo.UpdateFields(ctx, group.ID, map[string]interface{}{
				"namespace":              nil,
				"judge_chain_url":        nil,
				"judge_contract_address": nil,
				"updated_at":             time.Now(),
			}); err != nil {
				return err
			}
			return txQuotaRepo.ReleaseNamespaceSlot(ctx, competitionID)
		}); err != nil {
			return err
		}
	}
	return nil
}

// reserveAdGroupNamespaceQuota 为攻防赛分组运行时预占一个 Namespace 配额槽位。
func (s *battleService) reserveAdGroupNamespaceQuota(ctx context.Context, competitionID int64) error {
	if s.quotaRepo == nil {
		return nil
	}
	acquired, err := s.quotaRepo.TryAcquireNamespaceSlot(ctx, competitionID)
	if err != nil {
		return err
	}
	if !acquired {
		return errcode.ErrCompetitionQuotaExceeded
	}
	return nil
}

// rollbackFailedAdGroupProvision 回滚攻防赛分组运行时创建失败后的残留资源与配额占用。
func (s *battleService) rollbackFailedAdGroupProvision(ctx context.Context, competitionID int64, namespace *string) {
	if s.runtimeProvisioner != nil && namespace != nil && strings.TrimSpace(*namespace) != "" {
		_ = s.runtimeProvisioner.DeleteADGroupRuntime(ctx, strings.TrimSpace(*namespace))
	}
	if s.quotaRepo != nil {
		_ = s.quotaRepo.ReleaseNamespaceSlot(ctx, competitionID)
	}
}
