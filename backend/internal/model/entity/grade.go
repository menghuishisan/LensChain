// grade.go
// 模块06 — 评测与成绩：数据库实体结构体。
// 该文件覆盖学期、等级映射、审核、成绩汇总、申诉、预警、成绩单等全部表映射。

package entity

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// Semester 学期表。
type Semester struct {
	ID        int64          `gorm:"primaryKey;autoIncrement:false" json:"id,string"`
	SchoolID  int64          `gorm:"not null;index;index:idx_semesters_is_current" json:"school_id,string"`
	Name      string         `gorm:"type:varchar(50);not null" json:"name"`
	Code      string         `gorm:"type:varchar(20);not null;uniqueIndex:uk_semesters_school_code,where:deleted_at IS NULL" json:"code"`
	StartDate time.Time      `gorm:"type:date;not null" json:"start_date"`
	EndDate   time.Time      `gorm:"type:date;not null" json:"end_date"`
	IsCurrent bool           `gorm:"not null;default:false;index:idx_semesters_is_current" json:"is_current"`
	CreatedAt time.Time      `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt time.Time      `gorm:"not null;default:now()" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName 指定学期表表名。
func (Semester) TableName() string {
	return "semesters"
}

// GradeLevelConfig 等级映射配置表。
type GradeLevelConfig struct {
	ID        int64     `gorm:"primaryKey;autoIncrement:false" json:"id,string"`
	SchoolID  int64     `gorm:"not null;index;uniqueIndex:uk_grade_level_configs_school_level" json:"school_id,string"`
	LevelName string    `gorm:"type:varchar(10);not null;uniqueIndex:uk_grade_level_configs_school_level" json:"level_name"`
	MinScore  float64   `gorm:"type:decimal(5,2);not null" json:"min_score"`
	MaxScore  float64   `gorm:"type:decimal(5,2);not null" json:"max_score"`
	GPAPoint  float64   `gorm:"type:decimal(3,2);not null" json:"gpa_point"`
	SortOrder int       `gorm:"not null;default:0" json:"sort_order"`
	CreatedAt time.Time `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt time.Time `gorm:"not null;default:now()" json:"updated_at"`
}

// TableName 指定等级映射配置表表名。
func (GradeLevelConfig) TableName() string {
	return "grade_level_configs"
}

// GradeReview 成绩审核表。
type GradeReview struct {
	ID            int64      `gorm:"primaryKey;autoIncrement:false" json:"id,string"`
	CourseID      int64      `gorm:"not null;uniqueIndex:uk_grade_reviews_course_semester" json:"course_id,string"`
	SchoolID      int64      `gorm:"not null;index" json:"school_id,string"`
	SemesterID    int64      `gorm:"not null;index;uniqueIndex:uk_grade_reviews_course_semester" json:"semester_id,string"`
	SubmittedBy   int64      `gorm:"not null;index" json:"submitted_by,string"`
	Status        int16      `gorm:"column:status;type:smallint;not null;default:1;index" json:"status"`
	SubmitNote    *string    `gorm:"type:text" json:"submit_note,omitempty"`
	SubmittedAt   *time.Time `gorm:"" json:"submitted_at,omitempty"`
	ReviewedBy    *int64     `gorm:"" json:"reviewed_by,omitempty,string"`
	ReviewedAt    *time.Time `gorm:"" json:"reviewed_at,omitempty"`
	ReviewComment *string    `gorm:"type:text" json:"review_comment,omitempty"`
	IsLocked      bool       `gorm:"not null;default:false" json:"is_locked"`
	LockedAt      *time.Time `gorm:"" json:"locked_at,omitempty"`
	UnlockedBy    *int64     `gorm:"" json:"unlocked_by,omitempty,string"`
	UnlockedAt    *time.Time `gorm:"" json:"unlocked_at,omitempty"`
	UnlockReason  *string    `gorm:"type:text" json:"unlock_reason,omitempty"`
	CreatedAt     time.Time  `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt     time.Time  `gorm:"not null;default:now()" json:"updated_at"`
}

// TableName 指定成绩审核表表名。
func (GradeReview) TableName() string {
	return "grade_reviews"
}

// StudentSemesterGrade 学生学期成绩汇总表。
type StudentSemesterGrade struct {
	ID         int64     `gorm:"primaryKey;autoIncrement:false" json:"id,string"`
	StudentID  int64     `gorm:"not null;uniqueIndex:uk_student_semester_grades_unique;index" json:"student_id,string"`
	SchoolID   int64     `gorm:"not null;index:idx_ssg_school_id" json:"school_id,string"`
	SemesterID int64     `gorm:"not null;uniqueIndex:uk_student_semester_grades_unique;index:idx_ssg_semester_id" json:"semester_id,string"`
	CourseID   int64     `gorm:"not null;uniqueIndex:uk_student_semester_grades_unique;index:idx_ssg_course_id" json:"course_id,string"`
	FinalScore float64   `gorm:"type:decimal(6,2);not null" json:"final_score"`
	GradeLevel string    `gorm:"type:varchar(10);not null;index:idx_ssg_grade_level" json:"grade_level"`
	GPAPoint   float64   `gorm:"type:decimal(3,2);not null" json:"gpa_point"`
	Credits    float64   `gorm:"type:decimal(3,1);not null" json:"credits"`
	IsAdjusted bool      `gorm:"not null;default:false" json:"is_adjusted"`
	ReviewID   int64     `gorm:"not null" json:"review_id,string"`
	CreatedAt  time.Time `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt  time.Time `gorm:"not null;default:now()" json:"updated_at"`
}

// TableName 指定学生学期成绩汇总表表名。
func (StudentSemesterGrade) TableName() string {
	return "student_semester_grades"
}

// GradeAppeal 成绩申诉表。
type GradeAppeal struct {
	ID            int64      `gorm:"primaryKey;autoIncrement:false" json:"id,string"`
	StudentID     int64      `gorm:"not null;uniqueIndex:uk_grade_appeals_student_course_semester;index" json:"student_id,string"`
	SchoolID      int64      `gorm:"not null;index" json:"school_id,string"`
	SemesterID    int64      `gorm:"not null;uniqueIndex:uk_grade_appeals_student_course_semester" json:"semester_id,string"`
	CourseID      int64      `gorm:"not null;uniqueIndex:uk_grade_appeals_student_course_semester;index" json:"course_id,string"`
	GradeID       int64      `gorm:"not null" json:"grade_id,string"`
	OriginalScore float64    `gorm:"type:decimal(6,2);not null" json:"original_score"`
	AppealReason  string     `gorm:"type:text;not null" json:"appeal_reason"`
	Status        int16      `gorm:"column:status;type:smallint;not null;default:1;index" json:"status"`
	HandledBy     *int64     `gorm:"" json:"handled_by,omitempty,string"`
	HandledAt     *time.Time `gorm:"" json:"handled_at,omitempty"`
	NewScore      *float64   `gorm:"type:decimal(6,2)" json:"new_score,omitempty"`
	HandleComment *string    `gorm:"type:text" json:"handle_comment,omitempty"`
	CreatedAt     time.Time  `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt     time.Time  `gorm:"not null;default:now()" json:"updated_at"`
}

// TableName 指定成绩申诉表表名。
func (GradeAppeal) TableName() string {
	return "grade_appeals"
}

// AcademicWarning 学业预警表。
type AcademicWarning struct {
	ID          int64          `gorm:"primaryKey;autoIncrement:false" json:"id,string"`
	StudentID   int64          `gorm:"not null;index" json:"student_id,string"`
	SchoolID    int64          `gorm:"not null;index" json:"school_id,string"`
	SemesterID  int64          `gorm:"not null;index" json:"semester_id,string"`
	WarningType int16          `gorm:"column:warning_type;type:smallint;not null" json:"warning_type"`
	Detail      datatypes.JSON `gorm:"column:detail;type:jsonb;not null" json:"detail"`
	Status      int16          `gorm:"column:status;type:smallint;not null;default:1;index" json:"status"`
	HandledBy   *int64         `gorm:"" json:"handled_by,omitempty,string"`
	HandledAt   *time.Time     `gorm:"" json:"handled_at,omitempty"`
	HandleNote  *string        `gorm:"type:text" json:"handle_note,omitempty"`
	CreatedAt   time.Time      `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt   time.Time      `gorm:"not null;default:now()" json:"updated_at"`
}

// TableName 指定学业预警表表名。
func (AcademicWarning) TableName() string {
	return "academic_warnings"
}

// WarningConfig 学业预警配置表。
type WarningConfig struct {
	ID                 int64     `gorm:"primaryKey;autoIncrement:false" json:"id,string"`
	SchoolID           int64     `gorm:"not null;uniqueIndex" json:"school_id,string"`
	GPAThreshold       float64   `gorm:"type:decimal(3,2);not null;default:2.00" json:"gpa_threshold"`
	FailCountThreshold int       `gorm:"not null;default:2" json:"fail_count_threshold"`
	IsEnabled          bool      `gorm:"not null;default:true" json:"is_enabled"`
	CreatedAt          time.Time `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt          time.Time `gorm:"not null;default:now()" json:"updated_at"`
}

// TableName 指定学业预警配置表表名。
func (WarningConfig) TableName() string {
	return "warning_configs"
}

// TranscriptRecord 成绩单生成记录表。
type TranscriptRecord struct {
	ID               int64          `gorm:"primaryKey;autoIncrement:false" json:"id,string"`
	SchoolID         int64          `gorm:"not null;index" json:"school_id,string"`
	StudentID        int64          `gorm:"not null;index" json:"student_id,string"`
	GeneratedBy      int64          `gorm:"not null" json:"generated_by,string"`
	FileURL          string         `gorm:"type:varchar(500);not null" json:"file_url"`
	FileSize         int64          `gorm:"not null" json:"file_size"`
	IncludeSemesters datatypes.JSON `gorm:"column:include_semesters;type:jsonb;not null" json:"include_semesters"`
	GeneratedAt      time.Time      `gorm:"not null;default:now()" json:"generated_at"`
	ExpiresAt        *time.Time     `gorm:"" json:"expires_at,omitempty"`
	CreatedAt        time.Time      `gorm:"not null;default:now()" json:"created_at"`
}

// TableName 指定成绩单生成记录表表名。
func (TranscriptRecord) TableName() string {
	return "transcript_records"
}
