// course.go
// 模块03 — 课程与教学：枚举常量定义
// 对照 docs/modules/03-课程与教学/02-数据库设计.md
// 包含课程状态、课程类型、难度、内容类型、加入方式、作业类型、题目类型、
// 提交状态、迟交策略、学习进度状态等枚举

package enum

// ========== 课程状态（courses.status） ==========

const (
	CourseStatusDraft     = 1 // 草稿
	CourseStatusPublished = 2 // 已发布
	CourseStatusActive    = 3 // 进行中
	CourseStatusEnded     = 4 // 已结束
	CourseStatusArchived  = 5 // 已归档
)

// CourseStatusText 课程状态文本映射
var CourseStatusText = map[int16]string{
	CourseStatusDraft:     "草稿",
	CourseStatusPublished: "已发布",
	CourseStatusActive:    "进行中",
	CourseStatusEnded:     "已结束",
	CourseStatusArchived:  "已归档",
}

// GetCourseStatusText 获取课程状态文本
func GetCourseStatusText(status int16) string {
	if text, ok := CourseStatusText[status]; ok {
		return text
	}
	return "未知"
}

// IsValidCourseStatus 校验课程状态是否合法
func IsValidCourseStatus(status int16) bool {
	_, ok := CourseStatusText[status]
	return ok
}

// ========== 课程类型（courses.course_type） ==========

const (
	CourseTypeTheory  = 1 // 理论课
	CourseTypeLab     = 2 // 实验课
	CourseTypeMixed   = 3 // 混合课
	CourseTypeProject = 4 // 项目实战
)

// CourseTypeText 课程类型文本映射
var CourseTypeText = map[int16]string{
	CourseTypeTheory:  "理论课",
	CourseTypeLab:     "实验课",
	CourseTypeMixed:   "混合课",
	CourseTypeProject: "项目实战",
}

// GetCourseTypeText 获取课程类型文本
func GetCourseTypeText(t int16) string {
	if text, ok := CourseTypeText[t]; ok {
		return text
	}
	return "未知"
}

// IsValidCourseType 校验课程类型是否合法
func IsValidCourseType(t int16) bool {
	_, ok := CourseTypeText[t]
	return ok
}

// ========== 课程难度（courses.difficulty） ==========

const (
	DifficultyBeginner     = 1 // 入门
	DifficultyIntermediate = 2 // 进阶
	DifficultyAdvanced     = 3 // 高级
	DifficultyResearch     = 4 // 研究
)

// DifficultyText 课程难度文本映射
var DifficultyText = map[int16]string{
	DifficultyBeginner:     "入门",
	DifficultyIntermediate: "进阶",
	DifficultyAdvanced:     "高级",
	DifficultyResearch:     "研究",
}

// GetDifficultyText 获取课程难度文本
func GetDifficultyText(d int16) string {
	if text, ok := DifficultyText[d]; ok {
		return text
	}
	return "未知"
}

// IsValidDifficulty 校验课程难度是否合法
func IsValidDifficulty(d int16) bool {
	_, ok := DifficultyText[d]
	return ok
}

// ========== 课时内容类型（lessons.content_type） ==========

const (
	ContentTypeVideo      = 1 // 视频
	ContentTypeRichText   = 2 // 图文
	ContentTypeAttachment = 3 // 附件
	ContentTypeExperiment = 4 // 实验
)

// ContentTypeText 课时内容类型文本映射
var ContentTypeText = map[int16]string{
	ContentTypeVideo:      "视频",
	ContentTypeRichText:   "图文",
	ContentTypeAttachment: "附件",
	ContentTypeExperiment: "实验",
}

// GetContentTypeText 获取课时内容类型文本
func GetContentTypeText(t int16) string {
	if text, ok := ContentTypeText[t]; ok {
		return text
	}
	return "未知"
}

// IsValidContentType 校验课时内容类型是否合法
func IsValidContentType(t int16) bool {
	_, ok := ContentTypeText[t]
	return ok
}

// ========== 加入方式（course_enrollments.join_method） ==========

const (
	JoinMethodTeacher = 1 // 教师指定
	JoinMethodInvite  = 2 // 邀请码加入
)

// JoinMethodText 加入方式文本映射
var JoinMethodText = map[int16]string{
	JoinMethodTeacher: "教师指定",
	JoinMethodInvite:  "邀请码加入",
}

// GetJoinMethodText 获取加入方式文本
func GetJoinMethodText(m int16) string {
	if text, ok := JoinMethodText[m]; ok {
		return text
	}
	return "未知"
}

// IsValidJoinMethod 校验加入方式是否合法
func IsValidJoinMethod(m int16) bool {
	_, ok := JoinMethodText[m]
	return ok
}

// ========== 作业类型（assignments.assignment_type） ==========

const (
	AssignmentTypeHomework = 1 // 作业
	AssignmentTypeQuiz     = 2 // 测验
)

// AssignmentTypeText 作业类型文本映射
var AssignmentTypeText = map[int16]string{
	AssignmentTypeHomework: "作业",
	AssignmentTypeQuiz:     "测验",
}

// GetAssignmentTypeText 获取作业类型文本
func GetAssignmentTypeText(t int16) string {
	if text, ok := AssignmentTypeText[t]; ok {
		return text
	}
	return "未知"
}

// IsValidAssignmentType 校验作业类型是否合法
func IsValidAssignmentType(t int16) bool {
	_, ok := AssignmentTypeText[t]
	return ok
}

// ========== 题目类型（assignment_questions.question_type） ==========

const (
	QuestionTypeSingleChoice = 1 // 单选题
	QuestionTypeMultiChoice  = 2 // 多选题
	QuestionTypeTrueFalse    = 3 // 判断题
	QuestionTypeFillBlank    = 4 // 填空题
	QuestionTypeShortAnswer  = 5 // 简答题
	QuestionTypeCoding       = 6 // 编程题
	QuestionTypeLabReport    = 7 // 实验报告
)

// QuestionTypeText 题目类型文本映射
var QuestionTypeText = map[int16]string{
	QuestionTypeSingleChoice: "单选题",
	QuestionTypeMultiChoice:  "多选题",
	QuestionTypeTrueFalse:    "判断题",
	QuestionTypeFillBlank:    "填空题",
	QuestionTypeShortAnswer:  "简答题",
	QuestionTypeCoding:       "编程题",
	QuestionTypeLabReport:    "实验报告",
}

// GetQuestionTypeText 获取题目类型文本
func GetQuestionTypeText(t int16) string {
	if text, ok := QuestionTypeText[t]; ok {
		return text
	}
	return "未知"
}

// IsValidQuestionType 校验题目类型是否合法
func IsValidQuestionType(t int16) bool {
	_, ok := QuestionTypeText[t]
	return ok
}

// IsObjectiveQuestion 判断是否为客观题（可自动批改）
func IsObjectiveQuestion(t int16) bool {
	return t == QuestionTypeSingleChoice ||
		t == QuestionTypeMultiChoice ||
		t == QuestionTypeTrueFalse ||
		t == QuestionTypeFillBlank
}

// ========== 提交状态（assignment_submissions.status） ==========

const (
	SubmissionStatusSubmitted = 1 // 已提交
	SubmissionStatusReviewing = 2 // 待批改
	SubmissionStatusGraded    = 3 // 已批改
)

// SubmissionStatusText 提交状态文本映射
var SubmissionStatusText = map[int16]string{
	SubmissionStatusSubmitted: "已提交",
	SubmissionStatusReviewing: "待批改",
	SubmissionStatusGraded:    "已批改",
}

// GetSubmissionStatusText 获取提交状态文本
func GetSubmissionStatusText(s int16) string {
	if text, ok := SubmissionStatusText[s]; ok {
		return text
	}
	return "未知"
}

// IsValidSubmissionStatus 校验提交状态是否合法
func IsValidSubmissionStatus(s int16) bool {
	_, ok := SubmissionStatusText[s]
	return ok
}

// ========== 迟交策略（assignments.late_policy） ==========

const (
	LatePolicyNotAllowed    = 1 // 不允许迟交
	LatePolicyWithDeduction = 2 // 允许迟交（扣分）
	LatePolicyNoDeduction   = 3 // 允许迟交（不扣分）
)

// LatePolicyText 迟交策略文本映射
var LatePolicyText = map[int16]string{
	LatePolicyNotAllowed:    "不允许迟交",
	LatePolicyWithDeduction: "允许迟交（扣分）",
	LatePolicyNoDeduction:   "允许迟交（不扣分）",
}

// GetLatePolicyText 获取迟交策略文本
func GetLatePolicyText(p int16) string {
	if text, ok := LatePolicyText[p]; ok {
		return text
	}
	return "未知"
}

// IsValidLatePolicy 校验迟交策略是否合法
func IsValidLatePolicy(p int16) bool {
	_, ok := LatePolicyText[p]
	return ok
}

// ========== 学习进度状态（learning_progresses.status） ==========

const (
	LearningStatusNotStarted = 1 // 未开始
	LearningStatusInProgress = 2 // 进行中
	LearningStatusCompleted  = 3 // 已完成
)

// LearningStatusText 学习进度状态文本映射
var LearningStatusText = map[int16]string{
	LearningStatusNotStarted: "未开始",
	LearningStatusInProgress: "进行中",
	LearningStatusCompleted:  "已完成",
}

// GetLearningStatusText 获取学习进度状态文本
func GetLearningStatusText(s int16) string {
	if text, ok := LearningStatusText[s]; ok {
		return text
	}
	return "未知"
}

// IsValidLearningStatus 校验学习进度状态是否合法
func IsValidLearningStatus(s int16) bool {
	_, ok := LearningStatusText[s]
	return ok
}
