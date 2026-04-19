// grade.go
// 模块06 — 评测与成绩：请求/响应 DTO 定义。
// 该文件对齐 docs/modules/06-评测与成绩/03-API接口设计.md，覆盖学期、等级映射、审核、申诉、预警、成绩单、分析接口。

package dto

// SemesterReq 学期创建/更新请求。
type SemesterReq struct {
	Name      string `json:"name" binding:"required,max=50"`
	Code      string `json:"code" binding:"required,max=20"`
	StartDate string `json:"start_date" binding:"required"`
	EndDate   string `json:"end_date" binding:"required"`
}

// SemesterListReq 学期列表查询参数。
type SemesterListReq struct {
	Page      int    `form:"page" binding:"omitempty,min=1"`
	PageSize  int    `form:"page_size" binding:"omitempty,min=1,max=100"`
	SortBy    string `form:"sort_by"`
	SortOrder string `form:"sort_order" binding:"omitempty,oneof=asc desc"`
}

// GradeSemesterResp 学期响应。
type GradeSemesterResp struct {
	ID                  string                     `json:"id"`
	SchoolID            *string                    `json:"school_id,omitempty"`
	Name                string                     `json:"name"`
	Code                string                     `json:"code"`
	StartDate           string                     `json:"start_date"`
	EndDate             string                     `json:"end_date"`
	IsCurrent           bool                       `json:"is_current"`
	CreatedAt           *string                    `json:"created_at,omitempty"`
	CourseCount         *int                       `json:"course_count,omitempty"`
	ReviewStatusSummary *ReviewStatusSummaryCounts `json:"review_status_summary,omitempty"`
}

// ReviewStatusSummaryCounts 学期审核状态汇总。
// 该结构对应学期列表中的固定统计键，避免使用不稳定的泛型 map。
type ReviewStatusSummaryCounts struct {
	NotSubmitted int `json:"not_submitted"`
	Pending      int `json:"pending"`
	Approved     int `json:"approved"`
	Rejected     int `json:"rejected"`
}

// GradeSemesterListResp 学期列表响应。
type GradeSemesterListResp struct {
	List       []GradeSemesterResp `json:"list"`
	Pagination PaginationResp      `json:"pagination"`
}

// GradeLevelItem 等级映射项。
type GradeLevelItem struct {
	ID        *string `json:"id,omitempty"`
	LevelName string  `json:"level_name"`
	MinScore  float64 `json:"min_score"`
	MaxScore  float64 `json:"max_score"`
	GPAPoint  float64 `json:"gpa_point"`
	SortOrder *int    `json:"sort_order,omitempty"`
}

// GradeLevelConfigResp 等级映射配置响应。
type GradeLevelConfigResp struct {
	SchoolID string           `json:"school_id"`
	Levels   []GradeLevelItem `json:"levels"`
}

// UpdateGradeLevelConfigsReq 更新等级映射配置请求。
type UpdateGradeLevelConfigsReq struct {
	Levels []GradeLevelItem `json:"levels" binding:"required,min=2,dive"`
}

// SubmitGradeReviewReq 提交成绩审核请求。
type SubmitGradeReviewReq struct {
	CourseID   string  `json:"course_id" binding:"required"`
	SemesterID string  `json:"semester_id" binding:"required"`
	SubmitNote *string `json:"submit_note"`
}

// GradeReviewListReq 审核列表查询参数。
type GradeReviewListReq struct {
	Page       int    `form:"page" binding:"omitempty,min=1"`
	PageSize   int    `form:"page_size" binding:"omitempty,min=1,max=100"`
	Status     int16  `form:"status" binding:"omitempty,oneof=1 2 3 4"`
	SemesterID string `form:"semester_id"`
	CourseID   string `form:"course_id"`
}

// GradeReviewItem 审核列表项。
type GradeReviewItem struct {
	ID              string  `json:"id"`
	CourseID        string  `json:"course_id"`
	CourseName      string  `json:"course_name"`
	SemesterID      string  `json:"semester_id"`
	SemesterName    string  `json:"semester_name"`
	SubmittedBy     string  `json:"submitted_by"`
	SubmittedByName string  `json:"submitted_by_name"`
	Status          int16   `json:"status"`
	StatusText      string  `json:"status_text"`
	SubmittedAt     *string `json:"submitted_at"`
	ReviewedAt      *string `json:"reviewed_at"`
	IsLocked        bool    `json:"is_locked"`
}

// GradeReviewListResp 审核列表响应。
type GradeReviewListResp struct {
	List       []GradeReviewItem `json:"list"`
	Pagination PaginationResp    `json:"pagination"`
}

// GradeReviewDetailResp 审核详情响应。
type GradeReviewDetailResp struct {
	ID              string  `json:"id"`
	CourseID        string  `json:"course_id"`
	CourseName      string  `json:"course_name"`
	SemesterID      string  `json:"semester_id"`
	SemesterName    string  `json:"semester_name"`
	SubmittedBy     string  `json:"submitted_by"`
	SubmittedByName string  `json:"submitted_by_name"`
	Status          int16   `json:"status"`
	StatusText      string  `json:"status_text"`
	SubmitNote      *string `json:"submit_note"`
	SubmittedAt     *string `json:"submitted_at"`
	ReviewedBy      *string `json:"reviewed_by"`
	ReviewedByName  *string `json:"reviewed_by_name"`
	ReviewedAt      *string `json:"reviewed_at"`
	ReviewComment   *string `json:"review_comment"`
	IsLocked        bool    `json:"is_locked"`
	LockedAt        *string `json:"locked_at"`
	UnlockedBy      *string `json:"unlocked_by"`
	UnlockedAt      *string `json:"unlocked_at"`
	UnlockReason    *string `json:"unlock_reason"`
}

// ReviewHandleReq 审核通过/驳回请求。
type ReviewHandleReq struct {
	ReviewComment string `json:"review_comment" binding:"required"`
}

// UnlockGradeReviewReq 解锁成绩请求。
type UnlockGradeReviewReq struct {
	UnlockReason string `json:"unlock_reason" binding:"required"`
}

// GradeReviewStatusResp 审核状态响应。
type GradeReviewStatusResp struct {
	ID          string  `json:"id"`
	CourseID    string  `json:"course_id"`
	SemesterID  string  `json:"semester_id"`
	Status      int16   `json:"status"`
	StatusText  string  `json:"status_text"`
	SubmittedAt *string `json:"submitted_at,omitempty"`
}

// SemesterGradesReq 学期成绩查询参数。
type SemesterGradesReq struct {
	SemesterID string `form:"semester_id"`
}

// SemesterInfo 学期信息。
type SemesterInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Code string `json:"code"`
}

// SemesterGradeItem 学期成绩列表项。
type SemesterGradeItem struct {
	CourseID         string  `json:"course_id"`
	CourseName       string  `json:"course_name"`
	TeacherName      string  `json:"teacher_name"`
	Credits          float64 `json:"credits"`
	FinalScore       float64 `json:"final_score"`
	GradeLevel       string  `json:"grade_level"`
	GPAPoint         float64 `json:"gpa_point"`
	IsAdjusted       bool    `json:"is_adjusted"`
	ReviewStatus     string  `json:"review_status"`
	ReviewStatusText string  `json:"review_status_text"`
}

// SemesterGradeSummary 学期成绩汇总。
type SemesterGradeSummary struct {
	TotalCredits float64 `json:"total_credits"`
	SemesterGPA  float64 `json:"semester_gpa"`
	CourseCount  int     `json:"course_count"`
	PassedCount  int     `json:"passed_count"`
	FailedCount  int     `json:"failed_count"`
}

// SemesterGradesResp 学期成绩响应。
type SemesterGradesResp struct {
	Semester *SemesterInfo         `json:"semester"`
	Grades   []SemesterGradeItem   `json:"grades"`
	Summary  *SemesterGradeSummary `json:"summary"`
}

// GPAResp GPA响应。
type GPAResp struct {
	CumulativeGPA     float64           `json:"cumulative_gpa"`
	CumulativeCredits float64           `json:"cumulative_credits"`
	SemesterList      []GPASemesterItem `json:"semester_list"`
	GPATrend          []float64         `json:"gpa_trend"`
}

// GPASemesterItem GPA 学期项。
// 该结构对应 GPA 接口的 semester_list 固定字段集合。
type GPASemesterItem struct {
	SemesterID   string  `json:"semester_id"`
	SemesterName string  `json:"semester_name"`
	GPA          float64 `json:"gpa"`
	Credits      float64 `json:"credits"`
}

// CreateGradeAppealReq 提交成绩申诉请求。
type CreateGradeAppealReq struct {
	GradeID      string `json:"grade_id" binding:"required"`
	AppealReason string `json:"appeal_reason" binding:"required,min=20"`
}

// GradeAppealListReq 申诉列表查询参数。
type GradeAppealListReq struct {
	Page     int    `form:"page" binding:"omitempty,min=1"`
	PageSize int    `form:"page_size" binding:"omitempty,min=1,max=100"`
	Status   int16  `form:"status" binding:"omitempty,oneof=1 2 3"`
	CourseID string `form:"course_id"`
}

// GradeAppealItem 申诉列表项。
type GradeAppealItem struct {
	ID            string  `json:"id"`
	StudentID     string  `json:"student_id"`
	StudentName   string  `json:"student_name"`
	CourseID      string  `json:"course_id"`
	CourseName    string  `json:"course_name"`
	SemesterID    string  `json:"semester_id"`
	SemesterName  string  `json:"semester_name"`
	OriginalScore float64 `json:"original_score"`
	Status        int16   `json:"status"`
	StatusText    string  `json:"status_text"`
	CreatedAt     string  `json:"created_at"`
}

// GradeAppealListResp 申诉列表响应。
type GradeAppealListResp struct {
	List       []GradeAppealItem `json:"list"`
	Pagination PaginationResp    `json:"pagination"`
}

// GradeAppealDetailResp 申诉详情响应。
type GradeAppealDetailResp struct {
	ID            string   `json:"id"`
	StudentID     string   `json:"student_id"`
	StudentName   string   `json:"student_name"`
	CourseID      string   `json:"course_id"`
	CourseName    string   `json:"course_name"`
	SemesterID    string   `json:"semester_id"`
	SemesterName  string   `json:"semester_name"`
	GradeID       string   `json:"grade_id"`
	OriginalScore float64  `json:"original_score"`
	AppealReason  string   `json:"appeal_reason"`
	Status        int16    `json:"status"`
	StatusText    string   `json:"status_text"`
	HandledBy     *string  `json:"handled_by"`
	HandledByName *string  `json:"handled_by_name"`
	HandledAt     *string  `json:"handled_at"`
	NewScore      *float64 `json:"new_score"`
	HandleComment *string  `json:"handle_comment"`
	CreatedAt     string   `json:"created_at"`
	UpdatedAt     string   `json:"updated_at"`
}

// ApproveGradeAppealReq 同意申诉请求。
type ApproveGradeAppealReq struct {
	NewScore      float64 `json:"new_score" binding:"required"`
	HandleComment string  `json:"handle_comment" binding:"required"`
}

// RejectGradeAppealReq 驳回申诉请求。
type RejectGradeAppealReq struct {
	HandleComment string `json:"handle_comment" binding:"required"`
}

// AcademicWarningListReq 预警列表查询参数。
type AcademicWarningListReq struct {
	Page        int    `form:"page" binding:"omitempty,min=1"`
	PageSize    int    `form:"page_size" binding:"omitempty,min=1,max=100"`
	SemesterID  string `form:"semester_id"`
	WarningType int16  `form:"warning_type" binding:"omitempty,oneof=1 2"`
	Status      int16  `form:"status" binding:"omitempty,oneof=1 2 3"`
	Keyword     string `form:"keyword"`
}

// AcademicWarningItem 预警列表项。
type AcademicWarningItem struct {
	ID              string                `json:"id"`
	StudentID       string                `json:"student_id"`
	StudentName     string                `json:"student_name"`
	StudentNo       string                `json:"student_no"`
	SemesterName    string                `json:"semester_name"`
	WarningType     int16                 `json:"warning_type"`
	WarningTypeText string                `json:"warning_type_text"`
	Detail          AcademicWarningDetail `json:"detail"`
	Status          int16                 `json:"status"`
	StatusText      string                `json:"status_text"`
	CreatedAt       string                `json:"created_at"`
}

// AcademicWarningDetailResp 预警详情响应。
type AcademicWarningDetailResp struct {
	ID              string                `json:"id"`
	StudentID       string                `json:"student_id"`
	StudentName     string                `json:"student_name"`
	StudentNo       string                `json:"student_no"`
	SemesterID      string                `json:"semester_id"`
	SemesterName    string                `json:"semester_name"`
	WarningType     int16                 `json:"warning_type"`
	WarningTypeText string                `json:"warning_type_text"`
	Detail          AcademicWarningDetail `json:"detail"`
	Status          int16                 `json:"status"`
	StatusText      string                `json:"status_text"`
	HandledBy       *string               `json:"handled_by"`
	HandledByName   *string               `json:"handled_by_name"`
	HandledAt       *string               `json:"handled_at"`
	HandleNote      *string               `json:"handle_note"`
	CreatedAt       string                `json:"created_at"`
	UpdatedAt       string                `json:"updated_at"`
}

// AcademicWarningDetail 学业预警详情。
// 该结构按数据库设计中的 detail JSONB 固定字段建模，通过可选字段兼容两类预警明细。
type AcademicWarningDetail struct {
	CurrentGPA      *float64                      `json:"current_gpa,omitempty"`
	FailCount       *int                          `json:"fail_count,omitempty"`
	Threshold       float64                       `json:"threshold"`
	SemesterCourses []AcademicWarningCourseScore  `json:"semester_courses,omitempty"`
	FailedCourses   []AcademicWarningFailedCourse `json:"failed_courses,omitempty"`
}

// AcademicWarningCourseScore 低 GPA 预警课程项。
type AcademicWarningCourseScore struct {
	CourseID   string  `json:"course_id"`
	CourseName string  `json:"course_name"`
	Score      float64 `json:"score"`
	Grade      string  `json:"grade"`
	Credits    float64 `json:"credits"`
}

// AcademicWarningFailedCourse 连续挂科预警课程项。
type AcademicWarningFailedCourse struct {
	CourseID   string  `json:"course_id"`
	CourseName string  `json:"course_name"`
	Score      float64 `json:"score"`
	Semester   string  `json:"semester"`
}

// AcademicWarningListResp 预警列表响应。
type AcademicWarningListResp struct {
	List       []AcademicWarningItem `json:"list"`
	Pagination PaginationResp        `json:"pagination"`
}

// HandleAcademicWarningReq 处理预警请求。
type HandleAcademicWarningReq struct {
	HandleNote string `json:"handle_note" binding:"required"`
}

// WarningConfigResp 预警配置响应。
type WarningConfigResp struct {
	SchoolID           string  `json:"school_id"`
	GPAThreshold       float64 `json:"gpa_threshold"`
	FailCountThreshold int     `json:"fail_count_threshold"`
	IsEnabled          bool    `json:"is_enabled"`
}

// UpdateWarningConfigReq 更新预警配置请求。
type UpdateWarningConfigReq struct {
	GPAThreshold       float64 `json:"gpa_threshold" binding:"required,min=0,max=4"`
	FailCountThreshold int     `json:"fail_count_threshold" binding:"required,min=1"`
	IsEnabled          bool    `json:"is_enabled"`
}

// GenerateTranscriptReq 生成成绩单请求。
type GenerateTranscriptReq struct {
	StudentID   *string  `json:"student_id"`
	SemesterIDs []string `json:"semester_ids" binding:"required,min=1"`
}

// TranscriptResp 成绩单响应。
type TranscriptResp struct {
	ID          string  `json:"id"`
	FileURL     string  `json:"file_url"`
	GeneratedAt string  `json:"generated_at"`
	StudentID   *string `json:"student_id,omitempty"`
	FileSize    *int64  `json:"file_size,omitempty"`
}

// TranscriptListReq 成绩单列表查询参数。
type TranscriptListReq struct {
	Page      int    `form:"page" binding:"omitempty,min=1"`
	PageSize  int    `form:"page_size" binding:"omitempty,min=1,max=100"`
	StudentID string `form:"student_id"`
}

// TranscriptListItem 成绩单列表项。
type TranscriptListItem struct {
	ID               string   `json:"id"`
	StudentID        string   `json:"student_id"`
	StudentName      string   `json:"student_name"`
	FileURL          string   `json:"file_url"`
	FileSize         int64    `json:"file_size"`
	IncludeSemesters []string `json:"include_semesters"`
	GeneratedAt      string   `json:"generated_at"`
	ExpiresAt        *string  `json:"expires_at"`
}

// TranscriptListResp 成绩单列表响应。
type TranscriptListResp struct {
	List       []TranscriptListItem `json:"list"`
	Pagination PaginationResp       `json:"pagination"`
}

// ScoreDistributionItem 分数段分布项。
type ScoreDistributionItem struct {
	Range string `json:"range"`
	Count int    `json:"count"`
}

// CoursePerformanceItem 课程表现项。
type CoursePerformanceItem struct {
	CourseName   string  `json:"course_name"`
	AverageScore float64 `json:"average_score"`
	PassRate     float64 `json:"pass_rate"`
}

// SchoolComparisonItem 学校对比项。
type SchoolComparisonItem struct {
	SchoolName   string  `json:"school_name"`
	StudentCount int     `json:"student_count"`
	AverageGPA   float64 `json:"average_gpa"`
}

// CourseGradeAnalyticsResp 课程成绩分析响应。
type CourseGradeAnalyticsResp struct {
	CourseID          string                  `json:"course_id"`
	CourseName        string                  `json:"course_name"`
	SemesterName      string                  `json:"semester_name"`
	StudentCount      int                     `json:"student_count"`
	AverageScore      float64                 `json:"average_score"`
	MedianScore       float64                 `json:"median_score"`
	MaxScore          float64                 `json:"max_score"`
	MinScore          float64                 `json:"min_score"`
	PassRate          float64                 `json:"pass_rate"`
	GradeDistribution map[string]int          `json:"grade_distribution"`
	ScoreDistribution []ScoreDistributionItem `json:"score_distribution"`
}

// SchoolGradeAnalyticsResp 全校成绩分析响应。
type SchoolGradeAnalyticsResp struct {
	SemesterName    string                  `json:"semester_name"`
	TotalStudents   int                     `json:"total_students"`
	TotalCourses    int                     `json:"total_courses"`
	AverageGPA      float64                 `json:"average_gpa"`
	GPADistribution []ScoreDistributionItem `json:"gpa_distribution"`
	FailRate        float64                 `json:"fail_rate"`
	WarningCount    int                     `json:"warning_count"`
	TopCourses      []CoursePerformanceItem `json:"top_courses"`
	BottomCourses   []CoursePerformanceItem `json:"bottom_courses"`
}

// PlatformGradeAnalyticsResp 平台成绩总览响应。
type PlatformGradeAnalyticsResp struct {
	TotalSchools       int                    `json:"total_schools"`
	TotalStudents      int                    `json:"total_students"`
	PlatformAverageGPA float64                `json:"platform_average_gpa"`
	SchoolComparison   []SchoolComparisonItem `json:"school_comparison"`
}
