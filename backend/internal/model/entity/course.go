// course.go
// 模块03 — 课程与教学：数据库实体结构体
// 对照 docs/modules/03-课程与教学/02-数据库设计.md
// 包含 18 张表的 GORM 映射结构体

package entity

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// Course 课程主表。
// 该结构严格映射 courses 表字段，不承载课程聚合关系和业务逻辑。
type Course struct {
	ID           int64          `gorm:"primaryKey;autoIncrement:false" json:"id,string"`
	SchoolID     int64          `gorm:"not null;index" json:"school_id,string"`
	TeacherID    int64          `gorm:"not null;index" json:"teacher_id,string"`
	Title        string         `gorm:"type:varchar(200);not null" json:"title"`
	Description  *string        `gorm:"type:text" json:"description,omitempty"`
	CoverURL     *string        `gorm:"type:varchar(500)" json:"cover_url,omitempty"`
	CourseType   int16          `gorm:"column:course_type;type:smallint;not null" json:"course_type"`
	Difficulty   int16          `gorm:"column:difficulty;type:smallint;not null;default:1" json:"difficulty"`
	Topic        string         `gorm:"type:varchar(50);not null" json:"topic"`
	Status       int16          `gorm:"column:status;type:smallint;not null;default:1" json:"status"`
	IsShared     bool           `gorm:"not null;default:false" json:"is_shared"`
	InviteCode   *string        `gorm:"type:varchar(10)" json:"invite_code,omitempty"`
	StartAt      *time.Time     `gorm:"" json:"start_at,omitempty"`
	EndAt        *time.Time     `gorm:"" json:"end_at,omitempty"`
	Credits      *float64       `gorm:"type:decimal(3,1)" json:"credits,omitempty"`
	SemesterID   *int64         `gorm:"index" json:"semester_id,omitempty,string"`
	MaxStudents  *int           `gorm:"" json:"max_students,omitempty"`
	ClonedFromID *int64         `gorm:"" json:"cloned_from_id,omitempty,string"`
	CreatedAt    time.Time      `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt    time.Time      `gorm:"not null;default:now()" json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName 返回课程主表表名。
func (Course) TableName() string {
	return "courses"
}

// Chapter 章节表。
// 该结构严格映射 chapters 表字段，不承载课时聚合关系。
type Chapter struct {
	ID          int64          `gorm:"primaryKey;autoIncrement:false" json:"id,string"`
	CourseID    int64          `gorm:"not null;index" json:"course_id,string"`
	Title       string         `gorm:"type:varchar(200);not null" json:"title"`
	Description *string        `gorm:"type:text" json:"description,omitempty"`
	SortOrder   int            `gorm:"not null;default:0" json:"sort_order"`
	CreatedAt   time.Time      `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt   time.Time      `gorm:"not null;default:now()" json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName 返回章节表表名。
func (Chapter) TableName() string {
	return "chapters"
}

// Lesson 课时表。
// 该结构严格映射 lessons 表字段，不承载附件聚合关系。
type Lesson struct {
	ID               int64          `gorm:"primaryKey;autoIncrement:false" json:"id,string"`
	ChapterID        int64          `gorm:"not null;index" json:"chapter_id,string"`
	CourseID         int64          `gorm:"not null;index" json:"course_id,string"`
	Title            string         `gorm:"type:varchar(200);not null" json:"title"`
	ContentType      int16          `gorm:"column:content_type;type:smallint;not null" json:"content_type"`
	Content          *string        `gorm:"type:text" json:"content,omitempty"`
	VideoURL         *string        `gorm:"type:varchar(500)" json:"video_url,omitempty"`
	VideoDuration    *int           `gorm:"" json:"video_duration,omitempty"`
	ExperimentID     *int64         `gorm:"" json:"experiment_id,omitempty,string"`
	SortOrder        int            `gorm:"not null;default:0" json:"sort_order"`
	EstimatedMinutes *int           `gorm:"" json:"estimated_minutes,omitempty"`
	CreatedAt        time.Time      `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt        time.Time      `gorm:"not null;default:now()" json:"updated_at"`
	DeletedAt        gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName 返回课时表表名。
func (Lesson) TableName() string {
	return "lessons"
}

// LessonAttachment 课时附件表
// 对应 lesson_attachments 表
type LessonAttachment struct {
	ID        int64     `gorm:"primaryKey;autoIncrement:false" json:"id,string"`
	LessonID  int64     `gorm:"not null;index" json:"lesson_id,string"`
	FileName  string    `gorm:"type:varchar(200);not null" json:"file_name"`
	FileURL   string    `gorm:"type:varchar(500);not null" json:"file_url"`
	FileSize  int64     `gorm:"not null;default:0" json:"file_size"`
	FileType  string    `gorm:"type:varchar(50);not null" json:"file_type"`
	SortOrder int       `gorm:"not null;default:0" json:"sort_order"`
	CreatedAt time.Time `gorm:"not null;default:now()" json:"created_at"`
}

// TableName 返回课时附件表表名。
func (LessonAttachment) TableName() string {
	return "lesson_attachments"
}

// CourseEnrollment 选课记录表
// 对应 course_enrollments 表
type CourseEnrollment struct {
	ID         int64      `gorm:"primaryKey;autoIncrement:false" json:"id,string"`
	CourseID   int64      `gorm:"not null;index" json:"course_id,string"`
	StudentID  int64      `gorm:"not null;index" json:"student_id,string"`
	JoinMethod int16      `gorm:"column:join_method;type:smallint;not null" json:"join_method"`
	JoinedAt   time.Time  `gorm:"not null;default:now()" json:"joined_at"`
	RemovedAt  *time.Time `gorm:"" json:"removed_at,omitempty"`
}

// TableName 返回选课记录表表名。
func (CourseEnrollment) TableName() string {
	return "course_enrollments"
}

// Assignment 作业/测验表。
// 该结构严格映射 assignments 表字段，不承载题目聚合关系。
type Assignment struct {
	ID                  int64          `gorm:"primaryKey;autoIncrement:false" json:"id,string"`
	CourseID            int64          `gorm:"not null;index" json:"course_id,string"`
	ChapterID           *int64         `gorm:"" json:"chapter_id,omitempty,string"`
	Title               string         `gorm:"type:varchar(200);not null" json:"title"`
	Description         *string        `gorm:"type:text" json:"description,omitempty"`
	AssignmentType      int16          `gorm:"column:assignment_type;type:smallint;not null" json:"assignment_type"`
	TotalScore          float64        `gorm:"type:decimal(6,2);not null" json:"total_score"`
	DeadlineAt          time.Time      `gorm:"not null" json:"deadline_at"`
	MaxSubmissions      int            `gorm:"not null;default:1" json:"max_submissions"`
	LatePolicy          int16          `gorm:"column:late_policy;type:smallint;not null;default:1" json:"late_policy"`
	LateDeductionPerDay *float64       `gorm:"type:decimal(5,2)" json:"late_deduction_per_day,omitempty"`
	IsPublished         bool           `gorm:"not null;default:false" json:"is_published"`
	SortOrder           int            `gorm:"not null;default:0" json:"sort_order"`
	CreatedAt           time.Time      `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt           time.Time      `gorm:"not null;default:now()" json:"updated_at"`
	DeletedAt           gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName 返回作业表表名。
func (Assignment) TableName() string {
	return "assignments"
}

// AssignmentQuestion 作业题目表
// 对应 assignment_questions 表
type AssignmentQuestion struct {
	ID              int64          `gorm:"primaryKey;autoIncrement:false" json:"id,string"`
	AssignmentID    int64          `gorm:"not null;index" json:"assignment_id,string"`
	QuestionType    int16          `gorm:"column:question_type;type:smallint;not null" json:"question_type"`
	Title           string         `gorm:"type:text;not null" json:"title"`
	Options         datatypes.JSON `gorm:"column:options;type:jsonb" json:"options,omitempty"`
	CorrectAnswer   *string        `gorm:"type:text" json:"correct_answer,omitempty"`
	ReferenceAnswer *string        `gorm:"type:text" json:"reference_answer,omitempty"`
	Score           float64        `gorm:"type:decimal(6,2);not null" json:"score"`
	JudgeConfig     datatypes.JSON `gorm:"column:judge_config;type:jsonb" json:"judge_config,omitempty"`
	SortOrder       int            `gorm:"not null;default:0" json:"sort_order"`
	CreatedAt       time.Time      `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt       time.Time      `gorm:"not null;default:now()" json:"updated_at"`
}

// TableName 返回作业题目表表名。
func (AssignmentQuestion) TableName() string {
	return "assignment_questions"
}

// AssignmentSubmission 作业提交记录表。
// 该结构严格映射 assignment_submissions 表字段，不承载答案聚合关系。
type AssignmentSubmission struct {
	ID                   int64      `gorm:"primaryKey;autoIncrement:false" json:"id,string"`
	AssignmentID         int64      `gorm:"not null;index" json:"assignment_id,string"`
	StudentID            int64      `gorm:"not null;index" json:"student_id,string"`
	SubmissionNo         int        `gorm:"not null;default:1" json:"submission_no"`
	Status               int16      `gorm:"column:status;type:smallint;not null;default:1" json:"status"`
	TotalScore           *float64   `gorm:"type:decimal(10,2)" json:"total_score,omitempty"`
	IsLate               bool       `gorm:"not null;default:false" json:"is_late"`
	LateDays             *int       `gorm:"" json:"late_days,omitempty"`
	ScoreBeforeDeduction *float64   `gorm:"type:decimal(10,2)" json:"score_before_deduction,omitempty"`
	ScoreAfterDeduction  *float64   `gorm:"type:decimal(10,2)" json:"score_after_deduction,omitempty"`
	GradedBy             *int64     `gorm:"" json:"graded_by,omitempty,string"`
	GradedAt             *time.Time `gorm:"" json:"graded_at,omitempty"`
	TeacherComment       *string    `gorm:"type:text" json:"teacher_comment,omitempty"`
	SubmittedAt          time.Time  `gorm:"not null;default:now()" json:"submitted_at"`
}

// TableName 返回作业提交记录表表名。
func (AssignmentSubmission) TableName() string {
	return "assignment_submissions"
}

// SubmissionAnswer 提交答案明细表
// 对应 submission_answers 表
type SubmissionAnswer struct {
	ID              int64          `gorm:"primaryKey;autoIncrement:false" json:"id,string"`
	SubmissionID    int64          `gorm:"not null;index" json:"submission_id,string"`
	QuestionID      int64          `gorm:"not null" json:"question_id,string"`
	AnswerContent   *string        `gorm:"type:text" json:"answer_content,omitempty"`
	AnswerFileURL   *string        `gorm:"type:varchar(500)" json:"answer_file_url,omitempty"`
	IsCorrect       *bool          `gorm:"" json:"is_correct,omitempty"`
	Score           *float64       `gorm:"type:decimal(10,2)" json:"score,omitempty"`
	TeacherComment  *string        `gorm:"type:text" json:"teacher_comment,omitempty"`
	AutoJudgeResult datatypes.JSON `gorm:"column:auto_judge_result;type:jsonb" json:"auto_judge_result,omitempty"`
	CreatedAt       time.Time      `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt       time.Time      `gorm:"not null;default:now()" json:"updated_at"`
}

// TableName 返回答题记录表表名。
func (SubmissionAnswer) TableName() string {
	return "submission_answers"
}

// LearningProgress 学习进度表
// 对应 learning_progresses 表
type LearningProgress struct {
	ID             int64      `gorm:"primaryKey;autoIncrement:false" json:"id,string"`
	CourseID       int64      `gorm:"not null;index" json:"course_id,string"`
	StudentID      int64      `gorm:"not null;index" json:"student_id,string"`
	LessonID       int64      `gorm:"not null" json:"lesson_id,string"`
	Status         int16      `gorm:"column:status;type:smallint;not null;default:1" json:"status"`
	VideoProgress  *int       `gorm:"" json:"video_progress,omitempty"`
	StudyDuration  int        `gorm:"not null;default:0" json:"study_duration"`
	CompletedAt    *time.Time `gorm:"" json:"completed_at,omitempty"`
	LastAccessedAt *time.Time `gorm:"" json:"last_accessed_at,omitempty"`
	CreatedAt      time.Time  `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt      time.Time  `gorm:"not null;default:now()" json:"updated_at"`
}

// TableName 返回学习进度表表名。
func (LearningProgress) TableName() string {
	return "learning_progresses"
}

// CourseSchedule 课程时间表
// 对应 course_schedules 表
type CourseSchedule struct {
	ID        int64     `gorm:"primaryKey;autoIncrement:false" json:"id,string"`
	CourseID  int64     `gorm:"not null;index" json:"course_id,string"`
	DayOfWeek int16     `gorm:"column:day_of_week;type:smallint;not null" json:"day_of_week"`
	StartTime string    `gorm:"type:time;not null" json:"start_time"`
	EndTime   string    `gorm:"type:time;not null" json:"end_time"`
	Location  *string   `gorm:"type:varchar(100)" json:"location,omitempty"`
	CreatedAt time.Time `gorm:"not null;default:now()" json:"created_at"`
}

// TableName 返回课程时间表表名。
func (CourseSchedule) TableName() string {
	return "course_schedules"
}

// CourseAnnouncement 课程公告表
// 对应 course_announcements 表
type CourseAnnouncement struct {
	ID        int64          `gorm:"primaryKey;autoIncrement:false" json:"id,string"`
	CourseID  int64          `gorm:"not null;index" json:"course_id,string"`
	TeacherID int64          `gorm:"not null" json:"teacher_id,string"`
	Title     string         `gorm:"type:varchar(200);not null" json:"title"`
	Content   string         `gorm:"type:text;not null" json:"content"`
	IsPinned  bool           `gorm:"not null;default:false" json:"is_pinned"`
	CreatedAt time.Time      `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt time.Time      `gorm:"not null;default:now()" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName 返回课程公告表表名。
func (CourseAnnouncement) TableName() string {
	return "course_announcements"
}

// CourseDiscussion 课程讨论帖表。
// 该结构严格映射 course_discussions 表字段，不承载回复聚合关系。
type CourseDiscussion struct {
	ID            int64          `gorm:"primaryKey;autoIncrement:false" json:"id,string"`
	CourseID      int64          `gorm:"not null;index" json:"course_id,string"`
	AuthorID      int64          `gorm:"not null" json:"author_id,string"`
	Title         string         `gorm:"type:varchar(200);not null" json:"title"`
	Content       string         `gorm:"type:text;not null" json:"content"`
	IsPinned      bool           `gorm:"not null;default:false" json:"is_pinned"`
	ReplyCount    int            `gorm:"not null;default:0" json:"reply_count"`
	LikeCount     int            `gorm:"not null;default:0" json:"like_count"`
	LastRepliedAt *time.Time     `gorm:"" json:"last_replied_at,omitempty"`
	CreatedAt     time.Time      `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt     time.Time      `gorm:"not null;default:now()" json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName 返回课程讨论帖表表名。
func (CourseDiscussion) TableName() string {
	return "course_discussions"
}

// DiscussionReply 讨论回复表
// 对应 discussion_replies 表
type DiscussionReply struct {
	ID           int64          `gorm:"primaryKey;autoIncrement:false" json:"id,string"`
	DiscussionID int64          `gorm:"not null;index" json:"discussion_id,string"`
	AuthorID     int64          `gorm:"not null" json:"author_id,string"`
	Content      string         `gorm:"type:text;not null" json:"content"`
	ReplyToID    *int64         `gorm:"" json:"reply_to_id,omitempty,string"`
	CreatedAt    time.Time      `gorm:"not null;default:now()" json:"created_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName 返回讨论回复表表名。
func (DiscussionReply) TableName() string {
	return "discussion_replies"
}

// DiscussionLike 讨论点赞表
// 对应 discussion_likes 表
type DiscussionLike struct {
	ID           int64     `gorm:"primaryKey;autoIncrement:false" json:"id,string"`
	DiscussionID int64     `gorm:"not null;uniqueIndex:uk_discussion_likes" json:"discussion_id,string"`
	UserID       int64     `gorm:"not null;uniqueIndex:uk_discussion_likes" json:"user_id,string"`
	CreatedAt    time.Time `gorm:"not null;default:now()" json:"created_at"`
}

// TableName 返回讨论点赞表表名。
func (DiscussionLike) TableName() string {
	return "discussion_likes"
}

// CourseEvaluation 课程评价表
// 对应 course_evaluations 表
type CourseEvaluation struct {
	ID        int64     `gorm:"primaryKey;autoIncrement:false" json:"id,string"`
	CourseID  int64     `gorm:"not null;uniqueIndex:uk_course_evaluations;index" json:"course_id,string"`
	StudentID int64     `gorm:"not null;uniqueIndex:uk_course_evaluations" json:"student_id,string"`
	Rating    int16     `gorm:"column:rating;type:smallint;not null" json:"rating"`
	Comment   *string   `gorm:"type:text" json:"comment,omitempty"`
	CreatedAt time.Time `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt time.Time `gorm:"not null;default:now()" json:"updated_at"`
}

// TableName 返回课程评价表表名。
func (CourseEvaluation) TableName() string {
	return "course_evaluations"
}

// CourseGradeConfig 成绩权重配置表
// 对应 course_grade_configs 表
// config 字段为 JSONB，存储 {items: [{assignment_id, name, weight}], total_weight: 100}
type CourseGradeConfig struct {
	ID        int64          `gorm:"primaryKey;autoIncrement:false" json:"id,string"`
	CourseID  int64          `gorm:"not null;uniqueIndex" json:"course_id,string"`
	Config    datatypes.JSON `gorm:"column:config;type:jsonb;not null" json:"config"`
	CreatedAt time.Time      `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt time.Time      `gorm:"not null;default:now()" json:"updated_at"`
}

// TableName 返回成绩权重配置表表名。
func (CourseGradeConfig) TableName() string {
	return "course_grade_configs"
}

// CourseExperiment 课程实验关联表
// 对应 course_experiments 表
// 关联模块04的实验模板
type CourseExperiment struct {
	ID           int64     `gorm:"primaryKey;autoIncrement:false" json:"id,string"`
	CourseID     int64     `gorm:"not null;uniqueIndex:uk_course_experiments;index" json:"course_id,string"`
	ExperimentID int64     `gorm:"not null;uniqueIndex:uk_course_experiments" json:"experiment_id,string"`
	Title        *string   `gorm:"type:varchar(200)" json:"title,omitempty"`
	SortOrder    int       `gorm:"not null;default:0" json:"sort_order"`
	CreatedAt    time.Time `gorm:"not null;default:now()" json:"created_at"`
}

// TableName 返回课程实验关联表表名。
func (CourseExperiment) TableName() string {
	return "course_experiments"
}

// CourseGradeOverride 课程成绩调整记录表
// 对应 course_grade_overrides 表
type CourseGradeOverride struct {
	ID            int64     `gorm:"primaryKey;autoIncrement:false" json:"id,string"`
	CourseID      int64     `gorm:"not null;uniqueIndex:uk_course_grade_overrides_course_student;index" json:"course_id,string"`
	StudentID     int64     `gorm:"not null;uniqueIndex:uk_course_grade_overrides_course_student;index" json:"student_id,string"`
	WeightedTotal float64   `gorm:"type:decimal(6,2);not null" json:"weighted_total"`
	FinalScore    float64   `gorm:"type:decimal(6,2);not null" json:"final_score"`
	AdjustReason  string    `gorm:"type:varchar(200);not null" json:"adjust_reason"`
	AdjustedBy    int64     `gorm:"not null" json:"adjusted_by,string"`
	AdjustedAt    time.Time `gorm:"not null;default:now()" json:"adjusted_at"`
	CreatedAt     time.Time `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt     time.Time `gorm:"not null;default:now()" json:"updated_at"`
}

// TableName 返回课程成绩调整记录表表名。
func (CourseGradeOverride) TableName() string {
	return "course_grade_overrides"
}
