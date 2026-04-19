// grade.go
// 模块06 — 评测与成绩：统一维护成绩审核、申诉、预警等领域枚举。
// 所有状态与类型都集中在此，避免在业务层散落裸值判断。

package enum

const (
	GradeReviewStatusNotSubmitted = 1 // 未提交
	GradeReviewStatusPending      = 2 // 待审核
	GradeReviewStatusApproved     = 3 // 已通过
	GradeReviewStatusRejected     = 4 // 已驳回
)

// GradeReviewStatusText 成绩审核状态文本映射。
var GradeReviewStatusText = map[int16]string{
	GradeReviewStatusNotSubmitted: "未提交",
	GradeReviewStatusPending:      "待审核",
	GradeReviewStatusApproved:     "已通过",
	GradeReviewStatusRejected:     "已驳回",
}

// GetGradeReviewStatusText 获取成绩审核状态文本。
func GetGradeReviewStatusText(value int16) string {
	if text, ok := GradeReviewStatusText[value]; ok {
		return text
	}
	return "未知"
}

// IsValidGradeReviewStatus 校验成绩审核状态是否合法。
func IsValidGradeReviewStatus(value int16) bool {
	_, ok := GradeReviewStatusText[value]
	return ok
}

const (
	GradeAppealStatusPending  = 1 // 待处理
	GradeAppealStatusApproved = 2 // 已同意
	GradeAppealStatusRejected = 3 // 已驳回
)

// GradeAppealStatusText 成绩申诉状态文本映射。
var GradeAppealStatusText = map[int16]string{
	GradeAppealStatusPending:  "待处理",
	GradeAppealStatusApproved: "已同意",
	GradeAppealStatusRejected: "已驳回",
}

// GetGradeAppealStatusText 获取成绩申诉状态文本。
func GetGradeAppealStatusText(value int16) string {
	if text, ok := GradeAppealStatusText[value]; ok {
		return text
	}
	return "未知"
}

// IsValidGradeAppealStatus 校验成绩申诉状态是否合法。
func IsValidGradeAppealStatus(value int16) bool {
	_, ok := GradeAppealStatusText[value]
	return ok
}

const (
	AcademicWarningStatusPending  = 1 // 待处理
	AcademicWarningStatusHandled  = 2 // 已处理
	AcademicWarningStatusResolved = 3 // 已解除
)

// AcademicWarningStatusText 学业预警状态文本映射。
var AcademicWarningStatusText = map[int16]string{
	AcademicWarningStatusPending:  "待处理",
	AcademicWarningStatusHandled:  "已处理",
	AcademicWarningStatusResolved: "已解除",
}

// GetAcademicWarningStatusText 获取学业预警状态文本。
func GetAcademicWarningStatusText(value int16) string {
	if text, ok := AcademicWarningStatusText[value]; ok {
		return text
	}
	return "未知"
}

// IsValidAcademicWarningStatus 校验学业预警状态是否合法。
func IsValidAcademicWarningStatus(value int16) bool {
	_, ok := AcademicWarningStatusText[value]
	return ok
}

const (
	AcademicWarningTypeLowGPA          = 1 // 低GPA预警
	AcademicWarningTypeConsecutiveFail = 2 // 连续挂科预警
)

// AcademicWarningTypeText 学业预警类型文本映射。
var AcademicWarningTypeText = map[int16]string{
	AcademicWarningTypeLowGPA:          "低GPA",
	AcademicWarningTypeConsecutiveFail: "连续挂科",
}

// GetAcademicWarningTypeText 获取学业预警类型文本。
func GetAcademicWarningTypeText(value int16) string {
	if text, ok := AcademicWarningTypeText[value]; ok {
		return text
	}
	return "未知"
}

// IsValidAcademicWarningType 校验学业预警类型是否合法。
func IsValidAcademicWarningType(value int16) bool {
	_, ok := AcademicWarningTypeText[value]
	return ok
}
