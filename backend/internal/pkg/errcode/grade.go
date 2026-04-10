// grade.go
// 模块06 — 评测与成绩模块错误码
// 对照 docs/modules/06-评测与成绩/03-API接口设计.md

package errcode

import "net/http"

var (
	// 学期管理
	ErrSemesterNotFound       = New(40421, http.StatusNotFound, "学期不存在")
	ErrDuplicateSemesterCode  = New(40938, http.StatusConflict, "学期编码已存在")
	ErrSemesterDateInvalid    = New(40019, http.StatusBadRequest, "学期日期逻辑错误")

	// 等级映射
	ErrLevelConfigOverlap     = New(40020, http.StatusBadRequest, "分数区间存在重叠")
	ErrLevelConfigNotCovered  = New(40021, http.StatusBadRequest, "分数区间未覆盖0-100全范围")
	ErrLevelConfigGPARange    = New(40022, http.StatusBadRequest, "绩点值超出0.00-4.00范围")

	// 成绩审核
	ErrReviewNotFound         = New(40422, http.StatusNotFound, "审核记录不存在")
	ErrReviewAlreadyExists    = New(40939, http.StatusConflict, "该课程该学期已有审核记录")
	ErrGradesIncomplete       = New(40023, http.StatusBadRequest, "成绩未全部计算完成")
	ErrReviewNotPending       = New(40940, http.StatusConflict, "审核记录不在待审核状态")
	ErrGradeLocked            = New(40941, http.StatusConflict, "成绩已锁定")

	// 成绩申诉
	ErrAppealNotFound         = New(40423, http.StatusNotFound, "申诉记录不存在")
	ErrAppealAlreadyExists    = New(40942, http.StatusConflict, "该课程该学期已有申诉记录")
	ErrAppealNotPending       = New(40943, http.StatusConflict, "申诉不在待处理状态")
	ErrAppealReasonTooShort   = New(40024, http.StatusBadRequest, "申诉理由至少20字")

	// 学业预警
	ErrWarningNotFound        = New(40424, http.StatusNotFound, "预警记录不存在")
)
