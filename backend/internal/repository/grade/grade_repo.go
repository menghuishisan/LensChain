// grade_repo.go
// 模块06 — 评测与成绩：学生学期成绩汇总与成绩分析数据访问层。
// 负责 student_semester_grades 的写入、查询、GPA 聚合，以及对课程、学校、用户表的只读统计。

package graderepo

import (
	"context"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/model/enum"
	"github.com/lenschain/backend/internal/pkg/database"
	"github.com/lenschain/backend/internal/pkg/pagination"
	"github.com/lenschain/backend/internal/pkg/snowflake"
)

// StudentSemesterGradeRepository 学生学期成绩汇总数据访问接口。
type StudentSemesterGradeRepository interface {
	Create(ctx context.Context, grade *entity.StudentSemesterGrade) error
	BatchUpsert(ctx context.Context, grades []*entity.StudentSemesterGrade) error
	GetByID(ctx context.Context, id int64) (*entity.StudentSemesterGrade, error)
	GetByStudentCourseSemester(ctx context.Context, studentID, courseID, semesterID int64) (*entity.StudentSemesterGrade, error)
	List(ctx context.Context, params *StudentGradeListParams) ([]*entity.StudentSemesterGrade, int64, error)
	ListByReview(ctx context.Context, reviewID int64) ([]*entity.StudentSemesterGrade, error)
	ListByStudent(ctx context.Context, schoolID, studentID int64, semesterIDs []int64) ([]*entity.StudentSemesterGrade, error)
	UpdateAfterAppeal(ctx context.Context, id int64, finalScore float64, gradeLevel string, gpaPoint float64) error
	DeleteByReview(ctx context.Context, reviewID int64) error
	CalculateSemesterGPA(ctx context.Context, schoolID, studentID, semesterID int64) (*GPAStats, error)
	CalculateCumulativeGPA(ctx context.Context, schoolID, studentID int64) (*GPAStats, error)
	CourseAnalytics(ctx context.Context, schoolID, courseID int64) (*CourseGradeAnalytics, error)
	CourseGradeDistribution(ctx context.Context, schoolID, courseID int64) ([]*GradeDistributionItem, error)
	CourseScoreDistribution(ctx context.Context, schoolID, courseID int64) ([]*ScoreDistributionItem, error)
	SchoolAnalytics(ctx context.Context, schoolID, semesterID int64) (*SchoolGradeAnalytics, error)
	SchoolFailRate(ctx context.Context, schoolID, semesterID int64) (float64, error)
	SchoolGPADistribution(ctx context.Context, schoolID, semesterID int64) ([]*GPARangeDistributionItem, error)
	SchoolCoursePerformance(ctx context.Context, schoolID, semesterID int64, ascending bool, limit int) ([]*CoursePerformanceItem, error)
	PlatformAnalytics(ctx context.Context, semesterID int64) (*PlatformGradeAnalytics, error)
	PlatformSchoolComparison(ctx context.Context, semesterID int64) ([]*SchoolComparisonItem, error)
	ListFailingCourses(ctx context.Context, schoolID, semesterID int64, maxGPAPoint float64) ([]*FailingCourseItem, error)
}

// StudentGradeListParams 学生成绩列表查询参数。
type StudentGradeListParams struct {
	SchoolID   int64
	StudentID  int64
	SemesterID int64
	CourseID   int64
	GradeLevel string
	IsAdjusted *bool
	SortBy     string
	SortOrder  string
	Page       int
	PageSize   int
}

// GPAStats GPA 聚合结果。
type GPAStats struct {
	TotalCredits       float64 `gorm:"column:total_credits"`
	TotalWeightedPoint float64 `gorm:"column:total_weighted_point"`
	GPA                float64 `gorm:"column:gpa"`
	CourseCount        int64   `gorm:"column:course_count"`
}

// CourseGradeAnalytics 课程成绩分析聚合结果。
type CourseGradeAnalytics struct {
	CourseID     int64   `gorm:"column:course_id"`
	StudentCount int64   `gorm:"column:student_count"`
	AverageScore float64 `gorm:"column:average_score"`
	MedianScore  float64 `gorm:"column:median_score"`
	HighestScore float64 `gorm:"column:highest_score"`
	LowestScore  float64 `gorm:"column:lowest_score"`
	AverageGPA   float64 `gorm:"column:average_gpa"`
	PassRate     float64 `gorm:"column:pass_rate"`
}

// SchoolGradeAnalytics 学校成绩分析聚合结果。
type SchoolGradeAnalytics struct {
	SchoolID      int64   `gorm:"column:school_id"`
	SemesterID    int64   `gorm:"column:semester_id"`
	StudentCount  int64   `gorm:"column:student_count"`
	CourseCount   int64   `gorm:"column:course_count"`
	AverageScore  float64 `gorm:"column:average_score"`
	AverageGPA    float64 `gorm:"column:average_gpa"`
	WarningCount  int64   `gorm:"column:warning_count"`
	AppealCount   int64   `gorm:"column:appeal_count"`
	ReviewedCount int64   `gorm:"column:reviewed_count"`
}

// PlatformGradeAnalytics 平台成绩总览聚合结果。
type PlatformGradeAnalytics struct {
	SchoolCount   int64   `gorm:"column:school_count"`
	StudentCount  int64   `gorm:"column:student_count"`
	CourseCount   int64   `gorm:"column:course_count"`
	AverageScore  float64 `gorm:"column:average_score"`
	AverageGPA    float64 `gorm:"column:average_gpa"`
	WarningCount  int64   `gorm:"column:warning_count"`
	ReviewedCount int64   `gorm:"column:reviewed_count"`
}

// GradeDistributionItem 等级分布项。
type GradeDistributionItem struct {
	GradeLevel string `gorm:"column:grade_level"`
	Count      int64  `gorm:"column:count"`
}

// ScoreDistributionItem 分数段分布项。
type ScoreDistributionItem struct {
	Range string `gorm:"column:range"`
	Count int64  `gorm:"column:count"`
}

// GPARangeDistributionItem GPA 区间分布项。
type GPARangeDistributionItem struct {
	Range string `gorm:"column:range"`
	Count int64  `gorm:"column:count"`
}

// CoursePerformanceItem 课程表现统计项。
type CoursePerformanceItem struct {
	CourseID     int64   `gorm:"column:course_id"`
	CourseName   string  `gorm:"column:course_name"`
	AverageScore float64 `gorm:"column:average_score"`
	PassRate     float64 `gorm:"column:pass_rate"`
	StudentCount int64   `gorm:"column:student_count"`
}

// SchoolComparisonItem 学校成绩对比项。
type SchoolComparisonItem struct {
	SchoolID     int64   `gorm:"column:school_id"`
	SchoolName   string  `gorm:"column:school_name"`
	StudentCount int64   `gorm:"column:student_count"`
	AverageGPA   float64 `gorm:"column:average_gpa"`
}

// FailingCourseItem 挂科课程统计项。
type FailingCourseItem struct {
	StudentID int64 `gorm:"column:student_id"`
	CourseID  int64 `gorm:"column:course_id"`
}

type studentSemesterGradeRepository struct {
	db *gorm.DB
}

// NewStudentSemesterGradeRepository 创建学生学期成绩汇总数据访问实例。
func NewStudentSemesterGradeRepository(db *gorm.DB) StudentSemesterGradeRepository {
	return &studentSemesterGradeRepository{db: db}
}

// Create 创建学生学期成绩汇总。
func (r *studentSemesterGradeRepository) Create(ctx context.Context, grade *entity.StudentSemesterGrade) error {
	if grade.ID == 0 {
		grade.ID = snowflake.Generate()
	}
	return r.db.WithContext(ctx).Create(grade).Error
}

// BatchUpsert 批量写入学生学期成绩汇总。
func (r *studentSemesterGradeRepository) BatchUpsert(ctx context.Context, grades []*entity.StudentSemesterGrade) error {
	if len(grades) == 0 {
		return nil
	}
	for i := range grades {
		if grades[i].ID == 0 {
			grades[i].ID = snowflake.Generate()
		}
	}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "student_id"},
			{Name: "semester_id"},
			{Name: "course_id"},
		},
		DoUpdates: clause.AssignmentColumns([]string{
			"final_score",
			"grade_level",
			"gpa_point",
			"credits",
			"is_adjusted",
			"review_id",
			"updated_at",
		}),
	}).CreateInBatches(grades, 100).Error
}

// GetByID 根据 ID 获取学生学期成绩。
func (r *studentSemesterGradeRepository) GetByID(ctx context.Context, id int64) (*entity.StudentSemesterGrade, error) {
	var grade entity.StudentSemesterGrade
	err := r.db.WithContext(ctx).First(&grade, id).Error
	if err != nil {
		return nil, err
	}
	return &grade, nil
}

// GetByStudentCourseSemester 获取学生某学期某课程成绩。
func (r *studentSemesterGradeRepository) GetByStudentCourseSemester(ctx context.Context, studentID, courseID, semesterID int64) (*entity.StudentSemesterGrade, error) {
	var grade entity.StudentSemesterGrade
	err := r.db.WithContext(ctx).
		Where("student_id = ? AND course_id = ? AND semester_id = ?", studentID, courseID, semesterID).
		First(&grade).Error
	if err != nil {
		return nil, err
	}
	return &grade, nil
}

// List 查询学生学期成绩列表。
func (r *studentSemesterGradeRepository) List(ctx context.Context, params *StudentGradeListParams) ([]*entity.StudentSemesterGrade, int64, error) {
	query := r.db.WithContext(ctx).Model(&entity.StudentSemesterGrade{}).
		Scopes(database.WithSchoolID(params.SchoolID))
	if params.StudentID > 0 {
		query = query.Where("student_id = ?", params.StudentID)
	}
	if params.SemesterID > 0 {
		query = query.Where("semester_id = ?", params.SemesterID)
	}
	if params.CourseID > 0 {
		query = query.Where("course_id = ?", params.CourseID)
	}
	if params.GradeLevel != "" {
		query = query.Where("grade_level = ?", params.GradeLevel)
	}
	if params.IsAdjusted != nil {
		query = query.Where("is_adjusted = ?", *params.IsAdjusted)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
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
	}).ApplyToGORM(query, map[string]string{
		"created_at":  "created_at",
		"final_score": "final_score",
		"gpa_point":   "gpa_point",
		"course_id":   "course_id",
	})

	var grades []*entity.StudentSemesterGrade
	if err := query.Find(&grades).Error; err != nil {
		return nil, 0, err
	}
	return grades, total, nil
}

// ListByReview 查询审核记录生成的学生成绩。
func (r *studentSemesterGradeRepository) ListByReview(ctx context.Context, reviewID int64) ([]*entity.StudentSemesterGrade, error) {
	var grades []*entity.StudentSemesterGrade
	err := r.db.WithContext(ctx).
		Where("review_id = ?", reviewID).
		Order("student_id asc").
		Find(&grades).Error
	return grades, err
}

// ListByStudent 查询学生指定学期范围内的成绩。
func (r *studentSemesterGradeRepository) ListByStudent(ctx context.Context, schoolID, studentID int64, semesterIDs []int64) ([]*entity.StudentSemesterGrade, error) {
	query := r.db.WithContext(ctx).
		Scopes(database.WithSchoolID(schoolID)).
		Where("student_id = ?", studentID)
	if len(semesterIDs) > 0 {
		query = query.Where("semester_id IN ?", semesterIDs)
	}
	var grades []*entity.StudentSemesterGrade
	err := query.Order("semester_id asc, course_id asc").Find(&grades).Error
	return grades, err
}

// UpdateAfterAppeal 更新申诉通过后的正式成绩。
func (r *studentSemesterGradeRepository) UpdateAfterAppeal(ctx context.Context, id int64, finalScore float64, gradeLevel string, gpaPoint float64) error {
	return r.db.WithContext(ctx).Model(&entity.StudentSemesterGrade{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"final_score": finalScore,
			"grade_level": gradeLevel,
			"gpa_point":   gpaPoint,
			"is_adjusted": true,
			"updated_at":  gorm.Expr("now()"),
		}).Error
}

// DeleteByReview 删除指定审核记录生成的汇总成绩。
func (r *studentSemesterGradeRepository) DeleteByReview(ctx context.Context, reviewID int64) error {
	return r.db.WithContext(ctx).Where("review_id = ?", reviewID).Delete(&entity.StudentSemesterGrade{}).Error
}

// CalculateSemesterGPA 计算学生指定学期 GPA。
func (r *studentSemesterGradeRepository) CalculateSemesterGPA(ctx context.Context, schoolID, studentID, semesterID int64) (*GPAStats, error) {
	var stats GPAStats
	err := r.db.WithContext(ctx).Model(&entity.StudentSemesterGrade{}).
		Select(`
			COALESCE(SUM(credits), 0) AS total_credits,
			COALESCE(SUM(gpa_point * credits), 0) AS total_weighted_point,
			COALESCE(SUM(gpa_point * credits) / NULLIF(SUM(credits), 0), 0) AS gpa,
			COUNT(*) AS course_count
		`).
		Where("school_id = ? AND student_id = ? AND semester_id = ?", schoolID, studentID, semesterID).
		Scan(&stats).Error
	if err != nil {
		return nil, err
	}
	return &stats, nil
}

// CalculateCumulativeGPA 计算学生累计 GPA。
func (r *studentSemesterGradeRepository) CalculateCumulativeGPA(ctx context.Context, schoolID, studentID int64) (*GPAStats, error) {
	var stats GPAStats
	err := r.db.WithContext(ctx).Model(&entity.StudentSemesterGrade{}).
		Select(`
			COALESCE(SUM(credits), 0) AS total_credits,
			COALESCE(SUM(gpa_point * credits), 0) AS total_weighted_point,
			COALESCE(SUM(gpa_point * credits) / NULLIF(SUM(credits), 0), 0) AS gpa,
			COUNT(*) AS course_count
		`).
		Where("school_id = ? AND student_id = ?", schoolID, studentID).
		Scan(&stats).Error
	if err != nil {
		return nil, err
	}
	return &stats, nil
}

// CourseAnalytics 聚合课程成绩分析数据。
func (r *studentSemesterGradeRepository) CourseAnalytics(ctx context.Context, schoolID, courseID int64) (*CourseGradeAnalytics, error) {
	var stats CourseGradeAnalytics
	err := r.db.WithContext(ctx).Model(&entity.StudentSemesterGrade{}).
		Select(`
			course_id,
			COUNT(DISTINCT student_id) AS student_count,
			COALESCE(AVG(final_score), 0) AS average_score,
			COALESCE(PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY final_score), 0) AS median_score,
			COALESCE(MAX(final_score), 0) AS highest_score,
			COALESCE(MIN(final_score), 0) AS lowest_score,
			COALESCE(AVG(gpa_point), 0) AS average_gpa,
			COALESCE(AVG(CASE WHEN final_score >= 60 THEN 1.0 ELSE 0 END), 0) AS pass_rate
		`).
		Where("school_id = ? AND course_id = ?", schoolID, courseID).
		Group("course_id").
		Scan(&stats).Error
	if err != nil {
		return nil, err
	}
	return &stats, nil
}

// CourseGradeDistribution 按等级统计课程成绩分布。
func (r *studentSemesterGradeRepository) CourseGradeDistribution(ctx context.Context, schoolID, courseID int64) ([]*GradeDistributionItem, error) {
	var items []*GradeDistributionItem
	err := r.db.WithContext(ctx).Model(&entity.StudentSemesterGrade{}).
		Select("grade_level, COUNT(*) AS count").
		Where("school_id = ? AND course_id = ?", schoolID, courseID).
		Group("grade_level").
		Order("grade_level asc").
		Find(&items).Error
	return items, err
}

// CourseScoreDistribution 按分数段统计课程成绩分布。
func (r *studentSemesterGradeRepository) CourseScoreDistribution(ctx context.Context, schoolID, courseID int64) ([]*ScoreDistributionItem, error) {
	var items []*ScoreDistributionItem
	err := r.db.WithContext(ctx).Table("student_semester_grades").
		Select(`
			CASE
				WHEN final_score >= 90 THEN '90-100'
				WHEN final_score >= 80 THEN '80-89'
				WHEN final_score >= 70 THEN '70-79'
				WHEN final_score >= 60 THEN '60-69'
				ELSE '0-59'
			END AS range,
			COUNT(*) AS count
		`).
		Where("school_id = ? AND course_id = ?", schoolID, courseID).
		Group("range").
		Order(`
			CASE range
				WHEN '90-100' THEN 1
				WHEN '80-89' THEN 2
				WHEN '70-79' THEN 3
				WHEN '60-69' THEN 4
				ELSE 5
			END
		`).
		Find(&items).Error
	return items, err
}

// SchoolAnalytics 聚合学校成绩分析数据。
func (r *studentSemesterGradeRepository) SchoolAnalytics(ctx context.Context, schoolID, semesterID int64) (*SchoolGradeAnalytics, error) {
	var stats SchoolGradeAnalytics
	query := r.db.WithContext(ctx).Table("student_semester_grades AS g").
		Select(`
			g.school_id,
			g.semester_id,
			COUNT(DISTINCT g.student_id) AS student_count,
			COUNT(DISTINCT g.course_id) AS course_count,
			COALESCE(AVG(g.final_score), 0) AS average_score,
			COALESCE(SUM(g.gpa_point * g.credits) / NULLIF(SUM(g.credits), 0), 0) AS average_gpa,
			(SELECT COUNT(*) FROM academic_warnings w WHERE w.school_id = g.school_id AND w.semester_id = g.semester_id AND w.status <> ?) AS warning_count,
			(SELECT COUNT(*) FROM grade_appeals a WHERE a.school_id = g.school_id AND a.semester_id = g.semester_id) AS appeal_count,
			(SELECT COUNT(*) FROM grade_reviews r WHERE r.school_id = g.school_id AND r.semester_id = g.semester_id AND r.status = ?) AS reviewed_count
		`, enum.AcademicWarningStatusResolved, enum.GradeReviewStatusApproved).
		Where("g.school_id = ?", schoolID)
	if semesterID > 0 {
		query = query.Where("g.semester_id = ?", semesterID)
	}
	err := query.Group("g.school_id, g.semester_id").Scan(&stats).Error
	if err != nil {
		return nil, err
	}
	return &stats, nil
}

// SchoolFailRate 统计学校学期内不及格率。
func (r *studentSemesterGradeRepository) SchoolFailRate(ctx context.Context, schoolID, semesterID int64) (float64, error) {
	var result struct {
		FailRate float64 `gorm:"column:fail_rate"`
	}
	err := r.db.WithContext(ctx).Model(&entity.StudentSemesterGrade{}).
		Select("COALESCE(AVG(CASE WHEN final_score < 60 THEN 1.0 ELSE 0 END), 0) AS fail_rate").
		Where("school_id = ? AND semester_id = ?", schoolID, semesterID).
		Scan(&result).Error
	if err != nil {
		return 0, err
	}
	return result.FailRate, nil
}

// SchoolGPADistribution 按学生学期 GPA 统计学校 GPA 分布。
func (r *studentSemesterGradeRepository) SchoolGPADistribution(ctx context.Context, schoolID, semesterID int64) ([]*GPARangeDistributionItem, error) {
	var items []*GPARangeDistributionItem
	err := r.db.WithContext(ctx).Raw(`
		SELECT bucket.range, COUNT(*) AS count
		FROM (
			SELECT
				CASE
					WHEN student_gpa >= 3.5 THEN '3.5-4.0'
					WHEN student_gpa >= 3.0 THEN '3.0-3.49'
					WHEN student_gpa >= 2.5 THEN '2.5-2.99'
					WHEN student_gpa >= 2.0 THEN '2.0-2.49'
					ELSE '0-1.99'
				END AS range
			FROM (
				SELECT
					student_id,
					COALESCE(SUM(gpa_point * credits) / NULLIF(SUM(credits), 0), 0) AS student_gpa
				FROM student_semester_grades
				WHERE school_id = ? AND semester_id = ?
				GROUP BY student_id
			) AS sg
		) AS bucket
		GROUP BY bucket.range
		ORDER BY
			CASE bucket.range
				WHEN '3.5-4.0' THEN 1
				WHEN '3.0-3.49' THEN 2
				WHEN '2.5-2.99' THEN 3
				WHEN '2.0-2.49' THEN 4
				ELSE 5
			END
	`, schoolID, semesterID).Scan(&items).Error
	return items, err
}

// SchoolCoursePerformance 查询学校学期内课程表现排行。
func (r *studentSemesterGradeRepository) SchoolCoursePerformance(ctx context.Context, schoolID, semesterID int64, ascending bool, limit int) ([]*CoursePerformanceItem, error) {
	order := "average_score desc"
	if ascending {
		order = "average_score asc"
	}
	var items []*CoursePerformanceItem
	err := r.db.WithContext(ctx).Table("student_semester_grades AS g").
		Select(`
			g.course_id,
			c.title AS course_name,
			COALESCE(AVG(g.final_score), 0) AS average_score,
			COALESCE(AVG(CASE WHEN g.final_score >= 60 THEN 1.0 ELSE 0 END), 0) AS pass_rate,
			COUNT(DISTINCT g.student_id) AS student_count
		`).
		Joins("JOIN courses c ON c.id = g.course_id").
		Where("g.school_id = ? AND g.semester_id = ?", schoolID, semesterID).
		Group("g.course_id, c.title").
		Order(order).
		Limit(limit).
		Find(&items).Error
	return items, err
}

// PlatformAnalytics 聚合平台成绩总览数据。
func (r *studentSemesterGradeRepository) PlatformAnalytics(ctx context.Context, semesterID int64) (*PlatformGradeAnalytics, error) {
	var stats PlatformGradeAnalytics
	query := r.db.WithContext(ctx).Table("student_semester_grades AS g")
	if semesterID > 0 {
		query = query.Select(`
			COUNT(DISTINCT g.school_id) AS school_count,
			COUNT(DISTINCT g.student_id) AS student_count,
			COUNT(DISTINCT g.course_id) AS course_count,
			COALESCE(AVG(g.final_score), 0) AS average_score,
			COALESCE(SUM(g.gpa_point * g.credits) / NULLIF(SUM(g.credits), 0), 0) AS average_gpa,
			(SELECT COUNT(*) FROM academic_warnings w WHERE w.semester_id = ? AND w.status <> ?) AS warning_count,
			(SELECT COUNT(*) FROM grade_reviews r WHERE r.semester_id = ? AND r.status = ?) AS reviewed_count
		`, semesterID, enum.AcademicWarningStatusResolved, semesterID, enum.GradeReviewStatusApproved).
			Where("g.semester_id = ?", semesterID)
	} else {
		query = query.Select(`
			COUNT(DISTINCT g.school_id) AS school_count,
			COUNT(DISTINCT g.student_id) AS student_count,
			COUNT(DISTINCT g.course_id) AS course_count,
			COALESCE(AVG(g.final_score), 0) AS average_score,
			COALESCE(SUM(g.gpa_point * g.credits) / NULLIF(SUM(g.credits), 0), 0) AS average_gpa,
			(SELECT COUNT(*) FROM academic_warnings w WHERE w.status <> ?) AS warning_count,
			(SELECT COUNT(*) FROM grade_reviews r WHERE r.status = ?) AS reviewed_count
		`, enum.AcademicWarningStatusResolved, enum.GradeReviewStatusApproved)
	}
	err := query.Scan(&stats).Error
	if err != nil {
		return nil, err
	}
	return &stats, nil
}

// PlatformSchoolComparison 查询平台学校成绩对比。
func (r *studentSemesterGradeRepository) PlatformSchoolComparison(ctx context.Context, semesterID int64) ([]*SchoolComparisonItem, error) {
	var items []*SchoolComparisonItem
	query := r.db.WithContext(ctx).Table("student_semester_grades AS g").
		Select(`
			g.school_id,
			s.name AS school_name,
			COUNT(DISTINCT g.student_id) AS student_count,
			COALESCE(SUM(g.gpa_point * g.credits) / NULLIF(SUM(g.credits), 0), 0) AS average_gpa
		`).
		Joins("JOIN schools s ON s.id = g.school_id")
	if semesterID > 0 {
		query = query.Where("g.semester_id = ?", semesterID)
	}
	err := query.Group("g.school_id, s.name").
		Order("average_gpa desc, student_count desc").
		Find(&items).Error
	return items, err
}

// ListFailingCourses 查询学期内低绩点课程，用于学业预警检测的数据准备。
func (r *studentSemesterGradeRepository) ListFailingCourses(ctx context.Context, schoolID, semesterID int64, maxGPAPoint float64) ([]*FailingCourseItem, error) {
	var items []*FailingCourseItem
	err := r.db.WithContext(ctx).Model(&entity.StudentSemesterGrade{}).
		Select("student_id, course_id").
		Where("school_id = ? AND semester_id = ? AND gpa_point <= ?", schoolID, semesterID, maxGPAPoint).
		Order("student_id asc, course_id asc").
		Find(&items).Error
	return items, err
}
