// submission_repo.go
// 模块05 — CTF竞赛：提交记录与排行榜快照数据访问层。
// 负责 submissions、leaderboard_snapshots 的写入、查询、统计和历史快照读取。

package ctfrepo

import (
	"context"
	"database/sql"
	"time"

	"gorm.io/gorm"

	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/pkg/pagination"
	"github.com/lenschain/backend/internal/pkg/snowflake"
)

// SubmissionRepository 提交记录数据访问接口。
type SubmissionRepository interface {
	Create(ctx context.Context, submission *entity.Submission) error
	GetByID(ctx context.Context, id int64) (*entity.Submission, error)
	List(ctx context.Context, params *SubmissionListParams) ([]*entity.Submission, int64, error)
	ListRecent(ctx context.Context, competitionID int64, limit int) ([]*entity.Submission, error)
	HasCorrectSubmission(ctx context.Context, competitionID, teamID, challengeID int64) (bool, error)
	CountAttempts(ctx context.Context, competitionID, teamID, challengeID int64, since time.Time) (int64, error)
	CountByCompetition(ctx context.Context, competitionID int64) (*SubmissionCountStats, error)
	CountByChallenge(ctx context.Context, competitionID int64) ([]*ChallengeSubmissionStats, error)
	CorrectSubmissionsByTeam(ctx context.Context, competitionID, teamID int64) ([]*entity.Submission, error)
	LastCorrectSubmissionAt(ctx context.Context, competitionID, teamID int64) (*time.Time, error)
}

// SubmissionListParams 提交记录列表查询参数。
type SubmissionListParams struct {
	CompetitionID  int64
	ChallengeID    int64
	TeamID         int64
	StudentID      int64
	SubmissionType int16
	IsCorrect      *bool
	From           *time.Time
	To             *time.Time
	SortBy         string
	SortOrder      string
	Page           int
	PageSize       int
}

// SubmissionCountStats 竞赛提交统计。
type SubmissionCountStats struct {
	TotalSubmissions   int64 `gorm:"column:total_submissions"`
	CorrectSubmissions int64 `gorm:"column:correct_submissions"`
}

// ChallengeSubmissionStats 题目提交聚合统计。
type ChallengeSubmissionStats struct {
	ChallengeID  int64 `gorm:"column:challenge_id"`
	AttemptCount int64 `gorm:"column:attempt_count"`
	CorrectCount int64 `gorm:"column:correct_count"`
}

type submissionRepository struct {
	db *gorm.DB
}

// NewSubmissionRepository 创建提交记录数据访问实例。
func NewSubmissionRepository(db *gorm.DB) SubmissionRepository {
	return &submissionRepository{db: db}
}

// Create 创建提交记录。
func (r *submissionRepository) Create(ctx context.Context, submission *entity.Submission) error {
	if submission.ID == 0 {
		submission.ID = snowflake.Generate()
	}
	return r.db.WithContext(ctx).Create(submission).Error
}

// GetByID 根据 ID 获取提交记录。
func (r *submissionRepository) GetByID(ctx context.Context, id int64) (*entity.Submission, error) {
	var submission entity.Submission
	err := r.db.WithContext(ctx).First(&submission, id).Error
	if err != nil {
		return nil, err
	}
	return &submission, nil
}

// List 查询提交记录列表。
func (r *submissionRepository) List(ctx context.Context, params *SubmissionListParams) ([]*entity.Submission, int64, error) {
	query := r.db.WithContext(ctx).Model(&entity.Submission{})
	if params.CompetitionID > 0 {
		query = query.Where("competition_id = ?", params.CompetitionID)
	}
	if params.ChallengeID > 0 {
		query = query.Where("challenge_id = ?", params.ChallengeID)
	}
	if params.TeamID > 0 {
		query = query.Where("team_id = ?", params.TeamID)
	}
	if params.StudentID > 0 {
		query = query.Where("student_id = ?", params.StudentID)
	}
	if params.SubmissionType > 0 {
		query = query.Where("submission_type = ?", params.SubmissionType)
	}
	if params.IsCorrect != nil {
		query = query.Where("is_correct = ?", *params.IsCorrect)
	}
	if params.From != nil {
		query = query.Where("created_at >= ?", *params.From)
	}
	if params.To != nil {
		query = query.Where("created_at <= ?", *params.To)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	allowedSortFields := map[string]string{
		"created_at":    "created_at",
		"is_correct":    "is_correct",
		"score_awarded": "score_awarded",
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

	var submissions []*entity.Submission
	if err := query.Find(&submissions).Error; err != nil {
		return nil, 0, err
	}
	return submissions, total, nil
}

// ListRecent 查询竞赛最近提交记录。
func (r *submissionRepository) ListRecent(ctx context.Context, competitionID int64, limit int) ([]*entity.Submission, error) {
	var submissions []*entity.Submission
	err := r.db.WithContext(ctx).
		Where("competition_id = ?", competitionID).
		Order("created_at desc").
		Limit(limit).
		Find(&submissions).Error
	return submissions, err
}

// HasCorrectSubmission 判断团队是否已正确解出题目。
func (r *submissionRepository) HasCorrectSubmission(ctx context.Context, competitionID, teamID, challengeID int64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&entity.Submission{}).
		Where("competition_id = ? AND team_id = ? AND challenge_id = ? AND is_correct = ?", competitionID, teamID, challengeID, true).
		Count(&count).Error
	return count > 0, err
}

// CountAttempts 统计指定时间之后的提交次数，供 Redis 限流失效时做数据库兜底参考。
func (r *submissionRepository) CountAttempts(ctx context.Context, competitionID, teamID, challengeID int64, since time.Time) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&entity.Submission{}).
		Where("competition_id = ? AND team_id = ? AND challenge_id = ? AND created_at >= ?", competitionID, teamID, challengeID, since).
		Count(&count).Error
	return count, err
}

// CountByCompetition 统计竞赛提交总数和正确数。
func (r *submissionRepository) CountByCompetition(ctx context.Context, competitionID int64) (*SubmissionCountStats, error) {
	var stats SubmissionCountStats
	err := r.db.WithContext(ctx).Model(&entity.Submission{}).
		Select(`
			COUNT(*) AS total_submissions,
			COALESCE(SUM(CASE WHEN is_correct THEN 1 ELSE 0 END), 0) AS correct_submissions
		`).
		Where("competition_id = ?", competitionID).
		Scan(&stats).Error
	if err != nil {
		return nil, err
	}
	return &stats, nil
}

// CountByChallenge 按题目统计提交次数和正确次数。
func (r *submissionRepository) CountByChallenge(ctx context.Context, competitionID int64) ([]*ChallengeSubmissionStats, error) {
	var stats []*ChallengeSubmissionStats
	err := r.db.WithContext(ctx).Model(&entity.Submission{}).
		Select(`
			challenge_id,
			COUNT(*) AS attempt_count,
			COALESCE(SUM(CASE WHEN is_correct THEN 1 ELSE 0 END), 0) AS correct_count
		`).
		Where("competition_id = ?", competitionID).
		Group("challenge_id").
		Find(&stats).Error
	return stats, err
}

// CorrectSubmissionsByTeam 查询团队正确提交记录。
func (r *submissionRepository) CorrectSubmissionsByTeam(ctx context.Context, competitionID, teamID int64) ([]*entity.Submission, error) {
	var submissions []*entity.Submission
	err := r.db.WithContext(ctx).
		Where("competition_id = ? AND team_id = ? AND is_correct = ?", competitionID, teamID, true).
		Order("created_at asc").
		Find(&submissions).Error
	return submissions, err
}

// LastCorrectSubmissionAt 获取团队最后一次正确提交时间。
func (r *submissionRepository) LastCorrectSubmissionAt(ctx context.Context, competitionID, teamID int64) (*time.Time, error) {
	var t sql.NullTime
	err := r.db.WithContext(ctx).Model(&entity.Submission{}).
		Where("competition_id = ? AND team_id = ? AND is_correct = ?", competitionID, teamID, true).
		Select("MAX(created_at)").
		Scan(&t).Error
	if err != nil || !t.Valid {
		return nil, err
	}
	return &t.Time, nil
}

// LeaderboardSnapshotRepository 排行榜快照数据访问接口。
type LeaderboardSnapshotRepository interface {
	Create(ctx context.Context, snapshot *entity.LeaderboardSnapshot) error
	BatchCreate(ctx context.Context, snapshots []*entity.LeaderboardSnapshot) error
	ListByCompetition(ctx context.Context, params *LeaderboardSnapshotListParams) ([]*entity.LeaderboardSnapshot, int64, error)
	ListLatestByCompetition(ctx context.Context, competitionID int64, isFrozen *bool, limit int) ([]*entity.LeaderboardSnapshot, error)
	ListSnapshotTimes(ctx context.Context, competitionID int64, page, pageSize int) ([]time.Time, int64, error)
	DeleteByCompetitionID(ctx context.Context, competitionID int64) error
}

// LeaderboardSnapshotListParams 排行榜快照查询参数。
type LeaderboardSnapshotListParams struct {
	CompetitionID int64
	TeamID        int64
	IsFrozen      *bool
	SnapshotAt    *time.Time
	Page          int
	PageSize      int
}

type leaderboardSnapshotRepository struct {
	db *gorm.DB
}

// NewLeaderboardSnapshotRepository 创建排行榜快照数据访问实例。
func NewLeaderboardSnapshotRepository(db *gorm.DB) LeaderboardSnapshotRepository {
	return &leaderboardSnapshotRepository{db: db}
}

// Create 创建排行榜快照。
func (r *leaderboardSnapshotRepository) Create(ctx context.Context, snapshot *entity.LeaderboardSnapshot) error {
	if snapshot.ID == 0 {
		snapshot.ID = snowflake.Generate()
	}
	return r.db.WithContext(ctx).Create(snapshot).Error
}

// BatchCreate 批量创建排行榜快照。
func (r *leaderboardSnapshotRepository) BatchCreate(ctx context.Context, snapshots []*entity.LeaderboardSnapshot) error {
	if len(snapshots) == 0 {
		return nil
	}
	for i := range snapshots {
		if snapshots[i].ID == 0 {
			snapshots[i].ID = snowflake.Generate()
		}
	}
	return r.db.WithContext(ctx).CreateInBatches(snapshots, 100).Error
}

// ListByCompetition 查询竞赛排行榜快照。
func (r *leaderboardSnapshotRepository) ListByCompetition(ctx context.Context, params *LeaderboardSnapshotListParams) ([]*entity.LeaderboardSnapshot, int64, error) {
	query := r.db.WithContext(ctx).Model(&entity.LeaderboardSnapshot{}).
		Where("competition_id = ?", params.CompetitionID)
	if params.TeamID > 0 {
		query = query.Where("team_id = ?", params.TeamID)
	}
	if params.IsFrozen != nil {
		query = query.Where("is_frozen = ?", *params.IsFrozen)
	}
	if params.SnapshotAt != nil {
		query = query.Where("snapshot_at = ?", *params.SnapshotAt)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	page, pageSize := pagination.NormalizeValues(params.Page, params.PageSize)
	query = query.Order("snapshot_at desc, rank asc").Offset(pagination.Offset(page, pageSize)).Limit(pageSize)
	var snapshots []*entity.LeaderboardSnapshot
	if err := query.Find(&snapshots).Error; err != nil {
		return nil, 0, err
	}
	return snapshots, total, nil
}

// ListLatestByCompetition 查询竞赛最近一次排行榜快照。
func (r *leaderboardSnapshotRepository) ListLatestByCompetition(ctx context.Context, competitionID int64, isFrozen *bool, limit int) ([]*entity.LeaderboardSnapshot, error) {
	sub := r.db.WithContext(ctx).Model(&entity.LeaderboardSnapshot{}).
		Select("MAX(snapshot_at)").
		Where("competition_id = ?", competitionID)
	if isFrozen != nil {
		sub = sub.Where("is_frozen = ?", *isFrozen)
	}
	var snapshots []*entity.LeaderboardSnapshot
	err := r.db.WithContext(ctx).
		Where("competition_id = ? AND snapshot_at = (?)", competitionID, sub).
		Order("rank asc").
		Limit(limit).
		Find(&snapshots).Error
	return snapshots, err
}

// ListSnapshotTimes 分页查询排行榜快照时间点。
func (r *leaderboardSnapshotRepository) ListSnapshotTimes(ctx context.Context, competitionID int64, page, pageSize int) ([]time.Time, int64, error) {
	query := r.db.WithContext(ctx).Model(&entity.LeaderboardSnapshot{}).
		Where("competition_id = ?", competitionID).
		Group("snapshot_at")
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	page, pageSize = pagination.NormalizeValues(page, pageSize)
	var times []time.Time
	err := query.Select("snapshot_at").
		Order("snapshot_at desc").
		Offset(pagination.Offset(page, pageSize)).
		Limit(pageSize).
		Find(&times).Error
	return times, total, err
}

// DeleteByCompetitionID 删除竞赛排行榜快照，供清理异常竞赛数据使用。
func (r *leaderboardSnapshotRepository) DeleteByCompetitionID(ctx context.Context, competitionID int64) error {
	return r.db.WithContext(ctx).
		Where("competition_id = ?", competitionID).
		Delete(&entity.LeaderboardSnapshot{}).Error
}
