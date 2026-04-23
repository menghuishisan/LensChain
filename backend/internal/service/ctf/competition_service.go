// competition_service.go
// 模块05 — CTF竞赛：竞赛主流程业务逻辑。
// 负责竞赛创建发布、题目配置、排行榜、公告、资源配额、监控统计和平台概览等能力。

package ctf

import (
	"context"
	"strconv"
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

// competitionService 竞赛主流程服务实现。
type competitionService struct {
	db                *gorm.DB
	competitionRepo   ctfrepo.CompetitionRepository
	challengeRepo     ctfrepo.ChallengeRepository
	contractRepo      ctfrepo.ChallengeContractRepository
	assertionRepo     ctfrepo.ChallengeAssertionRepository
	verificationRepo  ctfrepo.ChallengeVerificationRepository
	compChallengeRepo ctfrepo.CompetitionChallengeRepository
	teamRepo          ctfrepo.TeamRepository
	teamMemberRepo    ctfrepo.TeamMemberRepository
	registrationRepo  ctfrepo.CompetitionRegistrationRepository
	submissionRepo    ctfrepo.SubmissionRepository
	leaderboardRepo   ctfrepo.LeaderboardSnapshotRepository
	announcementRepo  ctfrepo.CtfAnnouncementRepository
	quotaRepo         ctfrepo.CtfResourceQuotaRepository
	environmentRepo   ctfrepo.ChallengeEnvironmentRepository
	adGroupRepo       ctfrepo.AdGroupRepository
	adRoundRepo       ctfrepo.AdRoundRepository
	adAttackRepo      ctfrepo.AdAttackRepository
	adDefenseRepo     ctfrepo.AdDefenseRepository
	adLedgerRepo      ctfrepo.AdTokenLedgerRepository
	environmentOps    competitionEnvironmentOperator
	battleRuntimeOps  competitionBattleRuntimeOperator
	realtimePublisher CTFRealtimePublisher
	userQuerier       UserSummaryQuerier
	schoolQuerier     SchoolNameQuerier
	eventDispatcher   NotificationEventDispatcher
	audienceResolver  CompetitionAudienceResolver
}

// competitionEnvironmentOperator 声明竞赛服务在终止/归档场景复用的题目环境最小能力。
type competitionEnvironmentOperator interface {
	destroyEnvironment(ctx context.Context, environment *entity.ChallengeEnvironment) error
}

// competitionBattleRuntimeOperator 声明竞赛服务在终止/归档场景复用的攻防赛运行时清理能力。
type competitionBattleRuntimeOperator interface {
	cleanupGroupRuntimesByCompetition(ctx context.Context, competitionID int64) error
}

var _ CompetitionService = (*competitionService)(nil)

// NewCompetitionService 创建竞赛主流程服务实例。
func NewCompetitionService(
	db *gorm.DB,
	competitionRepo ctfrepo.CompetitionRepository,
	challengeRepo ctfrepo.ChallengeRepository,
	contractRepo ctfrepo.ChallengeContractRepository,
	assertionRepo ctfrepo.ChallengeAssertionRepository,
	verificationRepo ctfrepo.ChallengeVerificationRepository,
	compChallengeRepo ctfrepo.CompetitionChallengeRepository,
	teamRepo ctfrepo.TeamRepository,
	teamMemberRepo ctfrepo.TeamMemberRepository,
	registrationRepo ctfrepo.CompetitionRegistrationRepository,
	submissionRepo ctfrepo.SubmissionRepository,
	leaderboardRepo ctfrepo.LeaderboardSnapshotRepository,
	announcementRepo ctfrepo.CtfAnnouncementRepository,
	quotaRepo ctfrepo.CtfResourceQuotaRepository,
	environmentRepo ctfrepo.ChallengeEnvironmentRepository,
	adGroupRepo ctfrepo.AdGroupRepository,
	adRoundRepo ctfrepo.AdRoundRepository,
	adAttackRepo ctfrepo.AdAttackRepository,
	adDefenseRepo ctfrepo.AdDefenseRepository,
	adLedgerRepo ctfrepo.AdTokenLedgerRepository,
	environmentOps competitionEnvironmentOperator,
	battleRuntimeOps competitionBattleRuntimeOperator,
	realtimePublisher CTFRealtimePublisher,
	userQuerier UserSummaryQuerier,
	schoolQuerier SchoolNameQuerier,
	eventDispatcher NotificationEventDispatcher,
	audienceResolver CompetitionAudienceResolver,
) *competitionService {
	return &competitionService{
		db:                db,
		competitionRepo:   competitionRepo,
		challengeRepo:     challengeRepo,
		contractRepo:      contractRepo,
		assertionRepo:     assertionRepo,
		verificationRepo:  verificationRepo,
		compChallengeRepo: compChallengeRepo,
		teamRepo:          teamRepo,
		teamMemberRepo:    teamMemberRepo,
		registrationRepo:  registrationRepo,
		submissionRepo:    submissionRepo,
		leaderboardRepo:   leaderboardRepo,
		announcementRepo:  announcementRepo,
		quotaRepo:         quotaRepo,
		environmentRepo:   environmentRepo,
		adGroupRepo:       adGroupRepo,
		adRoundRepo:       adRoundRepo,
		adAttackRepo:      adAttackRepo,
		adDefenseRepo:     adDefenseRepo,
		adLedgerRepo:      adLedgerRepo,
		environmentOps:    environmentOps,
		battleRuntimeOps:  battleRuntimeOps,
		realtimePublisher: realtimePublisher,
		userQuerier:       userQuerier,
		schoolQuerier:     schoolQuerier,
		eventDispatcher:   eventDispatcher,
		audienceResolver:  audienceResolver,
	}
}

// Create 创建竞赛。
func (s *competitionService) Create(ctx context.Context, sc *svcctx.ServiceContext, req *dto.CreateCompetitionReq) (*dto.CompetitionCreateResp, error) {
	if req.Title == "" {
		return nil, errcode.ErrInvalidParams.WithMessage("竞赛名称不能为空")
	}
	if !enum.IsValidCompetitionType(req.CompetitionType) {
		return nil, errcode.ErrCompetitionTypeInvalid
	}
	if !enum.IsValidCompetitionScope(req.Scope) {
		return nil, errcode.ErrCompetitionScopeInvalid
	}
	if sc.IsSchoolAdmin() && req.Scope != enum.CompetitionScopeSchool {
		return nil, errcode.ErrForbidden.WithMessage("学校管理员只能创建校级竞赛")
	}
	if err := validateCompetitionTypeConfig(req.CompetitionType, req.JeopardyConfig, req.AdConfig); err != nil {
		return nil, err
	}

	schoolID, err := parseOptionalSnowflake(req.SchoolID)
	if err != nil {
		return nil, err
	}
	if req.Scope == enum.CompetitionScopeSchool {
		// 校级竞赛必须绑定学校；学校管理员创建时进一步强制绑定到本人所属学校，
		// 避免通过构造请求把校级竞赛挂到其他租户。
		if sc.IsSchoolAdmin() {
			if sc.SchoolID == 0 {
				return nil, errcode.ErrForbidden.WithMessage("学校管理员缺少学校归属")
			}
			if schoolID != nil && *schoolID != sc.SchoolID {
				return nil, errcode.ErrForbidden.WithMessage("学校管理员只能为本校创建竞赛")
			}
			schoolID = int64Ptr(sc.SchoolID)
		}
		if schoolID == nil {
			return nil, errcode.ErrInvalidParams.WithMessage("校级竞赛必须指定学校")
		}
	}
	registrationStartAt, err := parseOptionalTime(req.RegistrationStartAt)
	if err != nil {
		return nil, err
	}
	registrationEndAt, err := parseOptionalTime(req.RegistrationEndAt)
	if err != nil {
		return nil, err
	}
	startAt, err := parseOptionalTime(req.StartAt)
	if err != nil {
		return nil, err
	}
	endAt, err := parseOptionalTime(req.EndAt)
	if err != nil {
		return nil, err
	}
	freezeAt, err := parseOptionalTime(req.FreezeAt)
	if err != nil {
		return nil, err
	}
	if err := validateCompetitionWindow(registrationStartAt, registrationEndAt, startAt, endAt, freezeAt); err != nil {
		return nil, err
	}

	competition := &entity.Competition{
		ID:                  snowflake.Generate(),
		Title:               req.Title,
		Description:         req.Description,
		BannerURL:           req.BannerURL,
		CompetitionType:     req.CompetitionType,
		Scope:               req.Scope,
		CreatedBy:           sc.UserID,
		TeamMode:            req.TeamMode,
		MaxTeamSize:         req.MaxTeamSize,
		MinTeamSize:         req.MinTeamSize,
		MaxTeams:            req.MaxTeams,
		Status:              enum.CompetitionStatusDraft,
		RegistrationStartAt: registrationStartAt,
		RegistrationEndAt:   registrationEndAt,
		StartAt:             startAt,
		EndAt:               endAt,
		FreezeAt:            freezeAt,
		Rules:               req.Rules,
	}
	if schoolID != nil {
		competition.SchoolID = schoolID
	}
	if req.JeopardyConfig != nil {
		competition.JeopardyConfig = mustJSON(req.JeopardyConfig)
	}
	if req.AdConfig != nil {
		competition.AdConfig = mustJSON(req.AdConfig)
	}
	if err := s.competitionRepo.Create(ctx, competition); err != nil {
		return nil, err
	}
	return &dto.CompetitionCreateResp{
		ID:                  int64String(competition.ID),
		Title:               competition.Title,
		CompetitionType:     competition.CompetitionType,
		CompetitionTypeText: enum.GetCompetitionTypeText(competition.CompetitionType),
		Scope:               competition.Scope,
		ScopeText:           enum.GetCompetitionScopeText(competition.Scope),
		TeamMode:            competition.TeamMode,
		TeamModeText:        enum.GetTeamModeText(competition.TeamMode),
		Status:              competition.Status,
		StatusText:          enum.GetCompetitionStatusText(competition.Status),
		CreatedAt:           timeString(competition.CreatedAt),
	}, nil
}

// List 查询竞赛列表。
func (s *competitionService) List(ctx context.Context, sc *svcctx.ServiceContext, req *dto.CompetitionListReq) (*dto.CompetitionListResp, error) {
	params := &ctfrepo.CompetitionListParams{
		CompetitionType: req.CompetitionType,
		Scope:           req.Scope,
		Status:          req.Status,
		Keyword:         req.Keyword,
		Page:            req.Page,
		PageSize:        req.PageSize,
	}
	if !sc.IsSuperAdmin() {
		params.SchoolID = sc.SchoolID
	}
	competitions, total, err := s.competitionRepo.List(ctx, params)
	if err != nil {
		return nil, err
	}

	items := make([]dto.CompetitionListItem, 0, len(competitions))
	for _, competition := range competitions {
		registeredTeams, _ := s.registrationRepo.CountActiveByCompetitionID(ctx, competition.ID)
		challengeCount, _ := s.compChallengeRepo.CountByCompetitionID(ctx, competition.ID)
		items = append(items, buildCompetitionListItem(ctx, s.userQuerier, competition, registeredTeams, challengeCount))
	}
	return &dto.CompetitionListResp{
		List:       items,
		Pagination: paginationResp(req.Page, req.PageSize, total),
	}, nil
}

// Get 获取竞赛详情。
func (s *competitionService) Get(ctx context.Context, sc *svcctx.ServiceContext, id int64) (*dto.CompetitionDetailResp, error) {
	competition, err := getCompetition(ctx, s.competitionRepo, id)
	if err != nil {
		return nil, err
	}
	if err := s.ensureCompetitionReadable(sc, competition); err != nil {
		return nil, err
	}
	registeredTeams, _ := s.registrationRepo.CountActiveByCompetitionID(ctx, competition.ID)
	challengeCount, _ := s.compChallengeRepo.CountByCompetitionID(ctx, competition.ID)
	return buildCompetitionDetail(ctx, s.userQuerier, competition, registeredTeams, challengeCount)
}

// Update 更新竞赛。
func (s *competitionService) Update(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.UpdateCompetitionReq) error {
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

	// 编辑竞赛时需要基于“更新后的最终状态”执行一次完整窗口与学校归属校验，
	// 防止通过局部字段更新写入非法时间配置或把校级竞赛切换到错误租户。
	effectiveScope := competition.Scope
	if req.Scope != nil {
		effectiveScope = *req.Scope
	}
	effectiveSchoolID := competition.SchoolID
	if req.SchoolID != nil {
		parsedSchoolID, parseErr := parseOptionalSnowflake(req.SchoolID)
		if parseErr != nil {
			return parseErr
		}
		effectiveSchoolID = parsedSchoolID
	}
	if sc.IsSchoolAdmin() {
		if effectiveScope != enum.CompetitionScopeSchool {
			return errcode.ErrForbidden.WithMessage("学校管理员只能创建校级竞赛")
		}
		if sc.SchoolID == 0 {
			return errcode.ErrForbidden.WithMessage("学校管理员缺少学校归属")
		}
		if effectiveSchoolID != nil && *effectiveSchoolID != sc.SchoolID {
			return errcode.ErrForbidden.WithMessage("学校管理员只能为本校创建竞赛")
		}
		effectiveSchoolID = int64Ptr(sc.SchoolID)
	}
	if effectiveScope == enum.CompetitionScopeSchool && effectiveSchoolID == nil {
		return errcode.ErrInvalidParams.WithMessage("校级竞赛必须指定学校")
	}

	registrationStartAt := competition.RegistrationStartAt
	if req.RegistrationStartAt != nil {
		value, parseErr := parseOptionalTime(req.RegistrationStartAt)
		if parseErr != nil {
			return parseErr
		}
		registrationStartAt = value
	}
	registrationEndAt := competition.RegistrationEndAt
	if req.RegistrationEndAt != nil {
		value, parseErr := parseOptionalTime(req.RegistrationEndAt)
		if parseErr != nil {
			return parseErr
		}
		registrationEndAt = value
	}
	startAt := competition.StartAt
	if req.StartAt != nil {
		value, parseErr := parseOptionalTime(req.StartAt)
		if parseErr != nil {
			return parseErr
		}
		startAt = value
	}
	endAt := competition.EndAt
	if req.EndAt != nil {
		value, parseErr := parseOptionalTime(req.EndAt)
		if parseErr != nil {
			return parseErr
		}
		endAt = value
	}
	freezeAt := competition.FreezeAt
	if req.FreezeAt != nil {
		value, parseErr := parseOptionalTime(req.FreezeAt)
		if parseErr != nil {
			return parseErr
		}
		freezeAt = value
	}
	if err := validateCompetitionWindow(registrationStartAt, registrationEndAt, startAt, endAt, freezeAt); err != nil {
		return err
	}

	fields := map[string]interface{}{}
	if req.Title != nil {
		fields["title"] = *req.Title
	}
	if req.Description != nil {
		fields["description"] = *req.Description
	}
	if req.BannerURL != nil {
		fields["banner_url"] = *req.BannerURL
	}
	if req.Scope != nil {
		fields["scope"] = effectiveScope
	}
	if req.TeamMode != nil {
		fields["team_mode"] = *req.TeamMode
	}
	if req.MaxTeamSize != nil {
		fields["max_team_size"] = *req.MaxTeamSize
	}
	if req.MinTeamSize != nil {
		fields["min_team_size"] = *req.MinTeamSize
	}
	if req.MaxTeams != nil {
		fields["max_teams"] = *req.MaxTeams
	}
	if req.Rules != nil {
		fields["rules"] = *req.Rules
	}
	if req.JeopardyConfig != nil {
		fields["jeopardy_config"] = mustJSON(req.JeopardyConfig)
	}
	if req.AdConfig != nil {
		fields["ad_config"] = mustJSON(req.AdConfig)
	}
	if err := validateCompetitionUpdateConfig(competition, req); err != nil {
		return err
	}
	if req.SchoolID != nil {
		fields["school_id"] = effectiveSchoolID
	} else if sc.IsSchoolAdmin() {
		fields["school_id"] = effectiveSchoolID
	}
	if req.RegistrationStartAt != nil {
		fields["registration_start_at"] = registrationStartAt
	}
	if req.RegistrationEndAt != nil {
		fields["registration_end_at"] = registrationEndAt
	}
	if req.StartAt != nil {
		fields["start_at"] = startAt
	}
	if req.EndAt != nil {
		fields["end_at"] = endAt
	}
	if req.FreezeAt != nil {
		fields["freeze_at"] = freezeAt
	}
	if len(fields) == 0 {
		return nil
	}
	return s.competitionRepo.UpdateFields(ctx, competition.ID, fields)
}

// Delete 删除竞赛。
func (s *competitionService) Delete(ctx context.Context, sc *svcctx.ServiceContext, id int64) error {
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
	return s.competitionRepo.SoftDelete(ctx, id)
}

// Publish 发布竞赛。
func (s *competitionService) Publish(ctx context.Context, sc *svcctx.ServiceContext, id int64) (*dto.CompetitionStatusResp, error) {
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
	challengeCount, err := s.compChallengeRepo.CountByCompetitionID(ctx, competition.ID)
	if err != nil {
		return nil, err
	}
	if challengeCount == 0 {
		return nil, errcode.ErrCompetitionChallengeEmpty
	}
	if competition.StartAt == nil || competition.EndAt == nil {
		return nil, errcode.ErrCompetitionTimeInvalid
	}

	targetStatus := int16(enum.CompetitionStatusRegistration)
	if err := s.competitionRepo.UpdateStatus(ctx, competition.ID, targetStatus); err != nil {
		return nil, err
	}
	writeCompetitionStatusCache(ctx, competition.ID, targetStatus)
	writeCompetitionFrozenCache(ctx, competition.ID, false)
	if err := s.dispatchCompetitionPublished(ctx, competition); err != nil {
		return nil, err
	}
	return &dto.CompetitionStatusResp{
		ID:         int64String(competition.ID),
		Status:     targetStatus,
		StatusText: enum.GetCompetitionStatusText(targetStatus),
	}, nil
}

// dispatchCompetitionPublished 在竞赛发布后向符合条件的学生发送站内信事件。
func (s *competitionService) dispatchCompetitionPublished(ctx context.Context, competition *entity.Competition) error {
	if s.eventDispatcher == nil || s.audienceResolver == nil || competition == nil {
		return nil
	}
	receiverIDs, err := s.audienceResolver.ListCompetitionPublishStudentIDs(ctx, competition.Scope, competition.SchoolID)
	if err != nil {
		return err
	}
	if len(receiverIDs) == 0 {
		return nil
	}
	targets := make([]string, 0, len(receiverIDs))
	for _, receiverID := range receiverIDs {
		if receiverID == 0 {
			continue
		}
		targets = append(targets, strconv.FormatInt(receiverID, 10))
	}
	if len(targets) == 0 {
		return nil
	}
	params := map[string]interface{}{
		"competition_name": competition.Title,
	}
	if competition.RegistrationEndAt != nil {
		params["deadline"] = competition.RegistrationEndAt.UTC().Format("2006-01-02 15:04")
	}
	return s.eventDispatcher.DispatchEvent(ctx, &dto.InternalSendNotificationEventReq{
		EventType:    "competition.published",
		ReceiverIDs:  targets,
		Params:       params,
		SourceModule: "module_05",
		SourceType:   "competition",
		SourceID:     strconv.FormatInt(competition.ID, 10),
	})
}

// Archive 归档竞赛。
func (s *competitionService) Archive(ctx context.Context, sc *svcctx.ServiceContext, id int64) (*dto.CompetitionStatusResp, error) {
	competition, err := getCompetition(ctx, s.competitionRepo, id)
	if err != nil {
		return nil, err
	}
	if err := s.ensureCompetitionOwned(sc, competition); err != nil {
		return nil, err
	}
	if competition.Status != enum.CompetitionStatusEnded {
		return nil, errcode.ErrCompetitionStatusInvalid
	}
	if err := s.cleanupCompetitionRuntimeResources(ctx, competition); err != nil {
		return nil, err
	}
	if err := s.competitionRepo.UpdateStatus(ctx, competition.ID, enum.CompetitionStatusArchived); err != nil {
		return nil, err
	}
	writeCompetitionStatusCache(ctx, competition.ID, enum.CompetitionStatusArchived)
	writeCompetitionFrozenCache(ctx, competition.ID, false)
	clearCompetitionCacheData(ctx, s.adGroupRepo, s.adRoundRepo, competition.ID)
	return &dto.CompetitionStatusResp{
		ID:         int64String(competition.ID),
		Status:     enum.CompetitionStatusArchived,
		StatusText: enum.GetCompetitionStatusText(enum.CompetitionStatusArchived),
	}, nil
}

// Terminate 强制终止竞赛。
func (s *competitionService) Terminate(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.TerminateCompetitionReq) (*dto.TerminateCompetitionResp, error) {
	competition, err := getCompetition(ctx, s.competitionRepo, id)
	if err != nil {
		return nil, err
	}
	if !sc.IsSuperAdmin() {
		return nil, errcode.ErrForbidden
	}
	// 强制终止只允许作用于进行中的竞赛，避免草稿/报名中竞赛被误收口为已结束。
	if competition.Status != enum.CompetitionStatusRunning {
		return nil, errcode.ErrCompetitionStatusInvalid.WithMessage("仅进行中的竞赛可强制终止")
	}
	envs, _ := s.environmentRepo.ListByCompetitionID(ctx, competition.ID)
	destroyed := 0
	for _, env := range envs {
		if env.Status != enum.ChallengeEnvStatusDestroyed {
			destroyed++
		}
	}
	if err := s.cleanupCompetitionRuntimeResources(ctx, competition); err != nil {
		return nil, err
	}
	err = database.TransactionWithDB(ctx, s.db, func(tx *gorm.DB) error {
		txCompetitionRepo := ctfrepo.NewCompetitionRepository(tx)
		txEnvironmentRepo := ctfrepo.NewChallengeEnvironmentRepository(tx)
		txAdGroupRepo := ctfrepo.NewAdGroupRepository(tx)
		txAdRoundRepo := ctfrepo.NewAdRoundRepository(tx)
		txAdTeamChainRepo := ctfrepo.NewAdTeamChainRepository(tx)
		if err := txCompetitionRepo.UpdateStatus(ctx, competition.ID, enum.CompetitionStatusEnded); err != nil {
			return err
		}
		if err := txEnvironmentRepo.DestroyByCompetitionID(ctx, competition.ID); err != nil {
			return err
		}
		// 强制终止需要与自然结束保持一致，攻防赛分组、回合和队伍链都要同步收口。
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
		}
		if err := txAdTeamChainRepo.StopByCompetitionID(ctx, competition.ID); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if err := s.persistCompetitionFinalArtifacts(ctx, competition); err != nil {
		return nil, err
	}
	writeCompetitionStatusCache(ctx, competition.ID, enum.CompetitionStatusEnded)
	writeCompetitionFrozenCache(ctx, competition.ID, false)
	if s.realtimePublisher != nil {
		_ = s.realtimePublisher.PublishLeaderboardUpdate(ctx, competition.ID, nil, nil)
	}
	audit.RecordFromContext(s.db, sc.UserID, sc.ClientIP, "terminate_ctf_competition", "competition", competition.ID, map[string]interface{}{
		"reason":                 req.Reason,
		"competition_type":       competition.CompetitionType,
		"environments_destroyed": destroyed,
		"terminated_at":          time.Now().UTC().Format(time.RFC3339),
	})
	return &dto.TerminateCompetitionResp{
		ID:                    int64String(competition.ID),
		Status:                enum.CompetitionStatusEnded,
		StatusText:            enum.GetCompetitionStatusText(enum.CompetitionStatusEnded),
		EnvironmentsDestroyed: destroyed,
	}, nil
}

// cleanupCompetitionRuntimeResources 按真实运行时路径回收竞赛题目环境与攻防赛分组资源。
func (s *competitionService) cleanupCompetitionRuntimeResources(ctx context.Context, competition *entity.Competition) error {
	if competition == nil {
		return nil
	}
	if s.environmentOps != nil {
		environments, err := s.environmentRepo.ListByCompetitionID(ctx, competition.ID)
		if err != nil {
			return err
		}
		for _, environment := range environments {
			if environment == nil || environment.Status == enum.ChallengeEnvStatusDestroyed {
				continue
			}
			if err := s.environmentOps.destroyEnvironment(ctx, environment); err != nil {
				return err
			}
		}
	}
	if competition.CompetitionType == enum.CompetitionTypeAttackDefense && s.battleRuntimeOps != nil {
		if err := s.battleRuntimeOps.cleanupGroupRuntimesByCompetition(ctx, competition.ID); err != nil {
			return err
		}
	}
	return nil
}

// persistCompetitionFinalArtifacts 在手动强制终止场景补齐最终快照与名次回写。
func (s *competitionService) persistCompetitionFinalArtifacts(ctx context.Context, competition *entity.Competition) error {
	if competition == nil {
		return nil
	}
	if err := s.snapshotCompetitionLeaderboardForService(ctx, competition); err != nil {
		return err
	}
	return s.updateCompetitionFinalRanksForService(ctx, competition)
}

// snapshotCompetitionLeaderboardForService 在竞赛服务内写入最终排行榜快照。
// 该方法用于强制终止等非调度器结束路径，保证最终结果与自然结束保持一致。
func (s *competitionService) snapshotCompetitionLeaderboardForService(ctx context.Context, competition *entity.Competition) error {
	if competition == nil {
		return nil
	}
	rankings, _, err := s.buildLeaderboardRankings(ctx, competition, nil, 0)
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
		// 攻防赛最终快照沿用调度器口径，score 字段持久化的是队伍 Token 余额；
		// 解题赛则持久化题目总分，确保最终榜和历史榜恢复逻辑一致。
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

// updateCompetitionFinalRanksForService 在竞赛服务内回写团队最终名次。
func (s *competitionService) updateCompetitionFinalRanksForService(ctx context.Context, competition *entity.Competition) error {
	if competition == nil {
		return nil
	}
	rankings, _, err := s.buildLeaderboardRankings(ctx, competition, nil, 0)
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

// CreateAnnouncement 发布公告。
func (s *competitionService) ensureCompetitionReadable(sc *svcctx.ServiceContext, competition *entity.Competition) error {
	if sc == nil || competition == nil {
		return errcode.ErrForbidden
	}
	if sc.IsSuperAdmin() {
		return nil
	}
	if competition.Scope == enum.CompetitionScopePlatform {
		return nil
	}
	if competition.SchoolID != nil && *competition.SchoolID == sc.SchoolID {
		return nil
	}
	return errcode.ErrCompetitionNotFound
}

// ensureCompetitionOwned 校验当前上下文是否为竞赛创建者。
// 编辑、发布、归档、题目配置和公告等写入口都遵循“仅竞赛创建者”规则，
// 超级管理员不能通过通用管理校验绕开文档约束。
func (s *competitionService) ensureCompetitionOwned(sc *svcctx.ServiceContext, competition *entity.Competition) error {
	if err := s.ensureCompetitionReadable(sc, competition); err != nil {
		return err
	}
	if competition.CreatedBy == sc.UserID {
		return nil
	}
	return errcode.ErrForbidden
}

// ensureCompetitionManageable 校验当前上下文是否可访问竞赛管理视图。
// 竞赛监控、统计和资源配额详情等只读管理入口允许竞赛创建者和超级管理员访问，
// 其余真正修改状态或配置的接口必须使用 ensureCompetitionOwned。
func (s *competitionService) ensureCompetitionManageable(sc *svcctx.ServiceContext, competition *entity.Competition) error {
	if err := s.ensureCompetitionReadable(sc, competition); err != nil {
		return err
	}
	if sc.IsSuperAdmin() {
		return nil
	}
	if competition.CreatedBy == sc.UserID {
		return nil
	}
	return errcode.ErrForbidden
}

// collectTeamIDs 提取团队 ID 列表，供批量仓储查询复用。
func collectTeamIDs(teams []*entity.Team) []int64 {
	ids := make([]int64, 0, len(teams))
	for _, team := range teams {
		ids = append(ids, team.ID)
	}
	return ids
}

// buildTeamMembersMap 按团队 ID 聚合团队成员列表。
func buildTeamMembersMap(members []*entity.TeamMember) map[int64][]*entity.TeamMember {
	result := make(map[int64][]*entity.TeamMember, len(members))
	for _, member := range members {
		result[member.TeamID] = append(result[member.TeamID], member)
	}
	return result
}
