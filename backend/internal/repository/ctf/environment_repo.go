// environment_repo.go
// 模块05 — CTF竞赛：公告、资源配额与题目环境数据访问层。
// 负责 announcements、ctf_resource_quotas、challenge_environments 的 CRUD、配额计数和环境状态维护。

package ctfrepo

import (
	"context"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/model/enum"
	"github.com/lenschain/backend/internal/pkg/pagination"
	"github.com/lenschain/backend/internal/pkg/snowflake"
)

// CtfAnnouncementRepository 竞赛公告数据访问接口。
type CtfAnnouncementRepository interface {
	Create(ctx context.Context, announcement *entity.CtfAnnouncement) error
	GetByID(ctx context.Context, id int64) (*entity.CtfAnnouncement, error)
	Delete(ctx context.Context, id int64) error
	ListByCompetitionID(ctx context.Context, competitionID int64, challengeID int64) ([]*entity.CtfAnnouncement, error)
}

type ctfAnnouncementRepository struct {
	db *gorm.DB
}

// NewCtfAnnouncementRepository 创建竞赛公告数据访问实例。
func NewCtfAnnouncementRepository(db *gorm.DB) CtfAnnouncementRepository {
	return &ctfAnnouncementRepository{db: db}
}

// Create 创建竞赛公告。
func (r *ctfAnnouncementRepository) Create(ctx context.Context, announcement *entity.CtfAnnouncement) error {
	if announcement.ID == 0 {
		announcement.ID = snowflake.Generate()
	}
	return r.db.WithContext(ctx).Create(announcement).Error
}

// GetByID 根据 ID 获取竞赛公告。
func (r *ctfAnnouncementRepository) GetByID(ctx context.Context, id int64) (*entity.CtfAnnouncement, error) {
	var announcement entity.CtfAnnouncement
	err := r.db.WithContext(ctx).First(&announcement, id).Error
	if err != nil {
		return nil, err
	}
	return &announcement, nil
}

// Delete 删除竞赛公告。
func (r *ctfAnnouncementRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&entity.CtfAnnouncement{}, id).Error
}

// ListByCompetitionID 查询竞赛公告列表；challengeID 为 0 时返回全局和单题公告。
func (r *ctfAnnouncementRepository) ListByCompetitionID(ctx context.Context, competitionID int64, challengeID int64) ([]*entity.CtfAnnouncement, error) {
	query := r.db.WithContext(ctx).
		Where("competition_id = ?", competitionID)
	if challengeID > 0 {
		query = query.Where("(challenge_id IS NULL OR challenge_id = ?)", challengeID)
	}
	var announcements []*entity.CtfAnnouncement
	err := query.Order("created_at desc").Find(&announcements).Error
	return announcements, err
}

// CtfResourceQuotaRepository CTF 资源配额数据访问接口。
type CtfResourceQuotaRepository interface {
	Create(ctx context.Context, quota *entity.CtfResourceQuota) error
	GetByCompetitionID(ctx context.Context, competitionID int64) (*entity.CtfResourceQuota, error)
	Upsert(ctx context.Context, quota *entity.CtfResourceQuota) error
	UpdateFields(ctx context.Context, competitionID int64, fields map[string]interface{}) error
	IncrementNamespaces(ctx context.Context, competitionID int64, delta int) error
}

type ctfResourceQuotaRepository struct {
	db *gorm.DB
}

// NewCtfResourceQuotaRepository 创建 CTF 资源配额数据访问实例。
func NewCtfResourceQuotaRepository(db *gorm.DB) CtfResourceQuotaRepository {
	return &ctfResourceQuotaRepository{db: db}
}

// Create 创建资源配额。
func (r *ctfResourceQuotaRepository) Create(ctx context.Context, quota *entity.CtfResourceQuota) error {
	if quota.ID == 0 {
		quota.ID = snowflake.Generate()
	}
	return r.db.WithContext(ctx).Create(quota).Error
}

// GetByCompetitionID 根据竞赛 ID 获取资源配额。
func (r *ctfResourceQuotaRepository) GetByCompetitionID(ctx context.Context, competitionID int64) (*entity.CtfResourceQuota, error) {
	var quota entity.CtfResourceQuota
	err := r.db.WithContext(ctx).
		Where("competition_id = ?", competitionID).
		First(&quota).Error
	if err != nil {
		return nil, err
	}
	return &quota, nil
}

// Upsert 创建或更新竞赛资源配额。
func (r *ctfResourceQuotaRepository) Upsert(ctx context.Context, quota *entity.CtfResourceQuota) error {
	if quota.ID == 0 {
		quota.ID = snowflake.Generate()
	}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "competition_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"max_cpu",
			"max_memory",
			"max_storage",
			"max_namespaces",
			"updated_at",
		}),
	}).Create(quota).Error
}

// UpdateFields 更新资源配额指定字段。
func (r *ctfResourceQuotaRepository) UpdateFields(ctx context.Context, competitionID int64, fields map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&entity.CtfResourceQuota{}).
		Where("competition_id = ?", competitionID).
		Updates(fields).Error
}

// IncrementNamespaces 增减当前 Namespace 数量。
func (r *ctfResourceQuotaRepository) IncrementNamespaces(ctx context.Context, competitionID int64, delta int) error {
	return r.db.WithContext(ctx).Model(&entity.CtfResourceQuota{}).
		Where("competition_id = ?", competitionID).
		Updates(map[string]interface{}{
			"current_namespaces": gorm.Expr("current_namespaces + ?", delta),
			"updated_at":         time.Now(),
		}).Error
}

// ChallengeEnvironmentRepository 题目环境实例数据访问接口。
type ChallengeEnvironmentRepository interface {
	Create(ctx context.Context, environment *entity.ChallengeEnvironment) error
	GetByID(ctx context.Context, id int64) (*entity.ChallengeEnvironment, error)
	GetActiveByTeamAndChallenge(ctx context.Context, competitionID, teamID, challengeID int64) (*entity.ChallengeEnvironment, error)
	UpdateFields(ctx context.Context, id int64, fields map[string]interface{}) error
	UpdateStatus(ctx context.Context, id int64, status int16) error
	MarkDestroyed(ctx context.Context, id int64) error
	ResetToCreating(ctx context.Context, id int64) error
	List(ctx context.Context, params *ChallengeEnvironmentListParams) ([]*entity.ChallengeEnvironment, int64, error)
	ListByCompetitionID(ctx context.Context, competitionID int64) ([]*entity.ChallengeEnvironment, error)
	ListByTeamID(ctx context.Context, competitionID, teamID int64) ([]*entity.ChallengeEnvironment, error)
	CountByCompetitionAndStatus(ctx context.Context, competitionID int64, statuses []int16) (int64, error)
	CountByChallenge(ctx context.Context, competitionID int64) (map[int64]int64, error)
	DestroyByCompetitionID(ctx context.Context, competitionID int64) error
}

// ChallengeEnvironmentListParams 题目环境列表查询参数。
type ChallengeEnvironmentListParams struct {
	CompetitionID int64
	ChallengeID   int64
	TeamID        int64
	Status        int16
	Statuses      []int16
	Page          int
	PageSize      int
}

type challengeEnvironmentRepository struct {
	db *gorm.DB
}

// NewChallengeEnvironmentRepository 创建题目环境实例数据访问实例。
func NewChallengeEnvironmentRepository(db *gorm.DB) ChallengeEnvironmentRepository {
	return &challengeEnvironmentRepository{db: db}
}

// Create 创建题目环境实例。
func (r *challengeEnvironmentRepository) Create(ctx context.Context, environment *entity.ChallengeEnvironment) error {
	if environment.ID == 0 {
		environment.ID = snowflake.Generate()
	}
	return r.db.WithContext(ctx).Create(environment).Error
}

// GetByID 根据 ID 获取题目环境实例。
func (r *challengeEnvironmentRepository) GetByID(ctx context.Context, id int64) (*entity.ChallengeEnvironment, error) {
	var environment entity.ChallengeEnvironment
	err := r.db.WithContext(ctx).First(&environment, id).Error
	if err != nil {
		return nil, err
	}
	return &environment, nil
}

// GetActiveByTeamAndChallenge 获取团队题目的非销毁环境，支撑启动环境幂等处理。
func (r *challengeEnvironmentRepository) GetActiveByTeamAndChallenge(ctx context.Context, competitionID, teamID, challengeID int64) (*entity.ChallengeEnvironment, error) {
	var environment entity.ChallengeEnvironment
	err := r.db.WithContext(ctx).
		Where("competition_id = ? AND team_id = ? AND challenge_id = ? AND status <> ?", competitionID, teamID, challengeID, enum.ChallengeEnvStatusDestroyed).
		Order("created_at desc").
		First(&environment).Error
	if err != nil {
		return nil, err
	}
	return &environment, nil
}

// UpdateFields 更新题目环境指定字段。
func (r *challengeEnvironmentRepository) UpdateFields(ctx context.Context, id int64, fields map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&entity.ChallengeEnvironment{}).
		Where("id = ?", id).
		Updates(fields).Error
}

// UpdateStatus 更新题目环境状态。
func (r *challengeEnvironmentRepository) UpdateStatus(ctx context.Context, id int64, status int16) error {
	return r.db.WithContext(ctx).Model(&entity.ChallengeEnvironment{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":     status,
			"updated_at": time.Now(),
		}).Error
}

// MarkDestroyed 标记题目环境已销毁。
func (r *challengeEnvironmentRepository) MarkDestroyed(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Model(&entity.ChallengeEnvironment{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":       enum.ChallengeEnvStatusDestroyed,
			"destroyed_at": time.Now(),
			"updated_at":   time.Now(),
		}).Error
}

// ResetToCreating 将环境重置为创建中，实际销毁重建由 service 调用编排层完成。
func (r *challengeEnvironmentRepository) ResetToCreating(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Model(&entity.ChallengeEnvironment{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":           enum.ChallengeEnvStatusCreating,
			"chain_rpc_url":    nil,
			"container_status": nil,
			"started_at":       nil,
			"destroyed_at":     nil,
			"updated_at":       time.Now(),
		}).Error
}

// List 查询题目环境实例列表。
func (r *challengeEnvironmentRepository) List(ctx context.Context, params *ChallengeEnvironmentListParams) ([]*entity.ChallengeEnvironment, int64, error) {
	query := r.db.WithContext(ctx).Model(&entity.ChallengeEnvironment{})
	if params.CompetitionID > 0 {
		query = query.Where("competition_id = ?", params.CompetitionID)
	}
	if params.ChallengeID > 0 {
		query = query.Where("challenge_id = ?", params.ChallengeID)
	}
	if params.TeamID > 0 {
		query = query.Where("team_id = ?", params.TeamID)
	}
	if len(params.Statuses) > 0 {
		query = query.Where("status IN ?", params.Statuses)
	} else if params.Status > 0 {
		query = query.Where("status = ?", params.Status)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	page, pageSize := pagination.NormalizeValues(params.Page, params.PageSize)
	query = query.Order("created_at desc").Offset(pagination.Offset(page, pageSize)).Limit(pageSize)
	var environments []*entity.ChallengeEnvironment
	if err := query.Find(&environments).Error; err != nil {
		return nil, 0, err
	}
	return environments, total, nil
}

// ListByCompetitionID 查询竞赛全部题目环境。
func (r *challengeEnvironmentRepository) ListByCompetitionID(ctx context.Context, competitionID int64) ([]*entity.ChallengeEnvironment, error) {
	var environments []*entity.ChallengeEnvironment
	err := r.db.WithContext(ctx).
		Where("competition_id = ?", competitionID).
		Order("created_at desc").
		Find(&environments).Error
	return environments, err
}

// ListByTeamID 查询团队在竞赛中的全部题目环境。
func (r *challengeEnvironmentRepository) ListByTeamID(ctx context.Context, competitionID, teamID int64) ([]*entity.ChallengeEnvironment, error) {
	var environments []*entity.ChallengeEnvironment
	err := r.db.WithContext(ctx).
		Where("competition_id = ? AND team_id = ?", competitionID, teamID).
		Order("created_at desc").
		Find(&environments).Error
	return environments, err
}

// CountByCompetitionAndStatus 按状态统计竞赛题目环境数量。
func (r *challengeEnvironmentRepository) CountByCompetitionAndStatus(ctx context.Context, competitionID int64, statuses []int16) (int64, error) {
	var count int64
	query := r.db.WithContext(ctx).Model(&entity.ChallengeEnvironment{}).
		Where("competition_id = ?", competitionID)
	if len(statuses) > 0 {
		query = query.Where("status IN ?", statuses)
	}
	err := query.Count(&count).Error
	return count, err
}

// CountByChallenge 按题目统计竞赛环境数量。
func (r *challengeEnvironmentRepository) CountByChallenge(ctx context.Context, competitionID int64) (map[int64]int64, error) {
	type row struct {
		ChallengeID int64 `gorm:"column:challenge_id"`
		Count       int64 `gorm:"column:count"`
	}
	var rows []row
	err := r.db.WithContext(ctx).Model(&entity.ChallengeEnvironment{}).
		Select("challenge_id, COUNT(*) AS count").
		Where("competition_id = ? AND status <> ?", competitionID, enum.ChallengeEnvStatusDestroyed).
		Group("challenge_id").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	result := make(map[int64]int64, len(rows))
	for _, item := range rows {
		result[item.ChallengeID] = item.Count
	}
	return result, nil
}

// DestroyByCompetitionID 将竞赛所有非销毁环境标记为已销毁。
func (r *challengeEnvironmentRepository) DestroyByCompetitionID(ctx context.Context, competitionID int64) error {
	return r.db.WithContext(ctx).Model(&entity.ChallengeEnvironment{}).
		Where("competition_id = ? AND status <> ?", competitionID, enum.ChallengeEnvStatusDestroyed).
		Updates(map[string]interface{}{
			"status":       enum.ChallengeEnvStatusDestroyed,
			"destroyed_at": time.Now(),
			"updated_at":   time.Now(),
		}).Error
}
