// course_repo.go
// 模块03 — 课程与教学：课程数据访问层
// 负责课程主表的 CRUD 操作
// 对照 docs/modules/03-课程与教学/02-数据库设计.md

package courserepo

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/model/enum"
	"github.com/lenschain/backend/internal/pkg/database"
	"github.com/lenschain/backend/internal/pkg/snowflake"
)

// CourseRepository 课程数据访问接口
type CourseRepository interface {
	Create(ctx context.Context, course *entity.Course) error
	GetByID(ctx context.Context, id int64) (*entity.Course, error)
	GetByInviteCode(ctx context.Context, code string) (*entity.Course, error)
	UpdateFields(ctx context.Context, id int64, fields map[string]interface{}) error
	SoftDelete(ctx context.Context, id int64) error
	List(ctx context.Context, params *CourseListParams) ([]*entity.Course, int64, error)
	ListShared(ctx context.Context, params *SharedCourseListParams) ([]*entity.Course, int64, error)
	ListByStudentID(ctx context.Context, studentID int64, params *StudentCourseListParams) ([]*entity.Course, int64, error)
	CountStudents(ctx context.Context, courseID int64) (int, error)
	ListPublishedToActivate(ctx context.Context, now time.Time) ([]*entity.Course, error)
	ListActiveToEnd(ctx context.Context, now time.Time) ([]*entity.Course, error)
	UpdateStatus(ctx context.Context, id int64, status int) error
}

// CourseListParams 课程列表查询参数（教师视角）
type CourseListParams struct {
	SchoolID   int64
	TeacherID  int64
	Keyword    string
	Status     int
	CourseType int
	SortBy     string
	SortOrder  string
	Page       int
	PageSize   int
}

// SharedCourseListParams 共享课程列表查询参数
type SharedCourseListParams struct {
	Keyword    string
	CourseType int
	Difficulty int
	Topic      string
	Page       int
	PageSize   int
}

// StudentCourseListParams 学生课程列表查询参数
type StudentCourseListParams struct {
	Status   int
	Page     int
	PageSize int
}

// courseRepository 课程数据访问实现
type courseRepository struct {
	db *gorm.DB
}

// NewCourseRepository 创建课程数据访问实例
func NewCourseRepository(db *gorm.DB) CourseRepository {
	return &courseRepository{db: db}
}

// Create 创建课程
func (r *courseRepository) Create(ctx context.Context, course *entity.Course) error {
	if course.ID == 0 {
		course.ID = snowflake.Generate()
	}
	return r.db.WithContext(ctx).Create(course).Error
}

// GetByID 根据ID获取课程
func (r *courseRepository) GetByID(ctx context.Context, id int64) (*entity.Course, error) {
	var course entity.Course
	err := r.db.WithContext(ctx).First(&course, id).Error
	if err != nil {
		return nil, err
	}
	return &course, nil
}

// GetByInviteCode 根据邀请码获取课程
func (r *courseRepository) GetByInviteCode(ctx context.Context, code string) (*entity.Course, error) {
	var course entity.Course
	err := r.db.WithContext(ctx).
		Where("invite_code = ? AND status IN ?", code,
			[]int{enum.CourseStatusPublished, enum.CourseStatusActive}).
		First(&course).Error
	if err != nil {
		return nil, err
	}
	return &course, nil
}

// UpdateFields 更新课程指定字段
func (r *courseRepository) UpdateFields(ctx context.Context, id int64, fields map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&entity.Course{}).Where("id = ?", id).Updates(fields).Error
}

// SoftDelete 软删除课程
func (r *courseRepository) SoftDelete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&entity.Course{}, id).Error
}

// List 教师课程列表查询
func (r *courseRepository) List(ctx context.Context, params *CourseListParams) ([]*entity.Course, int64, error) {
	query := r.db.WithContext(ctx).Model(&entity.Course{})

	// 多租户隔离
	if params.SchoolID > 0 {
		query = query.Scopes(database.WithSchoolID(params.SchoolID))
	}

	// 教师筛选
	if params.TeacherID > 0 {
		query = query.Where("teacher_id = ?", params.TeacherID)
	}

	// 关键字搜索
	if params.Keyword != "" {
		query = query.Scopes(database.WithKeywordSearch(params.Keyword, "title", "topic"))
	}

	// 状态筛选
	if params.Status > 0 {
		query = query.Scopes(database.WithStatus(params.Status))
	}

	// 课程类型筛选
	if params.CourseType > 0 {
		query = query.Where("course_type = ?", params.CourseType)
	}

	// 统计总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 排序
	sortField := "created_at"
	sortOrder := "desc"
	allowedSortFields := map[string]string{
		"created_at": "created_at",
		"title":      "title",
		"status":     "status",
		"start_at":   "start_at",
	}
	if field, ok := allowedSortFields[params.SortBy]; ok {
		sortField = field
	}
	if params.SortOrder == "asc" {
		sortOrder = "asc"
	}
	query = query.Order(fmt.Sprintf("%s %s", sortField, sortOrder))

	// 分页
	page, pageSize := normalizePagination(params.Page, params.PageSize)
	query = query.Offset((page - 1) * pageSize).Limit(pageSize)

	var courses []*entity.Course
	if err := query.Find(&courses).Error; err != nil {
		return nil, 0, err
	}
	return courses, total, nil
}

// ListShared 共享课程列表查询
func (r *courseRepository) ListShared(ctx context.Context, params *SharedCourseListParams) ([]*entity.Course, int64, error) {
	query := r.db.WithContext(ctx).Model(&entity.Course{}).
		Where("is_shared = ?", true).
		Where("status IN ?", []int{enum.CourseStatusPublished, enum.CourseStatusActive, enum.CourseStatusEnded})

	if params.Keyword != "" {
		query = query.Scopes(database.WithKeywordSearch(params.Keyword, "title", "topic"))
	}
	if params.CourseType > 0 {
		query = query.Where("course_type = ?", params.CourseType)
	}
	if params.Difficulty > 0 {
		query = query.Where("difficulty = ?", params.Difficulty)
	}
	if params.Topic != "" {
		query = query.Where("topic = ?", params.Topic)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	page, pageSize := normalizePagination(params.Page, params.PageSize)
	query = query.Order("created_at desc").Offset((page - 1) * pageSize).Limit(pageSize)

	var courses []*entity.Course
	if err := query.Find(&courses).Error; err != nil {
		return nil, 0, err
	}
	return courses, total, nil
}

// ListByStudentID 学生已选课程列表
func (r *courseRepository) ListByStudentID(ctx context.Context, studentID int64, params *StudentCourseListParams) ([]*entity.Course, int64, error) {
	subQuery := r.db.Model(&entity.CourseEnrollment{}).
		Select("course_id").
		Where("student_id = ? AND removed_at IS NULL", studentID)

	query := r.db.WithContext(ctx).Model(&entity.Course{}).
		Where("id IN (?)", subQuery)

	if params.Status > 0 {
		query = query.Scopes(database.WithStatus(params.Status))
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	page, pageSize := normalizePagination(params.Page, params.PageSize)
	query = query.Order("created_at desc").Offset((page - 1) * pageSize).Limit(pageSize)

	var courses []*entity.Course
	if err := query.Find(&courses).Error; err != nil {
		return nil, 0, err
	}
	return courses, total, nil
}

// CountStudents 统计课程学生数
func (r *courseRepository) CountStudents(ctx context.Context, courseID int64) (int, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&entity.CourseEnrollment{}).
		Where("course_id = ? AND removed_at IS NULL", courseID).
		Count(&count).Error
	return int(count), err
}

// ListPublishedToActivate 查询需要自动激活的课程（已发布且到达开始时间）
func (r *courseRepository) ListPublishedToActivate(ctx context.Context, now time.Time) ([]*entity.Course, error) {
	var courses []*entity.Course
	err := r.db.WithContext(ctx).
		Where("status = ? AND start_at IS NOT NULL AND start_at <= ?", enum.CourseStatusPublished, now).
		Find(&courses).Error
	return courses, err
}

// ListActiveToEnd 查询需要自动结束的课程（进行中且到达结束时间）
func (r *courseRepository) ListActiveToEnd(ctx context.Context, now time.Time) ([]*entity.Course, error) {
	var courses []*entity.Course
	err := r.db.WithContext(ctx).
		Where("status = ? AND end_at IS NOT NULL AND end_at <= ?", enum.CourseStatusActive, now).
		Find(&courses).Error
	return courses, err
}

// UpdateStatus 更新课程状态
func (r *courseRepository) UpdateStatus(ctx context.Context, id int64, status int) error {
	return r.db.WithContext(ctx).Model(&entity.Course{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":     status,
			"updated_at": time.Now(),
		}).Error
}

// normalizePagination 规范化分页参数
func normalizePagination(page, pageSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return page, pageSize
}
