// user_summary_helper.go
// 模块03 — 课程与教学：跨模块用户摘要辅助方法
// 统一处理模块01返回的用户摘要，避免各 service 重复判空。

package course

// getSummaryStudentNo 获取用户学号
func getSummaryStudentNo(summary *CourseUserSummary) *string {
	if summary == nil {
		return nil
	}
	return summary.StudentNo
}

// getSummaryCollege 获取用户学院
func getSummaryCollege(summary *CourseUserSummary) *string {
	if summary == nil {
		return nil
	}
	return summary.College
}

// getSummaryMajor 获取用户专业
func getSummaryMajor(summary *CourseUserSummary) *string {
	if summary == nil {
		return nil
	}
	return summary.Major
}

// getSummaryClassName 获取用户班级
func getSummaryClassName(summary *CourseUserSummary) *string {
	if summary == nil {
		return nil
	}
	return summary.ClassName
}
