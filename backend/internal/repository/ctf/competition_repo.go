// competition_repo.go
// 模块05 — CTF竞赛：竞赛主表数据访问层。
// 负责 competitions 的 CRUD、列表筛选、状态流转候选查询，以及竞赛监控所需的数据库聚合。

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

// CompetitionRepository 竞赛主表数据访问接口。
type CompetitionRepository interface {
	Create(ctx context.Context, competition *entity.Competition) error
	GetByID(ctx context.Context, id int64) (*entity.Competition, error)
	GetByIDUnscoped(ctx context.Context, id int64) (*entity.Competition, error)
	UpdateFields(ctx context.Context, id int64, fields map[string]interface{}) error
	UpdateStatus(ctx context.Context, id int64, status int16) error
	SoftDelete(ctx context.Context, id int64) error
	List(ctx context.Context, params *CompetitionListParams) ([]*entity.Competition, int64, error)
	ListByCreator(ctx context.Context, creatorID int64, params *CompetitionListParams) ([]*entity.Competition, int64, error)
	ListRegistrationToStart(ctx context.Context, now time.Time) ([]*entity.Competition, error)
	ListRunningToEnd(ctx context.Context, now time.Time) ([]*entity.Competition, error)
	ListEndedBefore(ctx context.Context, endedBefore time.Time) ([]*entity.Competition, error)
	CountByStatus(ctx context.Context, statuses []int16) (map[int16]int64, error)
	CountParticipants(ctx context.Context, competitionID int64) (int64, error)
	Overview(ctx context.Context) (*CompetitionOverviewStats, error)
}

// CompetitionListParams 竞赛列表查询参数。
type CompetitionListParams struct {
	SchoolID        int64
	CompetitionType int16
	Scope           int16
	Status          int16
	Statuses        []int16
	Keyword         string
	SortBy          string
	SortOrder       string
	Page            int
	PageSize        int
}

// CompetitionOverviewStats 全平台竞赛概览聚合结果。
type CompetitionOverviewStats struct {
	TotalCompetitions    int64 `gorm:"column:total_competitions"`
	RunningCompetitions  int64 `gorm:"column:running_competitions"`
	UpcomingCompetitions int64 `gorm:"column:upcoming_competitions"`
}

// competitionRepository 竞赛主表数据访问实现。
type competitionRepository struct {
	db *gorm.DB
}

// NewCompetitionRepository 创建竞赛主表数据访问实例。
func NewCompetitionRepository(db *gorm.DB) CompetitionRepository {
	return &competitionRepository{db: db}
}

// Create 创建竞赛。
func (r *competitionRepository) Create(ctx context.Context, competition *entity.Competition) error {
	if competition.ID == 0 {
		competition.ID = snowflake.Generate()
	}
	return r.db.WithContext(ctx).Create(competition).Error
}

// GetByID 根据 ID 获取竞赛。
func (r *competitionRepository) GetByID(ctx context.Context, id int64) (*entity.Competition, error) {
	var competition entity.Competition
	err := r.db.WithContext(ctx).First(&competition, id).Error
	if err != nil {
		return nil, err
	}
	return &competition, nil
}

// GetByIDUnscoped 根据 ID 获取竞赛，包含已软删除记录，供审计和恢复场景使用。
func (r *competitionRepository) GetByIDUnscoped(ctx context.Context, id int64) (*entity.Competition, error) {
	var competition entity.Competition
	err := r.db.WithContext(ctx).Unscoped().First(&competition, id).Error
	if err != nil {
		return nil, err
	}
	return &competition, nil
}

// UpdateFields 更新竞赛指定字段。
func (r *competitionRepository) UpdateFields(ctx context.Context, id int64, fields map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&entity.Competition{}).
		Where("id = ?", id).
		Updates(fields).Error
}

// UpdateStatus 更新竞赛状态。
func (r *competitionRepository) UpdateStatus(ctx context.Context, id int64, status int16) error {
	return r.db.WithContext(ctx).Model(&entity.Competition{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":     status,
			"updated_at": time.Now(),
		}).Error
}

// SoftDelete 软删除竞赛。
func (r *competitionRepository) SoftDelete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&entity.Competition{}, id).Error
}

// List 查询竞赛列表。
func (r *competitionRepository) List(ctx context.Context, params *CompetitionListParams) ([]*entity.Competition, int64, error) {
	query := r.applyListFilters(r.db.WithContext(ctx).Model(&entity.Competition{}), params)
	return r.listByQuery(query, params)
}

// ListByCreator 查询创建者名下的竞赛列表。
func (r *competitionRepository) ListByCreator(ctx context.Context, creatorID int64, params *CompetitionListParams) ([]*entity.Competition, int64, error) {
	query := r.applyListFilters(r.db.WithContext(ctx).Model(&entity.Competition{}), params).
		Where("created_by = ?", creatorID)
	return r.listByQuery(query, params)
}

// ListRegistrationToStart 查询到达开始时间、需要从报名中流转为进行中的竞赛。
func (r *competitionRepository) ListRegistrationToStart(ctx context.Context, now time.Time) ([]*entity.Competition, error) {
	var competitions []*entity.Competition
	err := r.db.WithContext(ctx).
		Where("status = ?", enum.CompetitionStatusRegistration).
		Where("start_at IS NOT NULL AND start_at <= ?", now).
		Find(&competitions).Error
	return competitions, err
}

// ListRunningToEnd 查询到达结束时间、需要从进行中流转为已结束的竞赛。
func (r *competitionRepository) ListRunningToEnd(ctx context.Context, now time.Time) ([]*entity.Competition, error) {
	var competitions []*entity.Competition
	err := r.db.WithContext(ctx).
		Where("status = ?", enum.CompetitionStatusRunning).
		Where("end_at IS NOT NULL AND end_at <= ?", now).
		Find(&competitions).Error
	return competitions, err
}

// ListEndedBefore 查询已结束超过指定时间的竞赛，供归档定时任务使用。
func (r *competitionRepository) ListEndedBefore(ctx context.Context, endedBefore time.Time) ([]*entity.Competition, error) {
	var competitions []*entity.Competition
	err := r.db.WithContext(ctx).
		Where("status = ?", enum.CompetitionStatusEnded).
		Where("end_at IS NOT NULL AND end_at < ?", endedBefore).
		Find(&competitions).Error
	return competitions, err
}

// CountByStatus 按状态统计竞赛数量。
func (r *competitionRepository) CountByStatus(ctx context.Context, statuses []int16) (map[int16]int64, error) {
	type row struct {
		Status int16 `gorm:"column:status"`
		Count  int64 `gorm:"column:count"`
	}
	var rows []row
	query := r.db.WithContext(ctx).Model(&entity.Competition{}).
		Select("status, COUNT(*) AS count").
		Group("status")
	if len(statuses) > 0 {
		query = query.Where("status IN ?", statuses)
	}
	if err := query.Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make(map[int16]int64, len(rows))
	for _, item := range rows {
		result[item.Status] = item.Count
	}
	return result, nil
}

// CountParticipants 统计竞赛参赛人数。
func (r *competitionRepository) CountParticipants(ctx context.Context, competitionID int64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&entity.TeamMember{}).
		Joins("JOIN teams ON teams.id = team_members.team_id").
		Where("teams.competition_id = ?", competitionID).
		Count(&count).Error
	return count, err
}

// Overview 获取全平台竞赛概览基础统计。
func (r *competitionRepository) Overview(ctx context.Context) (*CompetitionOverviewStats, error) {
	var stats CompetitionOverviewStats
	err := r.db.WithContext(ctx).Model(&entity.Competition{}).
		Select(`
			COUNT(*) AS total_competitions,
			COALESCE(SUM(CASE WHEN status = ? THEN 1 ELSE 0 END), 0) AS running_competitions,
			COALESCE(SUM(CASE WHEN status = ? THEN 1 ELSE 0 END), 0) AS upcoming_competitions
		`, enum.CompetitionStatusRunning, enum.CompetitionStatusRegistration).
		Scan(&stats).Error
	if err != nil {
		return nil, err
	}
	return &stats, nil
}

func (r *competitionRepository) applyListFilters(query *gorm.DB, params *CompetitionListParams) *gorm.DB {
	if params.SchoolID > 0 {
		// 学校视角可见平台级竞赛和本校校级竞赛。
		query = query.Where("(scope = ? OR school_id = ?)", enum.CompetitionScopePlatform, params.SchoolID)
	}
	if params.CompetitionType > 0 {
		query = query.Where("competition_type = ?", params.CompetitionType)
	}
	if params.Scope > 0 {
		query = query.Where("scope = ?", params.Scope)
	}
	if len(params.Statuses) > 0 {
		query = query.Where("status IN ?", params.Statuses)
	} else if params.Status > 0 {
		query = query.Scopes(database.WithStatus(params.Status))
	}
	if params.Keyword != "" {
		query = query.Scopes(database.WithKeywordSearch(params.Keyword, "title"))
	}
	return query
}

func (r *competitionRepository) listByQuery(query *gorm.DB, params *CompetitionListParams) ([]*entity.Competition, int64, error) {
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	allowedSortFields := map[string]string{
		"created_at": "created_at",
		"start_at":   "start_at",
		"end_at":     "end_at",
		"status":     "status",
		"title":      "title",
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

	var competitions []*entity.Competition
	if err := query.Find(&competitions).Error; err != nil {
		return nil, 0, err
	}
	return competitions, total, nil
}
