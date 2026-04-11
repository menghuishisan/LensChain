// school_service.go
// 模块02 — 学校与租户管理：学校管理业务逻辑
// 负责学校 CRUD、查询、有效期设置、本校信息、授权状态
// 冻结/解冻、注销/恢复等生命周期操作在 school_lifecycle.go 中
// 对照 docs/modules/02-学校与租户管理/03-API接口设计.md

package school

import (
	"context"
	"github.com/lenschain/backend/internal/pkg/database"
	"math"
	"strconv"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/lenschain/backend/internal/model/dto"
	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/model/enum"
	"github.com/lenschain/backend/internal/pkg/audit"
	svcctx "github.com/lenschain/backend/internal/pkg/context"
	"github.com/lenschain/backend/internal/pkg/errcode"
	"github.com/lenschain/backend/internal/pkg/logger"
	"github.com/lenschain/backend/internal/pkg/sms"
	"github.com/lenschain/backend/internal/pkg/snowflake"
	schoolrepo "github.com/lenschain/backend/internal/repository/school"
)

// SchoolService 学校管理服务接口
type SchoolService interface {
	List(ctx context.Context, sc *svcctx.ServiceContext, req *dto.SchoolListReq) ([]*dto.SchoolListItem, int64, error)
	GetByID(ctx context.Context, sc *svcctx.ServiceContext, id int64) (*dto.SchoolDetailResp, error)
	Create(ctx context.Context, sc *svcctx.ServiceContext, req *dto.CreateSchoolReq) (*dto.CreateSchoolResp, error)
	Update(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.UpdateSchoolReq) error
	SetLicense(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.SetLicenseReq) error
	Freeze(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.FreezeSchoolReq) error
	Unfreeze(ctx context.Context, sc *svcctx.ServiceContext, id int64) error
	Cancel(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.CancelSchoolReq) error
	Restore(ctx context.Context, sc *svcctx.ServiceContext, id int64) error
	GetProfile(ctx context.Context, sc *svcctx.ServiceContext) (*dto.SchoolProfileResp, error)
	UpdateProfile(ctx context.Context, sc *svcctx.ServiceContext, req *dto.UpdateSchoolProfileReq) error
	GetLicenseStatus(ctx context.Context, sc *svcctx.ServiceContext) (*dto.LicenseStatusResp, error)
	GetSSOSchoolList(ctx context.Context) ([]*dto.SSOSchoolItem, error)
}

// schoolService 学校管理服务实现
type schoolService struct {
	db            *gorm.DB
	schoolRepo    schoolrepo.SchoolRepository
	adminCreator  AdminCreator
	sessionKicker SessionKicker
}

// SessionKicker 跨模块接口：踢出学校所有用户的Session
type SessionKicker interface {
	KickSchoolUsers(ctx context.Context, schoolID int64) error
}

// NewSchoolService 创建学校管理服务实例
func NewSchoolService(
	db *gorm.DB,
	schoolRepo schoolrepo.SchoolRepository,
	adminCreator AdminCreator,
	sessionKicker SessionKicker,
) SchoolService {
	return &schoolService{
		db:            db,
		schoolRepo:    schoolRepo,
		adminCreator:  adminCreator,
		sessionKicker: sessionKicker,
	}
}

// List 学校列表
func (s *schoolService) List(ctx context.Context, sc *svcctx.ServiceContext, req *dto.SchoolListReq) ([]*dto.SchoolListItem, int64, error) {
	params := &schoolrepo.SchoolListParams{
		Keyword:         req.Keyword,
		Status:          req.Status,
		LicenseExpiring: req.LicenseExpiring,
		SortBy:          req.SortBy,
		SortOrder:       req.SortOrder,
		Page:            req.Page,
		PageSize:        req.PageSize,
	}

	schools, total, err := s.schoolRepo.List(ctx, params)
	if err != nil {
		return nil, 0, errcode.ErrInternal.WithMessage("查询学校列表失败")
	}

	items := make([]*dto.SchoolListItem, 0, len(schools))
	for _, sch := range schools {
		items = append(items, schoolToListItem(sch))
	}

	return items, total, nil
}

// GetByID 学校详情
func (s *schoolService) GetByID(ctx context.Context, sc *svcctx.ServiceContext, id int64) (*dto.SchoolDetailResp, error) {
	sch, err := s.schoolRepo.GetByID(ctx, id)
	if err != nil {
		return nil, errcode.ErrSchoolNotFound
	}
	return schoolToDetailResp(sch), nil
}

// Create 后台直接创建学校
func (s *schoolService) Create(ctx context.Context, sc *svcctx.ServiceContext, req *dto.CreateSchoolReq) (*dto.CreateSchoolResp, error) {
	// 唯一性校验
	if existing, _ := s.schoolRepo.GetByName(ctx, req.Name); existing != nil {
		return nil, errcode.ErrDuplicateSchoolName
	}
	if existing, _ := s.schoolRepo.GetByCode(ctx, req.Code); existing != nil {
		return nil, errcode.ErrDuplicateSchoolCode
	}

	licenseStart, err := time.Parse(time.RFC3339, req.LicenseStartAt)
	if err != nil {
		return nil, errcode.ErrInvalidParams.WithMessage("授权开始时间格式不正确")
	}
	licenseEnd, err := time.Parse(time.RFC3339, req.LicenseEndAt)
	if err != nil {
		return nil, errcode.ErrInvalidParams.WithMessage("授权结束时间格式不正确")
	}

	var schoolID, adminUserID int64
	var adminPassword string

	err = database.TransactionWithDB(ctx, s.db, func(tx *gorm.DB) error {
		schoolID = snowflake.Generate()
		schoolEntity := &entity.School{
			ID:             schoolID,
			Name:           req.Name,
			Code:           req.Code,
			LogoURL:        req.LogoURL,
			Address:        req.Address,
			Website:        req.Website,
			Description:    req.Description,
			Status:         enum.SchoolStatusActive,
			LicenseStartAt: &licenseStart,
			LicenseEndAt:   &licenseEnd,
			ContactName:    req.ContactName,
			ContactPhone:   req.ContactPhone,
			ContactEmail:   req.ContactEmail,
			ContactTitle:   req.ContactTitle,
			CreatedBy:      &sc.UserID,
		}
		if err := tx.Create(schoolEntity).Error; err != nil {
			return err
		}

		var createErr error
		adminUserID, adminPassword, createErr = s.adminCreator.CreateSchoolAdmin(ctx, tx, schoolID, req.ContactPhone, req.ContactName, sc.UserID)
		return createErr
	})
	if err != nil {
		logger.L.Error("创建学校失败", zap.Error(err))
		return nil, errcode.ErrInternal.WithMessage("创建学校失败")
	}

	// 刷新缓存
	refreshSchoolStatusCache(ctx, schoolID, enum.SchoolStatusActive, &licenseEnd)

	// 异步发送短信
	go func() {
		_ = sms.Send(req.ContactPhone, sms.TemplateSchoolApproved, map[string]string{
			"school_name": req.Name,
			"phone":       req.ContactPhone,
			"password":    adminPassword,
		})
	}()

	audit.RecordFromContext(s.db, sc.UserID, sc.ClientIP, "create_school", "school", schoolID, map[string]interface{}{
		"name": req.Name,
		"code": req.Code,
	})

	return &dto.CreateSchoolResp{
		SchoolID:    strconv.FormatInt(schoolID, 10),
		AdminUserID: strconv.FormatInt(adminUserID, 10),
		AdminPhone:  req.ContactPhone,
		SMSSent:     true,
	}, nil
}

// Update 编辑学校信息（超管）
func (s *schoolService) Update(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.UpdateSchoolReq) error {
	sch, err := s.schoolRepo.GetByID(ctx, id)
	if err != nil {
		return errcode.ErrSchoolNotFound
	}

	// 唯一性校验
	if req.Name != nil && *req.Name != sch.Name {
		if existing, _ := s.schoolRepo.GetByName(ctx, *req.Name); existing != nil {
			return errcode.ErrDuplicateSchoolName
		}
	}
	if req.Code != nil && *req.Code != sch.Code {
		if existing, _ := s.schoolRepo.GetByCode(ctx, *req.Code); existing != nil {
			return errcode.ErrDuplicateSchoolCode
		}
	}

	fields := make(map[string]interface{})
	if req.Name != nil {
		fields["name"] = *req.Name
	}
	if req.Code != nil {
		fields["code"] = *req.Code
	}
	if req.Address != nil {
		fields["address"] = *req.Address
	}
	if req.Website != nil {
		fields["website"] = *req.Website
	}
	if req.LogoURL != nil {
		fields["logo_url"] = *req.LogoURL
	}
	if req.Description != nil {
		fields["description"] = *req.Description
	}
	if req.ContactName != nil {
		fields["contact_name"] = *req.ContactName
	}
	if req.ContactPhone != nil {
		fields["contact_phone"] = *req.ContactPhone
	}
	if req.ContactEmail != nil {
		fields["contact_email"] = *req.ContactEmail
	}
	if req.ContactTitle != nil {
		fields["contact_title"] = *req.ContactTitle
	}

	if len(fields) == 0 {
		return nil
	}
	fields["updated_at"] = time.Now()

	if err := s.schoolRepo.UpdateFields(ctx, id, fields); err != nil {
		return errcode.ErrInternal.WithMessage("更新学校信息失败")
	}

	audit.RecordFromContext(s.db, sc.UserID, sc.ClientIP, "update_school", "school", id, nil)
	return nil
}

// SetLicense 设置有效期
func (s *schoolService) SetLicense(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.SetLicenseReq) error {
	sch, err := s.schoolRepo.GetByID(ctx, id)
	if err != nil {
		return errcode.ErrSchoolNotFound
	}

	licenseEnd, err := time.Parse(time.RFC3339, req.LicenseEndAt)
	if err != nil {
		return errcode.ErrInvalidParams.WithMessage("授权结束时间格式不正确")
	}

	fields := map[string]interface{}{
		"license_end_at": licenseEnd,
		"updated_at":     time.Now(),
	}
	// 如果之前没有开始时间，设置为当前时间
	if sch.LicenseStartAt == nil {
		now := time.Now()
		fields["license_start_at"] = now
	}

	if err := s.schoolRepo.UpdateFields(ctx, id, fields); err != nil {
		return errcode.ErrInternal.WithMessage("设置有效期失败")
	}

	// 刷新缓存
	refreshSchoolStatusCache(ctx, id, sch.Status, &licenseEnd)

	audit.RecordFromContext(s.db, sc.UserID, sc.ClientIP, "set_license", "school", id, map[string]interface{}{
		"license_end_at": req.LicenseEndAt,
	})
	return nil
}

// GetProfile 获取本校信息（校管）
func (s *schoolService) GetProfile(ctx context.Context, sc *svcctx.ServiceContext) (*dto.SchoolProfileResp, error) {
	sch, err := s.schoolRepo.GetByID(ctx, sc.SchoolID)
	if err != nil {
		return nil, errcode.ErrSchoolNotFound
	}
	return &dto.SchoolProfileResp{
		ID:          strconv.FormatInt(sch.ID, 10),
		Name:        sch.Name,
		Code:        sch.Code,
		LogoURL:     sch.LogoURL,
		Address:     sch.Address,
		Website:     sch.Website,
		Description: sch.Description,
		Status:      sch.Status,
		StatusText:  enum.GetSchoolStatusText(sch.Status),
	}, nil
}

// UpdateProfile 编辑本校信息（校管，仅可修改 logo/description/address/website）
func (s *schoolService) UpdateProfile(ctx context.Context, sc *svcctx.ServiceContext, req *dto.UpdateSchoolProfileReq) error {
	_, err := s.schoolRepo.GetByID(ctx, sc.SchoolID)
	if err != nil {
		return errcode.ErrSchoolNotFound
	}

	fields := make(map[string]interface{})
	if req.LogoURL != nil {
		fields["logo_url"] = *req.LogoURL
	}
	if req.Description != nil {
		fields["description"] = *req.Description
	}
	if req.Address != nil {
		fields["address"] = *req.Address
	}
	if req.Website != nil {
		fields["website"] = *req.Website
	}

	if len(fields) == 0 {
		return nil
	}
	fields["updated_at"] = time.Now()

	return s.schoolRepo.UpdateFields(ctx, sc.SchoolID, fields)
}

// GetLicenseStatus 查看授权状态（校管）
func (s *schoolService) GetLicenseStatus(ctx context.Context, sc *svcctx.ServiceContext) (*dto.LicenseStatusResp, error) {
	sch, err := s.schoolRepo.GetByID(ctx, sc.SchoolID)
	if err != nil {
		return nil, errcode.ErrSchoolNotFound
	}

	resp := &dto.LicenseStatusResp{
		Status:     sch.Status,
		StatusText: enum.GetSchoolStatusText(sch.Status),
	}
	if sch.LicenseStartAt != nil {
		t := sch.LicenseStartAt.Format(time.RFC3339)
		resp.LicenseStartAt = &t
	}
	if sch.LicenseEndAt != nil {
		t := sch.LicenseEndAt.Format(time.RFC3339)
		resp.LicenseEndAt = &t
		remaining := int(math.Ceil(time.Until(*sch.LicenseEndAt).Hours() / 24))
		resp.RemainingDays = remaining
		resp.IsExpiringSoon = remaining >= 0 && remaining <= 7
	}

	return resp, nil
}

// GetSSOSchoolList 获取已配置SSO的学校列表（公开接口）
func (s *schoolService) GetSSOSchoolList(ctx context.Context) ([]*dto.SSOSchoolItem, error) {
	schools, err := s.schoolRepo.GetSSOEnabledSchools(ctx)
	if err != nil {
		return nil, errcode.ErrInternal.WithMessage("查询SSO学校列表失败")
	}

	items := make([]*dto.SSOSchoolItem, 0, len(schools))
	for _, sch := range schools {
		items = append(items, &dto.SSOSchoolItem{
			ID:      strconv.FormatInt(sch.ID, 10),
			Name:    sch.Name,
			LogoURL: sch.LogoURL,
		})
	}
	return items, nil
}
