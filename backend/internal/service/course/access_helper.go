// access_helper.go
// 模块03 — 课程与教学：课程访问控制辅助方法
// 统一封装课程成员、课程教师、跨学校访问等校验逻辑

package course

import (
	"context"
	"time"

	"github.com/lenschain/backend/internal/model/dto"
	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/model/enum"
	svcctx "github.com/lenschain/backend/internal/pkg/context"
	"github.com/lenschain/backend/internal/pkg/errcode"
	courserepo "github.com/lenschain/backend/internal/repository/course"
)

// loadCourseWithAccess 加载课程并完成基础跨学校校验
func loadCourseWithAccess(
	ctx context.Context,
	sc *svcctx.ServiceContext,
	courseRepo courserepo.CourseRepository,
	courseID int64,
) (*entity.Course, error) {
	course, err := courseRepo.GetByID(ctx, courseID)
	if err != nil {
		return nil, errcode.ErrCourseNotFound
	}

	if !sc.IsSuperAdmin() && sc.SchoolID > 0 && course.SchoolID != sc.SchoolID {
		return nil, errcode.ErrForbidden
	}

	return course, nil
}

// ensureCourseTeacher 校验当前用户是否为课程负责教师
func ensureCourseTeacher(
	ctx context.Context,
	sc *svcctx.ServiceContext,
	courseRepo courserepo.CourseRepository,
	courseID int64,
) (*entity.Course, error) {
	course, err := loadCourseWithAccess(ctx, sc, courseRepo, courseID)
	if err != nil {
		return nil, err
	}

	if course.TeacherID != sc.UserID {
		return nil, errcode.ErrNotCourseTeacher
	}

	return course, nil
}

// ensureCourseStudent 校验当前用户是否已选课
func ensureCourseStudent(
	ctx context.Context,
	sc *svcctx.ServiceContext,
	courseRepo courserepo.CourseRepository,
	enrollmentRepo courserepo.EnrollmentRepository,
	courseID int64,
) (*entity.Course, error) {
	course, err := loadCourseWithAccess(ctx, sc, courseRepo, courseID)
	if err != nil {
		return nil, err
	}
	if course.Status == enum.CourseStatusArchived {
		return nil, errcode.ErrForbidden
	}

	enrolled, err := enrollmentRepo.IsEnrolled(ctx, sc.UserID, courseID)
	if err != nil {
		return nil, err
	}
	if !enrolled {
		return nil, errcode.ErrNotCourseStudent
	}

	return course, nil
}

// ensureCourseMember 校验用户是否为课程成员（课程教师或已选课学生）
func ensureCourseMember(
	ctx context.Context,
	sc *svcctx.ServiceContext,
	courseRepo courserepo.CourseRepository,
	enrollmentRepo courserepo.EnrollmentRepository,
	courseID int64,
) (*entity.Course, error) {
	course, err := loadCourseWithAccess(ctx, sc, courseRepo, courseID)
	if err != nil {
		return nil, err
	}

	if course.TeacherID == sc.UserID {
		return course, nil
	}
	if course.Status == enum.CourseStatusArchived {
		return nil, errcode.ErrForbidden
	}

	enrolled, err := enrollmentRepo.IsEnrolled(ctx, sc.UserID, courseID)
	if err != nil {
		return nil, err
	}
	if !enrolled {
		return nil, errcode.ErrNotCourseStudent
	}

	return course, nil
}

// ensureCourseWriteAllowed 校验课程当前是否仍允许写操作。
// 已归档课程按验收标准进入只读状态，教师只能查看和导出数据。
func ensureCourseWriteAllowed(course *entity.Course) error {
	if course != nil && course.Status == enum.CourseStatusArchived {
		return errcode.ErrForbidden.WithMessage("课程已归档，仅支持查看和导出")
	}
	return nil
}

// ensureCourseContentEditable 校验课程当前是否允许编辑内容与教学配置。
// 文档要求仅草稿、已发布、进行中的课程可继续编辑内容；已结束后进入查看/批改阶段。
func ensureCourseContentEditable(course *entity.Course) error {
	if course == nil {
		return nil
	}
	switch course.Status {
	case enum.CourseStatusDraft, enum.CourseStatusPublished, enum.CourseStatusActive:
		return nil
	case enum.CourseStatusEnded:
		return errcode.ErrForbidden.WithMessage("课程已结束，不可继续编辑内容")
	default:
		return errcode.ErrForbidden.WithMessage("课程已归档，仅支持查看和导出")
	}
}

// ensureCourseInteractionAllowed 校验课程当前是否允许讨论区互动与公告写入。
// 课程互动只在学生可正常参与教学阶段开放，即已发布和进行中。
func ensureCourseInteractionAllowed(course *entity.Course) error {
	if course == nil {
		return nil
	}
	switch course.Status {
	case enum.CourseStatusPublished, enum.CourseStatusActive:
		return nil
	case enum.CourseStatusEnded:
		return errcode.ErrForbidden.WithMessage("课程已结束，不可继续互动")
	default:
		return errcode.ErrForbidden.WithMessage("课程当前状态不可互动")
	}
}

// ensureCourseEnrollmentManageable 校验课程当前是否允许管理学生名单。
// 文档要求学生加入与教师管理学生发生在已发布、进行中两个阶段；已结束后进入只读数据阶段。
func ensureCourseEnrollmentManageable(course *entity.Course) error {
	if course == nil {
		return nil
	}
	switch course.Status {
	case enum.CourseStatusPublished, enum.CourseStatusActive:
		return nil
	case enum.CourseStatusEnded:
		return errcode.ErrForbidden.WithMessage("课程已结束，无法管理学生")
	case enum.CourseStatusArchived:
		return errcode.ErrForbidden.WithMessage("课程已归档，仅支持查看和导出")
	default:
		return errcode.ErrForbidden.WithMessage("课程未发布，无法管理学生")
	}
}

// ensureCourseGradingAllowed 校验课程当前是否允许教师批改作业。
// 验收标准要求进行中与已结束课程允许批改；已归档仅允许查看与导出。
func ensureCourseGradingAllowed(course *entity.Course) error {
	if course == nil {
		return nil
	}
	switch course.Status {
	case enum.CourseStatusActive, enum.CourseStatusEnded:
		return nil
	case enum.CourseStatusArchived:
		return errcode.ErrForbidden.WithMessage("课程已归档，仅支持查看和导出")
	default:
		return errcode.ErrForbidden.WithMessage("课程当前状态不可批改作业")
	}
}

// ensureCourseLearningAllowed 校验课程当前是否允许学生继续学习并上报学习进度。
// 验收标准要求学生仅在已发布、进行中阶段可学习；已结束后只能查看内容与成绩。
func ensureCourseLearningAllowed(course *entity.Course) error {
	if course == nil {
		return nil
	}
	switch course.Status {
	case enum.CourseStatusPublished, enum.CourseStatusActive:
		return nil
	case enum.CourseStatusEnded:
		return errcode.ErrForbidden.WithMessage("课程已结束，不可继续学习")
	case enum.CourseStatusArchived:
		return errcode.ErrForbidden.WithMessage("课程已归档，仅支持查看和导出")
	default:
		return errcode.ErrForbidden.WithMessage("课程未发布，不可开始学习")
	}
}

// ensureCourseSubmissionAllowed 校验课程当前是否允许学生提交作业。
// 验收标准要求学生仅在课程进行中时提交作业；已发布阶段只允许加入和浏览内容。
func ensureCourseSubmissionAllowed(course *entity.Course) error {
	if course == nil {
		return nil
	}
	switch course.Status {
	case enum.CourseStatusActive:
		return nil
	case enum.CourseStatusPublished:
		return errcode.ErrForbidden.WithMessage("课程未开始，暂不可提交作业")
	case enum.CourseStatusEnded:
		return errcode.ErrInvalidParams.WithMessage("课程已结束，无法提交作业")
	case enum.CourseStatusArchived:
		return errcode.ErrForbidden.WithMessage("课程已归档，仅支持查看和导出")
	default:
		return errcode.ErrForbidden.WithMessage("课程未发布，不可提交作业")
	}
}

// ensureCourseCloneAllowed 校验课程是否允许被当前教师克隆。
// 自有课程允许克隆；共享课程必须仍处于共享课程库可见状态，避免通过接口绕过共享库可见性限制。
func ensureCourseCloneAllowed(course *entity.Course, userID int64) error {
	if course == nil {
		return errcode.ErrCourseNotFound
	}
	if course.TeacherID == userID {
		return nil
	}
	if !course.IsShared {
		return errcode.ErrForbidden.WithMessage("仅可克隆自己的课程或共享课程")
	}
	switch course.Status {
	case enum.CourseStatusPublished, enum.CourseStatusActive, enum.CourseStatusEnded:
		return nil
	default:
		return errcode.ErrCourseNotFound
	}
}

// validateScheduleItems 校验课程表时间段的基本合法性。
// 课程表属于课程模块的业务数据，服务层需要在入库前拦截明显无效的时间段。
func validateScheduleItems(items []dto.ScheduleItemReq) error {
	for _, item := range items {
		startTime, err := time.Parse("15:04", item.StartTime)
		if err != nil {
			return errcode.ErrInvalidParams.WithMessage("课程表开始时间格式错误")
		}
		endTime, err := time.Parse("15:04", item.EndTime)
		if err != nil {
			return errcode.ErrInvalidParams.WithMessage("课程表结束时间格式错误")
		}
		if !startTime.Before(endTime) {
			return errcode.ErrInvalidParams.WithMessage("课程表开始时间必须早于结束时间")
		}
	}
	return nil
}
