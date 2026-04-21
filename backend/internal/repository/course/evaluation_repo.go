// evaluation_repo.go
// 模块03 — 课程与教学：课程评价数据访问层
// 对照 docs/modules/03-课程与教学/02-数据库设计.md

package courserepo

import (
	"context"
	"github.com/lenschain/backend/internal/pkg/pagination"

	"gorm.io/gorm"

	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/pkg/snowflake"
)

// EvaluationRepository 评价数据访问接口
type EvaluationRepository interface {
	Create(ctx context.Context, evaluation *entity.CourseEvaluation) error
	GetByID(ctx context.Context, id int64) (*entity.CourseEvaluation, error)
	GetByStudentAndCourse(ctx context.Context, studentID, courseID int64) (*entity.CourseEvaluation, error)
	UpdateFields(ctx context.Context, id int64, fields map[string]interface{}) error
	List(ctx context.Context, courseID int64, page, pageSize int) ([]*entity.CourseEvaluation, int64, error)
	GetAvgRating(ctx context.Context, courseID int64) (float64, error)
	GetDistribution(ctx context.Context, courseID int64) ([courseRatingLevels]int, error)
}

// ========== Evaluation 实现 ==========

type evaluationRepository struct {
	db *gorm.DB
}

// NewEvaluationRepository 创建评价数据访问实例
func NewEvaluationRepository(db *gorm.DB) EvaluationRepository {
	return &evaluationRepository{db: db}
}

func (r *evaluationRepository) Create(ctx context.Context, evaluation *entity.CourseEvaluation) error {
	if evaluation.ID == 0 {
		evaluation.ID = snowflake.Generate()
	}
	return r.db.WithContext(ctx).Create(evaluation).Error
}

func (r *evaluationRepository) GetByID(ctx context.Context, id int64) (*entity.CourseEvaluation, error) {
	var e entity.CourseEvaluation
	err := r.db.WithContext(ctx).First(&e, id).Error
	if err != nil {
		return nil, err
	}
	return &e, nil
}

func (r *evaluationRepository) GetByStudentAndCourse(ctx context.Context, studentID, courseID int64) (*entity.CourseEvaluation, error) {
	var e entity.CourseEvaluation
	err := r.db.WithContext(ctx).
		Where("student_id = ? AND course_id = ?", studentID, courseID).
		First(&e).Error
	if err != nil {
		return nil, err
	}
	return &e, nil
}

func (r *evaluationRepository) UpdateFields(ctx context.Context, id int64, fields map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&entity.CourseEvaluation{}).Where("id = ?", id).Updates(fields).Error
}

func (r *evaluationRepository) List(ctx context.Context, courseID int64, page, pageSize int) ([]*entity.CourseEvaluation, int64, error) {
	query := r.db.WithContext(ctx).Model(&entity.CourseEvaluation{}).
		Where("course_id = ?", courseID)

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	page, pageSize = pagination.NormalizeValues(page, pageSize)
	query = query.Order("created_at desc").
		Offset(pagination.Offset(page, pageSize)).Limit(pageSize)

	var evaluations []*entity.CourseEvaluation
	if err := query.Find(&evaluations).Error; err != nil {
		return nil, 0, err
	}
	return evaluations, total, nil
}

func (r *evaluationRepository) GetAvgRating(ctx context.Context, courseID int64) (float64, error) {
	var avg *float64
	err := r.db.WithContext(ctx).Model(&entity.CourseEvaluation{}).
		Where("course_id = ?", courseID).
		Select("COALESCE(AVG(rating), 0)").
		Scan(&avg).Error
	if err != nil || avg == nil {
		return 0, err
	}
	return *avg, nil
}

func (r *evaluationRepository) GetDistribution(ctx context.Context, courseID int64) ([courseRatingLevels]int, error) {
	var dist [courseRatingLevels]int
	type result struct {
		Rating int
		Count  int
	}
	var results []result
	err := r.db.WithContext(ctx).Model(&entity.CourseEvaluation{}).
		Where("course_id = ?", courseID).
		Select("rating, COUNT(*) as count").
		Group("rating").
		Scan(&results).Error
	if err != nil {
		return dist, err
	}
	for _, r := range results {
		if r.Rating >= courseRatingMin && r.Rating <= courseRatingMax {
			dist[r.Rating-courseRatingMin] = r.Count
		}
	}
	return dist, nil
}
