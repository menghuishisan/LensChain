// course.go
// 该文件集中维护模块03“课程与教学”相关的业务错误码，覆盖课程、章节、课时、选课、作业、
// 讨论与统计导出等子域。这样课程模块各层在返回错误时可以直接复用文档定义的语义。

package errcode

import "net/http"

var (
	// 课程管理
	ErrCourseNotFound     = New(40404, http.StatusNotFound, "课程不存在")
	ErrChapterNotFound    = New(40405, http.StatusNotFound, "章节不存在")
	ErrLessonNotFound     = New(40406, http.StatusNotFound, "课时不存在")
	ErrAssignmentNotFound = New(40407, http.StatusNotFound, "作业不存在")
	ErrSubmissionNotFound = New(40408, http.StatusNotFound, "提交记录不存在")
	ErrDiscussionNotFound = New(40409, http.StatusNotFound, "讨论帖不存在")
	ErrReplyNotFound      = New(40412, http.StatusNotFound, "回复不存在")
	ErrEvaluationNotFound = New(40413, http.StatusNotFound, "评价不存在")

	// 权限
	ErrNotCourseTeacher = New(40301, http.StatusForbidden, "非课程负责教师")
	ErrNotCourseStudent = New(40304, http.StatusForbidden, "未加入该课程")

	// 业务冲突
	ErrCourseNotDraft          = New(40911, http.StatusConflict, "仅草稿状态的课程可删除")
	ErrCourseAlreadyPublished  = New(40912, http.StatusConflict, "课程已发布")
	ErrInviteCodeInvalid       = New(40410, http.StatusNotFound, "邀请码无效")
	ErrAlreadyEnrolled         = New(40913, http.StatusConflict, "您已加入该课程")
	ErrCourseStudentFull       = New(40914, http.StatusConflict, "课程人数已满")
	ErrSubmissionExceedMax     = New(40915, http.StatusConflict, "已达最大提交次数")
	ErrAssignmentDeadline      = New(40015, http.StatusBadRequest, "作业已截止且不允许迟交")
	ErrAlreadyEvaluated        = New(40916, http.StatusConflict, "已评价过该课程")
	ErrCourseProgressRateLimit = New(42912, http.StatusTooManyRequests, "视频进度上报过于频繁，请稍后再试")
)
