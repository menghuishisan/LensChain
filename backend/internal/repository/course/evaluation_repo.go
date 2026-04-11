// evaluation_repo.go
// 模块03 — 课程与教学：评价、成绩配置、课表、课程实验数据访问层
// 从 discussion_repo.go 拆分而来，保持单文件 ≤ 500 行
// 对照 docs/modules/03-课程与教学/02-数据库设计.md

package courserepo

import (
	"context"

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
	GetDistribution(ctx context.Context, courseID int64) ([5]int, error)
}

// GradeConfigRepository 成绩配置数据访问接口
type GradeConfigRepository interface {
	Upsert(ctx context.Context, config *entity.CourseGradeConfig) error
	GetByCourseID(ctx context.Context, courseID int64) (*entity.CourseGradeConfig, error)
}

// ScheduleRepository 课表数据访问接口
type ScheduleRepository interface {
	ReplaceByCourseID(ctx context.Context, courseID int64, schedules []*entity.CourseSchedule) error
	ListByCourseID(ctx context.Context, courseID int64) ([]*entity.CourseSchedule, error)
	ListByStudentCourses(ctx context.Context, courseIDs []int64) ([]*entity.CourseSchedule, error)
}

// CourseExperimentRepository 课程实验关联数据访问接口
type CourseExperimentRepository interface {
	Create(ctx context.Context, exp *entity.CourseExperiment) error
	Delete(ctx context.Context, id int64) error
	ListByCourseID(ctx context.Context, courseID int64) ([]*entity.CourseExperiment, error)
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

	page, pageSize = normalizePagination(page, pageSize)
	query = query.Order("created_at desc").
		Offset((page - 1) * pageSize).Limit(pageSize)

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

func (r *evaluationRepository) GetDistribution(ctx context.Context, courseID int64) ([5]int, error) {
	var dist [5]int
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
		if r.Rating >= 1 && r.Rating <= 5 {
			dist[r.Rating-1] = r.Count
		}
	}
	return dist, nil
}

// ========== GradeConfig 实现 ==========

type gradeConfigRepository struct {
	db *gorm.DB
}

// NewGradeConfigRepository 创建成绩配置数据访问实例
func NewGradeConfigRepository(db *gorm.DB) GradeConfigRepository {
	return &gradeConfigRepository{db: db}
}

func (r *gradeConfigRepository) Upsert(ctx context.Context, config *entity.CourseGradeConfig) error {
	existing, err := r.GetByCourseID(ctx, config.CourseID)
	if err != nil {
		if config.ID == 0 {
			config.ID = snowflake.Generate()
		}
		return r.db.WithContext(ctx).Create(config).Error
	}
	return r.db.WithContext(ctx).Model(&entity.CourseGradeConfig{}).
		Where("id = ?", existing.ID).
		Updates(map[string]interface{}{
			"config":     config.Config,
			"updated_at": gorm.Expr("now()"),
		}).Error
}

func (r *gradeConfigRepository) GetByCourseID(ctx context.Context, courseID int64) (*entity.CourseGradeConfig, error) {
	var config entity.CourseGradeConfig
	err := r.db.WithContext(ctx).Where("course_id = ?", courseID).First(&config).Error
	if err != nil {
		return nil, err
	}
	return &config, nil
}

// ========== Schedule 实现 ==========

type scheduleRepository struct {
	db *gorm.DB
}

// NewScheduleRepository 创建课表数据访问实例
func NewScheduleRepository(db *gorm.DB) ScheduleRepository {
	return &scheduleRepository{db: db}
}

func (r *scheduleRepository) ReplaceByCourseID(ctx context.Context, courseID int64, schedules []*entity.CourseSchedule) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 先删除旧的
		if err := tx.Where("course_id = ?", courseID).Delete(&entity.CourseSchedule{}).Error; err != nil {
			return err
		}
		// 再创建新的
		for i := range schedules {
			if schedules[i].ID == 0 {
				schedules[i].ID = snowflake.Generate()
			}
			schedules[i].CourseID = courseID
		}
		if len(schedules) > 0 {
			return tx.CreateInBatches(schedules, 20).Error
		}
		return nil
	})
}

func (r *scheduleRepository) ListByCourseID(ctx context.Context, courseID int64) ([]*entity.CourseSchedule, error) {
	var schedules []*entity.CourseSchedule
	err := r.db.WithContext(ctx).
		Where("course_id = ?", courseID).
		Order("day_of_week asc, start_time asc").
		Find(&schedules).Error
	return schedules, err
}

func (r *scheduleRepository) ListByStudentCourses(ctx context.Context, courseIDs []int64) ([]*entity.CourseSchedule, error) {
	var schedules []*entity.CourseSchedule
	err := r.db.WithContext(ctx).
		Where("course_id IN ?", courseIDs).
		Order("day_of_week asc, start_time asc").
		Find(&schedules).Error
	return schedules, err
}

// ========== CourseExperiment 实现 ==========

type courseExperimentRepository struct {
	db *gorm.DB
}

// NewCourseExperimentRepository 创建课程实验关联数据访问实例
func NewCourseExperimentRepository(db *gorm.DB) CourseExperimentRepository {
	return &courseExperimentRepository{db: db}
}

func (r *courseExperimentRepository) Create(ctx context.Context, exp *entity.CourseExperiment) error {
	if exp.ID == 0 {
		exp.ID = snowflake.Generate()
	}
	return r.db.WithContext(ctx).Create(exp).Error
}

func (r *courseExperimentRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&entity.CourseExperiment{}, id).Error
}

func (r *courseExperimentRepository) ListByCourseID(ctx context.Context, courseID int64) ([]*entity.CourseExperiment, error) {
	var exps []*entity.CourseExperiment
	err := r.db.WithContext(ctx).
		Where("course_id = ?", courseID).
		Order("sort_order asc, created_at asc").
		Find(&exps).Error
	return exps, err
}
