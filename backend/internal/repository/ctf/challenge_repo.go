// challenge_repo.go
// 模块05 — CTF竞赛：题目与参数化模板数据访问层。
// 负责 challenges、challenge_templates 的 CRUD、题库筛选、审核候选和使用次数维护。

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

// ChallengeRepository 题目主表数据访问接口。
type ChallengeRepository interface {
	Create(ctx context.Context, challenge *entity.Challenge) error
	GetByID(ctx context.Context, id int64) (*entity.Challenge, error)
	GetByIDs(ctx context.Context, ids []int64) ([]*entity.Challenge, error)
	UpdateFields(ctx context.Context, id int64, fields map[string]interface{}) error
	UpdateStatus(ctx context.Context, id int64, status int16) error
	SoftDelete(ctx context.Context, id int64) error
	List(ctx context.Context, params *ChallengeListParams) ([]*entity.Challenge, int64, error)
	ListVisibleToTeacher(ctx context.Context, teacherID, schoolID int64, params *ChallengeListParams) ([]*entity.Challenge, int64, error)
	ListPendingReview(ctx context.Context, params *ChallengeListParams) ([]*entity.Challenge, int64, error)
	CountByStatus(ctx context.Context, schoolID int64) (map[int16]int64, error)
	CountContracts(ctx context.Context, challengeID int64) (int64, error)
	CountAssertions(ctx context.Context, challengeID int64) (int64, error)
	HasPassedVerification(ctx context.Context, challengeID int64) (bool, error)
	IncrementUsage(ctx context.Context, id int64, delta int) error
}

// ChallengeListParams 题目列表查询参数。
type ChallengeListParams struct {
	SchoolID   int64
	AuthorID   int64
	Category   string
	Difficulty int16
	FlagType   int16
	Status     int16
	Statuses   []int16
	IsPublic   *bool
	Keyword    string
	SortBy     string
	SortOrder  string
	Page       int
	PageSize   int
}

// challengeRepository 题目主表数据访问实现。
type challengeRepository struct {
	db *gorm.DB
}

// NewChallengeRepository 创建题目主表数据访问实例。
func NewChallengeRepository(db *gorm.DB) ChallengeRepository {
	return &challengeRepository{db: db}
}

// Create 创建题目。
func (r *challengeRepository) Create(ctx context.Context, challenge *entity.Challenge) error {
	if challenge.ID == 0 {
		challenge.ID = snowflake.Generate()
	}
	return r.db.WithContext(ctx).Create(challenge).Error
}

// GetByID 根据 ID 获取题目。
func (r *challengeRepository) GetByID(ctx context.Context, id int64) (*entity.Challenge, error) {
	var challenge entity.Challenge
	err := r.db.WithContext(ctx).First(&challenge, id).Error
	if err != nil {
		return nil, err
	}
	return &challenge, nil
}

// GetByIDs 批量获取题目，供竞赛题目配置和列表聚合使用。
func (r *challengeRepository) GetByIDs(ctx context.Context, ids []int64) ([]*entity.Challenge, error) {
	if len(ids) == 0 {
		return []*entity.Challenge{}, nil
	}
	var challenges []*entity.Challenge
	err := r.db.WithContext(ctx).
		Where("id IN ?", ids).
		Find(&challenges).Error
	return challenges, err
}

// UpdateFields 更新题目指定字段。
func (r *challengeRepository) UpdateFields(ctx context.Context, id int64, fields map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&entity.Challenge{}).
		Where("id = ?", id).
		Updates(fields).Error
}

// UpdateStatus 更新题目状态。
func (r *challengeRepository) UpdateStatus(ctx context.Context, id int64, status int16) error {
	return r.db.WithContext(ctx).Model(&entity.Challenge{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":     status,
			"updated_at": time.Now(),
		}).Error
}

// SoftDelete 软删除题目。
func (r *challengeRepository) SoftDelete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&entity.Challenge{}, id).Error
}

// List 查询题库列表。
func (r *challengeRepository) List(ctx context.Context, params *ChallengeListParams) ([]*entity.Challenge, int64, error) {
	query := r.applyListFilters(r.db.WithContext(ctx).Model(&entity.Challenge{}), params)
	return r.listByQuery(query, params)
}

// ListVisibleToTeacher 查询教师可见题目：自己创建的题目，或公共题库中已通过题目。
func (r *challengeRepository) ListVisibleToTeacher(ctx context.Context, teacherID, schoolID int64, params *ChallengeListParams) ([]*entity.Challenge, int64, error) {
	query := r.applyListFilters(r.db.WithContext(ctx).Model(&entity.Challenge{}), params).
		Where("(author_id = ? OR (is_public = ? AND status = ?))", teacherID, true, enum.ChallengeStatusApproved)
	if schoolID > 0 {
		query = query.Where("(school_id = ? OR is_public = ?)", schoolID, true)
	}
	return r.listByQuery(query, params)
}

// ListPendingReview 查询待审核题目列表。
func (r *challengeRepository) ListPendingReview(ctx context.Context, params *ChallengeListParams) ([]*entity.Challenge, int64, error) {
	query := r.applyListFilters(r.db.WithContext(ctx).Model(&entity.Challenge{}), params).
		Where("status = ?", enum.ChallengeStatusPending)
	return r.listByQuery(query, params)
}

// CountByStatus 按题目状态统计数量。
func (r *challengeRepository) CountByStatus(ctx context.Context, schoolID int64) (map[int16]int64, error) {
	type row struct {
		Status int16 `gorm:"column:status"`
		Count  int64 `gorm:"column:count"`
	}
	var rows []row
	query := r.db.WithContext(ctx).Model(&entity.Challenge{}).
		Select("status, COUNT(*) AS count").
		Group("status")
	if schoolID > 0 {
		query = query.Scopes(database.WithSchoolID(schoolID))
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

// CountContracts 统计题目合约数量，支撑链上验证题目提交预验证校验。
func (r *challengeRepository) CountContracts(ctx context.Context, challengeID int64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&entity.ChallengeContract{}).
		Where("challenge_id = ?", challengeID).
		Count(&count).Error
	return count, err
}

// CountAssertions 统计题目断言数量，支撑链上验证题目提交预验证校验。
func (r *challengeRepository) CountAssertions(ctx context.Context, challengeID int64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&entity.ChallengeAssertion{}).
		Where("challenge_id = ?", challengeID).
		Count(&count).Error
	return count, err
}

// HasPassedVerification 判断题目是否存在通过的预验证记录。
func (r *challengeRepository) HasPassedVerification(ctx context.Context, challengeID int64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&entity.ChallengeVerification{}).
		Where("challenge_id = ? AND status = ?", challengeID, enum.VerificationStatusPassed).
		Count(&count).Error
	return count > 0, err
}

// IncrementUsage 增减题目被竞赛使用次数。
func (r *challengeRepository) IncrementUsage(ctx context.Context, id int64, delta int) error {
	return r.db.WithContext(ctx).Model(&entity.Challenge{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"usage_count": gorm.Expr("usage_count + ?", delta),
			"updated_at":  time.Now(),
		}).Error
}

func (r *challengeRepository) applyListFilters(query *gorm.DB, params *ChallengeListParams) *gorm.DB {
	if params.SchoolID > 0 {
		query = query.Scopes(database.WithSchoolID(params.SchoolID))
	}
	if params.AuthorID > 0 {
		query = query.Where("author_id = ?", params.AuthorID)
	}
	if params.Category != "" {
		query = query.Where("category = ?", params.Category)
	}
	if params.Difficulty > 0 {
		query = query.Where("difficulty = ?", params.Difficulty)
	}
	if params.FlagType > 0 {
		query = query.Where("flag_type = ?", params.FlagType)
	}
	if len(params.Statuses) > 0 {
		query = query.Where("status IN ?", params.Statuses)
	} else if params.Status > 0 {
		query = query.Scopes(database.WithStatus(params.Status))
	}
	if params.IsPublic != nil {
		query = query.Where("is_public = ?", *params.IsPublic)
	}
	if params.Keyword != "" {
		query = query.Scopes(database.WithKeywordSearch(params.Keyword, "title", "description"))
	}
	return query
}

func (r *challengeRepository) listByQuery(query *gorm.DB, params *ChallengeListParams) ([]*entity.Challenge, int64, error) {
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	allowedSortFields := map[string]string{
		"created_at":  "created_at",
		"difficulty":  "difficulty",
		"base_score":  "base_score",
		"usage_count": "usage_count",
		"status":      "status",
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

	var challenges []*entity.Challenge
	if err := query.Find(&challenges).Error; err != nil {
		return nil, 0, err
	}
	return challenges, total, nil
}

// ChallengeTemplateRepository 参数化模板库数据访问接口。
type ChallengeTemplateRepository interface {
	Create(ctx context.Context, template *entity.ChallengeTemplate) error
	GetByID(ctx context.Context, id int64) (*entity.ChallengeTemplate, error)
	GetByCode(ctx context.Context, code string) (*entity.ChallengeTemplate, error)
	UpdateFields(ctx context.Context, id int64, fields map[string]interface{}) error
	List(ctx context.Context, params *ChallengeTemplateListParams) ([]*entity.ChallengeTemplate, int64, error)
	IncrementUsage(ctx context.Context, id int64, delta int) error
}

// ChallengeTemplateListParams 模板库列表查询参数。
type ChallengeTemplateListParams struct {
	VulnerabilityType string
	Keyword           string
	SortBy            string
	SortOrder         string
	Page              int
	PageSize          int
}

// challengeTemplateRepository 参数化模板库数据访问实现。
type challengeTemplateRepository struct {
	db *gorm.DB
}

// NewChallengeTemplateRepository 创建参数化模板库数据访问实例。
func NewChallengeTemplateRepository(db *gorm.DB) ChallengeTemplateRepository {
	return &challengeTemplateRepository{db: db}
}

// Create 创建模板。
func (r *challengeTemplateRepository) Create(ctx context.Context, template *entity.ChallengeTemplate) error {
	if template.ID == 0 {
		template.ID = snowflake.Generate()
	}
	return r.db.WithContext(ctx).Create(template).Error
}

// GetByID 根据 ID 获取模板。
func (r *challengeTemplateRepository) GetByID(ctx context.Context, id int64) (*entity.ChallengeTemplate, error) {
	var template entity.ChallengeTemplate
	err := r.db.WithContext(ctx).First(&template, id).Error
	if err != nil {
		return nil, err
	}
	return &template, nil
}

// GetByCode 根据模板编码获取模板。
func (r *challengeTemplateRepository) GetByCode(ctx context.Context, code string) (*entity.ChallengeTemplate, error) {
	var template entity.ChallengeTemplate
	err := r.db.WithContext(ctx).Where("code = ?", code).First(&template).Error
	if err != nil {
		return nil, err
	}
	return &template, nil
}

// UpdateFields 更新模板指定字段。
func (r *challengeTemplateRepository) UpdateFields(ctx context.Context, id int64, fields map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&entity.ChallengeTemplate{}).
		Where("id = ?", id).
		Updates(fields).Error
}

// List 查询模板列表。
func (r *challengeTemplateRepository) List(ctx context.Context, params *ChallengeTemplateListParams) ([]*entity.ChallengeTemplate, int64, error) {
	query := r.db.WithContext(ctx).Model(&entity.ChallengeTemplate{})
	if params.VulnerabilityType != "" {
		query = query.Where("vulnerability_type = ?", params.VulnerabilityType)
	}
	if params.Keyword != "" {
		query = query.Scopes(database.WithKeywordSearch(params.Keyword, "name", "code", "description", "vulnerability_type"))
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	allowedSortFields := map[string]string{
		"created_at":  "created_at",
		"usage_count": "usage_count",
		"name":        "name",
		"updated_at":  "updated_at",
	}
	sortBy := params.SortBy
	if sortBy == "" {
		sortBy = "usage_count"
	}
	query = (&pagination.Query{
		Page:      params.Page,
		PageSize:  params.PageSize,
		SortBy:    sortBy,
		SortOrder: params.SortOrder,
	}).ApplyToGORM(query, allowedSortFields)

	var templates []*entity.ChallengeTemplate
	if err := query.Find(&templates).Error; err != nil {
		return nil, 0, err
	}
	return templates, total, nil
}

// IncrementUsage 增减模板使用次数。
func (r *challengeTemplateRepository) IncrementUsage(ctx context.Context, id int64, delta int) error {
	return r.db.WithContext(ctx).Model(&entity.ChallengeTemplate{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"usage_count": gorm.Expr("usage_count + ?", delta),
			"updated_at":  time.Now(),
		}).Error
}
