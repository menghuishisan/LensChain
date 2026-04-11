// access_helper.go
// 模块03 — 课程与教学：课程访问控制辅助方法
// 统一封装课程成员、课程教师、跨学校访问等校验逻辑

package course

import (
	"context"

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
