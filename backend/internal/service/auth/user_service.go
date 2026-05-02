// user_service.go
// 模块01 — 用户与认证：用户管理业务逻辑
// 负责用户 CRUD、状态变更、密码重置、解锁、批量删除等
// 对照 docs/modules/01-用户与认证/03-API接口设计.md

package auth

import (
	"context"
	"errors"
	"maps"
	"slices"
	"strconv"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/lenschain/backend/internal/model/dto"
	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/model/enum"
	"github.com/lenschain/backend/internal/pkg/audit"
	"github.com/lenschain/backend/internal/pkg/cache"
	svcctx "github.com/lenschain/backend/internal/pkg/context"
	"github.com/lenschain/backend/internal/pkg/crypto"
	"github.com/lenschain/backend/internal/pkg/database"
	"github.com/lenschain/backend/internal/pkg/errcode"
	"github.com/lenschain/backend/internal/pkg/logger"
	"github.com/lenschain/backend/internal/pkg/mask"
	"github.com/lenschain/backend/internal/pkg/snowflake"
	"github.com/lenschain/backend/internal/repository/auth"
	schoolrepo "github.com/lenschain/backend/internal/repository/school"
)

// UserService 用户管理服务接口
type UserService interface {
	List(ctx context.Context, sc *svcctx.ServiceContext, req *dto.UserListReq) ([]*dto.UserListItem, int64, error)
	GetByID(ctx context.Context, sc *svcctx.ServiceContext, id int64) (*dto.UserDetailResp, error)
	Create(ctx context.Context, sc *svcctx.ServiceContext, req *dto.CreateUserReq) (*dto.CreateUserResp, error)
	CreateSuperAdmin(ctx context.Context, sc *svcctx.ServiceContext, req *dto.CreateSuperAdminReq) (*dto.CreateUserResp, error)
	Update(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.UpdateUserReq) error
	Delete(ctx context.Context, sc *svcctx.ServiceContext, id int64) error
	BatchDelete(ctx context.Context, sc *svcctx.ServiceContext, ids []int64) error
	UpdateStatus(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.UpdateStatusReq) error
	ResetPassword(ctx context.Context, sc *svcctx.ServiceContext, id int64, newPassword string) error
	UnlockUser(ctx context.Context, sc *svcctx.ServiceContext, id int64) error
}

// userService 用户管理服务实现
type userService struct {
	db             *gorm.DB
	userRepo       authrepo.UserRepository
	profileRepo    authrepo.ProfileRepository
	roleRepo       authrepo.RoleRepository
	schoolRepo     schoolrepo.SchoolRepository
	policyProvider runtimePolicyProvider
}

// NewUserService 创建用户管理服务实例
func NewUserService(
	db *gorm.DB,
	userRepo authrepo.UserRepository,
	profileRepo authrepo.ProfileRepository,
	roleRepo authrepo.RoleRepository,
	schoolRepo schoolrepo.SchoolRepository,
) UserService {
	return &userService{
		db:             db,
		userRepo:       userRepo,
		profileRepo:    profileRepo,
		roleRepo:       roleRepo,
		schoolRepo:     schoolRepo,
		policyProvider: &cacheRuntimePolicyProvider{},
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

	profileMap, roleCodeMap, schoolMap, err := s.buildUserExtraData(ctx, users)
	if err != nil {
		return nil, 0, err
	}

	items := make([]*dto.UserListItem, 0, len(users))
	for _, user := range users {
		item := userToListItem(user, profileMap[user.ID], roleCodeMap[user.ID], schoolMap[user.SchoolID])
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

	profileMap, roleCodeMap, _, err := s.buildUserExtraData(ctx, []*entity.User{user})
	if err != nil {
		return nil, err
	}
	return userToDetailResp(user, profileMap[user.ID], roleCodeMap[user.ID]), nil
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
	if err := s.validatePasswordByPolicy(ctx, req.Password); err != nil {
		return nil, err
	}
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
	err = database.TransactionWithDB(ctx, s.db, func(tx *gorm.DB) error {
		txUserRepo := authrepo.NewUserRepository(tx)
		txProfileRepo := authrepo.NewProfileRepository(tx)
		txRoleRepo := authrepo.NewRoleRepository(tx)

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
		if err := txUserRepo.Create(ctx, user); err != nil {
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
		if err := txProfileRepo.Create(ctx, profile); err != nil {
			return err
		}

		// 分配角色
		return txRoleRepo.AssignRole(ctx, userID, role.ID)
	})
	if err != nil {
		logger.L.Error("创建用户失败", zap.Error(err))
		return nil, errcode.ErrInternal.WithMessage("创建用户失败")
	}

	// 记录操作日志（使用 pkg/audit 公共包）
	audit.RecordFromContext(s.db, sc.UserID, sc.ClientIP, "create_user", "user", userID, map[string]interface{}{
		"phone": mask.Phone(req.Phone),
		"name":  req.Name,
		"role":  req.Role,
	})

	return &dto.CreateUserResp{
		ID: strconv.FormatInt(userID, 10),
	}, nil
}

// CreateSuperAdmin 创建超级管理员账号。
// 超级管理员为平台级账号，不隶属任何学校，school_id 固定写入 0。
// 超级管理员权限边界由 super_admin 角色决定，不依赖 school_id 做数据范围收缩。
func (s *userService) CreateSuperAdmin(ctx context.Context, sc *svcctx.ServiceContext, req *dto.CreateSuperAdminReq) (*dto.CreateUserResp, error) {
	if !sc.IsSuperAdmin() {
		return nil, errcode.ErrForbidden
	}

	schoolID := int64(0)

	if existing, err := s.userRepo.GetByPhone(ctx, req.Phone); err == nil && existing != nil {
		return nil, errcode.ErrDuplicatePhone
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errcode.ErrInternal.WithMessage("校验手机号失败")
	}

	if err := s.validatePasswordByPolicy(ctx, req.Password); err != nil {
		return nil, err
	}
	hash, err := crypto.HashPassword(req.Password)
	if err != nil {
		return nil, errcode.ErrInternal.WithMessage("密码加密失败")
	}

	role, err := s.roleRepo.GetByCode(ctx, enum.RoleSuperAdmin)
	if err != nil {
		return nil, errcode.ErrInternal.WithMessage("超级管理员角色不存在")
	}

	userID := snowflake.Generate()
	err = database.TransactionWithDB(ctx, s.db, func(tx *gorm.DB) error {
		txUserRepo := authrepo.NewUserRepository(tx)
		txProfileRepo := authrepo.NewProfileRepository(tx)
		txRoleRepo := authrepo.NewRoleRepository(tx)

		user := &entity.User{
			ID:            userID,
			Phone:         req.Phone,
			PasswordHash:  hash,
			Name:          req.Name,
			SchoolID:      schoolID,
			Status:        enum.UserStatusActive,
			IsFirstLogin:  true,
			IsSchoolAdmin: false,
			CreatedBy:     &sc.UserID,
		}
		if err := txUserRepo.Create(ctx, user); err != nil {
			return err
		}

		profile := &entity.UserProfile{
			ID:     snowflake.Generate(),
			UserID: userID,
			Email:  req.Email,
			Remark: req.Remark,
		}
		if err := txProfileRepo.Create(ctx, profile); err != nil {
			return err
		}

		return txRoleRepo.AssignRole(ctx, userID, role.ID)
	})
	if err != nil {
		logger.L.Error("创建超级管理员失败", zap.Error(err))
		return nil, errcode.ErrInternal.WithMessage("创建超级管理员失败")
	}

	audit.RecordFromContext(s.db, sc.UserID, sc.ClientIP, "create_super_admin", "user", userID, map[string]interface{}{
		"phone": mask.Phone(req.Phone),
		"name":  req.Name,
	})

	return &dto.CreateUserResp{ID: strconv.FormatInt(userID, 10)}, nil
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

	var currentProfile *entity.UserProfile
	currentProfile, err = s.profileRepo.GetByUserID(ctx, id)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return errcode.ErrInternal.WithMessage("查询用户扩展信息失败")
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		currentProfile = nil
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
	err = database.TransactionWithDB(ctx, s.db, func(tx *gorm.DB) error {
		txUserRepo := authrepo.NewUserRepository(tx)
		txProfileRepo := authrepo.NewProfileRepository(tx)

		if len(userFields) > 0 {
			if err := txUserRepo.UpdateFields(ctx, id, userFields); err != nil {
				return err
			}
		}

		if len(profileFields) == 0 {
			return nil
		}

		if _, err := txProfileRepo.GetByUserID(ctx, id); err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			profile := &entity.UserProfile{
				UserID: id,
			}
			applyProfileCreateFields(profile, profileFields)
			return txProfileRepo.Create(ctx, profile)
		}

		return txProfileRepo.UpdateFields(ctx, id, profileFields)
	})
	if err != nil {
		return errcode.ErrInternal.WithMessage("更新用户信息失败")
	}

	beforeSnapshot := buildUserUpdateSnapshot(user, currentProfile)
	afterSnapshot := buildUserUpdateSnapshot(user, currentProfile)
	applyUserUpdateSnapshot(afterSnapshot, userFields, profileFields)

	// 记录操作日志（使用标准库 maps.Keys 替代自定义 mapKeys）
	allKeys := slices.Concat(slices.Collect(maps.Keys(userFields)), slices.Collect(maps.Keys(profileFields)))
	audit.RecordFromContext(s.db, sc.UserID, sc.ClientIP, "update_user", "user", id, map[string]interface{}{
		"updated_fields":  allKeys,
		"before_snapshot": beforeSnapshot,
		"after_snapshot":  afterSnapshot,
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
		return errcode.ErrForbidden.WithMessage("不能删除自己的账号")
	}
	if err := s.ensureSuperAdminRetained(ctx, []*entity.User{user}); err != nil {
		return err
	}

	if err := s.userRepo.SoftDelete(ctx, id); err != nil {
		return errcode.ErrInternal.WithMessage("删除用户失败")
	}

	// 踢下线
	s.kickUser(ctx, id)

	// 记录操作日志
	audit.RecordFromContext(s.db, sc.UserID, sc.ClientIP, "delete_user", "user", id, map[string]interface{}{
		"phone": mask.Phone(user.Phone),
		"name":  user.Name,
	})

	return nil
}

// BatchDelete 批量删除用户
func (s *userService) BatchDelete(ctx context.Context, sc *svcctx.ServiceContext, ids []int64) error {
	// 边界检查：不允许批量删除中包含自己
	for _, id := range ids {
		if id == sc.UserID {
			return errcode.ErrForbidden.WithMessage("不能删除自己的账号")
		}
	}

	users, err := s.userRepo.GetByIDs(ctx, ids)
	if err != nil {
		return errcode.ErrInternal.WithMessage("查询批量删除用户失败")
	}
	if len(users) != len(ids) {
		return errcode.ErrUserNotFound
	}
	for _, user := range users {
		if sc.SchoolID > 0 && user.SchoolID != sc.SchoolID {
			return errcode.ErrUserNotFound
		}
	}
	if err := s.ensureSuperAdminRetained(ctx, users); err != nil {
		return err
	}

	if err := s.userRepo.BatchSoftDelete(ctx, ids, sc.SchoolID); err != nil {
		return errcode.ErrInternal.WithMessage("批量删除用户失败")
	}

	// 批量踢下线
	for _, user := range users {
		s.kickUser(ctx, user.ID)
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
	if (req.Status == enum.UserStatusDisabled || req.Status == enum.UserStatusArchived) && user.Status == enum.UserStatusActive {
		if err := s.ensureSuperAdminRetained(ctx, []*entity.User{user}); err != nil {
			return err
		}
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
	if err := s.validatePasswordByPolicy(ctx, newPassword); err != nil {
		return err
	}
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

// buildUserExtraData 批量装配用户扩展信息和角色编码。
func (s *userService) buildUserExtraData(ctx context.Context, users []*entity.User) (map[int64]*entity.UserProfile, map[int64][]string, map[int64]*entity.School, error) {
	profileMap := make(map[int64]*entity.UserProfile, len(users))
	roleCodeMap := make(map[int64][]string, len(users))
	schoolMap := make(map[int64]*entity.School)
	if len(users) == 0 {
		return profileMap, roleCodeMap, schoolMap, nil
	}

	userIDs := make([]int64, 0, len(users))
	schoolIDs := make([]int64, 0, len(users))
	schoolIDSet := make(map[int64]bool)
	for _, user := range users {
		userIDs = append(userIDs, user.ID)
		// 收集唯一的学校ID（排除0，即超管）
		if user.SchoolID > 0 && !schoolIDSet[user.SchoolID] {
			schoolIDs = append(schoolIDs, user.SchoolID)
			schoolIDSet[user.SchoolID] = true
		}
	}

	profiles, err := s.profileRepo.GetByUserIDs(ctx, userIDs)
	if err != nil {
		return nil, nil, nil, errcode.ErrInternal.WithMessage("查询用户扩展信息失败")
	}
	for _, profile := range profiles {
		profileMap[profile.UserID] = profile
	}

	roleCodeMap, err = s.roleRepo.GetUserRoleCodesMap(ctx, userIDs)
	if err != nil {
		return nil, nil, nil, errcode.ErrInternal.WithMessage("查询用户角色失败")
	}

	// 批量查询学校信息
	if len(schoolIDs) > 0 {
		schools, err := s.schoolRepo.GetByIDs(ctx, schoolIDs)
		if err != nil {
			return nil, nil, nil, errcode.ErrInternal.WithMessage("查询学校信息失败")
		}
		for _, school := range schools {
			schoolMap[school.ID] = school
		}
	}

	return profileMap, roleCodeMap, schoolMap, nil
}

// applyProfileCreateFields 将更新字段映射到新建的扩展信息实体。
func applyProfileCreateFields(profile *entity.UserProfile, fields map[string]interface{}) {
	if value, ok := fields["college"].(string); ok {
		profile.College = &value
	}
	if value, ok := fields["major"].(string); ok {
		profile.Major = &value
	}
	if value, ok := fields["class_name"].(string); ok {
		profile.ClassName = &value
	}
	if value, ok := fields["email"].(string); ok {
		profile.Email = &value
	}
	if value, ok := fields["remark"].(string); ok {
		profile.Remark = &value
	}
	if value, ok := fields["enrollment_year"].(int16); ok {
		profile.EnrollmentYear = &value
	}
	if value, ok := fields["education_level"].(int16); ok {
		profile.EducationLevel = &value
	}
	if value, ok := fields["grade"].(int16); ok {
		profile.Grade = &value
	}
}

// kickUser 踢用户下线（清除Session）
func (s *userService) kickUser(ctx context.Context, userID int64) {
	invalidateUserSession(ctx, s.userRepo, userID, resolveAccessTokenTTLByProvider(ctx, s.policyProvider))
}

// buildUserUpdateSnapshot 构建用户更新前后的资料快照。
func buildUserUpdateSnapshot(user *entity.User, profile *entity.UserProfile) map[string]interface{} {
	snapshot := map[string]interface{}{
		"name":       user.Name,
		"student_no": user.StudentNo,
	}
	if profile == nil {
		snapshot["college"] = nil
		snapshot["major"] = nil
		snapshot["class_name"] = nil
		snapshot["enrollment_year"] = nil
		snapshot["education_level"] = nil
		snapshot["grade"] = nil
		snapshot["email"] = nil
		snapshot["remark"] = nil
		return snapshot
	}

	snapshot["college"] = profile.College
	snapshot["major"] = profile.Major
	snapshot["class_name"] = profile.ClassName
	snapshot["enrollment_year"] = profile.EnrollmentYear
	snapshot["education_level"] = profile.EducationLevel
	snapshot["grade"] = profile.Grade
	snapshot["email"] = profile.Email
	snapshot["remark"] = profile.Remark
	return snapshot
}

// applyUserUpdateSnapshot 根据更新字段生成更新后的快照内容。
func applyUserUpdateSnapshot(snapshot map[string]interface{}, userFields, profileFields map[string]interface{}) {
	for key, value := range userFields {
		snapshot[key] = value
	}
	for key, value := range profileFields {
		snapshot[key] = value
	}
}

// validatePasswordByPolicy 按运行时安全策略校验密码复杂度
func (s *userService) validatePasswordByPolicy(ctx context.Context, password string) error {
	if s.policyProvider == nil {
		return validatePasswordWithPolicy(password, defaultRuntimeSecurityPolicy())
	}
	policy, err := s.policyProvider.GetRuntimeSecurityPolicy(ctx)
	if err != nil {
		return err
	}
	return validatePasswordWithPolicy(password, policy)
}

// ensureSuperAdminRetained 校验操作完成后是否仍会保留至少一个正常超级管理员。
// 删除、批量删除、禁用、归档都会让账号失去可用性，因此在真正写库前必须先通过这里校验。
func (s *userService) ensureSuperAdminRetained(ctx context.Context, users []*entity.User) error {
	if len(users) == 0 {
		return nil
	}

	userIDs := make([]int64, 0, len(users))
	activeUserMap := make(map[int64]bool, len(users))
	for _, user := range users {
		userIDs = append(userIDs, user.ID)
		activeUserMap[user.ID] = user.Status == enum.UserStatusActive
	}

	roleCodeMap, err := s.roleRepo.GetUserRoleCodesMap(ctx, userIDs)
	if err != nil {
		return errcode.ErrInternal.WithMessage("查询用户角色失败")
	}

	affectedActiveSuperAdmins := 0
	for _, user := range users {
		if !activeUserMap[user.ID] {
			continue
		}
		if slices.Contains(roleCodeMap[user.ID], enum.RoleSuperAdmin) {
			affectedActiveSuperAdmins++
		}
	}
	if affectedActiveSuperAdmins == 0 {
		return nil
	}

	superAdminCount, err := s.roleRepo.CountActiveUsersByRoleCode(ctx, enum.RoleSuperAdmin)
	if err != nil {
		return errcode.ErrInternal.WithMessage("查询超级管理员数量失败")
	}
	if superAdminCount-int64(affectedActiveSuperAdmins) < 1 {
		return errcode.ErrForbidden.WithMessage("至少保留一个超级管理员")
	}

	return nil
}
