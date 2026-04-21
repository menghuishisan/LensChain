// course_source_repo.go
// 模块06 — 评测与成绩：课程成绩来源只读数据访问层。
// 负责读取模块03课程、选课、成绩配置和作业提交数据，为成绩审核汇总提供数据库来源。

package graderepo

import (
	"context"

	"gorm.io/gorm"

	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/model/enum"
)

// CourseGradeSourceRepository 课程成绩来源只读数据访问接口。
type CourseGradeSourceRepository interface {
	GetCourse(ctx context.Context, courseID int64) (*entity.Course, error)
	ListCoursesBySemester(ctx context.Context, schoolID, semesterID int64) ([]*entity.Course, error)
	ListEnrolledStudentIDs(ctx context.Context, courseID int64) ([]int64, error)
	GetGradeConfig(ctx context.Context, courseID int64) (*entity.CourseGradeConfig, error)
	ListLatestGradedSubmissions(ctx context.Context, courseID int64) ([]*CourseSubmissionScore, error)
	ListGradeOverrides(ctx context.Context, courseID int64) ([]*entity.CourseGradeOverride, error)
	CountReviewedCourses(ctx context.Context, schoolID, semesterID int64) (int64, error)
}

// CourseSubmissionScore 课程作业提交成绩。
type CourseSubmissionScore struct {
	AssignmentID int64   `gorm:"column:assignment_id"`
	StudentID    int64   `gorm:"column:student_id"`
	TotalScore   float64 `gorm:"column:total_score"`
}

type courseGradeSourceRepository struct {
	db *gorm.DB
}

// NewCourseGradeSourceRepository 创建课程成绩来源只读数据访问实例。
func NewCourseGradeSourceRepository(db *gorm.DB) CourseGradeSourceRepository {
	return &courseGradeSourceRepository{db: db}
}

// GetCourse 获取课程基础信息。
func (r *courseGradeSourceRepository) GetCourse(ctx context.Context, courseID int64) (*entity.Course, error) {
	var course entity.Course
	err := r.db.WithContext(ctx).First(&course, courseID).Error
	if err != nil {
		return nil, err
	}
	return &course, nil
}

// ListCoursesBySemester 查询学校指定学期课程。
func (r *courseGradeSourceRepository) ListCoursesBySemester(ctx context.Context, schoolID, semesterID int64) ([]*entity.Course, error) {
	var courses []*entity.Course
	err := r.db.WithContext(ctx).
		Where("school_id = ? AND semester_id = ?", schoolID, semesterID).
		Order("id asc").
		Find(&courses).Error
	return courses, err
}

// ListEnrolledStudentIDs 查询课程当前在课学生 ID。
func (r *courseGradeSourceRepository) ListEnrolledStudentIDs(ctx context.Context, courseID int64) ([]int64, error) {
	var studentIDs []int64
	err := r.db.WithContext(ctx).Model(&entity.CourseEnrollment{}).
		Where("course_id = ? AND removed_at IS NULL", courseID).
		Order("student_id asc").
		Pluck("student_id", &studentIDs).Error
	return studentIDs, err
}

// GetGradeConfig 获取课程成绩权重配置。
func (r *courseGradeSourceRepository) GetGradeConfig(ctx context.Context, courseID int64) (*entity.CourseGradeConfig, error) {
	var config entity.CourseGradeConfig
	err := r.db.WithContext(ctx).Where("course_id = ?", courseID).First(&config).Error
	if err != nil {
		return nil, err
	}
	return &config, nil
}

// ListLatestGradedSubmissions 查询课程内学生每个作业的最新已批改提交成绩。
func (r *courseGradeSourceRepository) ListLatestGradedSubmissions(ctx context.Context, courseID int64) ([]*CourseSubmissionScore, error) {
	var scores []*CourseSubmissionScore
	err := r.db.WithContext(ctx).Table("assignment_submissions AS s").
		Select("s.assignment_id, s.student_id, COALESCE(s.total_score, 0) AS total_score").
		Joins("JOIN assignments a ON a.id = s.assignment_id").
		Joins(`
			JOIN (
				SELECT assignment_id, student_id, MAX(submission_no) AS submission_no
				FROM assignment_submissions
				WHERE status = ?
				GROUP BY assignment_id, student_id
			) latest ON latest.assignment_id = s.assignment_id
				AND latest.student_id = s.student_id
				AND latest.submission_no = s.submission_no
		`, enum.SubmissionStatusGraded).
		Where("a.course_id = ? AND s.status = ?", courseID, enum.SubmissionStatusGraded).
		Order("s.student_id asc, s.assignment_id asc").
		Find(&scores).Error
	return scores, err
}

// ListGradeOverrides 查询课程成绩手动调整记录。
func (r *courseGradeSourceRepository) ListGradeOverrides(ctx context.Context, courseID int64) ([]*entity.CourseGradeOverride, error) {
	var overrides []*entity.CourseGradeOverride
	err := r.db.WithContext(ctx).
		Where("course_id = ?", courseID).
		Order("student_id asc").
		Find(&overrides).Error
	return overrides, err
}

// CountReviewedCourses 统计学校学期内已审核通过课程数。
func (r *courseGradeSourceRepository) CountReviewedCourses(ctx context.Context, schoolID, semesterID int64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&entity.GradeReview{}).
		Where("school_id = ? AND semester_id = ? AND status = ?", schoolID, semesterID, enum.GradeReviewStatusApproved).
		Count(&count).Error
	return count, err
}
