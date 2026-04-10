// course.go
// 模块03 — 课程与教学模块错误码
// 对照 docs/modules/03-课程与教学/03-API接口设计.md

package errcode

import "net/http"

var (
	// 课程管理
	ErrCourseNotFound        = New(40404, http.StatusNotFound, "课程不存在")
	ErrChapterNotFound       = New(40405, http.StatusNotFound, "章节不存在")
	ErrLessonNotFound        = New(40406, http.StatusNotFound, "课时不存在")
	ErrAssignmentNotFound    = New(40407, http.StatusNotFound, "作业不存在")
	ErrSubmissionNotFound    = New(40408, http.StatusNotFound, "提交记录不存在")
	ErrDiscussionNotFound    = New(40409, http.StatusNotFound, "讨论帖不存在")

	// 权限
	ErrNotCourseTeacher      = New(40301, http.StatusForbidden, "非课程负责教师")
	ErrNotCourseStudent      = New(40304, http.StatusForbidden, "未加入该课程")

	// 业务冲突
	ErrCourseNotDraft        = New(40911, http.StatusConflict, "仅草稿状态的课程可删除")
	ErrCourseAlreadyPublished = New(40912, http.StatusConflict, "课程已发布")
	ErrInviteCodeInvalid     = New(40014, http.StatusBadRequest, "邀请码无效或已过期")
	ErrAlreadyEnrolled       = New(40913, http.StatusConflict, "已加入该课程")
	ErrCourseStudentFull     = New(40914, http.StatusConflict, "课程学生人数已满")
	ErrSubmissionExceedMax   = New(40915, http.StatusConflict, "已达到最大提交次数")
	ErrAssignmentDeadline    = New(40015, http.StatusBadRequest, "已超过截止时间且不允许迟交")
	ErrAlreadyEvaluated      = New(40916, http.StatusConflict, "已评价过该课程")
)
