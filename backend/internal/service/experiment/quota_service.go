// quota_service.go
// 模块04 — 实验环境：资源配额业务逻辑
// 负责学校/课程配额管理、学校资源使用情况、课程配额分配

package experiment

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
	"github.com/lenschain/backend/internal/pkg/snowflake"
	experimentrepo "github.com/lenschain/backend/internal/repository/experiment"
)

// QuotaService 资源配额服务接口。
type QuotaService interface {
	Create(ctx context.Context, sc *svcctx.ServiceContext, req *dto.CreateQuotaReq) (*dto.QuotaResp, error)
	List(ctx context.Context, sc *svcctx.ServiceContext, req *dto.QuotaListReq) ([]dto.QuotaResp, int64, error)
	GetByID(ctx context.Context, sc *svcctx.ServiceContext, id int64) (*dto.QuotaResp, error)
	Update(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.UpdateQuotaReq) (*dto.QuotaResp, error)
	GetSchoolUsage(ctx context.Context, sc *svcctx.ServiceContext, schoolID int64) (*dto.ResourceUsageResp, error)
	AssignCourseQuota(ctx context.Context, sc *svcctx.ServiceContext, courseID int64, req *dto.CourseQuotaReq) (*dto.QuotaResp, error)
}

// quotaService 资源配额服务实现。
// 统一封装学校级、课程级配额的创建、查询、更新和资源使用汇总。
type quotaService struct {
	quotaRepo         experimentrepo.QuotaRepository
	instanceRepo      experimentrepo.InstanceRepository
	schoolNameQuerier SchoolNameQuerier
	courseQuerier     CourseQuerier
}

// NewQuotaService 创建资源配额服务实例。
func NewQuotaService(
	quotaRepo experimentrepo.QuotaRepository,
	instanceRepo experimentrepo.InstanceRepository,
	schoolNameQuerier SchoolNameQuerier,
	courseQuerier CourseQuerier,
) QuotaService {
	return &quotaService{
		quotaRepo:         quotaRepo,
		instanceRepo:      instanceRepo,
		schoolNameQuerier: schoolNameQuerier,
		courseQuerier:     courseQuerier,
	}
}

// Create 创建学校级或课程级资源配额。
func (s *quotaService) Create(ctx context.Context, sc *svcctx.ServiceContext, req *dto.CreateQuotaReq) (*dto.QuotaResp, error) {
	if !sc.IsSuperAdmin() {
		return nil, errcode.ErrForbidden
	}

	schoolID, err := snowflake.ParseString(req.SchoolID)
	if err != nil {
		return nil, errcode.ErrInvalidParams.WithMessage("学校ID无效")
	}

	var courseID *int64
	if req.CourseID != nil && *req.CourseID != "" {
		parsedCourseID, parseErr := snowflake.ParseString(*req.CourseID)
		if parseErr != nil {
			return nil, errcode.ErrInvalidParams.WithMessage("课程ID无效")
		}
		courseID = &parsedCourseID
	}

	if req.QuotaLevel == enum.QuotaLevelCourse && courseID == nil {
		return nil, errcode.ErrInvalidParams.WithMessage("课程级配额必须指定课程ID")
	}

	if _, err := s.quotaRepo.GetBySchoolAndCourse(ctx, schoolID, courseID); err == nil {
		return nil, errcode.ErrQuotaAlreadyExists
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	if err := s.validateQuotaWithinSchool(ctx, schoolID, courseID, req.MaxConcurrency, req.MaxPerStudent); err != nil {
		return nil, err
	}

	quota := &entity.ResourceQuota{
		ID:             snowflake.Generate(),
		QuotaLevel:     req.QuotaLevel,
		SchoolID:       schoolID,
		CourseID:       courseID,
		MaxCPU:         quotaString(req.MaxCPU),
		MaxMemory:      quotaString(req.MaxMemory),
		MaxStorage:     quotaString(req.MaxStorage),
		MaxConcurrency: req.MaxConcurrency,
		MaxPerStudent:  req.MaxPerStudent,
	}
	if err := s.quotaRepo.Create(ctx, quota); err != nil {
		return nil, err
	}
	return s.buildQuotaResp(ctx, quota), nil
}

// List 获取资源配额列表。
func (s *quotaService) List(ctx context.Context, sc *svcctx.ServiceContext, req *dto.QuotaListReq) ([]dto.QuotaResp, int64, error) {
	params := &experimentrepo.QuotaListParams{
		QuotaLevel: req.QuotaLevel,
		Page:       req.Page,
		PageSize:   req.PageSize,
	}
	if sc.IsSchoolAdmin() && !sc.IsSuperAdmin() {
		params.SchoolID = sc.SchoolID
	}
	if sc.IsSuperAdmin() && req.SchoolID != "" {
		schoolID, err := snowflake.ParseString(req.SchoolID)
		if err != nil {
			return nil, 0, errcode.ErrInvalidParams.WithMessage("学校ID无效")
		}
		params.SchoolID = schoolID
	}

	quotas, total, err := s.quotaRepo.List(ctx, params)
	if err != nil {
		return nil, 0, err
	}
	items := make([]dto.QuotaResp, 0, len(quotas))
	for _, quota := range quotas {
		items = append(items, *s.buildQuotaResp(ctx, quota))
	}
	return items, total, nil
}

// GetByID 获取配额详情。
func (s *quotaService) GetByID(ctx context.Context, sc *svcctx.ServiceContext, id int64) (*dto.QuotaResp, error) {
	quota, err := s.quotaRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errcode.ErrQuotaNotFound
		}
		return nil, err
	}
	if !sc.IsSuperAdmin() && quota.SchoolID != sc.SchoolID {
		return nil, errcode.ErrForbidden
	}
	return s.buildQuotaResp(ctx, quota), nil
}

// Update 更新资源配额。
func (s *quotaService) Update(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.UpdateQuotaReq) (*dto.QuotaResp, error) {
	quota, err := s.quotaRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errcode.ErrQuotaNotFound
		}
		return nil, err
	}
	if !sc.IsSuperAdmin() && quota.SchoolID != sc.SchoolID {
		return nil, errcode.ErrForbidden
	}

	targetConcurrency := quota.MaxConcurrency
	targetPerStudent := quota.MaxPerStudent
	if req.MaxConcurrency != nil {
		targetConcurrency = *req.MaxConcurrency
	}
	if req.MaxPerStudent != nil {
		targetPerStudent = *req.MaxPerStudent
	}
	if err := s.validateQuotaWithinSchool(ctx, quota.SchoolID, quota.CourseID, targetConcurrency, targetPerStudent); err != nil {
		return nil, err
	}

	fields := map[string]interface{}{
		"updated_at": time.Now(),
	}
	if req.MaxCPU != nil {
		fields["max_cpu"] = *req.MaxCPU
	}
	if req.MaxMemory != nil {
		fields["max_memory"] = *req.MaxMemory
	}
	if req.MaxStorage != nil {
		fields["max_storage"] = *req.MaxStorage
	}
	if req.MaxConcurrency != nil {
		fields["max_concurrency"] = *req.MaxConcurrency
	}
	if req.MaxPerStudent != nil {
		fields["max_per_student"] = *req.MaxPerStudent
	}

	if err := s.quotaRepo.UpdateFields(ctx, id, fields); err != nil {
		return nil, err
	}
	quota, _ = s.quotaRepo.GetByID(ctx, id)
	return s.buildQuotaResp(ctx, quota), nil
}

// GetSchoolUsage 获取学校资源使用情况。
func (s *quotaService) GetSchoolUsage(ctx context.Context, sc *svcctx.ServiceContext, schoolID int64) (*dto.ResourceUsageResp, error) {
	if !sc.IsSuperAdmin() && schoolID != sc.SchoolID {
		return nil, errcode.ErrForbidden
	}

	schoolQuota, err := s.quotaRepo.GetBySchoolID(ctx, schoolID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errcode.ErrQuotaNotFound
		}
		return nil, err
	}

	courseQuotas, err := s.quotaRepo.ListBySchoolID(ctx, schoolID)
	if err != nil {
		return nil, err
	}

	breakdown := make([]dto.CourseBreakdownItem, 0)
	for _, quota := range courseQuotas {
		if quota.CourseID == nil {
			continue
		}
		breakdown = append(breakdown, dto.CourseBreakdownItem{
			CourseID:           strconv.FormatInt(*quota.CourseID, 10),
			CourseTitle:        s.courseQuerier.GetCourseTitle(ctx, *quota.CourseID),
			CurrentConcurrency: quota.UsedConcurrency,
			MaxConcurrency:     quota.MaxConcurrency,
			CPUUsed:            quota.UsedCPU,
			MemoryUsed:         quota.UsedMemory,
		})
	}

	return &dto.ResourceUsageResp{
		SchoolID:   strconv.FormatInt(schoolID, 10),
		SchoolName: s.schoolNameQuerier.GetSchoolName(ctx, schoolID),
		Quota: dto.ResourceQuotaInfo{
			MaxCPU:         schoolQuota.MaxCPU,
			MaxMemory:      schoolQuota.MaxMemory,
			MaxStorage:     schoolQuota.MaxStorage,
			MaxConcurrency: schoolQuota.MaxConcurrency,
		},
		Usage: dto.ResourceUsageInfo{
			UsedCPU:                 schoolQuota.UsedCPU,
			UsedMemory:              schoolQuota.UsedMemory,
			UsedStorage:             schoolQuota.UsedStorage,
			CurrentConcurrency:      schoolQuota.UsedConcurrency,
			CPUUsagePercent:         percentString(schoolQuota.UsedCPU, schoolQuota.MaxCPU),
			MemoryUsagePercent:      percentString(schoolQuota.UsedMemory, schoolQuota.MaxMemory),
			StorageUsagePercent:     percentString(schoolQuota.UsedStorage, schoolQuota.MaxStorage),
			ConcurrencyUsagePercent: percentInt(schoolQuota.UsedConcurrency, schoolQuota.MaxConcurrency),
		},
		CourseBreakdown: breakdown,
	}, nil
}

// AssignCourseQuota 为课程分配或更新课程级并发配额。
func (s *quotaService) AssignCourseQuota(ctx context.Context, sc *svcctx.ServiceContext, courseID int64, req *dto.CourseQuotaReq) (*dto.QuotaResp, error) {
	if !sc.IsSchoolAdmin() && !sc.IsSuperAdmin() {
		return nil, errcode.ErrForbidden
	}

	courseSchoolID, err := s.courseQuerier.GetCourseSchoolID(ctx, courseID)
	if err != nil {
		return nil, err
	}
	if !sc.IsSuperAdmin() && courseSchoolID != sc.SchoolID {
		return nil, errcode.ErrForbidden
	}

	if err := s.validateQuotaWithinSchool(ctx, courseSchoolID, &courseID, req.MaxConcurrency, req.MaxPerStudent); err != nil {
		return nil, err
	}

	quota, err := s.quotaRepo.GetByCourseID(ctx, courseID)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		schoolQuota, schoolErr := s.quotaRepo.GetBySchoolID(ctx, courseSchoolID)
		if schoolErr != nil {
			return nil, schoolErr
		}
		quota = &entity.ResourceQuota{
			ID:             snowflake.Generate(),
			QuotaLevel:     enum.QuotaLevelCourse,
			SchoolID:       courseSchoolID,
			CourseID:       &courseID,
			MaxCPU:         schoolQuota.MaxCPU,
			MaxMemory:      schoolQuota.MaxMemory,
			MaxStorage:     schoolQuota.MaxStorage,
			MaxConcurrency: req.MaxConcurrency,
			MaxPerStudent:  req.MaxPerStudent,
		}
		if createErr := s.quotaRepo.Create(ctx, quota); createErr != nil {
			return nil, createErr
		}
		return s.buildQuotaResp(ctx, quota), nil
	}

	if err := s.quotaRepo.UpdateFields(ctx, quota.ID, map[string]interface{}{
		"max_concurrency": req.MaxConcurrency,
		"max_per_student": req.MaxPerStudent,
		"updated_at":      time.Now(),
	}); err != nil {
		return nil, err
	}
	quota, _ = s.quotaRepo.GetByID(ctx, quota.ID)
	return s.buildQuotaResp(ctx, quota), nil
}

// buildQuotaResp 构建配额响应。
func (s *quotaService) buildQuotaResp(ctx context.Context, quota *entity.ResourceQuota) *dto.QuotaResp {
	if quota == nil {
		return nil
	}
	resp := &dto.QuotaResp{
		ID:              strconv.FormatInt(quota.ID, 10),
		QuotaLevel:      quota.QuotaLevel,
		QuotaLevelText:  enum.GetQuotaLevelText(quota.QuotaLevel),
		SchoolID:        strconv.FormatInt(quota.SchoolID, 10),
		SchoolName:      s.schoolNameQuerier.GetSchoolName(ctx, quota.SchoolID),
		MaxCPU:          quota.MaxCPU,
		MaxMemory:       quota.MaxMemory,
		MaxStorage:      quota.MaxStorage,
		MaxConcurrency:  quota.MaxConcurrency,
		MaxPerStudent:   quota.MaxPerStudent,
		UsedCPU:         quota.UsedCPU,
		UsedMemory:      quota.UsedMemory,
		UsedStorage:     quota.UsedStorage,
		UsedConcurrency: quota.UsedConcurrency,
		CreatedAt:       quota.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:       quota.UpdatedAt.UTC().Format(time.RFC3339),
	}
	if quota.CourseID != nil {
		courseID := strconv.FormatInt(*quota.CourseID, 10)
		resp.CourseID = &courseID
		title := s.courseQuerier.GetCourseTitle(ctx, *quota.CourseID)
		if title != "" {
			resp.CourseTitle = &title
		}
	}
	return resp
}

// validateQuotaWithinSchool 校验课程级配额是否超过学校级配额。
func (s *quotaService) validateQuotaWithinSchool(ctx context.Context, schoolID int64, courseID *int64, maxConcurrency, maxPerStudent int) error {
	if courseID == nil {
		return nil
	}
	schoolQuota, err := s.quotaRepo.GetBySchoolID(ctx, schoolID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errcode.ErrQuotaNotFound.WithMessage("学校级配额不存在")
		}
		return err
	}
	quotas, err := s.quotaRepo.ListBySchoolID(ctx, schoolID)
	if err != nil {
		return err
	}
	if maxConcurrency > calculateRemainingConcurrency(schoolQuota.MaxConcurrency, quotas, courseID) {
		return errcode.ErrInvalidParams.WithMessage("课程配额不能超过学校剩余可分配量")
	}
	if maxPerStudent > schoolQuota.MaxPerStudent {
		return errcode.ErrInvalidParams.WithMessage("课程单学生配额不能超过学校级限制")
	}
	return nil
}

// quotaString 将空字符串资源值规范化为 0，避免数据库出现空值语义分歧。
func quotaString(value string) string {
	if value == "" {
		return "0"
	}
	return value
}

// percentString 计算字符串数值表示的资源使用百分比。
func percentString(used, total string) float64 {
	usedValue, usedErr := strconv.ParseFloat(used, 64)
	totalValue, totalErr := strconv.ParseFloat(total, 64)
	if usedErr != nil || totalErr != nil || totalValue <= 0 {
		return 0
	}
	return usedValue / totalValue * 100
}

// percentInt 计算整数配额的使用百分比。
func percentInt(used, total int) float64 {
	if total <= 0 {
		return 0
	}
	return float64(used) / float64(total) * 100
}
