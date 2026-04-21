// schedule_repo.go
// 模块03 — 课程与教学：课程表数据访问层
// 对照 docs/modules/03-课程与教学/02-数据库设计.md

package courserepo

import (
	"context"

	"gorm.io/gorm"

	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/pkg/database"
	"github.com/lenschain/backend/internal/pkg/snowflake"
)

// ScheduleRepository 课表数据访问接口
type ScheduleRepository interface {
	ReplaceByCourseID(ctx context.Context, courseID int64, schedules []*entity.CourseSchedule) error
	ListByCourseID(ctx context.Context, courseID int64) ([]*entity.CourseSchedule, error)
	ListByStudentCourses(ctx context.Context, courseIDs []int64) ([]*entity.CourseSchedule, error)
}

type scheduleRepository struct {
	db *gorm.DB
}

// NewScheduleRepository 创建课表数据访问实例
func NewScheduleRepository(db *gorm.DB) ScheduleRepository {
	return &scheduleRepository{db: db}
}

// ReplaceByCourseID 整组替换课程课表
func (r *scheduleRepository) ReplaceByCourseID(ctx context.Context, courseID int64, schedules []*entity.CourseSchedule) error {
	return database.TransactionWithDB(ctx, r.db, func(tx *gorm.DB) error {
		// 课程表采用整组替换，删除旧记录和创建新记录必须保持原子性。
		if err := tx.Where("course_id = ?", courseID).Delete(&entity.CourseSchedule{}).Error; err != nil {
			return err
		}
		for i := range schedules {
			if schedules[i].ID == 0 {
				schedules[i].ID = snowflake.Generate()
			}
			schedules[i].CourseID = courseID
		}
		if len(schedules) > 0 {
			return tx.CreateInBatches(schedules, courseScheduleBatchSize).Error
		}
		return nil
	})
}

// ListByCourseID 获取课程课表
func (r *scheduleRepository) ListByCourseID(ctx context.Context, courseID int64) ([]*entity.CourseSchedule, error) {
	var schedules []*entity.CourseSchedule
	err := r.db.WithContext(ctx).
		Where("course_id = ?", courseID).
		Order("day_of_week asc, start_time asc").
		Find(&schedules).Error
	return schedules, err
}

// ListByStudentCourses 获取学生已选课程的课表
func (r *scheduleRepository) ListByStudentCourses(ctx context.Context, courseIDs []int64) ([]*entity.CourseSchedule, error) {
	if len(courseIDs) == 0 {
		return []*entity.CourseSchedule{}, nil
	}
	var schedules []*entity.CourseSchedule
	err := r.db.WithContext(ctx).
		Where("course_id IN ?", courseIDs).
		Order("day_of_week asc, start_time asc").
		Find(&schedules).Error
	return schedules, err
}
