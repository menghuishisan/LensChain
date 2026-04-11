// course_assignment.go
// 模块03 — 课程与教学：请求/响应 DTO 定义（作业 + 提交 + 讨论 + 公告 + 评价 + 成绩）
// 对照 docs/modules/03-课程与教学/03-API接口设计.md

package dto

// ========== 作业管理 DTO ==========

// CreateAssignmentReq 创建作业请求
// POST /api/v1/courses/:id/assignments
type CreateAssignmentReq struct {
	Title               string   `json:"title" binding:"required,max=200"`
	Description         *string  `json:"description"`
	ChapterID           *string  `json:"chapter_id"`
	AssignmentType      int      `json:"assignment_type" binding:"required,oneof=1 2"`
	TotalScore          float64  `json:"total_score" binding:"required,min=1"`
	DeadlineAt          *string  `json:"deadline_at" binding:"required"`
	MaxSubmissions      *int     `json:"max_submissions" binding:"omitempty,min=1"`
	LatePolicy          int      `json:"late_policy" binding:"required,oneof=1 2 3"`
	LateDeductionPerDay *float64 `json:"late_deduction_per_day" binding:"omitempty,min=0,max=100"`
}

// UpdateAssignmentReq 编辑作业请求
// PUT /api/v1/assignments/:id
type UpdateAssignmentReq struct {
	Title               *string  `json:"title" binding:"omitempty,max=200"`
	Description         *string  `json:"description"`
	ChapterID           *string  `json:"chapter_id"`
	AssignmentType      *int     `json:"assignment_type" binding:"omitempty,oneof=1 2"`
	TotalScore          *float64 `json:"total_score" binding:"omitempty,min=1"`
	DeadlineAt          *string  `json:"deadline_at"`
	MaxSubmissions      *int     `json:"max_submissions" binding:"omitempty,min=1"`
	LatePolicy          *int     `json:"late_policy" binding:"omitempty,oneof=1 2 3"`
	LateDeductionPerDay *float64 `json:"late_deduction_per_day" binding:"omitempty,min=0,max=100"`
}

// AssignmentListReq 作业列表查询参数
// GET /api/v1/courses/:id/assignments
type AssignmentListReq struct {
	Page           int `form:"page" binding:"omitempty,min=1"`
	PageSize       int `form:"page_size" binding:"omitempty,min=1,max=100"`
	AssignmentType int `form:"assignment_type" binding:"omitempty,oneof=1 2"`
}

// AssignmentListItem 作业列表项
type AssignmentListItem struct {
	ID                 string  `json:"id"`
	Title              string  `json:"title"`
	AssignmentType     int     `json:"assignment_type"`
	AssignmentTypeText string  `json:"assignment_type_text"`
	TotalScore         float64 `json:"total_score"`
	DeadlineAt         *string `json:"deadline_at"`
	IsPublished        bool    `json:"is_published"`
	SubmitCount        int     `json:"submit_count"`
	TotalStudents      int     `json:"total_students"`
	SortOrder          int     `json:"sort_order"`
}

// AssignmentDetailResp 作业详情响应（含题目）
// GET /api/v1/assignments/:id
type AssignmentDetailResp struct {
	ID                  string               `json:"id"`
	CourseID            string               `json:"course_id"`
	ChapterID           *string              `json:"chapter_id"`
	Title               string               `json:"title"`
	Description         *string              `json:"description"`
	AssignmentType      int                  `json:"assignment_type"`
	AssignmentTypeText  string               `json:"assignment_type_text"`
	TotalScore          float64              `json:"total_score"`
	DeadlineAt          *string              `json:"deadline_at"`
	MaxSubmissions      int                  `json:"max_submissions"`
	LatePolicy          int                  `json:"late_policy"`
	LatePolicyText      string               `json:"late_policy_text"`
	LateDeductionPerDay *float64             `json:"late_deduction_per_day"`
	IsPublished         bool                 `json:"is_published"`
	Questions           []QuestionDetailItem `json:"questions"`
}

// ========== 题目 DTO ==========

// AddQuestionReq 添加题目请求
// POST /api/v1/assignments/:id/questions
type AddQuestionReq struct {
	QuestionType    int     `json:"question_type" binding:"required,oneof=1 2 3 4 5 6 7"`
	Title           string  `json:"title" binding:"required"`
	Options         *string `json:"options"`
	CorrectAnswer   *string `json:"correct_answer"`
	ReferenceAnswer *string `json:"reference_answer"`
	Score           float64 `json:"score" binding:"required,min=0"`
	JudgeConfig     *string `json:"judge_config"`
}

// UpdateQuestionReq 编辑题目请求
// PUT /api/v1/assignment-questions/:id
type UpdateQuestionReq struct {
	QuestionType    *int     `json:"question_type" binding:"omitempty,oneof=1 2 3 4 5 6 7"`
	Title           *string  `json:"title"`
	Options         *string  `json:"options"`
	CorrectAnswer   *string  `json:"correct_answer"`
	ReferenceAnswer *string  `json:"reference_answer"`
	Score           *float64 `json:"score" binding:"omitempty,min=0"`
	JudgeConfig     *string  `json:"judge_config"`
}

// QuestionDetailItem 题目详情项
type QuestionDetailItem struct {
	ID               string  `json:"id"`
	QuestionType     int     `json:"question_type"`
	QuestionTypeText string  `json:"question_type_text"`
	Title            string  `json:"title"`
	Options          *string `json:"options"`
	CorrectAnswer    *string `json:"correct_answer"`
	ReferenceAnswer  *string `json:"reference_answer"`
	Score            float64 `json:"score"`
	JudgeConfig      *string `json:"judge_config"`
	SortOrder        int     `json:"sort_order"`
}

// ========== 提交与批改 DTO ==========

// SubmitAssignmentReq 提交作业请求
// POST /api/v1/assignments/:id/submit
type SubmitAssignmentReq struct {
	Answers []SubmitAnswerReq `json:"answers" binding:"required,min=1,dive"`
}

// SubmitAnswerReq 提交答案项
type SubmitAnswerReq struct {
	QuestionID    string  `json:"question_id" binding:"required"`
	AnswerContent *string `json:"answer_content"`
	AnswerFileURL *string `json:"answer_file_url" binding:"omitempty,url,max=500"`
}

// SubmitAssignmentResp 提交作业响应
// POST /api/v1/assignments/:id/submit
type SubmitAssignmentResp struct {
	SubmissionID         string                `json:"submission_id"`
	SubmissionNo         int                   `json:"submission_no"`
	RemainingSubmissions int                   `json:"remaining_submissions"`
	IsLate               bool                  `json:"is_late"`
	InstantFeedback      SubmitFeedbackSummary `json:"instant_feedback"`
}

// SubmitFeedbackSummary 提交即时反馈
type SubmitFeedbackSummary struct {
	AutoGradedScore float64                `json:"auto_graded_score"`
	AutoGradedTotal float64                `json:"auto_graded_total"`
	Details         []SubmitFeedbackDetail `json:"details"`
}

// SubmitFeedbackDetail 单题即时反馈
type SubmitFeedbackDetail struct {
	QuestionID string   `json:"question_id"`
	IsCorrect  *bool    `json:"is_correct,omitempty"`
	Score      *float64 `json:"score,omitempty"`
	Status     string   `json:"status,omitempty"`
}

// SubmissionListReq 提交列表查询参数
// GET /api/v1/assignments/:id/submissions
type SubmissionListReq struct {
	Page     int    `form:"page" binding:"omitempty,min=1"`
	PageSize int    `form:"page_size" binding:"omitempty,min=1,max=100"`
	Status   int    `form:"status" binding:"omitempty,oneof=1 2 3"`
	Keyword  string `form:"keyword"`
}

// SubmissionListItem 提交列表项
type SubmissionListItem struct {
	ID           string   `json:"id"`
	StudentID    string   `json:"student_id"`
	StudentName  string   `json:"student_name"`
	StudentNo    *string  `json:"student_no"`
	SubmissionNo int      `json:"submission_no"`
	Status       int      `json:"status"`
	StatusText   string   `json:"status_text"`
	TotalScore   *float64 `json:"total_score"`
	IsLate       bool     `json:"is_late"`
	SubmittedAt  string   `json:"submitted_at"`
}

// SubmissionDetailResp 提交详情响应
// GET /api/v1/submissions/:id
type SubmissionDetailResp struct {
	ID                   string                 `json:"id"`
	AssignmentID         string                 `json:"assignment_id"`
	StudentID            string                 `json:"student_id"`
	StudentName          string                 `json:"student_name"`
	SubmissionNo         int                    `json:"submission_no"`
	Status               int                    `json:"status"`
	StatusText           string                 `json:"status_text"`
	TotalScore           *float64               `json:"total_score"`
	IsLate               bool                   `json:"is_late"`
	LateDays             *int                   `json:"late_days"`
	ScoreBeforeDeduction *float64               `json:"score_before_deduction"`
	ScoreAfterDeduction  *float64               `json:"score_after_deduction"`
	TeacherComment       *string                `json:"teacher_comment"`
	SubmittedAt          string                 `json:"submitted_at"`
	GradedAt             *string                `json:"graded_at"`
	Answers              []SubmissionAnswerItem `json:"answers"`
}

// SubmissionAnswerItem 提交答案详情项
type SubmissionAnswerItem struct {
	ID              string   `json:"id"`
	QuestionID      string   `json:"question_id"`
	QuestionTitle   string   `json:"question_title"`
	QuestionType    int      `json:"question_type"`
	AnswerContent   *string  `json:"answer_content"`
	AnswerFileURL   *string  `json:"answer_file_url"`
	IsCorrect       *bool    `json:"is_correct"`
	Score           *float64 `json:"score"`
	TeacherComment  *string  `json:"teacher_comment"`
	AutoJudgeResult *string  `json:"auto_judge_result"`
}

// GradeSubmissionReq 批改提交请求
// POST /api/v1/submissions/:id/grade
type GradeSubmissionReq struct {
	TeacherComment *string          `json:"teacher_comment"`
	Answers        []GradeAnswerReq `json:"answers" binding:"required,min=1,dive"`
}

// GradeAnswerReq 批改答案项
type GradeAnswerReq struct {
	QuestionID     string  `json:"question_id" binding:"required"`
	Score          float64 `json:"score" binding:"min=0"`
	TeacherComment *string `json:"teacher_comment"`
}

// MySubmissionsResp 我的提交列表响应
// GET /api/v1/assignments/:id/my-submissions
type MySubmissionsResp struct {
	Submissions []MySubmissionItem `json:"submissions"`
}

// MySubmissionItem 我的提交项
type MySubmissionItem struct {
	ID           string   `json:"id"`
	SubmissionNo int      `json:"submission_no"`
	Status       int      `json:"status"`
	StatusText   string   `json:"status_text"`
	TotalScore   *float64 `json:"total_score"`
	IsLate       bool     `json:"is_late"`
	SubmittedAt  string   `json:"submitted_at"`
}

// ========== 公告 DTO ==========

// CreateAnnouncementReq 发布公告请求
// POST /api/v1/courses/:id/announcements
type CreateAnnouncementReq struct {
	Title   string `json:"title" binding:"required,max=200"`
	Content string `json:"content" binding:"required"`
}

// UpdateAnnouncementReq 编辑公告请求
// PUT /api/v1/announcements/:id
type UpdateAnnouncementReq struct {
	Title   *string `json:"title" binding:"omitempty,max=200"`
	Content *string `json:"content"`
}

// AnnouncementListReq 公告列表查询参数
// GET /api/v1/courses/:id/announcements
type AnnouncementListReq struct {
	Page     int `form:"page" binding:"omitempty,min=1"`
	PageSize int `form:"page_size" binding:"omitempty,min=1,max=100"`
}

// AnnouncementItem 公告列表项
type AnnouncementItem struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Content     string `json:"content"`
	IsPinned    bool   `json:"is_pinned"`
	TeacherName string `json:"teacher_name"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// ========== 讨论 DTO ==========

// CreateDiscussionReq 创建讨论帖请求
// POST /api/v1/courses/:id/discussions
type CreateDiscussionReq struct {
	Title   string `json:"title" binding:"required,max=200"`
	Content string `json:"content" binding:"required"`
}

// DiscussionListReq 讨论列表查询参数
// GET /api/v1/courses/:id/discussions
type DiscussionListReq struct {
	Page     int    `form:"page" binding:"omitempty,min=1"`
	PageSize int    `form:"page_size" binding:"omitempty,min=1,max=100"`
	Keyword  string `form:"keyword"`
	SortBy   string `form:"sort_by" binding:"omitempty"`
}

// DiscussionListItem 讨论列表项
type DiscussionListItem struct {
	ID            string  `json:"id"`
	Title         string  `json:"title"`
	AuthorID      string  `json:"author_id"`
	AuthorName    string  `json:"author_name"`
	IsPinned      bool    `json:"is_pinned"`
	ReplyCount    int     `json:"reply_count"`
	LikeCount     int     `json:"like_count"`
	IsLiked       bool    `json:"is_liked"`
	LastRepliedAt *string `json:"last_replied_at"`
	CreatedAt     string  `json:"created_at"`
}

// DiscussionDetailResp 讨论详情响应（含回复）
// GET /api/v1/discussions/:id
type DiscussionDetailResp struct {
	ID         string                `json:"id"`
	CourseID   string                `json:"course_id"`
	Title      string                `json:"title"`
	Content    string                `json:"content"`
	AuthorID   string                `json:"author_id"`
	AuthorName string                `json:"author_name"`
	IsPinned   bool                  `json:"is_pinned"`
	ReplyCount int                   `json:"reply_count"`
	LikeCount  int                   `json:"like_count"`
	IsLiked    bool                  `json:"is_liked"`
	CreatedAt  string                `json:"created_at"`
	Replies    []DiscussionReplyItem `json:"replies"`
}

// PinDiscussionReq 置顶/取消置顶请求
// PATCH /api/v1/discussions/:id/pin
type PinDiscussionReq struct {
	IsPinned bool `json:"is_pinned"`
}

// CreateReplyReq 回复讨论请求
// POST /api/v1/discussions/:id/replies
type CreateReplyReq struct {
	Content   string  `json:"content" binding:"required"`
	ReplyToID *string `json:"reply_to_id"`
}

// DiscussionReplyItem 讨论回复项
type DiscussionReplyItem struct {
	ID          string  `json:"id"`
	AuthorID    string  `json:"author_id"`
	AuthorName  string  `json:"author_name"`
	Content     string  `json:"content"`
	ReplyToID   *string `json:"reply_to_id"`
	ReplyToName *string `json:"reply_to_name"`
	CreatedAt   string  `json:"created_at"`
}

// ========== 课程评价 DTO ==========

// CreateEvaluationReq 提交课程评价请求
// POST /api/v1/courses/:id/evaluations
type CreateEvaluationReq struct {
	Rating  int     `json:"rating" binding:"required,min=1,max=5"`
	Comment *string `json:"comment"`
}

// UpdateEvaluationReq 编辑评价请求
// PUT /api/v1/course-evaluations/:id
type UpdateEvaluationReq struct {
	Rating  *int    `json:"rating" binding:"omitempty,min=1,max=5"`
	Comment *string `json:"comment"`
}

// EvaluationListReq 评价列表查询参数
// GET /api/v1/courses/:id/evaluations
type EvaluationListReq struct {
	Page     int `form:"page" binding:"omitempty,min=1"`
	PageSize int `form:"page_size" binding:"omitempty,min=1,max=100"`
}

// EvaluationItem 评价列表项
type EvaluationItem struct {
	ID          string  `json:"id"`
	StudentID   string  `json:"student_id"`
	StudentName string  `json:"student_name"`
	Rating      int     `json:"rating"`
	Comment     *string `json:"comment"`
	CreatedAt   string  `json:"created_at"`
}

// EvaluationSummary 评价汇总
type EvaluationSummary struct {
	AvgRating    float64 `json:"avg_rating"`
	TotalCount   int     `json:"total_count"`
	Distribution [5]int  `json:"distribution"` // 1-5星分布
}

// ========== 成绩管理 DTO ==========

// GradeConfigReq 设置成绩权重请求
// PUT /api/v1/courses/:id/grade-config
type GradeConfigReq struct {
	Items []GradeConfigItem `json:"items" binding:"required,dive"`
}

// GradeConfigItem 成绩权重项
type GradeConfigItem struct {
	AssignmentID string  `json:"assignment_id" binding:"required"`
	Name         string  `json:"name" binding:"required,max=100"`
	Weight       float64 `json:"weight" binding:"required,min=0,max=100"`
}

// GradeConfigResp 成绩权重配置响应
// GET /api/v1/courses/:id/grade-config
type GradeConfigResp struct {
	Items       []GradeConfigItem `json:"items"`
	TotalWeight float64           `json:"total_weight"`
}

// GradeSummaryReq 成绩汇总查询参数
// GET /api/v1/courses/:id/grades
type GradeSummaryReq struct {
	Page      int    `form:"page" binding:"omitempty,min=1"`
	PageSize  int    `form:"page_size" binding:"omitempty,min=1,max=100"`
	Keyword   string `form:"keyword"`
	SortBy    string `form:"sort_by" binding:"omitempty"`
	SortOrder string `form:"sort_order" binding:"omitempty,oneof=asc desc"`
}

// GradeSummaryItem 成绩汇总项
type GradeSummaryItem struct {
	StudentID        string                `json:"student_id"`
	StudentName      string                `json:"student_name"`
	StudentNo        *string               `json:"student_no"`
	AssignmentScores []AssignmentScoreItem `json:"assignment_scores"`
	WeightedTotal    float64               `json:"weighted_total"`
	FinalScore       float64               `json:"final_score"`
	IsAdjusted       bool                  `json:"is_adjusted"`
}

// AssignmentScoreItem 作业成绩项
type AssignmentScoreItem struct {
	AssignmentID string   `json:"assignment_id"`
	Name         string   `json:"name"`
	Score        *float64 `json:"score"`
	Weight       float64  `json:"weight"`
}

// AdjustGradeReq 手动调整成绩请求
// PATCH /api/v1/courses/:id/grades/:student_id
type AdjustGradeReq struct {
	FinalScore float64 `json:"final_score" binding:"min=0"`
	Reason     string  `json:"reason" binding:"required,max=200"`
}

// MyGradesResp 我的成绩响应
// GET /api/v1/courses/:id/my-grades
type MyGradesResp struct {
	AssignmentScores []AssignmentScoreItem `json:"assignment_scores"`
	WeightedTotal    float64               `json:"weighted_total"`
	FinalScore       float64               `json:"final_score"`
	IsAdjusted       bool                  `json:"is_adjusted"`
}
