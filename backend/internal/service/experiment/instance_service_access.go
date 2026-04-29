// instance_service_access.go
// 模块04 — 实验环境：实例访问控制辅助逻辑
// 统一封装学生本人、课程教师、学校管理员和超管对实验实例的访问校验

package experiment

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/lenschain/backend/internal/model/entity"
	svcctx "github.com/lenschain/backend/internal/pkg/context"
	"github.com/lenschain/backend/internal/pkg/errcode"
)

// getAccessibleInstance 获取当前用户可查看的实验实例。
// 学生仅可访问本人实例，教师仅可访问本人负责课程下的实例，超管可访问全部实例。
func (s *instanceService) getAccessibleInstance(ctx context.Context, sc *svcctx.ServiceContext, id int64) (*entity.ExperimentInstance, error) {
	instance, err := s.loadInstanceRecord(ctx, id)
	if err != nil {
		return nil, err
	}
	allowed, err := s.canViewInstance(ctx, sc, instance)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, errcode.ErrForbidden
	}
	return instance, nil
}

// getOwnedInstance 获取当前学生本人拥有的实验实例。
func (s *instanceService) getOwnedInstance(ctx context.Context, sc *svcctx.ServiceContext, id int64) (*entity.ExperimentInstance, error) {
	instance, err := s.loadInstanceRecord(ctx, id)
	if err != nil {
		return nil, err
	}
	if sc.IsSuperAdmin() {
		return instance, nil
	}
	if !sc.IsStudent() || instance.StudentID != sc.UserID {
		return nil, errcode.ErrForbidden
	}
	return instance, nil
}

// getManageableInstance 获取当前用户可执行管理动作的实验实例。
// 管理动作允许实例所属学生、课程教师、学校管理员和超管访问。
func (s *instanceService) getManageableInstance(ctx context.Context, sc *svcctx.ServiceContext, id int64) (*entity.ExperimentInstance, error) {
	instance, err := s.loadInstanceRecord(ctx, id)
	if err != nil {
		return nil, err
	}
	allowed, err := s.canManageInstance(ctx, sc, instance)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, errcode.ErrForbidden
	}
	return instance, nil
}

// loadInstanceRecord 加载实例记录并统一转换不存在错误。
func (s *instanceService) loadInstanceRecord(ctx context.Context, id int64) (*entity.ExperimentInstance, error) {
	instance, err := s.instanceRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errcode.ErrInstanceNotFound
		}
		return nil, err
	}
	return instance, nil
}

// canViewInstance 判断当前用户是否可查看实例详情类数据。
func (s *instanceService) canViewInstance(ctx context.Context, sc *svcctx.ServiceContext, instance *entity.ExperimentInstance) (bool, error) {
	if instance == nil {
		return false, nil
	}
	if sc.IsSuperAdmin() {
		return true, nil
	}
	if instance.SchoolID != sc.SchoolID {
		return false, nil
	}
	if sc.IsStudent() {
		return instance.StudentID == sc.UserID, nil
	}
	if sc.IsTeacher() {
		return s.isCourseTeacher(ctx, sc, instance.CourseID)
	}
	return false, nil
}

// canManageInstance 判断当前用户是否可执行销毁等管理动作。
func (s *instanceService) canManageInstance(ctx context.Context, sc *svcctx.ServiceContext, instance *entity.ExperimentInstance) (bool, error) {
	if instance == nil {
		return false, nil
	}
	if sc.IsSuperAdmin() {
		return true, nil
	}
	if instance.SchoolID != sc.SchoolID {
		return false, nil
	}
	if sc.IsStudent() {
		return instance.StudentID == sc.UserID, nil
	}
	if sc.IsSchoolAdmin() {
		return true, nil
	}
	if sc.IsTeacher() {
		return s.isCourseTeacher(ctx, sc, instance.CourseID)
	}
	return false, nil
}

// canTeachInstance 判断当前用户是否可执行课程教师专属操作。
// 仅课程教师可执行评分、终端查看、教师指导等动作。
func (s *instanceService) canTeachInstance(ctx context.Context, sc *svcctx.ServiceContext, instance *entity.ExperimentInstance) (bool, error) {
	if instance == nil {
		return false, nil
	}
	if !sc.IsTeacher() || instance.SchoolID != sc.SchoolID {
		return false, nil
	}
	return s.isCourseTeacher(ctx, sc, instance.CourseID)
}

// isCourseTeacher 判断当前教师是否为实例关联课程的负责教师。
func (s *instanceService) isCourseTeacher(ctx context.Context, sc *svcctx.ServiceContext, courseID *int64) (bool, error) {
	if courseID == nil || *courseID == 0 {
		return false, nil
	}
	teacherID, err := s.courseQuerier.GetCourseTeacherID(ctx, *courseID)
	if err != nil {
		return false, err
	}
	return teacherID == sc.UserID, nil
}
