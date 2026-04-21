// profile_service.go
// 模块01 — 用户与认证：个人中心业务逻辑
// 负责获取个人信息、更新个人信息
// 对照 docs/modules/01-用户与认证/03-API接口设计.md

package auth

import (
	"context"
	"errors"
	"strconv"
	"time"

	"gorm.io/gorm"

	"github.com/lenschain/backend/internal/model/dto"
	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/model/enum"
	svcctx "github.com/lenschain/backend/internal/pkg/context"
	"github.com/lenschain/backend/internal/pkg/errcode"
	"github.com/lenschain/backend/internal/pkg/mask"
	"github.com/lenschain/backend/internal/repository/auth"
)

// ProfileService 个人中心服务接口
type ProfileService interface {
	GetProfile(ctx context.Context, sc *svcctx.ServiceContext) (*dto.ProfileResp, error)
	UpdateProfile(ctx context.Context, sc *svcctx.ServiceContext, req *dto.UpdateProfileReq) error
}

// LearningOverviewQuerier 跨模块接口：查询学习概览
// 由模块03/04/05按需聚合实现，避免模块01直接依赖其他模块 service
type LearningOverviewQuerier interface {
	GetLearningOverview(ctx context.Context, userID int64) (*dto.LearningOverview, error)
}

// profileService 个人中心服务实现
type profileService struct {
	db                *gorm.DB
	userRepo          authrepo.UserRepository
	profileRepo       authrepo.ProfileRepository
	roleRepo          authrepo.RoleRepository
	schoolNameQuerier SchoolNameQuerier
	overviewQuerier   LearningOverviewQuerier
}

// NewProfileService 创建个人中心服务实例
func NewProfileService(
	db *gorm.DB,
	userRepo authrepo.UserRepository,
	profileRepo authrepo.ProfileRepository,
	roleRepo authrepo.RoleRepository,
	schoolNameQuerier SchoolNameQuerier,
	overviewQuerier LearningOverviewQuerier,
) ProfileService {
	ps := &profileService{
		db:                db,
		userRepo:          userRepo,
		profileRepo:       profileRepo,
		roleRepo:          roleRepo,
		schoolNameQuerier: schoolNameQuerier,
		overviewQuerier:   overviewQuerier,
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
	roleCodes, err := s.roleRepo.GetUserRoleCodes(ctx, user.ID)
	if err != nil {
		return nil, errcode.ErrInternal.WithMessage("查询用户角色失败")
	}
	profile, err := s.profileRepo.GetByUserID(ctx, user.ID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errcode.ErrInternal.WithMessage("查询个人资料失败")
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		profile = nil
	}

	resp := &dto.ProfileResp{
		ID:         strconv.FormatInt(user.ID, 10),
		Phone:      mask.Phone(user.Phone), // 手机号脱敏
		Name:       user.Name,
		StudentNo:  user.StudentNo,
		SchoolName: "", // 下方通过跨模块查询填充
		Roles:      roleCodes,
		LearningOverview: dto.LearningOverview{
			CourseCount:      0,
			ExperimentCount:  0,
			CompetitionCount: 0,
			TotalStudyHours:  0,
		},
	}

	// 扩展信息
	if profile != nil {
		resp.Nickname = profile.Nickname
		resp.AvatarURL = profile.AvatarURL
		resp.Email = profile.Email
		resp.College = profile.College
		resp.Major = profile.Major
		resp.ClassName = profile.ClassName
		resp.EducationLevel = profile.EducationLevel
		if profile.EducationLevel != nil {
			text := enum.GetEduLevelText(*profile.EducationLevel)
			resp.EducationLevelText = &text
		}
	}

	// 跨模块查询学校名称
	if s.schoolNameQuerier != nil {
		resp.SchoolName = s.schoolNameQuerier.GetSchoolName(ctx, user.SchoolID)
	}

	// 学习概览通过跨模块接口聚合，未实现的模块维持默认值
	if s.overviewQuerier != nil {
		overview, err := s.overviewQuerier.GetLearningOverview(ctx, sc.UserID)
		if err == nil && overview != nil {
			resp.LearningOverview = *overview
		}
	}

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
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return errcode.ErrInternal.WithMessage("查询个人资料失败")
		}
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
