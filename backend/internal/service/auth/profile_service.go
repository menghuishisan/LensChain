// profile_service.go
// 模块01 — 用户与认证：个人中心业务逻辑
// 负责获取个人信息、更新个人信息
// 对照 docs/modules/01-用户与认证/03-API接口设计.md

package auth

import (
	"context"
	"strconv"
	"time"

	"gorm.io/gorm"

	"github.com/lenschain/backend/internal/model/dto"
	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/model/enum"
	"github.com/lenschain/backend/internal/repository/auth"
	svcctx "github.com/lenschain/backend/internal/pkg/context"
	"github.com/lenschain/backend/internal/pkg/errcode"
	"github.com/lenschain/backend/internal/pkg/mask"
)

// ProfileService 个人中心服务接口
type ProfileService interface {
	GetProfile(ctx context.Context, sc *svcctx.ServiceContext) (*dto.ProfileResp, error)
	UpdateProfile(ctx context.Context, sc *svcctx.ServiceContext, req *dto.UpdateProfileReq) error
}

// profileService 个人中心服务实现
type profileService struct {
	db                *gorm.DB
	userRepo          authrepo.UserRepository
	profileRepo       authrepo.ProfileRepository
	roleRepo          authrepo.RoleRepository
	schoolNameQuerier SchoolNameQuerier
}

// NewProfileService 创建个人中心服务实例
func NewProfileService(
	db *gorm.DB,
	userRepo authrepo.UserRepository,
	profileRepo authrepo.ProfileRepository,
	roleRepo authrepo.RoleRepository,
	schoolNameQuerier ...SchoolNameQuerier,
) ProfileService {
	ps := &profileService{
		db:          db,
		userRepo:    userRepo,
		profileRepo: profileRepo,
		roleRepo:    roleRepo,
	}
	if len(schoolNameQuerier) > 0 && schoolNameQuerier[0] != nil {
		ps.schoolNameQuerier = schoolNameQuerier[0]
	}
	return ps
}

// GetProfile 获取个人信息
// 手机号脱敏展示，包含学习概览
func (s *profileService) GetProfile(ctx context.Context, sc *svcctx.ServiceContext) (*dto.ProfileResp, error) {
	user, err := s.userRepo.GetByID(ctx, sc.UserID)
	if err != nil {
		return nil, errcode.ErrUserNotFound
	}

	resp := &dto.ProfileResp{
		ID:        strconv.FormatInt(user.ID, 10),
		Phone:     mask.Phone(user.Phone), // 手机号脱敏
		Name:      user.Name,
		StudentNo: user.StudentNo,
		SchoolName: "", // 下方通过跨模块查询填充
		Roles:     make([]string, 0),
		LearningOverview: dto.LearningOverview{
			CourseCount:      0,
			ExperimentCount:  0,
			CompetitionCount: 0,
			TotalStudyHours:  0,
		},
	}

	// 角色
	for _, ur := range user.Roles {
		if ur.Role != nil {
			resp.Roles = append(resp.Roles, ur.Role.Code)
		}
	}

	// 扩展信息
	if user.Profile != nil {
		resp.Nickname = user.Profile.Nickname
		resp.AvatarURL = user.Profile.AvatarURL
		resp.Email = user.Profile.Email
		resp.College = user.Profile.College
		resp.Major = user.Profile.Major
		resp.ClassName = user.Profile.ClassName
		resp.EducationLevel = user.Profile.EducationLevel
		if user.Profile.EducationLevel != nil {
			text := enum.GetEduLevelText(*user.Profile.EducationLevel)
			resp.EducationLevelText = &text
		}
	}

	// 跨模块查询学校名称
	if s.schoolNameQuerier != nil {
		resp.SchoolName = s.schoolNameQuerier.GetSchoolName(ctx, user.SchoolID)
	}

	// TODO: 学习概览数据需要从其他模块聚合
	// 当前返回默认值，待模块03/04/05实现后补充

	return resp, nil
}

// UpdateProfile 更新个人信息
// 学生仅可修改：昵称、头像、邮箱
func (s *profileService) UpdateProfile(ctx context.Context, sc *svcctx.ServiceContext, req *dto.UpdateProfileReq) error {
	// 确保用户存在
	_, err := s.userRepo.GetByID(ctx, sc.UserID)
	if err != nil {
		return errcode.ErrUserNotFound
	}

	// 构建更新字段
	fields := make(map[string]interface{})
	if req.Nickname != nil {
		fields["nickname"] = *req.Nickname
	}
	if req.AvatarURL != nil {
		fields["avatar_url"] = *req.AvatarURL
	}
	if req.Email != nil {
		fields["email"] = *req.Email
	}

	if len(fields) == 0 {
		return nil // 无需更新
	}

	fields["updated_at"] = time.Now()

	// 确保 profile 存在，不存在则创建
	profile, err := s.profileRepo.GetByUserID(ctx, sc.UserID)
	if err != nil {
		// profile 不存在，创建
		newProfile := &entity.UserProfile{
			UserID: sc.UserID,
		}
		if req.Nickname != nil {
			newProfile.Nickname = req.Nickname
		}
		if req.AvatarURL != nil {
			newProfile.AvatarURL = req.AvatarURL
		}
		if req.Email != nil {
			newProfile.Email = req.Email
		}
		return s.profileRepo.Create(ctx, newProfile)
	}

	_ = profile // 已存在，更新字段
	return s.profileRepo.UpdateFields(ctx, sc.UserID, fields)
}
