// team_registration.go
// 模块05 — CTF竞赛：团队报名业务逻辑。

package ctf

import (
	"context"
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

// RegisterCompetition 报名竞赛。
func (s *teamService) RegisterCompetition(ctx context.Context, sc *svcctx.ServiceContext, competitionID int64, req *dto.RegisterCompetitionReq) (*dto.RegistrationResp, error) {
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
	if competition.Status != enum.CompetitionStatusRegistration {
		return nil, errcode.ErrRegistrationClosed
	}
	registeredCount, err := s.teamRepo.CountRegisteredByCompetitionID(ctx, competitionID)
	if err != nil {
		return nil, err
	}
	if competition.MaxTeams != nil && *competition.MaxTeams > 0 && int(registeredCount) >= *competition.MaxTeams {
		return nil, errcode.ErrMaxTeamsReached
	}

	var team *entity.Team
	if req.TeamID != nil && *req.TeamID != "" {
		teamID, err := snowflake.ParseString(*req.TeamID)
		if err != nil {
			return nil, errcode.ErrInvalidID
		}
		team, err = s.teamRepo.GetByID(ctx, teamID)
		if err != nil {
			return nil, errcode.ErrTeamNotFound
		}
		if team.CaptainID != sc.UserID {
			return nil, errcode.ErrTeamCaptainOnly.WithMessage("只有队长可以报名")
		}
	} else {
		if competition.TeamMode != enum.TeamModeIndividual {
			return nil, errcode.ErrInvalidParams.WithMessage("组队赛必须指定 team_id")
		}
		createdTeam, err := s.createSoloTeam(ctx, sc, competitionID)
		if err != nil {
			return nil, err
		}
		team = createdTeam
	}
	if team.CompetitionID != competitionID {
		return nil, errcode.ErrTeamNotFound
	}
	memberCount, err := s.teamMemberRepo.CountByTeamID(ctx, team.ID)
	if err != nil {
		return nil, err
	}
	if int(memberCount) < competition.MinTeamSize {
		return nil, errcode.ErrTeamMinMemberNotReached.WithMessage("团队人数不满足最低要求")
	}
	if registration, err := s.registrationRepo.GetByCompetitionAndTeam(ctx, competitionID, team.ID); err == nil && registration.Status == enum.RegistrationStatusRegistered {
		return nil, errcode.ErrAlreadyRegistered
	}

	registration := &entity.CompetitionRegistration{
		ID:            snowflake.Generate(),
		CompetitionID: competitionID,
		TeamID:        team.ID,
		RegisteredBy:  sc.UserID,
		Status:        enum.RegistrationStatusRegistered,
		CreatedAt:     time.Now(),
	}
	err = database.TransactionWithDB(ctx, s.db, func(tx *gorm.DB) error {
		txRegistrationRepo := ctfrepo.NewCompetitionRegistrationRepository(tx)
		if err := txRegistrationRepo.Create(ctx, registration); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &dto.RegistrationResp{
		RegistrationID: int64String(registration.ID),
		CompetitionID:  int64String(competitionID),
		TeamID:         int64String(team.ID),
		TeamName:       team.Name,
		Status:         registration.Status,
		StatusText:     enum.GetRegistrationStatusText(registration.Status),
		RegisteredAt:   timeString(registration.CreatedAt),
	}, nil
}

// CancelRegistration 取消报名。
func (s *teamService) CancelRegistration(ctx context.Context, sc *svcctx.ServiceContext, competitionID int64) error {
	competition, err := getCompetition(ctx, s.competitionRepo, competitionID)
	if err != nil {
		return err
	}
	if competition.Status != enum.CompetitionStatusRegistration {
		return errcode.ErrRegistrationClosed
	}
	registration, err := s.registrationRepo.GetByCompetitionAndStudent(ctx, competitionID, sc.UserID)
	if err != nil {
		return errcode.ErrRegistrationNotFound
	}
	team, err := s.teamRepo.GetByID(ctx, registration.TeamID)
	if err != nil {
		return err
	}
	if team.CaptainID != sc.UserID {
		return errcode.ErrTeamCaptainOnly
	}
	return database.TransactionWithDB(ctx, s.db, func(tx *gorm.DB) error {
		txRegistrationRepo := ctfrepo.NewCompetitionRegistrationRepository(tx)
		txTeamRepo := ctfrepo.NewTeamRepository(tx)
		if err := txRegistrationRepo.UpdateStatus(ctx, registration.ID, enum.RegistrationStatusCanceled); err != nil {
			return err
		}
		return txTeamRepo.UpdateStatus(ctx, team.ID, enum.TeamStatusForming)
	})
}

// ListRegistrations 查询竞赛报名列表。
func (s *teamService) ListRegistrations(ctx context.Context, sc *svcctx.ServiceContext, competitionID int64, req *dto.RegistrationListReq) (*dto.RegistrationListResp, error) {
	competition, err := getCompetition(ctx, s.competitionRepo, competitionID)
	if err != nil {
		return nil, err
	}
	if sc == nil {
		return nil, errcode.ErrForbidden
	}
	if err := s.ensureCompetitionOwned(sc, competition); err != nil {
		return nil, err
	}
	registrations, _, err := s.registrationRepo.ListByCompetitionID(ctx, competitionID, req.Page, req.PageSize)
	if err != nil {
		return nil, err
	}
	teamIDs := make([]int64, 0, len(registrations))
	for _, item := range registrations {
		teamIDs = append(teamIDs, item.TeamID)
	}
	teams, _ := s.teamRepo.ListByIDs(ctx, teamIDs)
	teamMap := make(map[int64]*entity.Team, len(teams))
	for _, team := range teams {
		teamMap[team.ID] = team
	}
	memberCounts, _ := s.teamMemberRepo.CountByTeamIDs(ctx, teamIDs)
	items := make([]dto.RegistrationListItem, 0, len(registrations))
	for _, registration := range registrations {
		team := teamMap[registration.TeamID]
		if team == nil {
			continue
		}
		items = append(items, dto.RegistrationListItem{
			ID:           int64String(registration.ID),
			TeamID:       int64String(team.ID),
			TeamName:     team.Name,
			CaptainName:  s.userQuerier.GetUserName(ctx, team.CaptainID),
			MemberCount:  int(memberCounts[team.ID]),
			Status:       registration.Status,
			StatusText:   enum.GetRegistrationStatusText(registration.Status),
			RegisteredAt: timeString(registration.CreatedAt),
		})
	}
	return &dto.RegistrationListResp{List: items}, nil
}

// GetMyRegistration 获取我的报名状态。
func (s *teamService) GetMyRegistration(ctx context.Context, sc *svcctx.ServiceContext, competitionID int64) (*dto.MyRegistrationResp, error) {
	competition, err := getCompetition(ctx, s.competitionRepo, competitionID)
	if err != nil {
		return nil, err
	}
	if err := s.ensureCompetitionReadable(sc, competition); err != nil {
		return nil, err
	}
	registration, err := s.registrationRepo.GetByCompetitionAndStudent(ctx, competitionID, sc.UserID)
	if err != nil {
		return &dto.MyRegistrationResp{IsRegistered: false}, nil
	}
	team, _ := s.teamRepo.GetByID(ctx, registration.TeamID)
	statusText := enum.GetRegistrationStatusText(registration.Status)
	registrationID := int64String(registration.ID)
	teamID := int64String(registration.TeamID)
	teamName := ""
	if team != nil {
		teamName = team.Name
	}
	return &dto.MyRegistrationResp{
		IsRegistered:   registration.Status == enum.RegistrationStatusRegistered,
		RegistrationID: &registrationID,
		TeamID:         &teamID,
		TeamName:       &teamName,
		Status:         &registration.Status,
		StatusText:     &statusText,
		RegisteredAt:   optionalTimeString(&registration.CreatedAt),
	}, nil
}
