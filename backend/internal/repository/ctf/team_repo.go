// team_repo.go
// 模块05 — CTF竞赛：团队、成员与报名数据访问层。
// 负责 teams、team_members、competition_registrations 的 CRUD、成员统计、报名状态和攻防赛分组分配。

package ctfrepo

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/model/enum"
	"github.com/lenschain/backend/internal/pkg/database"
	"github.com/lenschain/backend/internal/pkg/pagination"
	"github.com/lenschain/backend/internal/pkg/snowflake"
)

// TeamRepository 参赛团队数据访问接口。
type TeamRepository interface {
	Create(ctx context.Context, team *entity.Team) error
	GetByID(ctx context.Context, id int64) (*entity.Team, error)
	GetByInviteCode(ctx context.Context, inviteCode string) (*entity.Team, error)
	UpdateFields(ctx context.Context, id int64, fields map[string]interface{}) error
	UpdateStatus(ctx context.Context, id int64, status int16) error
	UpdateTokenBalance(ctx context.Context, id int64, balance int) error
	UpdateTotalScore(ctx context.Context, id int64, totalScore int) error
	IncrementTokenBalance(ctx context.Context, id int64, delta int) error
	IncrementTotalScore(ctx context.Context, id int64, delta int) error
	ListByCompetitionID(ctx context.Context, competitionID int64, params *TeamListParams) ([]*entity.Team, int64, error)
	ListByAdGroupID(ctx context.Context, adGroupID int64) ([]*entity.Team, error)
	ListRegisteredByCompetitionID(ctx context.Context, competitionID int64) ([]*entity.Team, error)
	ListByIDs(ctx context.Context, ids []int64) ([]*entity.Team, error)
	CountByCompetitionID(ctx context.Context, competitionID int64) (int64, error)
	CountRegisteredByCompetitionID(ctx context.Context, competitionID int64) (int64, error)
	LockByCompetitionID(ctx context.Context, competitionID int64) error
	AssignAdGroup(ctx context.Context, teamIDs []int64, adGroupID int64) error
	UpdateFinalRanks(ctx context.Context, ranks []TeamRankUpdate) error
}

// TeamListParams 团队列表查询参数。
type TeamListParams struct {
	Status    int16
	AdGroupID int64
	Keyword   string
	SortBy    string
	SortOrder string
	Page      int
	PageSize  int
}

// TeamRankUpdate 团队最终排名更新项。
type TeamRankUpdate struct {
	TeamID    int64
	FinalRank int
}

type teamRepository struct {
	db *gorm.DB
}

// NewTeamRepository 创建参赛团队数据访问实例。
func NewTeamRepository(db *gorm.DB) TeamRepository {
	return &teamRepository{db: db}
}

// Create 创建参赛团队。
func (r *teamRepository) Create(ctx context.Context, team *entity.Team) error {
	if team.ID == 0 {
		team.ID = snowflake.Generate()
	}
	return r.db.WithContext(ctx).Create(team).Error
}

// GetByID 根据 ID 获取团队。
func (r *teamRepository) GetByID(ctx context.Context, id int64) (*entity.Team, error) {
	var team entity.Team
	err := r.db.WithContext(ctx).First(&team, id).Error
	if err != nil {
		return nil, err
	}
	return &team, nil
}

// GetByInviteCode 根据邀请码获取团队。
func (r *teamRepository) GetByInviteCode(ctx context.Context, inviteCode string) (*entity.Team, error) {
	var team entity.Team
	err := r.db.WithContext(ctx).
		Where("invite_code = ?", inviteCode).
		First(&team).Error
	if err != nil {
		return nil, err
	}
	return &team, nil
}

// UpdateFields 更新团队指定字段。
func (r *teamRepository) UpdateFields(ctx context.Context, id int64, fields map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&entity.Team{}).
		Where("id = ?", id).
		Updates(fields).Error
}

// UpdateStatus 更新团队状态。
func (r *teamRepository) UpdateStatus(ctx context.Context, id int64, status int16) error {
	return r.db.WithContext(ctx).Model(&entity.Team{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":     status,
			"updated_at": time.Now(),
		}).Error
}

// UpdateTokenBalance 更新团队 Token 余额。
func (r *teamRepository) UpdateTokenBalance(ctx context.Context, id int64, balance int) error {
	return r.db.WithContext(ctx).Model(&entity.Team{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"token_balance": balance,
			"updated_at":    time.Now(),
		}).Error
}

// UpdateTotalScore 更新团队总分。
func (r *teamRepository) UpdateTotalScore(ctx context.Context, id int64, totalScore int) error {
	return r.db.WithContext(ctx).Model(&entity.Team{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"total_score": totalScore,
			"updated_at":  time.Now(),
		}).Error
}

// IncrementTokenBalance 增减团队 Token 余额。
func (r *teamRepository) IncrementTokenBalance(ctx context.Context, id int64, delta int) error {
	return r.db.WithContext(ctx).Model(&entity.Team{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"token_balance": gorm.Expr("COALESCE(token_balance, 0) + ?", delta),
			"updated_at":    time.Now(),
		}).Error
}

// IncrementTotalScore 增加团队总分。
func (r *teamRepository) IncrementTotalScore(ctx context.Context, id int64, delta int) error {
	return r.db.WithContext(ctx).Model(&entity.Team{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"total_score": gorm.Expr("COALESCE(total_score, 0) + ?", delta),
			"updated_at":  time.Now(),
		}).Error
}

// ListByCompetitionID 查询竞赛团队列表。
func (r *teamRepository) ListByCompetitionID(ctx context.Context, competitionID int64, params *TeamListParams) ([]*entity.Team, int64, error) {
	query := r.db.WithContext(ctx).Model(&entity.Team{}).
		Where("competition_id = ?", competitionID)
	if params.Status > 0 {
		query = query.Where("status = ?", params.Status)
	}
	if params.AdGroupID > 0 {
		query = query.Where("ad_group_id = ?", params.AdGroupID)
	}
	if params.Keyword != "" {
		query = query.Scopes(database.WithKeywordSearch(params.Keyword, "name"))
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	allowedSortFields := map[string]string{
		"created_at":    "created_at",
		"total_score":   "total_score",
		"token_balance": "token_balance",
		"final_rank":    "final_rank",
		"name":          "name",
	}
	sortBy := params.SortBy
	if sortBy == "" {
		sortBy = "created_at"
	}
	query = (&pagination.Query{
		Page:      params.Page,
		PageSize:  params.PageSize,
		SortBy:    sortBy,
		SortOrder: params.SortOrder,
	}).ApplyToGORM(query, allowedSortFields)

	var teams []*entity.Team
	if err := query.Find(&teams).Error; err != nil {
		return nil, 0, err
	}
	return teams, total, nil
}

// ListByAdGroupID 查询攻防赛分组下的团队。
func (r *teamRepository) ListByAdGroupID(ctx context.Context, adGroupID int64) ([]*entity.Team, error) {
	var teams []*entity.Team
	err := r.db.WithContext(ctx).
		Where("ad_group_id = ?", adGroupID).
		Order("id asc").
		Find(&teams).Error
	return teams, err
}

// ListRegisteredByCompetitionID 查询竞赛已报名团队。
func (r *teamRepository) ListRegisteredByCompetitionID(ctx context.Context, competitionID int64) ([]*entity.Team, error) {
	var teams []*entity.Team
	err := r.db.WithContext(ctx).Model(&entity.Team{}).
		Joins("JOIN competition_registrations ON competition_registrations.team_id = teams.id").
		Where("teams.competition_id = ?", competitionID).
		Where("competition_registrations.status = ?", enum.RegistrationStatusRegistered).
		Order("teams.created_at asc").
		Find(&teams).Error
	return teams, err
}

// ListByIDs 批量查询团队。
func (r *teamRepository) ListByIDs(ctx context.Context, ids []int64) ([]*entity.Team, error) {
	if len(ids) == 0 {
		return []*entity.Team{}, nil
	}
	var teams []*entity.Team
	err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&teams).Error
	return teams, err
}

// CountByCompetitionID 统计竞赛团队数。
func (r *teamRepository) CountByCompetitionID(ctx context.Context, competitionID int64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&entity.Team{}).
		Where("competition_id = ? AND status <> ?", competitionID, enum.TeamStatusDisbanded).
		Count(&count).Error
	return count, err
}

// CountRegisteredByCompetitionID 统计竞赛已报名团队数。
func (r *teamRepository) CountRegisteredByCompetitionID(ctx context.Context, competitionID int64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&entity.CompetitionRegistration{}).
		Where("competition_id = ? AND status = ?", competitionID, enum.RegistrationStatusRegistered).
		Count(&count).Error
	return count, err
}

// LockByCompetitionID 锁定竞赛下所有组建中的已报名团队。
func (r *teamRepository) LockByCompetitionID(ctx context.Context, competitionID int64) error {
	return r.db.WithContext(ctx).Model(&entity.Team{}).
		Where("competition_id = ? AND status = ?", competitionID, enum.TeamStatusForming).
		Updates(map[string]interface{}{
			"status":     enum.TeamStatusLocked,
			"updated_at": time.Now(),
		}).Error
}

// AssignAdGroup 批量分配团队到攻防赛分组。
func (r *teamRepository) AssignAdGroup(ctx context.Context, teamIDs []int64, adGroupID int64) error {
	if len(teamIDs) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Model(&entity.Team{}).
		Where("id IN ?", teamIDs).
		Updates(map[string]interface{}{
			"ad_group_id": adGroupID,
			"updated_at":  time.Now(),
		}).Error
}

// UpdateFinalRanks 批量更新团队最终排名。
func (r *teamRepository) UpdateFinalRanks(ctx context.Context, ranks []TeamRankUpdate) error {
	if len(ranks) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, item := range ranks {
			if err := tx.Model(&entity.Team{}).
				Where("id = ?", item.TeamID).
				Updates(map[string]interface{}{
					"final_rank": item.FinalRank,
					"updated_at": time.Now(),
				}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// TeamMemberRepository 团队成员数据访问接口。
type TeamMemberRepository interface {
	Create(ctx context.Context, member *entity.TeamMember) error
	Delete(ctx context.Context, teamID, studentID int64) error
	DeleteByTeamID(ctx context.Context, teamID int64) error
	GetByTeamAndStudent(ctx context.Context, teamID, studentID int64) (*entity.TeamMember, error)
	GetByCompetitionAndStudent(ctx context.Context, competitionID, studentID int64) (*entity.TeamMember, error)
	ListByTeamID(ctx context.Context, teamID int64) ([]*entity.TeamMember, error)
	ListByTeamIDs(ctx context.Context, teamIDs []int64) ([]*entity.TeamMember, error)
	CountByTeamID(ctx context.Context, teamID int64) (int64, error)
	CountByTeamIDs(ctx context.Context, teamIDs []int64) (map[int64]int64, error)
	IsTeamMember(ctx context.Context, teamID, studentID int64) (bool, error)
}

type teamMemberRepository struct {
	db *gorm.DB
}

// NewTeamMemberRepository 创建团队成员数据访问实例。
func NewTeamMemberRepository(db *gorm.DB) TeamMemberRepository {
	return &teamMemberRepository{db: db}
}

// Create 创建团队成员。
func (r *teamMemberRepository) Create(ctx context.Context, member *entity.TeamMember) error {
	if member.ID == 0 {
		member.ID = snowflake.Generate()
	}
	return r.db.WithContext(ctx).Create(member).Error
}

// Delete 删除团队成员。
func (r *teamMemberRepository) Delete(ctx context.Context, teamID, studentID int64) error {
	return r.db.WithContext(ctx).
		Where("team_id = ? AND student_id = ?", teamID, studentID).
		Delete(&entity.TeamMember{}).Error
}

// DeleteByTeamID 删除团队下所有成员。
func (r *teamMemberRepository) DeleteByTeamID(ctx context.Context, teamID int64) error {
	return r.db.WithContext(ctx).
		Where("team_id = ?", teamID).
		Delete(&entity.TeamMember{}).Error
}

// GetByTeamAndStudent 查询学生在指定团队的成员记录。
func (r *teamMemberRepository) GetByTeamAndStudent(ctx context.Context, teamID, studentID int64) (*entity.TeamMember, error) {
	var member entity.TeamMember
	err := r.db.WithContext(ctx).
		Where("team_id = ? AND student_id = ?", teamID, studentID).
		First(&member).Error
	if err != nil {
		return nil, err
	}
	return &member, nil
}

// GetByCompetitionAndStudent 查询学生在某竞赛中的团队成员记录。
func (r *teamMemberRepository) GetByCompetitionAndStudent(ctx context.Context, competitionID, studentID int64) (*entity.TeamMember, error) {
	var member entity.TeamMember
	err := r.db.WithContext(ctx).Model(&entity.TeamMember{}).
		Joins("JOIN teams ON teams.id = team_members.team_id").
		Where("teams.competition_id = ? AND team_members.student_id = ? AND teams.status <> ?", competitionID, studentID, enum.TeamStatusDisbanded).
		First(&member).Error
	if err != nil {
		return nil, err
	}
	return &member, nil
}

// ListByTeamID 查询团队成员列表。
func (r *teamMemberRepository) ListByTeamID(ctx context.Context, teamID int64) ([]*entity.TeamMember, error) {
	var members []*entity.TeamMember
	err := r.db.WithContext(ctx).
		Where("team_id = ?", teamID).
		Order("role asc, joined_at asc").
		Find(&members).Error
	return members, err
}

// ListByTeamIDs 批量查询多个团队的成员。
func (r *teamMemberRepository) ListByTeamIDs(ctx context.Context, teamIDs []int64) ([]*entity.TeamMember, error) {
	if len(teamIDs) == 0 {
		return []*entity.TeamMember{}, nil
	}
	var members []*entity.TeamMember
	err := r.db.WithContext(ctx).
		Where("team_id IN ?", teamIDs).
		Order("team_id asc, role asc, joined_at asc").
		Find(&members).Error
	return members, err
}

// CountByTeamID 统计团队成员数。
func (r *teamMemberRepository) CountByTeamID(ctx context.Context, teamID int64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&entity.TeamMember{}).
		Where("team_id = ?", teamID).
		Count(&count).Error
	return count, err
}

// CountByTeamIDs 批量统计团队成员数。
func (r *teamMemberRepository) CountByTeamIDs(ctx context.Context, teamIDs []int64) (map[int64]int64, error) {
	if len(teamIDs) == 0 {
		return map[int64]int64{}, nil
	}
	type row struct {
		TeamID int64 `gorm:"column:team_id"`
		Count  int64 `gorm:"column:count"`
	}
	var rows []row
	err := r.db.WithContext(ctx).Model(&entity.TeamMember{}).
		Select("team_id, COUNT(*) AS count").
		Where("team_id IN ?", teamIDs).
		Group("team_id").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	result := make(map[int64]int64, len(rows))
	for _, item := range rows {
		result[item.TeamID] = item.Count
	}
	return result, nil
}

// IsTeamMember 判断学生是否属于指定团队。
func (r *teamMemberRepository) IsTeamMember(ctx context.Context, teamID, studentID int64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&entity.TeamMember{}).
		Where("team_id = ? AND student_id = ?", teamID, studentID).
		Count(&count).Error
	return count > 0, err
}

// CompetitionRegistrationRepository 竞赛报名数据访问接口。
type CompetitionRegistrationRepository interface {
	Create(ctx context.Context, registration *entity.CompetitionRegistration) error
	GetByCompetitionAndTeam(ctx context.Context, competitionID, teamID int64) (*entity.CompetitionRegistration, error)
	GetByCompetitionAndStudent(ctx context.Context, competitionID, studentID int64) (*entity.CompetitionRegistration, error)
	UpdateStatus(ctx context.Context, id int64, status int16) error
	ListByCompetitionID(ctx context.Context, competitionID int64, page, pageSize int) ([]*entity.CompetitionRegistration, int64, error)
	ListByTeamID(ctx context.Context, teamID int64) ([]*entity.CompetitionRegistration, error)
	CountActiveByCompetitionID(ctx context.Context, competitionID int64) (int64, error)
}

type competitionRegistrationRepository struct {
	db *gorm.DB
}

// NewCompetitionRegistrationRepository 创建竞赛报名数据访问实例。
func NewCompetitionRegistrationRepository(db *gorm.DB) CompetitionRegistrationRepository {
	return &competitionRegistrationRepository{db: db}
}

// Create 创建报名记录。
func (r *competitionRegistrationRepository) Create(ctx context.Context, registration *entity.CompetitionRegistration) error {
	if registration.ID == 0 {
		registration.ID = snowflake.Generate()
	}
	return r.db.WithContext(ctx).Create(registration).Error
}

// GetByCompetitionAndTeam 查询团队报名记录。
func (r *competitionRegistrationRepository) GetByCompetitionAndTeam(ctx context.Context, competitionID, teamID int64) (*entity.CompetitionRegistration, error) {
	var registration entity.CompetitionRegistration
	err := r.db.WithContext(ctx).
		Where("competition_id = ? AND team_id = ?", competitionID, teamID).
		First(&registration).Error
	if err != nil {
		return nil, err
	}
	return &registration, nil
}

// GetByCompetitionAndStudent 查询学生在竞赛中的报名记录。
func (r *competitionRegistrationRepository) GetByCompetitionAndStudent(ctx context.Context, competitionID, studentID int64) (*entity.CompetitionRegistration, error) {
	var registration entity.CompetitionRegistration
	err := r.db.WithContext(ctx).Model(&entity.CompetitionRegistration{}).
		Joins("JOIN team_members ON team_members.team_id = competition_registrations.team_id").
		Where("competition_registrations.competition_id = ? AND team_members.student_id = ?", competitionID, studentID).
		First(&registration).Error
	if err != nil {
		return nil, err
	}
	return &registration, nil
}

// UpdateStatus 更新报名状态。
func (r *competitionRegistrationRepository) UpdateStatus(ctx context.Context, id int64, status int16) error {
	return r.db.WithContext(ctx).Model(&entity.CompetitionRegistration{}).
		Where("id = ?", id).
		Update("status", status).Error
}

// ListByCompetitionID 查询竞赛报名列表。
func (r *competitionRegistrationRepository) ListByCompetitionID(ctx context.Context, competitionID int64, page, pageSize int) ([]*entity.CompetitionRegistration, int64, error) {
	query := r.db.WithContext(ctx).Model(&entity.CompetitionRegistration{}).
		Where("competition_id = ?", competitionID)
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	page, pageSize = pagination.NormalizeValues(page, pageSize)
	query = query.Order("created_at desc").Offset(pagination.Offset(page, pageSize)).Limit(pageSize)
	var registrations []*entity.CompetitionRegistration
	if err := query.Find(&registrations).Error; err != nil {
		return nil, 0, err
	}
	return registrations, total, nil
}

// ListByTeamID 查询团队报名记录。
func (r *competitionRegistrationRepository) ListByTeamID(ctx context.Context, teamID int64) ([]*entity.CompetitionRegistration, error) {
	var registrations []*entity.CompetitionRegistration
	err := r.db.WithContext(ctx).
		Where("team_id = ?", teamID).
		Order("created_at desc").
		Find(&registrations).Error
	return registrations, err
}

// CountActiveByCompetitionID 统计竞赛有效报名数。
func (r *competitionRegistrationRepository) CountActiveByCompetitionID(ctx context.Context, competitionID int64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&entity.CompetitionRegistration{}).
		Where("competition_id = ? AND status = ?", competitionID, enum.RegistrationStatusRegistered).
		Count(&count).Error
	return count, err
}
