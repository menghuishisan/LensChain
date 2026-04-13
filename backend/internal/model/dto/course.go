// course.go
// 模块03 — 课程与教学：请求/响应 DTO 定义（课程管理 + 章节课时 + 选课 + 课表 + 共享 + 统计）
// 对照 docs/modules/03-课程与教学/03-API接口设计.md

package dto

import "time"

// ========== 课程管理 DTO ==========

// CreateCourseReq 创建课程请求
// POST /api/v1/courses
type CreateCourseReq struct {
	Title       string   `json:"title" binding:"required,max=200"`
	Description *string  `json:"description"`
	CoverURL    *string  `json:"cover_url" binding:"omitempty,url,max=500"`
	CourseType  int      `json:"course_type" binding:"required,oneof=1 2 3 4"`
	Difficulty  int      `json:"difficulty" binding:"required,oneof=1 2 3 4"`
	Topic       string   `json:"topic" binding:"required,max=50"`
	Credits     *float64 `json:"credits" binding:"omitempty,min=0,max=99.9"`
	SemesterID  *string  `json:"semester_id"`
	StartAt     *string  `json:"start_at"`
	EndAt       *string  `json:"end_at"`
	MaxStudents *int     `json:"max_students" binding:"omitempty,min=1"`
}

// CreateCourseResp 创建课程响应
// POST /api/v1/courses
type CreateCourseResp struct {
	ID         string  `json:"id"`
	Title      string  `json:"title"`
	Status     int     `json:"status"`
	StatusText string  `json:"status_text"`
	InviteCode string  `json:"invite_code"`
	CoverURL   *string `json:"cover_url"`
}

// UpdateCourseReq 编辑课程请求
// PUT /api/v1/courses/:id
type UpdateCourseReq struct {
	Title       *string  `json:"title" binding:"omitempty,max=200"`
	Description *string  `json:"description"`
	CoverURL    *string  `json:"cover_url" binding:"omitempty,url,max=500"`
	CourseType  *int     `json:"course_type" binding:"omitempty,oneof=1 2 3 4"`
	Difficulty  *int     `json:"difficulty" binding:"omitempty,oneof=1 2 3 4"`
	Topic       *string  `json:"topic" binding:"omitempty,max=50"`
	Credits     *float64 `json:"credits" binding:"omitempty,min=0,max=99.9"`
	SemesterID  *string  `json:"semester_id"`
	StartAt     *string  `json:"start_at"`
	EndAt       *string  `json:"end_at"`
	MaxStudents *int     `json:"max_students" binding:"omitempty,min=1"`
}

// CourseListReq 课程列表查询参数（教师视角）
// GET /api/v1/courses
type CourseListReq struct {
	Page       int    `form:"page" binding:"omitempty,min=1"`
	PageSize   int    `form:"page_size" binding:"omitempty,min=1,max=100"`
	Keyword    string `form:"keyword"`
	Status     int    `form:"status" binding:"omitempty,oneof=1 2 3 4 5"`
	CourseType int    `form:"course_type" binding:"omitempty,oneof=1 2 3 4"`
	SortBy     string `form:"sort_by" binding:"omitempty"`
	SortOrder  string `form:"sort_order" binding:"omitempty,oneof=asc desc"`
}

// CourseListItem 课程列表项
type CourseListItem struct {
	ID             string  `json:"id"`
	Title          string  `json:"title"`
	CoverURL       *string `json:"cover_url"`
	CourseType     int     `json:"course_type"`
	CourseTypeText string  `json:"course_type_text"`
	Difficulty     int     `json:"difficulty"`
	DifficultyText string  `json:"difficulty_text"`
	Topic          string  `json:"topic"`
	Status         int     `json:"status"`
	StatusText     string  `json:"status_text"`
	IsShared       bool    `json:"is_shared"`
	StudentCount   int     `json:"student_count"`
	StartAt        *string `json:"start_at"`
	EndAt          *string `json:"end_at"`
	CreatedAt      string  `json:"created_at"`
}

// CourseDetailResp 课程详情响应
// GET /api/v1/courses/:id
type CourseDetailResp struct {
	ID             string   `json:"id"`
	Title          string   `json:"title"`
	Description    *string  `json:"description"`
	CoverURL       *string  `json:"cover_url"`
	CourseType     int      `json:"course_type"`
	CourseTypeText string   `json:"course_type_text"`
	Difficulty     int      `json:"difficulty"`
	DifficultyText string   `json:"difficulty_text"`
	Topic          string   `json:"topic"`
	Status         int      `json:"status"`
	StatusText     string   `json:"status_text"`
	IsShared       bool     `json:"is_shared"`
	InviteCode     *string  `json:"invite_code"`
	Credits        *float64 `json:"credits"`
	SemesterID     *string  `json:"semester_id"`
	StartAt        *string  `json:"start_at"`
	EndAt          *string  `json:"end_at"`
	MaxStudents    *int     `json:"max_students"`
	StudentCount   int      `json:"student_count"`
	TeacherID      string   `json:"teacher_id"`
	TeacherName    string   `json:"teacher_name"`
	ClonedFromID   *string  `json:"cloned_from_id"`
	CreatedAt      string   `json:"created_at"`
	UpdatedAt      string   `json:"updated_at"`
}

// ToggleShareReq 切换共享状态请求
// PATCH /api/v1/courses/:id/share
type ToggleShareReq struct {
	IsShared bool `json:"is_shared"`
}

// ========== 章节 DTO ==========

// CreateChapterReq 创建章节请求
// POST /api/v1/courses/:id/chapters
type CreateChapterReq struct {
	Title       string  `json:"title" binding:"required,max=200"`
	Description *string `json:"description"`
}

// UpdateChapterReq 编辑章节请求
// PUT /api/v1/chapters/:id
type UpdateChapterReq struct {
	Title       *string `json:"title" binding:"omitempty,max=200"`
	Description *string `json:"description"`
}

// SortItemReq 排序项（通用）
type SortItemReq struct {
	ID        string `json:"id" binding:"required"`
	SortOrder int    `json:"sort_order" binding:"min=0"`
}

// SortReq 排序请求
type SortReq struct {
	Items []SortItemReq `json:"items" binding:"required,min=1,dive"`
}

// ChapterWithLessonsResp 章节（含课时列表）响应
type ChapterWithLessonsResp struct {
	ID          string           `json:"id"`
	Title       string           `json:"title"`
	Description *string          `json:"description"`
	SortOrder   int              `json:"sort_order"`
	Lessons     []LessonListItem `json:"lessons"`
}

// ========== 课时 DTO ==========

// CreateLessonReq 创建课时请求
// POST /api/v1/chapters/:id/lessons
type CreateLessonReq struct {
	Title            string  `json:"title" binding:"required,max=200"`
	ContentType      int     `json:"content_type" binding:"required,oneof=1 2 3 4"`
	Content          *string `json:"content"`
	VideoURL         *string `json:"video_url" binding:"omitempty,url,max=500"`
	VideoDuration    *int    `json:"video_duration" binding:"omitempty,min=0"`
	ExperimentID     *string `json:"experiment_id"`
	EstimatedMinutes *int    `json:"estimated_minutes" binding:"omitempty,min=1"`
}

// UpdateLessonReq 编辑课时请求
// PUT /api/v1/lessons/:id
type UpdateLessonReq struct {
	Title            *string `json:"title" binding:"omitempty,max=200"`
	ContentType      *int    `json:"content_type" binding:"omitempty,oneof=1 2 3 4"`
	Content          *string `json:"content"`
	VideoURL         *string `json:"video_url" binding:"omitempty,url,max=500"`
	VideoDuration    *int    `json:"video_duration" binding:"omitempty,min=0"`
	ExperimentID     *string `json:"experiment_id"`
	EstimatedMinutes *int    `json:"estimated_minutes" binding:"omitempty,min=1"`
}

// LessonListItem 课时列表项
type LessonListItem struct {
	ID               string  `json:"id"`
	Title            string  `json:"title"`
	ContentType      int     `json:"content_type"`
	ContentTypeText  string  `json:"content_type_text"`
	VideoDuration    *int    `json:"video_duration"`
	ExperimentID     *string `json:"experiment_id"`
	EstimatedMinutes *int    `json:"estimated_minutes"`
	SortOrder        int     `json:"sort_order"`
}

// LessonDetailResp 课时详情响应
// GET /api/v1/lessons/:id
type LessonDetailResp struct {
	ID               string                 `json:"id"`
	ChapterID        string                 `json:"chapter_id"`
	CourseID         string                 `json:"course_id"`
	Title            string                 `json:"title"`
	ContentType      int                    `json:"content_type"`
	ContentTypeText  string                 `json:"content_type_text"`
	Content          *string                `json:"content"`
	VideoURL         *string                `json:"video_url"`
	VideoDuration    *int                   `json:"video_duration"`
	ExperimentID     *string                `json:"experiment_id"`
	EstimatedMinutes *int                   `json:"estimated_minutes"`
	SortOrder        int                    `json:"sort_order"`
	Attachments      []LessonAttachmentItem `json:"attachments"`
}

// LessonAttachmentItem 课时附件项
type LessonAttachmentItem struct {
	ID       string `json:"id"`
	FileName string `json:"file_name"`
	FileURL  string `json:"file_url"`
	FileSize int64  `json:"file_size"`
	FileType string `json:"file_type"`
}

// UploadAttachmentReq 上传附件请求
// POST /api/v1/lessons/:id/attachments
type UploadAttachmentReq struct {
	FileName string `json:"file_name" binding:"required,max=200"`
	FileURL  string `json:"file_url" binding:"required,url,max=500"`
	FileSize int64  `json:"file_size" binding:"required,min=1"`
	FileType string `json:"file_type" binding:"required,max=100"`
}

// ========== 选课 DTO ==========

// JoinCourseReq 邀请码加入课程请求
// POST /api/v1/courses/join
type JoinCourseReq struct {
	InviteCode string `json:"invite_code" binding:"required,min=6,max=10"`
}

// AddStudentReq 教师添加学生请求
// POST /api/v1/courses/:id/students
type AddStudentReq struct {
	StudentID string `json:"student_id" binding:"required"`
}

// BatchAddStudentsReq 批量添加学生请求
// POST /api/v1/courses/:id/students/batch
type BatchAddStudentsReq struct {
	StudentIDs []string `json:"student_ids" binding:"required,min=1"`
}

// StudentListReq 学生列表查询参数
// GET /api/v1/courses/:id/students
type StudentListReq struct {
	Page     int    `form:"page" binding:"omitempty,min=1"`
	PageSize int    `form:"page_size" binding:"omitempty,min=1,max=100"`
	Keyword  string `form:"keyword"`
}

// EnrolledStudentItem 已选课学生项
type EnrolledStudentItem struct {
	ID             string  `json:"id"`
	Name           string  `json:"name"`
	StudentNo      *string `json:"student_no"`
	College        *string `json:"college"`
	Major          *string `json:"major"`
	ClassName      *string `json:"class_name"`
	JoinMethod     int     `json:"join_method"`
	JoinMethodText string  `json:"join_method_text"`
	JoinedAt       string  `json:"joined_at"`
	Progress       float64 `json:"progress"`
}

// ========== 课表 DTO ==========

// SetScheduleReq 设置课程时间表请求
// PUT /api/v1/courses/:id/schedules
type SetScheduleReq struct {
	Schedules []ScheduleItemReq `json:"schedules" binding:"required,dive"`
}

// ScheduleItemReq 课表项
type ScheduleItemReq struct {
	DayOfWeek int     `json:"day_of_week" binding:"required,min=1,max=7"`
	StartTime string  `json:"start_time" binding:"required"`
	EndTime   string  `json:"end_time" binding:"required"`
	Location  *string `json:"location" binding:"omitempty,max=100"`
}

// ScheduleItemResp 课表项响应
type ScheduleItemResp struct {
	ID        string  `json:"id"`
	DayOfWeek int     `json:"day_of_week"`
	StartTime string  `json:"start_time"`
	EndTime   string  `json:"end_time"`
	Location  *string `json:"location"`
}

// MyScheduleReq 我的课表查询参数
// GET /api/v1/my-schedule
type MyScheduleReq struct {
	Week string `form:"week"` // 可选，格式 2026-W15
}

// MyScheduleItem 我的课表项
type MyScheduleItem struct {
	CourseID    string  `json:"course_id"`
	CourseTitle string  `json:"course_title"`
	TeacherName string  `json:"teacher_name"`
	DayOfWeek   int     `json:"day_of_week"`
	StartTime   string  `json:"start_time"`
	EndTime     string  `json:"end_time"`
	Location    *string `json:"location"`
}

// MyScheduleResp 我的周课程表响应
// GET /api/v1/my-schedule
type MyScheduleResp struct {
	Schedules []*MyScheduleItem `json:"schedules"`
}

// ========== 共享课程库 DTO ==========

// SharedCourseListReq 共享课程列表查询参数
// GET /api/v1/shared-courses
type SharedCourseListReq struct {
	Page       int    `form:"page" binding:"omitempty,min=1"`
	PageSize   int    `form:"page_size" binding:"omitempty,min=1,max=100"`
	Keyword    string `form:"keyword"`
	CourseType int    `form:"course_type" binding:"omitempty,oneof=1 2 3 4"`
	Difficulty int    `form:"difficulty" binding:"omitempty,oneof=1 2 3 4"`
	Topic      string `form:"topic"`
}

// SharedCourseItem 共享课程列表项
type SharedCourseItem struct {
	ID             string  `json:"id"`
	Title          string  `json:"title"`
	Description    *string `json:"description"`
	CoverURL       *string `json:"cover_url"`
	CourseType     int     `json:"course_type"`
	CourseTypeText string  `json:"course_type_text"`
	Difficulty     int     `json:"difficulty"`
	DifficultyText string  `json:"difficulty_text"`
	Topic          string  `json:"topic"`
	TeacherName    string  `json:"teacher_name"`
	SchoolName     string  `json:"school_name"`
	StudentCount   int     `json:"student_count"`
	Rating         float64 `json:"rating"`
}

// ========== 课程统计 DTO ==========

// CourseOverviewStatsResp 课程概览统计响应
// GET /api/v1/courses/:id/statistics/overview
type CourseOverviewStatsResp struct {
	StudentCount    int     `json:"student_count"`
	LessonCount     int     `json:"lesson_count"`
	AssignmentCount int     `json:"assignment_count"`
	AvgProgress     float64 `json:"avg_progress"`
	AvgScore        float64 `json:"avg_score"`
	CompletionRate  float64 `json:"completion_rate"`
	TotalStudyHours float64 `json:"total_study_hours"`
}

// AssignmentStatsResp 作业统计响应
// GET /api/v1/courses/:id/statistics/assignments
type AssignmentStatsResp struct {
	Assignments []AssignmentStatItem `json:"assignments"`
}

// AssignmentStatItem 单个作业统计项
type AssignmentStatItem struct {
	ID            string  `json:"id"`
	Title         string  `json:"title"`
	SubmitCount   int     `json:"submit_count"`
	TotalStudents int     `json:"total_students"`
	SubmitRate    float64 `json:"submit_rate"`
	AvgScore      float64 `json:"avg_score"`
	MaxScore      float64 `json:"max_score"`
	MinScore      float64 `json:"min_score"`
}

// ========== 学生视角 DTO ==========

// MyCourseListReq 我的课程列表查询参数
// GET /api/v1/my-courses
type MyCourseListReq struct {
	Page     int `form:"page" binding:"omitempty,min=1"`
	PageSize int `form:"page_size" binding:"omitempty,min=1,max=100"`
	Status   int `form:"status" binding:"omitempty,oneof=2 3 4"`
}

// MyCourseItem 我的课程列表项
type MyCourseItem struct {
	ID             string  `json:"id"`
	Title          string  `json:"title"`
	CoverURL       *string `json:"cover_url"`
	CourseType     int     `json:"course_type"`
	CourseTypeText string  `json:"course_type_text"`
	TeacherName    string  `json:"teacher_name"`
	Status         int     `json:"status"`
	StatusText     string  `json:"status_text"`
	Progress       float64 `json:"progress"`
	JoinedAt       string  `json:"joined_at"`
}

// ========== 学习进度 DTO ==========

// UpdateProgressReq 更新学习进度请求
// POST /api/v1/lessons/:id/progress
type UpdateProgressReq struct {
	VideoProgress          *int `json:"video_progress" binding:"omitempty,min=0"`
	StudyDurationIncrement int  `json:"study_duration_increment" binding:"min=0"`
	Completed              bool `json:"completed"`
}

// MyProgressResp 我的课程学习进度响应
// GET /api/v1/courses/:id/my-progress
type MyProgressResp struct {
	CourseID        string               `json:"course_id"`
	TotalLessons    int                  `json:"total_lessons"`
	CompletedCount  int                  `json:"completed_count"`
	Progress        float64              `json:"progress"`
	TotalStudyHours float64              `json:"total_study_hours"`
	Lessons         []LessonProgressItem `json:"lessons"`
}

// LessonProgressItem 课时学习进度项
type LessonProgressItem struct {
	LessonID       string  `json:"lesson_id"`
	LessonTitle    string  `json:"lesson_title"`
	ChapterTitle   string  `json:"chapter_title"`
	Status         int     `json:"status"`
	StatusText     string  `json:"status_text"`
	VideoProgress  int     `json:"video_progress"`
	VideoDuration  *int    `json:"video_duration"`
	StudyDuration  int     `json:"study_duration"`
	CompletedAt    *string `json:"completed_at"`
	LastAccessedAt *string `json:"last_accessed_at"`
}

// StudentsProgressReq 全班学习进度查询参数
// GET /api/v1/courses/:id/students-progress
type StudentsProgressReq struct {
	Page     int    `form:"page" binding:"omitempty,min=1"`
	PageSize int    `form:"page_size" binding:"omitempty,min=1,max=100"`
	Keyword  string `form:"keyword"`
}

// StudentProgressItem 学生学习进度项
type StudentProgressItem struct {
	StudentID       string  `json:"student_id"`
	StudentName     string  `json:"student_name"`
	StudentNo       *string `json:"student_no"`
	CompletedCount  int     `json:"completed_count"`
	TotalLessons    int     `json:"total_lessons"`
	Progress        float64 `json:"progress"`
	TotalStudyHours float64 `json:"total_study_hours"`
	LastAccessedAt  *string `json:"last_accessed_at"`
}

// ========== 时间解析辅助 ==========

// ParseTime 解析时间字符串（ISO 8601）
func ParseTime(s string) (*time.Time, error) {
	if s == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return nil, err
	}
	return &t, nil
}
