// team_service.go
// 模块05 — CTF竞赛：团队、报名与解题提交业务逻辑。
// 负责团队生命周期、竞赛报名、题目提交判定与团队维度统计。

package ctf

import (
	"context"
	"errors"
	"fmt"
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

// teamService 团队、报名与提交服务实现。
type teamService struct {
	db                 *gorm.DB
	competitionRepo    ctfrepo.CompetitionRepository
	challengeRepo      ctfrepo.ChallengeRepository
	contractRepo       ctfrepo.ChallengeContractRepository
	assertionRepo      ctfrepo.ChallengeAssertionRepository
	compChallengeRepo  ctfrepo.CompetitionChallengeRepository
	teamRepo           ctfrepo.TeamRepository
	teamMemberRepo     ctfrepo.TeamMemberRepository
	registrationRepo   ctfrepo.CompetitionRegistrationRepository
	submissionRepo     ctfrepo.SubmissionRepository
	environmentRepo    ctfrepo.ChallengeEnvironmentRepository
	userQuerier        UserSummaryQuerier
	submissionExecutor ChallengeSubmissionExecutor
	realtimePublisher  CTFRealtimePublisher
}

// NewTeamService 创建团队服务实例。
func NewTeamService(
	db *gorm.DB,
	competitionRepo ctfrepo.CompetitionRepository,
	challengeRepo ctfrepo.ChallengeRepository,
	contractRepo ctfrepo.ChallengeContractRepository,
	assertionRepo ctfrepo.ChallengeAssertionRepository,
	compChallengeRepo ctfrepo.CompetitionChallengeRepository,
	teamRepo ctfrepo.TeamRepository,
	teamMemberRepo ctfrepo.TeamMemberRepository,
	registrationRepo ctfrepo.CompetitionRegistrationRepository,
	submissionRepo ctfrepo.SubmissionRepository,
	environmentRepo ctfrepo.ChallengeEnvironmentRepository,
	userQuerier UserSummaryQuerier,
	submissionExecutor ChallengeSubmissionExecutor,
	realtimePublisher CTFRealtimePublisher,
) TeamService {
	return &teamService{
		db:                 db,
		competitionRepo:    competitionRepo,
		challengeRepo:      challengeRepo,
		contractRepo:       contractRepo,
		assertionRepo:      assertionRepo,
		compChallengeRepo:  compChallengeRepo,
		teamRepo:           teamRepo,
		teamMemberRepo:     teamMemberRepo,
		registrationRepo:   registrationRepo,
		submissionRepo:     submissionRepo,
		environmentRepo:    environmentRepo,
		userQuerier:        userQuerier,
		submissionExecutor: submissionExecutor,
		realtimePublisher:  realtimePublisher,
	}
}

// CreateTeam 创建团队。
func (s *teamService) CreateTeam(ctx context.Context, sc *svcctx.ServiceContext, competitionID int64, req *dto.CreateTeamReq) (*dto.TeamResp, error) {
	if !sc.IsStudent() {
		return nil, errcode.ErrForbidden
	}
	competition, err := getCompetition(ctx, s.competitionRepo, competitionID)
	if err != nil {
		return nil, err
	}
	if err := s.ensureCompetitionReadable(sc, competition); err != nil {
		return nil, err
	}
	if competition.TeamMode == enum.TeamModeIndividual {
		return nil, errcode.ErrTeamModeNoNeedCreate
	}
	// 组队只允许发生在报名阶段，避免草稿期提前暴露参赛组织入口。
	if competition.Status != enum.CompetitionStatusRegistration {
		return nil, errcode.ErrRegistrationClosed
	}
	if _, err := s.teamMemberRepo.GetByCompetitionAndStudent(ctx, competitionID, sc.UserID); err == nil {
		return nil, errcode.ErrAlreadyInTeam.WithMessage("当前用户已加入该竞赛团队")
	}

	inviteCode := buildInviteCode()
	team := &entity.Team{
		ID:            snowflake.Generate(),
		CompetitionID: competitionID,
		Name:          req.Name,
		CaptainID:     sc.UserID,
		InviteCode:    &inviteCode,
		Status:        enum.TeamStatusForming,
	}
	member := &entity.TeamMember{
		ID:        snowflake.Generate(),
		TeamID:    team.ID,
		StudentID: sc.UserID,
		Role:      enum.TeamMemberRoleCaptain,
		JoinedAt:  time.Now(),
	}
	err = database.TransactionWithDB(ctx, s.db, func(tx *gorm.DB) error {
		txTeamRepo := ctfrepo.NewTeamRepository(tx)
		txMemberRepo := ctfrepo.NewTeamMemberRepository(tx)
		if err := txTeamRepo.Create(ctx, team); err != nil {
			return err
		}
		return txMemberRepo.Create(ctx, member)
	})
	if err != nil {
		return nil, err
	}
	return s.GetTeam(ctx, sc, team.ID)
}

// GetTeam 获取团队详情。
func (s *teamService) GetTeam(ctx context.Context, sc *svcctx.ServiceContext, id int64) (*dto.TeamResp, error) {
	team, err := s.teamRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errcode.ErrTeamNotFound
		}
		return nil, err
	}
	if err := s.ensureTeamReadable(ctx, sc, team); err != nil {
		return nil, err
	}
	return s.buildTeamResp(ctx, team)
}

// UpdateTeam 编辑团队信息。
func (s *teamService) UpdateTeam(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.UpdateTeamReq) error {
	team, err := s.teamRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errcode.ErrTeamNotFound
		}
		return err
	}
	if err := s.ensureTeamCaptain(ctx, sc, team); err != nil {
		return err
	}
	if team.Status != enum.TeamStatusForming {
		return errcode.ErrTeamLocked
	}
	return s.teamRepo.UpdateFields(ctx, id, map[string]interface{}{
		"name":       req.Name,
		"updated_at": time.Now(),
	})
}

// DisbandTeam 解散团队。
func (s *teamService) DisbandTeam(ctx context.Context, sc *svcctx.ServiceContext, id int64) error {
	team, err := s.teamRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errcode.ErrTeamNotFound
		}
		return err
	}
	if err := s.ensureTeamCaptain(ctx, sc, team); err != nil {
		return err
	}
	if team.Status == enum.TeamStatusLocked {
		return errcode.ErrTeamLocked
	}
	return database.TransactionWithDB(ctx, s.db, func(tx *gorm.DB) error {
		txRegistrationRepo := ctfrepo.NewCompetitionRegistrationRepository(tx)
		txTeamRepo := ctfrepo.NewTeamRepository(tx)
		registration, err := txRegistrationRepo.GetByCompetitionAndTeam(ctx, team.CompetitionID, team.ID)
		if err == nil && registration != nil && registration.Status == enum.RegistrationStatusRegistered {
			if err := txRegistrationRepo.UpdateStatus(ctx, registration.ID, enum.RegistrationStatusCanceled); err != nil {
				return err
			}
		}
		return txTeamRepo.UpdateStatus(ctx, team.ID, enum.TeamStatusDisbanded)
	})
}

// JoinTeam 通过邀请码加入团队。
func (s *teamService) JoinTeam(ctx context.Context, sc *svcctx.ServiceContext, req *dto.JoinTeamReq) (*dto.JoinTeamResp, error) {
	if !sc.IsStudent() {
		return nil, errcode.ErrForbidden
	}
	team, err := s.teamRepo.GetByInviteCode(ctx, req.InviteCode)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errcode.ErrTeamNotFound
		}
		return nil, err
	}
	if team.Status != enum.TeamStatusForming {
		return nil, errcode.ErrTeamLocked
	}
	competition, err := getCompetition(ctx, s.competitionRepo, team.CompetitionID)
	if err != nil {
		return nil, err
	}
	if err := s.ensureCompetitionReadable(sc, competition); err != nil {
		return nil, err
	}
	if competition.Status != enum.CompetitionStatusRegistration {
		return nil, errcode.ErrRegistrationClosed
	}
	if _, err := s.teamMemberRepo.GetByCompetitionAndStudent(ctx, team.CompetitionID, sc.UserID); err == nil {
		return nil, errcode.ErrAlreadyInTeam.WithMessage("当前用户已加入该竞赛团队")
	}
	memberCount, err := s.teamMemberRepo.CountByTeamID(ctx, team.ID)
	if err != nil {
		return nil, err
	}
	if int(memberCount) >= competition.MaxTeamSize {
		return nil, errcode.ErrTeamFull
	}
	member := &entity.TeamMember{
		ID:        snowflake.Generate(),
		TeamID:    team.ID,
		StudentID: sc.UserID,
		Role:      enum.TeamMemberRoleMember,
		JoinedAt:  time.Now(),
	}
	if err := s.teamMemberRepo.Create(ctx, member); err != nil {
		return nil, err
	}
	return &dto.JoinTeamResp{
		TeamID:         int64String(team.ID),
		TeamName:       team.Name,
		CompetitionID:  int64String(team.CompetitionID),
		Role:           member.Role,
		RoleText:       enum.GetTeamMemberRoleText(member.Role),
		CurrentMembers: int(memberCount) + 1,
		MaxTeamSize:    competition.MaxTeamSize,
	}, nil
}

// LeaveTeam 退出团队。
func (s *teamService) LeaveTeam(ctx context.Context, sc *svcctx.ServiceContext, id int64) error {
	team, err := s.teamRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errcode.ErrTeamNotFound
		}
		return err
	}
	if team.Status == enum.TeamStatusLocked {
		return errcode.ErrTeamLocked
	}
	member, err := s.teamMemberRepo.GetByTeamAndStudent(ctx, id, sc.UserID)
	if err != nil {
		return errcode.ErrForbidden
	}
	if member.Role == enum.TeamMemberRoleCaptain {
		return errcode.ErrTeamCaptainOnly.WithMessage("队长不能直接退出团队，请先解散或移交")
	}
	return s.teamMemberRepo.Delete(ctx, id, sc.UserID)
}

// RemoveMember 移除队员。
func (s *teamService) RemoveMember(ctx context.Context, sc *svcctx.ServiceContext, teamID, studentID int64) error {
	team, err := s.teamRepo.GetByID(ctx, teamID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errcode.ErrTeamNotFound
		}
		return err
	}
	if err := s.ensureTeamCaptain(ctx, sc, team); err != nil {
		return err
	}
	if team.Status == enum.TeamStatusLocked {
		return errcode.ErrTeamLocked
	}
	if studentID == team.CaptainID {
		return errcode.ErrTeamCaptainOnly
	}
	return s.teamMemberRepo.Delete(ctx, teamID, studentID)
}

// ListTeams 查询竞赛团队列表。
func (s *teamService) ListTeams(ctx context.Context, sc *svcctx.ServiceContext, competitionID int64) (*dto.TeamListResp, error) {
	competition, err := getCompetition(ctx, s.competitionRepo, competitionID)
	if err != nil {
		return nil, err
	}
	if err := s.ensureCompetitionReadable(sc, competition); err != nil {
		return nil, err
	}
	teams, _, err := s.teamRepo.ListByCompetitionID(ctx, competitionID, &ctfrepo.TeamListParams{Page: 1, PageSize: 1000})
	if err != nil {
		return nil, err
	}
	teamIDs := collectTeamIDs(teams)
	memberCounts, _ := s.teamMemberRepo.CountByTeamIDs(ctx, teamIDs)
	registrationMap := make(map[int64]bool, len(teams))
	for _, team := range teams {
		if registration, err := s.registrationRepo.GetByCompetitionAndTeam(ctx, competitionID, team.ID); err == nil && registration.Status == enum.RegistrationStatusRegistered {
			registrationMap[team.ID] = true
		}
	}
	items := make([]dto.TeamListItem, 0, len(teams))
	for _, team := range teams {
		items = append(items, dto.TeamListItem{
			ID:          int64String(team.ID),
			Name:        team.Name,
			CaptainName: s.userQuerier.GetUserName(ctx, team.CaptainID),
			MemberCount: int(memberCounts[team.ID]),
			Status:      team.Status,
			StatusText:  enum.GetTeamStatusText(team.Status),
			Registered:  registrationMap[team.ID],
			FinalRank:   team.FinalRank,
			TotalScore:  team.TotalScore,
		})
	}
	return &dto.TeamListResp{List: items}, nil
}

// buildTeamResp 构建团队详情响应。
func (s *teamService) buildTeamResp(ctx context.Context, team *entity.Team) (*dto.TeamResp, error) {
	members, err := s.teamMemberRepo.ListByTeamID(ctx, team.ID)
	if err != nil {
		return nil, err
	}
	resp := &dto.TeamResp{
		ID:            int64String(team.ID),
		CompetitionID: int64String(team.CompetitionID),
		Name:          team.Name,
		CaptainID:     int64String(team.CaptainID),
		InviteCode:    team.InviteCode,
		Status:        team.Status,
		StatusText:    enum.GetTeamStatusText(team.Status),
		Members:       buildTeamMemberRespList(ctx, s.userQuerier, members),
	}
	return resp, nil
}

// ensureCompetitionOwned 校验当前上下文是否为竞赛创建者。
// 报名列表和提交统计都属于竞赛私有管理视图，不对超级管理员开放通用旁路。
func (s *teamService) ensureCompetitionOwned(sc *svcctx.ServiceContext, competition *entity.Competition) error {
	if sc == nil || competition == nil {
		return errcode.ErrForbidden
	}
	if competition.CreatedBy == sc.UserID {
		return nil
	}
	return errcode.ErrForbidden
}

// ensureCompetitionReadable 校验当前上下文是否可读取或参与指定竞赛。
// 团队列表、组队、报名和我的报名状态都必须遵循竞赛可见性，避免外校学生绕过校级隔离。
func (s *teamService) ensureCompetitionReadable(sc *svcctx.ServiceContext, competition *entity.Competition) error {
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

// ensureRegisteredCompetitionMember 校验学生已加入并完成指定竞赛报名。
// 该校验统一复用在解题提交、题目环境、排行榜个体视图等“仅参赛成员”入口，
// 确保用户不仅属于某支队伍，而且该队伍的报名状态已经生效。
func (s *teamService) ensureRegisteredCompetitionMember(ctx context.Context, competitionID, studentID int64) (*entity.TeamMember, *entity.Team, error) {
	member, team, err := s.getCurrentMemberTeam(ctx, competitionID, studentID)
	if err != nil {
		return nil, nil, err
	}
	registration, err := s.registrationRepo.GetByCompetitionAndTeam(ctx, competitionID, team.ID)
	if err != nil || registration == nil || registration.Status != enum.RegistrationStatusRegistered {
		return nil, nil, errcode.ErrRegistrationNotFound
	}
	return member, team, nil
}

// ensureTeamReadable 校验当前上下文是否可读取团队。
func (s *teamService) ensureTeamReadable(ctx context.Context, sc *svcctx.ServiceContext, team *entity.Team) error {
	if sc.IsSuperAdmin() || team.CaptainID == sc.UserID {
		return nil
	}
	if sc.IsSchoolAdmin() {
		competition, err := getCompetition(ctx, s.competitionRepo, team.CompetitionID)
		if err != nil {
			return err
		}
		return s.ensureCompetitionReadable(sc, competition)
	}
	isMember, err := s.teamMemberRepo.IsTeamMember(ctx, team.ID, sc.UserID)
	if err != nil {
		return err
	}
	if isMember {
		return nil
	}
	return errcode.ErrForbidden
}

// ensureTeamCaptain 校验当前上下文是否为团队队长。
func (s *teamService) ensureTeamCaptain(ctx context.Context, sc *svcctx.ServiceContext, team *entity.Team) error {
	member, err := s.teamMemberRepo.GetByTeamAndStudent(ctx, team.ID, sc.UserID)
	if err != nil {
		return errcode.ErrForbidden
	}
	if member.Role != enum.TeamMemberRoleCaptain {
		return errcode.ErrTeamCaptainOnly
	}
	return nil
}

// getCurrentMemberTeam 获取当前学生在竞赛中的团队成员关系和团队实体。
func (s *teamService) getCurrentMemberTeam(ctx context.Context, competitionID, studentID int64) (*entity.TeamMember, *entity.Team, error) {
	return getCompetitionMemberTeam(ctx, s.teamMemberRepo, s.teamRepo, competitionID, studentID)
}

// createSoloTeam 为个人赛自动创建单人团队。
func (s *teamService) createSoloTeam(ctx context.Context, sc *svcctx.ServiceContext, competitionID int64) (*entity.Team, error) {
	team := &entity.Team{
		ID:            snowflake.Generate(),
		CompetitionID: competitionID,
		Name:          fmt.Sprintf("%s的个人队", s.userQuerier.GetUserName(ctx, sc.UserID)),
		CaptainID:     sc.UserID,
		Status:        enum.TeamStatusForming,
	}
	member := &entity.TeamMember{
		ID:        snowflake.Generate(),
		TeamID:    team.ID,
		StudentID: sc.UserID,
		Role:      enum.TeamMemberRoleCaptain,
		JoinedAt:  time.Now(),
	}
	err := database.TransactionWithDB(ctx, s.db, func(tx *gorm.DB) error {
		txTeamRepo := ctfrepo.NewTeamRepository(tx)
		txMemberRepo := ctfrepo.NewTeamMemberRepository(tx)
		if err := txTeamRepo.Create(ctx, team); err != nil {
			return err
		}
		return txMemberRepo.Create(ctx, member)
	})
	if err != nil {
		return nil, err
	}
	return team, nil
}

// buildInviteCode 生成团队邀请码。
func buildInviteCode() string {
	id := snowflake.Generate()
	return strings.ToUpper(fmt.Sprintf("T%06d", id%1000000))
}

// timePtr 返回时间指针。
func timePtr(v time.Time) *time.Time {
	return &v
}

// buildTeamMemberRespList 构建团队成员响应列表。
func buildTeamMemberRespList(ctx context.Context, userQuerier UserSummaryQuerier, members []*entity.TeamMember) []dto.TeamMemberResp {
	items := make([]dto.TeamMemberResp, 0, len(members))
	for _, member := range members {
		items = append(items, dto.TeamMemberResp{
			StudentID: int64String(member.StudentID),
			Name:      userQuerier.GetUserName(ctx, member.StudentID),
			Role:      member.Role,
			RoleText:  enum.GetTeamMemberRoleText(member.Role),
			JoinedAt:  timeString(member.JoinedAt),
		})
	}
	return items
}
