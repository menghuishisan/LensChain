// user_service.go
// 模块01 — 用户与认证：用户管理业务逻辑
// 负责用户 CRUD、状态变更、密码重置、解锁、批量删除等
// 对照 docs/modules/01-用户与认证/03-API接口设计.md

package auth

import (
	"context"
	"maps"
	"slices"
	"strconv"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/lenschain/backend/internal/model/dto"
	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/model/enum"
	"github.com/lenschain/backend/internal/repository/auth"
	"github.com/lenschain/backend/internal/pkg/audit"
	"github.com/lenschain/backend/internal/pkg/cache"
	"github.com/lenschain/backend/internal/pkg/crypto"
	svcctx "github.com/lenschain/backend/internal/pkg/context"
	"github.com/lenschain/backend/internal/pkg/errcode"
	"github.com/lenschain/backend/internal/pkg/logger"
	"github.com/lenschain/backend/internal/pkg/snowflake"
)

// UserService 用户管理服务接口
type UserService interface {
	List(ctx context.Context, sc *svcctx.ServiceContext, req *dto.UserListReq) ([]*dto.UserListItem, int64, error)
	GetByID(ctx context.Context, sc *svcctx.ServiceContext, id int64) (*dto.UserDetailResp, error)
	Create(ctx context.Context, sc *svcctx.ServiceContext, req *dto.CreateUserReq) (*dto.CreateUserResp, error)
	Update(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.UpdateUserReq) error
	Delete(ctx context.Context, sc *svcctx.ServiceContext, id int64) error
	BatchDelete(ctx context.Context, sc *svcctx.ServiceContext, ids []int64) error
	UpdateStatus(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.UpdateStatusReq) error
	ResetPassword(ctx context.Context, sc *svcctx.ServiceContext, id int64, newPassword string) error
	UnlockUser(ctx context.Context, sc *svcctx.ServiceContext, id int64) error
}

// userService 用户管理服务实现
type userService struct {
	db          *gorm.DB
	userRepo    authrepo.UserRepository
	profileRepo authrepo.ProfileRepository
	roleRepo    authrepo.RoleRepository
}

// NewUserService 创建用户管理服务实例
func NewUserService(
	db *gorm.DB,
	userRepo authrepo.UserRepository,
	profileRepo authrepo.ProfileRepository,
	roleRepo authrepo.RoleRepository,
) UserService {
	return &userService{
		db:          db,
		userRepo:    userRepo,
		profileRepo: profileRepo,
		roleRepo:    roleRepo,
	}
}

// List 用户列表
func (s *userService) List(ctx context.Context, sc *svcctx.ServiceContext, req *dto.UserListReq) ([]*dto.UserListItem, int64, error) {
	params := &authrepo.UserListParams{
		SchoolID:       sc.SchoolID,
		Keyword:        req.Keyword,
		Status:         req.Status,
		Role:           req.Role,
		College:        req.College,
		EducationLevel: req.EducationLevel,
		SortBy:         req.SortBy,
		SortOrder:      req.SortOrder,
		Page:           req.Page,
		PageSize:       req.PageSize,
	}

	users, total, err := s.userRepo.List(ctx, params)
	if err != nil {
		return nil, 0, errcode.ErrInternal.WithMessage("查询用户列表失败")
	}

	// 转换为 DTO
	items := make([]*dto.UserListItem, 0, len(users))
	for _, user := range users {
		item := userToListItem(user)
		items = append(items, item)
	}

	return items, total, nil
}

// GetByID 用户详情
func (s *userService) GetByID(ctx context.Context, sc *svcctx.ServiceContext, id int64) (*dto.UserDetailResp, error) {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return nil, errcode.ErrUserNotFound
	}

	// 多租户校验：校管只能查看本校用户
	if sc.SchoolID > 0 && user.SchoolID != sc.SchoolID {
		return nil, errcode.ErrUserNotFound
	}

	return userToDetailResp(user), nil
}

// Create 创建用户
func (s *userService) Create(ctx context.Context, sc *svcctx.ServiceContext, req *dto.CreateUserReq) (*dto.CreateUserResp, error) {
	// 检查手机号是否已存在
	existing, err := s.userRepo.GetByPhone(ctx, req.Phone)
	if err == nil && existing != nil {
		return nil, errcode.ErrDuplicatePhone
	}

	// 检查学号是否已存在（校内唯一）
	schoolID := sc.SchoolID
	if req.StudentNo != nil && *req.StudentNo != "" {
		existing, err := s.userRepo.GetBySchoolAndStudentNo(ctx, schoolID, *req.StudentNo)
		if err == nil && existing != nil {
			return nil, errcode.ErrDuplicateStudentNo
		}
	}

	// 加密密码
	hash, err := crypto.HashPassword(req.Password)
	if err != nil {
		return nil, errcode.ErrInternal.WithMessage("密码加密失败")
	}

	// 获取角色
	role, err := s.roleRepo.GetByCode(ctx, req.Role)
	if err != nil {
		return nil, errcode.ErrInternal.WithMessage("角色不存在")
	}

	// 创建用户（事务）
	userID := snowflake.Generate()
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 创建用户主记录
		user := &entity.User{
			ID:            userID,
			Phone:         req.Phone,
			PasswordHash:  hash,
			Name:          req.Name,
			SchoolID:      schoolID,
			StudentNo:     req.StudentNo,
			Status:        enum.UserStatusActive,
			IsFirstLogin:  true,
			IsSchoolAdmin: false,
			CreatedBy:     &sc.UserID,
		}
		if err := tx.Create(user).Error; err != nil {
			return err
		}

		// 创建用户扩展信息
		profile := &entity.UserProfile{
			ID:             snowflake.Generate(),
			UserID:         userID,
			College:        req.College,
			Major:          req.Major,
			ClassName:      req.ClassName,
			EducationLevel: req.EducationLevel,
			Email:          req.Email,
			Remark:         req.Remark,
		}
		if err := tx.Create(profile).Error; err != nil {
			return err
		}

		// 分配角色
		userRole := &entity.UserRole{
			ID:     snowflake.Generate(),
			UserID: userID,
			RoleID: role.ID,
		}
		return tx.Create(userRole).Error
	})
	if err != nil {
		logger.L.Error("创建用户失败", zap.Error(err))
		return nil, errcode.ErrInternal.WithMessage("创建用户失败")
	}

	// 记录操作日志（使用 pkg/audit 公共包）
	audit.RecordFromContext(s.db, sc.UserID, sc.ClientIP, "create_user", "user", userID, map[string]interface{}{
		"phone": req.Phone,
		"name":  req.Name,
		"role":  req.Role,
	})

	return &dto.CreateUserResp{
		ID: strconv.FormatInt(userID, 10),
	}, nil
}

// Update 更新用户信息
func (s *userService) Update(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.UpdateUserReq) error {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return errcode.ErrUserNotFound
	}

	// 多租户校验
	if sc.SchoolID > 0 && user.SchoolID != sc.SchoolID {
		return errcode.ErrUserNotFound
	}

	// 检查学号唯一性
	if req.StudentNo != nil && *req.StudentNo != "" {
		existing, err := s.userRepo.GetBySchoolAndStudentNo(ctx, user.SchoolID, *req.StudentNo)
		if err == nil && existing != nil && existing.ID != id {
			return errcode.ErrDuplicateStudentNo
		}
	}

	// 更新用户主表字段
	userFields := make(map[string]interface{})
	if req.Name != nil {
		userFields["name"] = *req.Name
	}
	if req.StudentNo != nil {
		userFields["student_no"] = *req.StudentNo
	}
	if len(userFields) > 0 {
		if err := s.userRepo.UpdateFields(ctx, id, userFields); err != nil {
			return errcode.ErrInternal.WithMessage("更新用户信息失败")
		}
	}

	// 更新扩展信息
	profileFields := make(map[string]interface{})
	if req.College != nil {
		profileFields["college"] = *req.College
	}
	if req.Major != nil {
		profileFields["major"] = *req.Major
	}
	if req.ClassName != nil {
		profileFields["class_name"] = *req.ClassName
	}
	if req.EnrollmentYear != nil {
		profileFields["enrollment_year"] = *req.EnrollmentYear
	}
	if req.EducationLevel != nil {
		profileFields["education_level"] = *req.EducationLevel
	}
	if req.Grade != nil {
		profileFields["grade"] = *req.Grade
	}
	if req.Email != nil {
		profileFields["email"] = *req.Email
	}
	if req.Remark != nil {
		profileFields["remark"] = *req.Remark
	}
	if len(profileFields) > 0 {
		if err := s.profileRepo.UpdateFields(ctx, id, profileFields); err != nil {
			return errcode.ErrInternal.WithMessage("更新用户扩展信息失败")
		}
	}

	// 记录操作日志（使用标准库 maps.Keys 替代自定义 mapKeys）
	allKeys := slices.Concat(slices.Collect(maps.Keys(userFields)), slices.Collect(maps.Keys(profileFields)))
	audit.RecordFromContext(s.db, sc.UserID, sc.ClientIP, "update_user", "user", id, map[string]interface{}{
		"updated_fields": allKeys,
	})

	return nil
}

// Delete 删除用户（软删除）
func (s *userService) Delete(ctx context.Context, sc *svcctx.ServiceContext, id int64) error {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return errcode.ErrUserNotFound
	}

	// 多租户校验
	if sc.SchoolID > 0 && user.SchoolID != sc.SchoolID {
		return errcode.ErrUserNotFound
	}

	// 边界检查：不允许删除自己
	if id == sc.UserID {
		return errcode.ErrForbidden.WithMessage("不允许删除自己的账号")
	}

	if err := s.userRepo.SoftDelete(ctx, id); err != nil {
		return errcode.ErrInternal.WithMessage("删除用户失败")
	}

	// 踢下线
	s.kickUser(ctx, id)

	// 记录操作日志
	audit.RecordFromContext(s.db, sc.UserID, sc.ClientIP, "delete_user", "user", id, map[string]interface{}{
		"phone": user.Phone,
		"name":  user.Name,
	})

	return nil
}

// BatchDelete 批量删除用户
func (s *userService) BatchDelete(ctx context.Context, sc *svcctx.ServiceContext, ids []int64) error {
	// 边界检查：不允许批量删除中包含自己
	for _, id := range ids {
		if id == sc.UserID {
			return errcode.ErrForbidden.WithMessage("不允许删除自己的账号")
		}
	}

	if err := s.userRepo.BatchSoftDelete(ctx, ids, sc.SchoolID); err != nil {
		return errcode.ErrInternal.WithMessage("批量删除用户失败")
	}

	// 批量踢下线
	for _, id := range ids {
		s.kickUser(ctx, id)
	}

	// 记录操作日志
	audit.RecordFromContext(s.db, sc.UserID, sc.ClientIP, "batch_delete_users", "user", 0, map[string]interface{}{
		"count": len(ids),
	})

	return nil
}

// UpdateStatus 变更账号状态
func (s *userService) UpdateStatus(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.UpdateStatusReq) error {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return errcode.ErrUserNotFound
	}

	// 多租户校验
	if sc.SchoolID > 0 && user.SchoolID != sc.SchoolID {
		return errcode.ErrUserNotFound
	}

	// 边界检查：不允许禁用/归档自己
	if id == sc.UserID && (req.Status == enum.UserStatusDisabled || req.Status == enum.UserStatusArchived) {
		return errcode.ErrForbidden.WithMessage("不允许禁用或归档自己的账号")
	}

	// 更新状态
	if err := s.userRepo.UpdateFields(ctx, id, map[string]interface{}{
		"status": req.Status,
	}); err != nil {
		return errcode.ErrInternal.WithMessage("更新账号状态失败")
	}

	// 禁用或归档时立即踢下线
	if req.Status == enum.UserStatusDisabled || req.Status == enum.UserStatusArchived {
		s.kickUser(ctx, id)
	}

	// 记录操作日志
	audit.RecordFromContext(s.db, sc.UserID, sc.ClientIP, "update_user_status", "user", id, map[string]interface{}{
		"old_status": user.Status,
		"new_status": req.Status,
		"reason":     req.Reason,
	})

	return nil
}

// ResetPassword 重置用户密码
func (s *userService) ResetPassword(ctx context.Context, sc *svcctx.ServiceContext, id int64, newPassword string) error {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return errcode.ErrUserNotFound
	}

	// 多租户校验
	if sc.SchoolID > 0 && user.SchoolID != sc.SchoolID {
		return errcode.ErrUserNotFound
	}

	// 加密新密码
	hash, err := crypto.HashPassword(newPassword)
	if err != nil {
		return errcode.ErrInternal.WithMessage("密码加密失败")
	}

	// 更新密码 + 设置首次登录标记
	if err := s.userRepo.UpdateFields(ctx, id, map[string]interface{}{
		"password_hash":  hash,
		"is_first_login": true,
	}); err != nil {
		return errcode.ErrInternal.WithMessage("重置密码失败")
	}

	// 踢下线
	s.kickUser(ctx, id)

	// 记录操作日志
	audit.RecordFromContext(s.db, sc.UserID, sc.ClientIP, "reset_password", "user", id, nil)

	return nil
}

// UnlockUser 解锁账号
func (s *userService) UnlockUser(ctx context.Context, sc *svcctx.ServiceContext, id int64) error {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return errcode.ErrUserNotFound
	}

	// 多租户校验
	if sc.SchoolID > 0 && user.SchoolID != sc.SchoolID {
		return errcode.ErrUserNotFound
	}

	// 解锁数据库
	if err := s.userRepo.UnlockUser(ctx, id); err != nil {
		return errcode.ErrInternal.WithMessage("解锁账号失败")
	}

	// 清除 Redis 锁定标记和失败计数
	_ = cache.Del(ctx, cache.KeyAccountLocked+user.Phone)
	_ = cache.Del(ctx, cache.KeyLoginFail+user.Phone)

	// 记录操作日志
	audit.RecordFromContext(s.db, sc.UserID, sc.ClientIP, "unlock_user", "user", id, nil)

	return nil
}

// ========== 内部辅助方法 ==========

// kickUser 踢用户下线（清除Session）
func (s *userService) kickUser(ctx context.Context, userID int64) {
	sessionKey := cache.KeySession + strconv.FormatInt(userID, 10)
	_ = cache.Del(ctx, sessionKey)
}
