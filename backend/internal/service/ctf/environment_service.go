// environment_service.go
// 模块05 — CTF竞赛：题目环境业务逻辑。
// 负责题目环境的启动、查询、重置、销毁、竞赛级环境清单和资源配额校验。

package ctf

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/lenschain/backend/internal/model/dto"
	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/model/enum"
	"github.com/lenschain/backend/internal/pkg/audit"
	svcctx "github.com/lenschain/backend/internal/pkg/context"
	"github.com/lenschain/backend/internal/pkg/database"
	"github.com/lenschain/backend/internal/pkg/errcode"
	"github.com/lenschain/backend/internal/pkg/snowflake"
	ctfrepo "github.com/lenschain/backend/internal/repository/ctf"
)

// environmentService 题目环境服务实现。
type environmentService struct {
	db                 *gorm.DB
	competitionRepo    ctfrepo.CompetitionRepository
	challengeRepo      ctfrepo.ChallengeRepository
	contractRepo       ctfrepo.ChallengeContractRepository
	compChallengeRepo  ctfrepo.CompetitionChallengeRepository
	teamRepo           ctfrepo.TeamRepository
	teamMemberRepo     ctfrepo.TeamMemberRepository
	environmentRepo    ctfrepo.ChallengeEnvironmentRepository
	quotaRepo          ctfrepo.CtfResourceQuotaRepository
	userQuerier        UserSummaryQuerier
	provisioner        NamespaceProvisioner
	runtimeProvisioner ChallengeEnvironmentProvisioner
}

var _ EnvironmentService = (*environmentService)(nil)
var _ competitionEnvironmentOperator = (*environmentService)(nil)
var _ environmentSchedulerService = (*environmentService)(nil)

// NewEnvironmentService 创建题目环境服务实例。
func NewEnvironmentService(
	db *gorm.DB,
	competitionRepo ctfrepo.CompetitionRepository,
	challengeRepo ctfrepo.ChallengeRepository,
	contractRepo ctfrepo.ChallengeContractRepository,
	compChallengeRepo ctfrepo.CompetitionChallengeRepository,
	teamRepo ctfrepo.TeamRepository,
	teamMemberRepo ctfrepo.TeamMemberRepository,
	environmentRepo ctfrepo.ChallengeEnvironmentRepository,
	quotaRepo ctfrepo.CtfResourceQuotaRepository,
	userQuerier UserSummaryQuerier,
	provisioner NamespaceProvisioner,
	runtimeProvisioner ChallengeEnvironmentProvisioner,
) *environmentService {
	return &environmentService{
		db:                 db,
		competitionRepo:    competitionRepo,
		challengeRepo:      challengeRepo,
		contractRepo:       contractRepo,
		compChallengeRepo:  compChallengeRepo,
		teamRepo:           teamRepo,
		teamMemberRepo:     teamMemberRepo,
		environmentRepo:    environmentRepo,
		quotaRepo:          quotaRepo,
		userQuerier:        userQuerier,
		provisioner:        provisioner,
		runtimeProvisioner: runtimeProvisioner,
	}
}

// Start 启动题目环境。
func (s *environmentService) Start(ctx context.Context, sc *svcctx.ServiceContext, competitionID, challengeID int64) (*dto.StartChallengeEnvironmentResp, error) {
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
	if _, err := s.compChallengeRepo.GetByCompetitionAndChallenge(ctx, competitionID, challengeID); err != nil {
		return nil, errcode.ErrSubmissionChallengeGone.WithMessage("题目不在竞赛中")
	}
	challenge, err := getChallenge(ctx, s.challengeRepo, challengeID)
	if err != nil {
		return nil, err
	}
	_, team, err := s.getCurrentMemberTeam(ctx, competitionID, sc.UserID)
	if err != nil {
		return nil, err
	}
	if existing, err := s.environmentRepo.GetActiveByTeamAndChallenge(ctx, competitionID, team.ID, challengeID); err == nil && existing != nil {
		if existing.Status == enum.ChallengeEnvStatusError {
			return s.retryErroredEnvironmentStart(ctx, existing, challenge)
		}
		return buildStartChallengeEnvironmentResp(existing), nil
	}
	if err := s.ensureQuotaAvailable(ctx, competitionID); err != nil {
		return nil, err
	}

	namespace := buildChallengeEnvironmentNamespace(competitionID, team.ID, challengeID)
	environment := &entity.ChallengeEnvironment{
		ID:            snowflake.Generate(),
		CompetitionID: competitionID,
		ChallengeID:   challengeID,
		TeamID:        team.ID,
		Namespace:     namespace,
		Status:        enum.ChallengeEnvStatusCreating,
		CreatedAt:     time.Now(),
	}

	err = database.TransactionWithDB(ctx, s.db, func(tx *gorm.DB) error {
		txEnvRepo := ctfrepo.NewChallengeEnvironmentRepository(tx)
		txQuotaRepo := ctfrepo.NewCtfResourceQuotaRepository(tx)
		if err := txEnvRepo.Create(ctx, environment); err != nil {
			return err
		}
		quota, quotaErr := txQuotaRepo.GetByCompetitionID(ctx, competitionID)
		if quotaErr == nil && quota != nil {
			if err := txQuotaRepo.IncrementNamespaces(ctx, competitionID, 1); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	if s.runtimeProvisioner == nil {
		s.rollbackFailedEnvironmentStart(ctx, competitionID, environment)
		return nil, errcode.ErrInternal.WithMessage("CTF题目环境运行时未配置")
	}
	spec, err := s.buildChallengeEnvironmentSpec(ctx, challenge, competitionID, team.ID, namespace)
	if err != nil {
		s.rollbackFailedEnvironmentStart(ctx, competitionID, environment)
		return nil, err
	}
	result, err := s.runtimeProvisioner.ProvisionChallengeEnvironment(ctx, spec)
	if err != nil {
		s.rollbackFailedEnvironmentStart(ctx, competitionID, environment)
		return nil, errcode.ErrInternal.WithMessage("题目环境创建失败：" + err.Error())
	}
	if err := s.applyProvisionedEnvironmentResult(ctx, environment.ID, result); err != nil {
		s.rollbackFailedEnvironmentStart(ctx, competitionID, environment)
		return nil, err
	}
	environment.Status = enum.ChallengeEnvStatusRunning
	environment.ChainRPCURL = result.ChainRPCURL
	now := time.Now()
	environment.StartedAt = &now
	environment.ContainerStatus = mustJSON(buildChallengeEnvironmentRuntimeState(result))

	return buildStartChallengeEnvironmentResp(environment), nil
}

// retryErroredEnvironmentStart 对已处于异常状态的题目环境执行原地重试，避免重复创建新记录。
func (s *environmentService) retryErroredEnvironmentStart(ctx context.Context, environment *entity.ChallengeEnvironment, challenge *entity.Challenge) (*dto.StartChallengeEnvironmentResp, error) {
	if environment == nil || challenge == nil {
		return nil, errcode.ErrEnvironmentNotFound
	}
	if err := s.environmentRepo.ResetToCreating(ctx, environment.ID); err != nil {
		return nil, err
	}
	if s.provisioner != nil && environment.Namespace != "" {
		_ = s.provisioner.DeleteNamespace(ctx, environment.Namespace)
	}
	if s.runtimeProvisioner == nil {
		s.markEnvironmentError(ctx, environment.ID)
		return nil, errcode.ErrInternal.WithMessage("CTF题目环境运行时未配置")
	}
	spec, err := s.buildChallengeEnvironmentSpec(ctx, challenge, environment.CompetitionID, environment.TeamID, environment.Namespace)
	if err != nil {
		s.markEnvironmentError(ctx, environment.ID)
		return nil, err
	}
	result, err := s.runtimeProvisioner.ProvisionChallengeEnvironment(ctx, spec)
	if err != nil {
		s.markEnvironmentError(ctx, environment.ID)
		return nil, errcode.ErrInternal.WithMessage("题目环境重试失败：" + err.Error())
	}
	if err := s.applyProvisionedEnvironmentResult(ctx, environment.ID, result); err != nil {
		s.markEnvironmentError(ctx, environment.ID)
		return nil, err
	}
	environment.Status = enum.ChallengeEnvStatusRunning
	environment.ChainRPCURL = result.ChainRPCURL
	now := time.Now()
	environment.StartedAt = &now
	environment.ContainerStatus = mustJSON(buildChallengeEnvironmentRuntimeState(result))
	return buildStartChallengeEnvironmentResp(environment), nil
}

// Get 获取题目环境详情。
func (s *environmentService) Get(ctx context.Context, sc *svcctx.ServiceContext, id int64) (*dto.ChallengeEnvironmentResp, error) {
	environment, err := s.environmentRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errcode.ErrEnvironmentNotFound
		}
		return nil, err
	}
	if err := s.ensureEnvironmentReadable(ctx, sc, environment); err != nil {
		return nil, err
	}
	return buildChallengeEnvironmentResp(environment), nil
}

// Reset 重置题目环境。
func (s *environmentService) Reset(ctx context.Context, sc *svcctx.ServiceContext, id int64) (*dto.ResetChallengeEnvironmentResp, error) {
	environment, err := s.environmentRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errcode.ErrEnvironmentNotFound
		}
		return nil, err
	}
	if err := s.ensureEnvironmentWritable(ctx, sc, environment); err != nil {
		return nil, err
	}
	if environment.Status == enum.ChallengeEnvStatusDestroyed {
		return nil, errcode.ErrEnvironmentNotFound
	}
	challenge, err := getChallenge(ctx, s.challengeRepo, environment.ChallengeID)
	if err != nil {
		return nil, err
	}
	if err := s.environmentRepo.ResetToCreating(ctx, id); err != nil {
		return nil, err
	}
	if s.provisioner != nil {
		_ = s.provisioner.DeleteNamespace(ctx, environment.Namespace)
	}
	if s.runtimeProvisioner == nil {
		s.markEnvironmentError(ctx, id)
		return nil, errcode.ErrInternal.WithMessage("CTF题目环境运行时未配置")
	}
	spec, err := s.buildChallengeEnvironmentSpec(ctx, challenge, environment.CompetitionID, environment.TeamID, environment.Namespace)
	if err != nil {
		s.markEnvironmentError(ctx, id)
		return nil, err
	}
	result, err := s.runtimeProvisioner.ProvisionChallengeEnvironment(ctx, spec)
	if err != nil {
		s.markEnvironmentError(ctx, id)
		return nil, errcode.ErrInternal.WithMessage("题目环境重置失败：" + err.Error())
	}
	if err := s.applyProvisionedEnvironmentResult(ctx, id, result); err != nil {
		s.markEnvironmentError(ctx, id)
		return nil, err
	}
	return &dto.ResetChallengeEnvironmentResp{
		EnvironmentID: int64String(id),
		Status:        enum.ChallengeEnvStatusRunning,
		StatusText:    enum.GetChallengeEnvStatusText(enum.ChallengeEnvStatusRunning),
	}, nil
}

// Destroy 销毁题目环境。
func (s *environmentService) Destroy(ctx context.Context, sc *svcctx.ServiceContext, id int64) error {
	environment, err := s.environmentRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errcode.ErrEnvironmentNotFound
		}
		return err
	}
	if err := s.ensureEnvironmentWritable(ctx, sc, environment); err != nil {
		return err
	}
	return s.destroyEnvironment(ctx, environment)
}

// ForceDestroy 强制回收题目环境。
func (s *environmentService) ForceDestroy(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.ForceDestroyChallengeEnvironmentReq) (*dto.ForceDestroyChallengeEnvironmentResp, error) {
	if !sc.IsSuperAdmin() {
		return nil, errcode.ErrForbidden
	}
	environment, err := s.environmentRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errcode.ErrEnvironmentNotFound
		}
		return nil, err
	}
	_ = req
	if err := s.destroyEnvironment(ctx, environment); err != nil {
		return nil, err
	}
	audit.RecordFromContext(s.db, sc.UserID, sc.ClientIP, "force_destroy_ctf_environment", "challenge_environment", id, map[string]interface{}{
		"competition_id": environment.CompetitionID,
		"challenge_id":   environment.ChallengeID,
		"team_id":        environment.TeamID,
		"namespace":      environment.Namespace,
		"reason":         req.Reason,
	})
	return &dto.ForceDestroyChallengeEnvironmentResp{
		EnvironmentID: int64String(id),
		Status:        enum.ChallengeEnvStatusDestroyed,
		StatusText:    enum.GetChallengeEnvStatusText(enum.ChallengeEnvStatusDestroyed),
	}, nil
}

// ListMyEnvironments 查询我的题目环境列表。
func (s *environmentService) ListMyEnvironments(ctx context.Context, sc *svcctx.ServiceContext, competitionID int64) (*dto.MyChallengeEnvironmentListResp, error) {
	_, team, err := s.getCurrentMemberTeam(ctx, competitionID, sc.UserID)
	if err != nil {
		return nil, err
	}
	environments, err := s.environmentRepo.ListByTeamID(ctx, competitionID, team.ID)
	if err != nil {
		return nil, err
	}
	challenges, err := s.loadEnvironmentChallenges(ctx, environments)
	if err != nil {
		return nil, err
	}
	items := make([]dto.MyChallengeEnvironmentItem, 0, len(environments))
	for _, environment := range environments {
		title := ""
		if challenge := challenges[environment.ChallengeID]; challenge != nil {
			title = challenge.Title
		}
		items = append(items, dto.MyChallengeEnvironmentItem{
			ID:             int64String(environment.ID),
			ChallengeID:    int64String(environment.ChallengeID),
			ChallengeTitle: title,
			Namespace:      environment.Namespace,
			Status:         environment.Status,
			StatusText:     enum.GetChallengeEnvStatusText(environment.Status),
			ChainRPCURL:    environment.ChainRPCURL,
			CreatedAt:      timeString(environment.CreatedAt),
		})
	}
	return &dto.MyChallengeEnvironmentListResp{List: items}, nil
}

// ListCompetitionEnvironments 查询竞赛环境资源列表。
func (s *environmentService) ListCompetitionEnvironments(ctx context.Context, sc *svcctx.ServiceContext, competitionID int64, req *dto.CompetitionEnvironmentListReq) (*dto.CompetitionEnvironmentListResp, error) {
	competition, err := getCompetition(ctx, s.competitionRepo, competitionID)
	if err != nil {
		return nil, err
	}
	if err := s.ensureCompetitionManageable(sc, competition); err != nil {
		return nil, err
	}
	params := &ctfrepo.ChallengeEnvironmentListParams{
		CompetitionID: competitionID,
		Status:        req.Status,
		Page:          req.Page,
		PageSize:      req.PageSize,
	}
	if req.ChallengeID != "" {
		challengeID, parseErr := snowflake.ParseString(req.ChallengeID)
		if parseErr != nil {
			return nil, errcode.ErrInvalidID
		}
		params.ChallengeID = challengeID
	}
	if req.TeamID != "" {
		teamID, parseErr := snowflake.ParseString(req.TeamID)
		if parseErr != nil {
			return nil, errcode.ErrInvalidID
		}
		params.TeamID = teamID
	}
	environments, total, err := s.environmentRepo.List(ctx, params)
	if err != nil {
		return nil, err
	}
	challenges, err := s.loadEnvironmentChallenges(ctx, environments)
	if err != nil {
		return nil, err
	}
	teams, err := s.loadEnvironmentTeams(ctx, environments)
	if err != nil {
		return nil, err
	}
	items := make([]dto.CompetitionEnvironmentListItem, 0, len(environments))
	for _, environment := range environments {
		challengeTitle := ""
		if challenge := challenges[environment.ChallengeID]; challenge != nil {
			challengeTitle = challenge.Title
		}
		teamName := ""
		if team := teams[environment.TeamID]; team != nil {
			teamName = team.Name
		}
		items = append(items, dto.CompetitionEnvironmentListItem{
			ID:             int64String(environment.ID),
			CompetitionID:  int64String(environment.CompetitionID),
			ChallengeID:    int64String(environment.ChallengeID),
			ChallengeTitle: challengeTitle,
			TeamID:         int64String(environment.TeamID),
			TeamName:       teamName,
			Namespace:      environment.Namespace,
			Status:         environment.Status,
			StatusText:     enum.GetChallengeEnvStatusText(environment.Status),
			ChainRPCURL:    environment.ChainRPCURL,
			StartedAt:      optionalTimeString(environment.StartedAt),
			CreatedAt:      timeString(environment.CreatedAt),
		})
	}
	return &dto.CompetitionEnvironmentListResp{
		List:       items,
		Pagination: paginationResp(req.Page, req.PageSize, total),
	}, nil
}

// ensureCompetitionManageable 校验当前上下文是否可管理竞赛环境与资源列表。
// 环境监控与资源列表遵循文档中的“竞赛创建者/超级管理员”权限，不放宽到同校管理员。
func (s *environmentService) ensureCompetitionManageable(sc *svcctx.ServiceContext, competition *entity.Competition) error {
	if sc == nil || competition == nil {
		return errcode.ErrForbidden
	}
	if sc.IsSuperAdmin() {
		return nil
	}
	if competition.CreatedBy == sc.UserID {
		return nil
	}
	return errcode.ErrForbidden
}

// ensureQuotaAvailable 校验竞赛资源配额是否还有可用命名空间。
func (s *environmentService) ensureQuotaAvailable(ctx context.Context, competitionID int64) error {
	quota, err := s.quotaRepo.GetByCompetitionID(ctx, competitionID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	if quota.MaxNamespaces != nil && quota.CurrentNamespaces >= *quota.MaxNamespaces {
		return errcode.ErrCompetitionQuotaExceeded
	}
	return nil
}

// ensureEnvironmentReadable 校验当前上下文是否可读取题目环境。
func (s *environmentService) ensureEnvironmentReadable(ctx context.Context, sc *svcctx.ServiceContext, environment *entity.ChallengeEnvironment) error {
	if sc.IsSuperAdmin() {
		return nil
	}
	team, err := s.teamRepo.GetByID(ctx, environment.TeamID)
	if err != nil {
		return err
	}
	if team.CaptainID == sc.UserID {
		return nil
	}
	isMember, err := s.teamMemberRepo.IsTeamMember(ctx, environment.TeamID, sc.UserID)
	if err != nil {
		return err
	}
	if isMember {
		return nil
	}
	competition, err := getCompetition(ctx, s.competitionRepo, environment.CompetitionID)
	if err != nil {
		return err
	}
	return s.ensureCompetitionManageable(sc, competition)
}

// ensureEnvironmentWritable 校验当前上下文是否可变更题目环境。
func (s *environmentService) ensureEnvironmentWritable(ctx context.Context, sc *svcctx.ServiceContext, environment *entity.ChallengeEnvironment) error {
	if sc.IsSuperAdmin() {
		return nil
	}
	isMember, err := s.teamMemberRepo.IsTeamMember(ctx, environment.TeamID, sc.UserID)
	if err != nil {
		return err
	}
	if !isMember {
		return errcode.ErrForbidden
	}
	return nil
}

// getCurrentMemberTeam 获取当前学生在竞赛中的团队成员关系和团队实体。
func (s *environmentService) getCurrentMemberTeam(ctx context.Context, competitionID, studentID int64) (*entity.TeamMember, *entity.Team, error) {
	return getCompetitionMemberTeam(ctx, s.teamMemberRepo, s.teamRepo, competitionID, studentID)
}

// destroyEnvironment 执行环境销毁和配额回收。
func (s *environmentService) destroyEnvironment(ctx context.Context, environment *entity.ChallengeEnvironment) error {
	if environment.Status == enum.ChallengeEnvStatusDestroyed {
		return nil
	}
	if s.provisioner != nil {
		_ = s.provisioner.DeleteNamespace(ctx, environment.Namespace)
	}
	return database.TransactionWithDB(ctx, s.db, func(tx *gorm.DB) error {
		txEnvRepo := ctfrepo.NewChallengeEnvironmentRepository(tx)
		txQuotaRepo := ctfrepo.NewCtfResourceQuotaRepository(tx)
		if err := txEnvRepo.MarkDestroyed(ctx, environment.ID); err != nil {
			return err
		}
		quota, quotaErr := txQuotaRepo.GetByCompetitionID(ctx, environment.CompetitionID)
		if quotaErr == nil && quota != nil && quota.CurrentNamespaces > 0 {
			if err := txQuotaRepo.IncrementNamespaces(ctx, environment.CompetitionID, -1); err != nil {
				return err
			}
		}
		return nil
	})
}

// loadEnvironmentChallenges 批量加载环境关联题目。
func (s *environmentService) loadEnvironmentChallenges(ctx context.Context, environments []*entity.ChallengeEnvironment) (map[int64]*entity.Challenge, error) {
	challengeIDs := make([]int64, 0, len(environments))
	for _, environment := range environments {
		challengeIDs = append(challengeIDs, environment.ChallengeID)
	}
	challenges, err := s.challengeRepo.GetByIDs(ctx, challengeIDs)
	if err != nil {
		return nil, err
	}
	result := make(map[int64]*entity.Challenge, len(challenges))
	for _, challenge := range challenges {
		result[challenge.ID] = challenge
	}
	return result, nil
}

// loadEnvironmentTeams 批量加载环境关联团队。
func (s *environmentService) loadEnvironmentTeams(ctx context.Context, environments []*entity.ChallengeEnvironment) (map[int64]*entity.Team, error) {
	teamIDs := make([]int64, 0, len(environments))
	for _, environment := range environments {
		teamIDs = append(teamIDs, environment.TeamID)
	}
	teams, err := s.teamRepo.ListByIDs(ctx, teamIDs)
	if err != nil {
		return nil, err
	}
	result := make(map[int64]*entity.Team, len(teams))
	for _, team := range teams {
		result[team.ID] = team
	}
	return result, nil
}

// buildChallengeEnvironmentNamespace 生成题目环境命名空间名称。
func buildChallengeEnvironmentNamespace(competitionID, teamID, challengeID int64) string {
	return fmt.Sprintf("ctf-%d-%d-%d", competitionID, teamID, challengeID)
}

// rollbackFailedEnvironmentStart 回滚首次启动失败的命名空间配额占用，并把环境标记为异常。
func (s *environmentService) rollbackFailedEnvironmentStart(ctx context.Context, competitionID int64, environment *entity.ChallengeEnvironment) {
	if environment == nil {
		return
	}
	if s.provisioner != nil && environment.Namespace != "" {
		_ = s.provisioner.DeleteNamespace(ctx, environment.Namespace)
	}
	_ = database.TransactionWithDB(ctx, s.db, func(tx *gorm.DB) error {
		txEnvRepo := ctfrepo.NewChallengeEnvironmentRepository(tx)
		txQuotaRepo := ctfrepo.NewCtfResourceQuotaRepository(tx)
		if err := txEnvRepo.UpdateFields(ctx, environment.ID, map[string]interface{}{
			"status":           enum.ChallengeEnvStatusError,
			"chain_rpc_url":    nil,
			"container_status": nil,
			"started_at":       nil,
			"destroyed_at":     nil,
		}); err != nil {
			return err
		}
		quota, quotaErr := txQuotaRepo.GetByCompetitionID(ctx, competitionID)
		if quotaErr == nil && quota != nil && quota.CurrentNamespaces > 0 {
			if err := txQuotaRepo.IncrementNamespaces(ctx, competitionID, -1); err != nil {
				return err
			}
		}
		return nil
	})
}

// buildChallengeEnvironmentSpec 根据题目配置构建运行时编排输入。
func (s *environmentService) buildChallengeEnvironmentSpec(ctx context.Context, challenge *entity.Challenge, competitionID, teamID int64, namespace string) (*ChallengeEnvironmentSpec, error) {
	contracts, err := s.contractRepo.ListByChallengeID(ctx, challenge.ID)
	if err != nil {
		return nil, err
	}
	return buildChallengeEnvironmentSpec(challenge, competitionID, teamID, namespace, contracts), nil
}

// applyProvisionedEnvironmentResult 把环境编排结果写回题目环境记录。
func (s *environmentService) applyProvisionedEnvironmentResult(ctx context.Context, environmentID int64, result *ChallengeEnvironmentResult) error {
	if result == nil {
		return errcode.ErrInternal.WithMessage("题目环境编排结果为空")
	}
	startedAt := time.Now()
	fields := map[string]interface{}{
		"status":           enum.ChallengeEnvStatusRunning,
		"chain_rpc_url":    result.ChainRPCURL,
		"container_status": mustJSON(buildChallengeEnvironmentRuntimeState(result)),
		"started_at":       startedAt,
		"updated_at":       startedAt,
	}
	return s.environmentRepo.UpdateFields(ctx, environmentID, fields)
}

// markEnvironmentError 把题目环境更新为异常状态，便于选手重试。
func (s *environmentService) markEnvironmentError(ctx context.Context, environmentID int64) {
	_ = s.environmentRepo.UpdateFields(ctx, environmentID, map[string]interface{}{
		"status":     enum.ChallengeEnvStatusError,
		"updated_at": time.Now(),
		"started_at": nil,
	})
}

// buildChallengeEnvironmentResp 构建题目环境详情响应。
func buildChallengeEnvironmentResp(environment *entity.ChallengeEnvironment) *dto.ChallengeEnvironmentResp {
	runtimeState := decodeChallengeEnvironmentRuntimeState(environment)
	return &dto.ChallengeEnvironmentResp{
		ID:              int64String(environment.ID),
		CompetitionID:   int64String(environment.CompetitionID),
		ChallengeID:     int64String(environment.ChallengeID),
		TeamID:          int64String(environment.TeamID),
		Namespace:       environment.Namespace,
		Status:          environment.Status,
		StatusText:      enum.GetChallengeEnvStatusText(environment.Status),
		ChainRPCURL:     environment.ChainRPCURL,
		ContainerStatus: runtimeState.Containers,
		StartedAt:       optionalTimeString(environment.StartedAt),
		CreatedAt:       timeString(environment.CreatedAt),
	}
}

// buildStartChallengeEnvironmentResp 构建启动题目环境接口响应，供首次创建和幂等返回共用。
func buildStartChallengeEnvironmentResp(environment *entity.ChallengeEnvironment) *dto.StartChallengeEnvironmentResp {
	if environment == nil {
		return nil
	}
	return &dto.StartChallengeEnvironmentResp{
		EnvironmentID: int64String(environment.ID),
		CompetitionID: int64String(environment.CompetitionID),
		ChallengeID:   int64String(environment.ChallengeID),
		TeamID:        int64String(environment.TeamID),
		Namespace:     environment.Namespace,
		Status:        environment.Status,
		StatusText:    enum.GetChallengeEnvStatusText(environment.Status),
		ChainRPCURL:   environment.ChainRPCURL,
		CreatedAt:     timeString(environment.CreatedAt),
	}
}
