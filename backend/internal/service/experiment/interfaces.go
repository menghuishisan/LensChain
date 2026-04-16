// interfaces.go
// 模块04 — 实验环境：跨模块接口定义
// 定义模块04需要从其他模块注入的接口
// 通过接口解耦，避免直接引用其他模块的具体实现

package experiment

import (
	"context"
	"time"
)

// ExperimentUserSummary 模块04需要的用户摘要
// 统一承载实验环境模块在监控、分组等场景下需要的用户最小信息。
type ExperimentUserSummary struct {
	UserID    int64
	Name      string
	StudentNo string
}

// CourseStudentSummary 课程学生摘要
// 用于多人实验自动分组、教师监控等需要课程学生名单的场景。
type CourseStudentSummary struct {
	StudentID int64
	Name      string
	StudentNo string
}

// UserNameQuerier 跨模块接口：查询用户名称（从模块01注入）
type UserNameQuerier interface {
	GetUserName(ctx context.Context, userID int64) string
}

// UserSummaryQuerier 跨模块接口：查询用户摘要（从模块01注入）
type UserSummaryQuerier interface {
	GetUserSummary(ctx context.Context, userID int64) *ExperimentUserSummary
}

// SchoolNameQuerier 跨模块接口：查询学校名称（从模块02注入）
type SchoolNameQuerier interface {
	GetSchoolName(ctx context.Context, schoolID int64) string
}

// CourseQuerier 跨模块接口：查询课程信息（从模块03注入）
type CourseQuerier interface {
	GetCourseTitle(ctx context.Context, courseID int64) string
	GetCourseSchoolID(ctx context.Context, courseID int64) (int64, error)
	GetCourseTeacherID(ctx context.Context, courseID int64) (int64, error)
}

// CourseGradeSyncer 跨模块接口：将实验最终成绩同步回课程成绩体系。
// 模块04 仅声明同步契约，具体写入 assignment_submissions 的实现由模块03适配层注入。
type CourseGradeSyncer interface {
	SyncExperimentScore(
		ctx context.Context,
		courseID int64,
		assignmentID int64,
		studentID int64,
		templateTitle string,
		score float64,
		submittedAt time.Time,
		teacherComment *string,
	) error
}

// EnrollmentChecker 跨模块接口：检查选课状态（从模块03注入）
type EnrollmentChecker interface {
	IsEnrolled(ctx context.Context, courseID, studentID int64) (bool, error)
}

// CourseRosterQuerier 跨模块接口：查询课程学生名单（从模块03注入）
type CourseRosterQuerier interface {
	ListCourseStudents(ctx context.Context, courseID int64) ([]CourseStudentSummary, error)
}

// EndedCourseQuerier 跨模块接口：查询已结束或已归档课程，用于实验环境回收调度。
type EndedCourseQuerier interface {
	ListEndedCourseIDs(ctx context.Context) ([]int64, error)
	ListCourseIDsEndingWithin(ctx context.Context, within time.Duration) (map[int64]time.Time, error)
}
