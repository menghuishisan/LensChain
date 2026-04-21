// challenge_sub_repo.go
// 模块05 — CTF竞赛：题目子资源数据访问层。
// 负责合约、断言、审核、预验证、竞赛题目关联等表的 CRUD、批量查询和排序更新。

package ctfrepo

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/model/enum"
	"github.com/lenschain/backend/internal/pkg/pagination"
	"github.com/lenschain/backend/internal/pkg/snowflake"
)

// ChallengeContractRepository 题目合约数据访问接口。
type ChallengeContractRepository interface {
	Create(ctx context.Context, contract *entity.ChallengeContract) error
	GetByID(ctx context.Context, id int64) (*entity.ChallengeContract, error)
	UpdateFields(ctx context.Context, id int64, fields map[string]interface{}) error
	Delete(ctx context.Context, id int64) error
	DeleteByChallengeID(ctx context.Context, challengeID int64) error
	ListByChallengeID(ctx context.Context, challengeID int64) ([]*entity.ChallengeContract, error)
	ListByChallengeIDs(ctx context.Context, challengeIDs []int64) ([]*entity.ChallengeContract, error)
	BatchCreate(ctx context.Context, contracts []*entity.ChallengeContract) error
}

type challengeContractRepository struct {
	db *gorm.DB
}

// NewChallengeContractRepository 创建题目合约数据访问实例。
func NewChallengeContractRepository(db *gorm.DB) ChallengeContractRepository {
	return &challengeContractRepository{db: db}
}

// Create 创建题目合约。
func (r *challengeContractRepository) Create(ctx context.Context, contract *entity.ChallengeContract) error {
	if contract.ID == 0 {
		contract.ID = snowflake.Generate()
	}
	return r.db.WithContext(ctx).Create(contract).Error
}

// GetByID 根据 ID 获取题目合约。
func (r *challengeContractRepository) GetByID(ctx context.Context, id int64) (*entity.ChallengeContract, error) {
	var contract entity.ChallengeContract
	err := r.db.WithContext(ctx).First(&contract, id).Error
	if err != nil {
		return nil, err
	}
	return &contract, nil
}

// UpdateFields 更新题目合约指定字段。
func (r *challengeContractRepository) UpdateFields(ctx context.Context, id int64, fields map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&entity.ChallengeContract{}).
		Where("id = ?", id).
		Updates(fields).Error
}

// Delete 删除题目合约。
func (r *challengeContractRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&entity.ChallengeContract{}, id).Error
}

// DeleteByChallengeID 删除题目下所有合约，供重新导入或模板生成回滚使用。
func (r *challengeContractRepository) DeleteByChallengeID(ctx context.Context, challengeID int64) error {
	return r.db.WithContext(ctx).
		Where("challenge_id = ?", challengeID).
		Delete(&entity.ChallengeContract{}).Error
}

// ListByChallengeID 查询题目合约列表。
func (r *challengeContractRepository) ListByChallengeID(ctx context.Context, challengeID int64) ([]*entity.ChallengeContract, error) {
	var contracts []*entity.ChallengeContract
	err := r.db.WithContext(ctx).
		Where("challenge_id = ?", challengeID).
		Order("deploy_order asc, created_at asc").
		Find(&contracts).Error
	return contracts, err
}

// ListByChallengeIDs 批量查询多个题目的合约，供详情聚合避免 N+1 查询。
func (r *challengeContractRepository) ListByChallengeIDs(ctx context.Context, challengeIDs []int64) ([]*entity.ChallengeContract, error) {
	if len(challengeIDs) == 0 {
		return []*entity.ChallengeContract{}, nil
	}
	var contracts []*entity.ChallengeContract
	err := r.db.WithContext(ctx).
		Where("challenge_id IN ?", challengeIDs).
		Order("challenge_id asc, deploy_order asc, created_at asc").
		Find(&contracts).Error
	return contracts, err
}

// BatchCreate 批量创建题目合约。
func (r *challengeContractRepository) BatchCreate(ctx context.Context, contracts []*entity.ChallengeContract) error {
	if len(contracts) == 0 {
		return nil
	}
	for i := range contracts {
		if contracts[i].ID == 0 {
			contracts[i].ID = snowflake.Generate()
		}
	}
	return r.db.WithContext(ctx).CreateInBatches(contracts, 50).Error
}

// ChallengeAssertionRepository 题目断言数据访问接口。
type ChallengeAssertionRepository interface {
	Create(ctx context.Context, assertion *entity.ChallengeAssertion) error
	GetByID(ctx context.Context, id int64) (*entity.ChallengeAssertion, error)
	UpdateFields(ctx context.Context, id int64, fields map[string]interface{}) error
	Delete(ctx context.Context, id int64) error
	DeleteByChallengeID(ctx context.Context, challengeID int64) error
	ListByChallengeID(ctx context.Context, challengeID int64) ([]*entity.ChallengeAssertion, error)
	ListByChallengeIDs(ctx context.Context, challengeIDs []int64) ([]*entity.ChallengeAssertion, error)
	BatchCreate(ctx context.Context, assertions []*entity.ChallengeAssertion) error
	BatchUpdateSort(ctx context.Context, items []AssertionSortItem) error
}

// AssertionSortItem 断言排序更新项。
type AssertionSortItem struct {
	ID        int64
	SortOrder int
}

type challengeAssertionRepository struct {
	db *gorm.DB
}

// NewChallengeAssertionRepository 创建题目断言数据访问实例。
func NewChallengeAssertionRepository(db *gorm.DB) ChallengeAssertionRepository {
	return &challengeAssertionRepository{db: db}
}

// Create 创建题目断言。
func (r *challengeAssertionRepository) Create(ctx context.Context, assertion *entity.ChallengeAssertion) error {
	if assertion.ID == 0 {
		assertion.ID = snowflake.Generate()
	}
	return r.db.WithContext(ctx).Create(assertion).Error
}

// GetByID 根据 ID 获取题目断言。
func (r *challengeAssertionRepository) GetByID(ctx context.Context, id int64) (*entity.ChallengeAssertion, error) {
	var assertion entity.ChallengeAssertion
	err := r.db.WithContext(ctx).First(&assertion, id).Error
	if err != nil {
		return nil, err
	}
	return &assertion, nil
}

// UpdateFields 更新题目断言指定字段。
func (r *challengeAssertionRepository) UpdateFields(ctx context.Context, id int64, fields map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&entity.ChallengeAssertion{}).
		Where("id = ?", id).
		Updates(fields).Error
}

// Delete 删除题目断言。
func (r *challengeAssertionRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&entity.ChallengeAssertion{}, id).Error
}

// DeleteByChallengeID 删除题目下所有断言。
func (r *challengeAssertionRepository) DeleteByChallengeID(ctx context.Context, challengeID int64) error {
	return r.db.WithContext(ctx).
		Where("challenge_id = ?", challengeID).
		Delete(&entity.ChallengeAssertion{}).Error
}

// ListByChallengeID 查询题目断言列表。
func (r *challengeAssertionRepository) ListByChallengeID(ctx context.Context, challengeID int64) ([]*entity.ChallengeAssertion, error) {
	var assertions []*entity.ChallengeAssertion
	err := r.db.WithContext(ctx).
		Where("challenge_id = ?", challengeID).
		Order("sort_order asc, created_at asc").
		Find(&assertions).Error
	return assertions, err
}

// ListByChallengeIDs 批量查询多个题目的断言。
func (r *challengeAssertionRepository) ListByChallengeIDs(ctx context.Context, challengeIDs []int64) ([]*entity.ChallengeAssertion, error) {
	if len(challengeIDs) == 0 {
		return []*entity.ChallengeAssertion{}, nil
	}
	var assertions []*entity.ChallengeAssertion
	err := r.db.WithContext(ctx).
		Where("challenge_id IN ?", challengeIDs).
		Order("challenge_id asc, sort_order asc, created_at asc").
		Find(&assertions).Error
	return assertions, err
}

// BatchCreate 批量创建题目断言。
func (r *challengeAssertionRepository) BatchCreate(ctx context.Context, assertions []*entity.ChallengeAssertion) error {
	if len(assertions) == 0 {
		return nil
	}
	for i := range assertions {
		if assertions[i].ID == 0 {
			assertions[i].ID = snowflake.Generate()
		}
	}
	return r.db.WithContext(ctx).CreateInBatches(assertions, 50).Error
}

// BatchUpdateSort 批量更新断言排序。
func (r *challengeAssertionRepository) BatchUpdateSort(ctx context.Context, items []AssertionSortItem) error {
	if len(items) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, item := range items {
			if err := tx.Model(&entity.ChallengeAssertion{}).
				Where("id = ?", item.ID).
				Update("sort_order", item.SortOrder).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// ChallengeReviewRepository 题目审核记录数据访问接口。
type ChallengeReviewRepository interface {
	Create(ctx context.Context, review *entity.ChallengeReview) error
	GetByID(ctx context.Context, id int64) (*entity.ChallengeReview, error)
	ListByChallengeID(ctx context.Context, challengeID int64, page, pageSize int) ([]*entity.ChallengeReview, int64, error)
	GetLatestByChallengeID(ctx context.Context, challengeID int64) (*entity.ChallengeReview, error)
}

type challengeReviewRepository struct {
	db *gorm.DB
}

// NewChallengeReviewRepository 创建题目审核记录数据访问实例。
func NewChallengeReviewRepository(db *gorm.DB) ChallengeReviewRepository {
	return &challengeReviewRepository{db: db}
}

// Create 创建题目审核记录。
func (r *challengeReviewRepository) Create(ctx context.Context, review *entity.ChallengeReview) error {
	if review.ID == 0 {
		review.ID = snowflake.Generate()
	}
	return r.db.WithContext(ctx).Create(review).Error
}

// GetByID 根据 ID 获取题目审核记录。
func (r *challengeReviewRepository) GetByID(ctx context.Context, id int64) (*entity.ChallengeReview, error) {
	var review entity.ChallengeReview
	err := r.db.WithContext(ctx).First(&review, id).Error
	if err != nil {
		return nil, err
	}
	return &review, nil
}

// ListByChallengeID 查询题目审核记录列表。
func (r *challengeReviewRepository) ListByChallengeID(ctx context.Context, challengeID int64, page, pageSize int) ([]*entity.ChallengeReview, int64, error) {
	query := r.db.WithContext(ctx).Model(&entity.ChallengeReview{}).
		Where("challenge_id = ?", challengeID)
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	page, pageSize = pagination.NormalizeValues(page, pageSize)
	query = query.Order("created_at desc").Offset(pagination.Offset(page, pageSize)).Limit(pageSize)
	var reviews []*entity.ChallengeReview
	if err := query.Find(&reviews).Error; err != nil {
		return nil, 0, err
	}
	return reviews, total, nil
}

// GetLatestByChallengeID 获取题目最新审核记录。
func (r *challengeReviewRepository) GetLatestByChallengeID(ctx context.Context, challengeID int64) (*entity.ChallengeReview, error) {
	var review entity.ChallengeReview
	err := r.db.WithContext(ctx).
		Where("challenge_id = ?", challengeID).
		Order("created_at desc").
		First(&review).Error
	if err != nil {
		return nil, err
	}
	return &review, nil
}

// ChallengeVerificationRepository 题目预验证记录数据访问接口。
type ChallengeVerificationRepository interface {
	Create(ctx context.Context, verification *entity.ChallengeVerification) error
	GetByID(ctx context.Context, id int64) (*entity.ChallengeVerification, error)
	GetLatestByChallengeID(ctx context.Context, challengeID int64) (*entity.ChallengeVerification, error)
	GetRunningByChallengeID(ctx context.Context, challengeID int64) (*entity.ChallengeVerification, error)
	UpdateFields(ctx context.Context, id int64, fields map[string]interface{}) error
	Complete(ctx context.Context, id int64, status int16, errorMessage *string) error
	ListByChallengeID(ctx context.Context, challengeID int64, page, pageSize int) ([]*entity.ChallengeVerification, int64, error)
	ListTimeoutRunning(ctx context.Context, startedBefore time.Time) ([]*entity.ChallengeVerification, error)
}

type challengeVerificationRepository struct {
	db *gorm.DB
}

// NewChallengeVerificationRepository 创建题目预验证记录数据访问实例。
func NewChallengeVerificationRepository(db *gorm.DB) ChallengeVerificationRepository {
	return &challengeVerificationRepository{db: db}
}

// Create 创建题目预验证记录。
func (r *challengeVerificationRepository) Create(ctx context.Context, verification *entity.ChallengeVerification) error {
	if verification.ID == 0 {
		verification.ID = snowflake.Generate()
	}
	return r.db.WithContext(ctx).Create(verification).Error
}

// GetByID 根据 ID 获取预验证记录。
func (r *challengeVerificationRepository) GetByID(ctx context.Context, id int64) (*entity.ChallengeVerification, error) {
	var verification entity.ChallengeVerification
	err := r.db.WithContext(ctx).First(&verification, id).Error
	if err != nil {
		return nil, err
	}
	return &verification, nil
}

// GetLatestByChallengeID 获取题目最新预验证记录。
func (r *challengeVerificationRepository) GetLatestByChallengeID(ctx context.Context, challengeID int64) (*entity.ChallengeVerification, error) {
	var verification entity.ChallengeVerification
	err := r.db.WithContext(ctx).
		Where("challenge_id = ?", challengeID).
		Order("created_at desc").
		First(&verification).Error
	if err != nil {
		return nil, err
	}
	return &verification, nil
}

// GetRunningByChallengeID 获取题目正在进行中的预验证记录。
func (r *challengeVerificationRepository) GetRunningByChallengeID(ctx context.Context, challengeID int64) (*entity.ChallengeVerification, error) {
	var verification entity.ChallengeVerification
	err := r.db.WithContext(ctx).
		Where("challenge_id = ? AND status = ?", challengeID, enum.VerificationStatusRunning).
		Order("created_at desc").
		First(&verification).Error
	if err != nil {
		return nil, err
	}
	return &verification, nil
}

// UpdateFields 更新预验证记录指定字段。
func (r *challengeVerificationRepository) UpdateFields(ctx context.Context, id int64, fields map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&entity.ChallengeVerification{}).
		Where("id = ?", id).
		Updates(fields).Error
}

// Complete 完成预验证并记录完成时间。
func (r *challengeVerificationRepository) Complete(ctx context.Context, id int64, status int16, errorMessage *string) error {
	return r.db.WithContext(ctx).Model(&entity.ChallengeVerification{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":        status,
			"error_message": errorMessage,
			"completed_at":  time.Now(),
		}).Error
}

// ListByChallengeID 查询题目预验证记录列表。
func (r *challengeVerificationRepository) ListByChallengeID(ctx context.Context, challengeID int64, page, pageSize int) ([]*entity.ChallengeVerification, int64, error) {
	query := r.db.WithContext(ctx).Model(&entity.ChallengeVerification{}).
		Where("challenge_id = ?", challengeID)
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	page, pageSize = pagination.NormalizeValues(page, pageSize)
	query = query.Order("created_at desc").Offset(pagination.Offset(page, pageSize)).Limit(pageSize)
	var verifications []*entity.ChallengeVerification
	if err := query.Find(&verifications).Error; err != nil {
		return nil, 0, err
	}
	return verifications, total, nil
}

// ListTimeoutRunning 查询超时未完成的预验证记录，供临时环境清理任务使用。
func (r *challengeVerificationRepository) ListTimeoutRunning(ctx context.Context, startedBefore time.Time) ([]*entity.ChallengeVerification, error) {
	var verifications []*entity.ChallengeVerification
	err := r.db.WithContext(ctx).
		Where("status = ? AND started_at < ?", enum.VerificationStatusRunning, startedBefore).
		Find(&verifications).Error
	return verifications, err
}

// CompetitionChallengeRepository 竞赛题目关联数据访问接口。
type CompetitionChallengeRepository interface {
	Create(ctx context.Context, item *entity.CompetitionChallenge) error
	BatchCreate(ctx context.Context, items []*entity.CompetitionChallenge) error
	GetByID(ctx context.Context, id int64) (*entity.CompetitionChallenge, error)
	GetByCompetitionAndChallenge(ctx context.Context, competitionID, challengeID int64) (*entity.CompetitionChallenge, error)
	Delete(ctx context.Context, id int64) error
	DeleteByCompetitionID(ctx context.Context, competitionID int64) error
	ListByCompetitionID(ctx context.Context, competitionID int64) ([]*entity.CompetitionChallenge, error)
	ListByCompetitionIDs(ctx context.Context, competitionIDs []int64) ([]*entity.CompetitionChallenge, error)
	CountByCompetitionID(ctx context.Context, competitionID int64) (int64, error)
	BatchUpdateSort(ctx context.Context, items []CompetitionChallengeSortItem) error
	UpdateScoreFields(ctx context.Context, id int64, fields map[string]interface{}) error
	MarkFirstBlood(ctx context.Context, id, teamID int64, firstBloodAt time.Time) error
	IncrementSolveCount(ctx context.Context, id int64, newScore int) error
}

// CompetitionChallengeSortItem 竞赛题目排序更新项。
type CompetitionChallengeSortItem struct {
	ID        int64
	SortOrder int
}

type competitionChallengeRepository struct {
	db *gorm.DB
}

// NewCompetitionChallengeRepository 创建竞赛题目关联数据访问实例。
func NewCompetitionChallengeRepository(db *gorm.DB) CompetitionChallengeRepository {
	return &competitionChallengeRepository{db: db}
}

// Create 创建竞赛题目关联。
func (r *competitionChallengeRepository) Create(ctx context.Context, item *entity.CompetitionChallenge) error {
	if item.ID == 0 {
		item.ID = snowflake.Generate()
	}
	return r.db.WithContext(ctx).Create(item).Error
}

// BatchCreate 批量创建竞赛题目关联。
func (r *competitionChallengeRepository) BatchCreate(ctx context.Context, items []*entity.CompetitionChallenge) error {
	if len(items) == 0 {
		return nil
	}
	for i := range items {
		if items[i].ID == 0 {
			items[i].ID = snowflake.Generate()
		}
	}
	return r.db.WithContext(ctx).CreateInBatches(items, 50).Error
}

// GetByID 根据 ID 获取竞赛题目关联。
func (r *competitionChallengeRepository) GetByID(ctx context.Context, id int64) (*entity.CompetitionChallenge, error) {
	var item entity.CompetitionChallenge
	err := r.db.WithContext(ctx).First(&item, id).Error
	if err != nil {
		return nil, err
	}
	return &item, nil
}

// GetByCompetitionAndChallenge 根据竞赛和题目获取关联记录。
func (r *competitionChallengeRepository) GetByCompetitionAndChallenge(ctx context.Context, competitionID, challengeID int64) (*entity.CompetitionChallenge, error) {
	var item entity.CompetitionChallenge
	err := r.db.WithContext(ctx).
		Where("competition_id = ? AND challenge_id = ?", competitionID, challengeID).
		First(&item).Error
	if err != nil {
		return nil, err
	}
	return &item, nil
}

// Delete 删除竞赛题目关联。
func (r *competitionChallengeRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&entity.CompetitionChallenge{}, id).Error
}

// DeleteByCompetitionID 删除竞赛下全部题目关联。
func (r *competitionChallengeRepository) DeleteByCompetitionID(ctx context.Context, competitionID int64) error {
	return r.db.WithContext(ctx).
		Where("competition_id = ?", competitionID).
		Delete(&entity.CompetitionChallenge{}).Error
}

// ListByCompetitionID 查询竞赛题目关联列表。
func (r *competitionChallengeRepository) ListByCompetitionID(ctx context.Context, competitionID int64) ([]*entity.CompetitionChallenge, error) {
	var items []*entity.CompetitionChallenge
	err := r.db.WithContext(ctx).
		Where("competition_id = ?", competitionID).
		Order("sort_order asc, created_at asc").
		Find(&items).Error
	return items, err
}

// ListByCompetitionIDs 批量查询多个竞赛的题目关联。
func (r *competitionChallengeRepository) ListByCompetitionIDs(ctx context.Context, competitionIDs []int64) ([]*entity.CompetitionChallenge, error) {
	if len(competitionIDs) == 0 {
		return []*entity.CompetitionChallenge{}, nil
	}
	var items []*entity.CompetitionChallenge
	err := r.db.WithContext(ctx).
		Where("competition_id IN ?", competitionIDs).
		Order("competition_id asc, sort_order asc, created_at asc").
		Find(&items).Error
	return items, err
}

// CountByCompetitionID 统计竞赛题目数量。
func (r *competitionChallengeRepository) CountByCompetitionID(ctx context.Context, competitionID int64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&entity.CompetitionChallenge{}).
		Where("competition_id = ?", competitionID).
		Count(&count).Error
	return count, err
}

// BatchUpdateSort 批量更新竞赛题目排序。
func (r *competitionChallengeRepository) BatchUpdateSort(ctx context.Context, items []CompetitionChallengeSortItem) error {
	if len(items) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, item := range items {
			if err := tx.Model(&entity.CompetitionChallenge{}).
				Where("id = ?", item.ID).
				Update("sort_order", item.SortOrder).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// UpdateScoreFields 更新竞赛题目计分相关字段。
func (r *competitionChallengeRepository) UpdateScoreFields(ctx context.Context, id int64, fields map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&entity.CompetitionChallenge{}).
		Where("id = ?", id).
		Updates(fields).Error
}

// MarkFirstBlood 标记题目 First Blood 队伍。
func (r *competitionChallengeRepository) MarkFirstBlood(ctx context.Context, id, teamID int64, firstBloodAt time.Time) error {
	return r.db.WithContext(ctx).Model(&entity.CompetitionChallenge{}).
		Where("id = ? AND first_blood_team_id IS NULL", id).
		Updates(map[string]interface{}{
			"first_blood_team_id": teamID,
			"first_blood_at":      firstBloodAt,
			"updated_at":          time.Now(),
		}).Error
}

// IncrementSolveCount 增加题目解出次数并更新当前动态分值。
func (r *competitionChallengeRepository) IncrementSolveCount(ctx context.Context, id int64, newScore int) error {
	return r.db.WithContext(ctx).Model(&entity.CompetitionChallenge{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"solve_count":   gorm.Expr("solve_count + ?", 1),
			"current_score": newScore,
			"updated_at":    time.Now(),
		}).Error
}
