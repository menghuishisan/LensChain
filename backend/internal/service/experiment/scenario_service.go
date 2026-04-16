// scenario_service.go
// 模块04 — 实验环境：仿真场景库业务逻辑
// 负责仿真场景CRUD、审核、联动组查询
// 对照 docs/modules/04-实验环境/03-API接口设计.md

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

// ScenarioService 仿真场景库服务接口
type ScenarioService interface {
	// 场景 CRUD
	Create(ctx context.Context, sc *svcctx.ServiceContext, req *dto.CreateScenarioReq) (string, error)
	GetByID(ctx context.Context, sc *svcctx.ServiceContext, id int64) (*dto.ScenarioResp, error)
	Update(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.UpdateScenarioReq) error
	Delete(ctx context.Context, sc *svcctx.ServiceContext, id int64) error
	List(ctx context.Context, sc *svcctx.ServiceContext, req *dto.ScenarioListReq) ([]*dto.ScenarioListItem, int64, error)

	// 审核
	Review(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.ReviewScenarioReq) error

	// 联动组查询
	ListLinkGroups(ctx context.Context, sc *svcctx.ServiceContext) ([]*dto.LinkGroupListItem, error)
	GetLinkGroup(ctx context.Context, sc *svcctx.ServiceContext, id int64) (*dto.LinkGroupResp, error)
}

// scenarioService 仿真场景库服务实现
type scenarioService struct {
	scenarioRepo       experimentrepo.ScenarioRepository
	linkGroupRepo      experimentrepo.LinkGroupRepository
	linkGroupSceneRepo experimentrepo.LinkGroupSceneRepository
	userNameQuerier    UserNameQuerier
}

// NewScenarioService 创建仿真场景库服务实例
func NewScenarioService(
	scenarioRepo experimentrepo.ScenarioRepository,
	linkGroupRepo experimentrepo.LinkGroupRepository,
	linkGroupSceneRepo experimentrepo.LinkGroupSceneRepository,
	userNameQuerier UserNameQuerier,
) ScenarioService {
	return &scenarioService{
		scenarioRepo:       scenarioRepo,
		linkGroupRepo:      linkGroupRepo,
		linkGroupSceneRepo: linkGroupSceneRepo,
		userNameQuerier:    userNameQuerier,
	}
}

// ---------------------------------------------------------------------------
// 场景 CRUD
// ---------------------------------------------------------------------------

// Create 上传自定义仿真场景
func (s *scenarioService) Create(ctx context.Context, sc *svcctx.ServiceContext, req *dto.CreateScenarioReq) (string, error) {
	scenario := &entity.SimScenario{
		ID:              snowflake.Generate(),
		Name:            req.Name,
		Code:            req.Code,
		Category:        req.Category,
		Description:     req.Description,
		SourceType:      enum.ScenarioSourceTypeCustom,
		UploadedBy:      &sc.UserID,
		SchoolID:        &sc.SchoolID,
		Status:          enum.ScenarioStatusPending,
		TimeControlMode: req.TimeControlMode,
		DefaultParams:   req.DefaultParams,
	}

	// 可选字段
	if req.ContainerImageURL != "" {
		scenario.ContainerImageURL = &req.ContainerImageURL
	}
	if req.ContainerImageSize != nil {
		if size, err := strconv.ParseInt(*req.ContainerImageSize, 10, 64); err == nil {
			scenario.ContainerImageSize = &size
		}
	}
	if req.Ecosystem != nil {
		scenario.AlgorithmType = *req.Ecosystem
	}
	if req.DefaultInitialState != nil {
		scenario.InteractionSchema = req.DefaultInitialState
	}
	if req.DataSourceModes != nil {
		scenario.DefaultSize = req.DataSourceModes
	}

	if err := s.scenarioRepo.Create(ctx, scenario); err != nil {
		return "", err
	}
	return strconv.FormatInt(scenario.ID, 10), nil
}

// GetByID 获取仿真场景详情
func (s *scenarioService) GetByID(ctx context.Context, sc *svcctx.ServiceContext, id int64) (*dto.ScenarioResp, error) {
	scenario, err := s.scenarioRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errcode.ErrScenarioNotFound
		}
		return nil, err
	}
	return s.toScenarioResp(ctx, scenario), nil
}

// Update 编辑场景信息
func (s *scenarioService) Update(ctx context.Context, _ *svcctx.ServiceContext, id int64, req *dto.UpdateScenarioReq) error {
	scenario, err := s.scenarioRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errcode.ErrScenarioNotFound
		}
		return err
	}

	// 审核中的场景不可编辑
	if scenario.Status == enum.ScenarioStatusPending {
		return errcode.ErrScenarioPendingReview
	}

	fields := make(map[string]interface{})
	if req.Name != nil {
		fields["name"] = *req.Name
	}
	if req.Description != nil {
		fields["description"] = *req.Description
	}
	if req.Category != nil {
		fields["category"] = *req.Category
	}
	if req.Ecosystem != nil {
		fields["algorithm_type"] = *req.Ecosystem
	}
	if req.TimeControlMode != nil {
		fields["time_control_mode"] = *req.TimeControlMode
	}
	if req.ContainerImageURL != nil {
		fields["container_image_url"] = *req.ContainerImageURL
	}
	if req.ContainerImageSize != nil {
		if size, err := strconv.ParseInt(*req.ContainerImageSize, 10, 64); err == nil {
			fields["container_image_size"] = size
		}
	}
	if req.DefaultParams != nil {
		fields["default_params"] = req.DefaultParams
	}
	if req.DefaultInitialState != nil {
		fields["interaction_schema"] = req.DefaultInitialState
	}
	if req.DataSourceModes != nil {
		fields["default_size"] = req.DataSourceModes
	}

	if len(fields) == 0 {
		return nil
	}
	fields["updated_at"] = time.Now()
	return s.scenarioRepo.UpdateFields(ctx, id, fields)
}

// Delete 删除/下架仿真场景
func (s *scenarioService) Delete(ctx context.Context, _ *svcctx.ServiceContext, id int64) error {
	_, err := s.scenarioRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errcode.ErrScenarioNotFound
		}
		return err
	}

	// 检查是否被模板引用
	hasRef, err := s.scenarioRepo.HasReferences(ctx, id)
	if err != nil {
		return err
	}
	if hasRef {
		return errcode.ErrScenarioHasReferences
	}

	return s.scenarioRepo.SoftDelete(ctx, id)
}

// List 仿真场景列表
func (s *scenarioService) List(ctx context.Context, _ *svcctx.ServiceContext, req *dto.ScenarioListReq) ([]*dto.ScenarioListItem, int64, error) {
	params := &experimentrepo.ScenarioListParams{
		Keyword:         req.Keyword,
		Category:        req.Category,
		SourceType:      req.SourceType,
		Status:          req.Status,
		TimeControlMode: req.TimeControlMode,
		SortBy:          req.SortBy,
		SortOrder:       req.SortOrder,
		Page:            req.Page,
		PageSize:        req.PageSize,
	}

	scenarios, total, err := s.scenarioRepo.List(ctx, params)
	if err != nil {
		return nil, 0, err
	}

	items := make([]*dto.ScenarioListItem, 0, len(scenarios))
	for _, sc := range scenarios {
		items = append(items, &dto.ScenarioListItem{
			ID:              strconv.FormatInt(sc.ID, 10),
			Name:            sc.Name,
			Code:            sc.Code,
			Category:        sc.Category,
			CategoryText:    enum.GetScenarioCategoryText(sc.Category),
			Ecosystem:       ptrIfNotEmpty(sc.AlgorithmType),
			SourceType:      sc.SourceType,
			SourceTypeText:  enum.ScenarioSourceTypeText[sc.SourceType],
			Status:          sc.Status,
			StatusText:      enum.ScenarioStatusText[sc.Status],
			TimeControlMode: sc.TimeControlMode,
			CreatedAt:       sc.CreatedAt.Format(time.RFC3339),
		})
	}
	return items, total, nil
}

// ---------------------------------------------------------------------------
// 审核
// ---------------------------------------------------------------------------

// Review 审核仿真场景
func (s *scenarioService) Review(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.ReviewScenarioReq) error {
	scenario, err := s.scenarioRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errcode.ErrScenarioNotFound
		}
		return err
	}

	if scenario.Status != enum.ScenarioStatusPending {
		return errcode.ErrScenarioPendingReview
	}

	fields := map[string]interface{}{
		"reviewed_by": sc.UserID,
		"reviewed_at": time.Now(),
		"updated_at":  time.Now(),
	}
	if req.Comment != nil {
		fields["review_comment"] = *req.Comment
	}

	switch req.Action {
	case "approve":
		fields["status"] = enum.ScenarioStatusNormal
	case "reject":
		fields["status"] = enum.ScenarioStatusRejected
	}

	return s.scenarioRepo.UpdateFields(ctx, id, fields)
}

// ---------------------------------------------------------------------------
// 联动组查询
// ---------------------------------------------------------------------------

// ListLinkGroups 联动组列表
func (s *scenarioService) ListLinkGroups(ctx context.Context, _ *svcctx.ServiceContext) ([]*dto.LinkGroupListItem, error) {
	groups, err := s.linkGroupRepo.ListAll(ctx)
	if err != nil {
		return nil, err
	}

	items := make([]*dto.LinkGroupListItem, 0, len(groups))
	for _, g := range groups {
		items = append(items, &dto.LinkGroupListItem{
			ID:          strconv.FormatInt(g.ID, 10),
			Name:        g.Name,
			Description: g.Description,
			SceneCount:  len(g.Scenes),
		})
	}
	return items, nil
}

// GetLinkGroup 联动组详情
func (s *scenarioService) GetLinkGroup(ctx context.Context, _ *svcctx.ServiceContext, id int64) (*dto.LinkGroupResp, error) {
	group, err := s.linkGroupRepo.GetByIDWithScenes(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errcode.ErrLinkGroupNotFound
		}
		return nil, err
	}

	scenes := make([]dto.LinkGroupSceneResp, 0, len(group.Scenes))
	for _, sc := range group.Scenes {
		sceneResp := dto.LinkGroupSceneResp{
			ID:         strconv.FormatInt(sc.ID, 10),
			ScenarioID: strconv.FormatInt(sc.ScenarioID, 10),
			SortOrder:  sc.SortOrder,
		}
		if sc.RoleInGroup != nil {
			sceneResp.LinkRole = *sc.RoleInGroup
		}
		// 查询场景名称和编码
		scenario, err := s.scenarioRepo.GetByID(ctx, sc.ScenarioID)
		if err == nil {
			sceneResp.SceneName = scenario.Name
			sceneResp.SceneCode = scenario.Code
		}
		scenes = append(scenes, sceneResp)
	}

	return &dto.LinkGroupResp{
		ID:          strconv.FormatInt(group.ID, 10),
		Name:        group.Name,
		Description: group.Description,
		Scenes:      scenes,
	}, nil
}

// ---------------------------------------------------------------------------
// 内部辅助
// ---------------------------------------------------------------------------

// toScenarioResp 将实体转换为场景详情响应
func (s *scenarioService) toScenarioResp(ctx context.Context, sc *entity.SimScenario) *dto.ScenarioResp {
	resp := &dto.ScenarioResp{
		ID:                  strconv.FormatInt(sc.ID, 10),
		Name:                sc.Name,
		Code:                sc.Code,
		Description:         sc.Description,
		Category:            sc.Category,
		CategoryText:        enum.GetScenarioCategoryText(sc.Category),
		Ecosystem:           ptrIfNotEmpty(sc.AlgorithmType),
		SourceType:          sc.SourceType,
		SourceTypeText:      enum.ScenarioSourceTypeText[sc.SourceType],
		Status:              sc.Status,
		StatusText:          enum.ScenarioStatusText[sc.Status],
		TimeControlMode:     sc.TimeControlMode,
		DefaultParams:       sc.DefaultParams,
		DefaultInitialState: sc.InteractionSchema,
		DataSourceModes:     sc.DefaultSize,
		GrpcPort:            0,
		CreatedAt:           sc.CreatedAt.Format(time.RFC3339),
		UpdatedAt:           sc.UpdatedAt.Format(time.RFC3339),
	}

	if sc.ContainerImageURL != nil {
		resp.ContainerImageURL = *sc.ContainerImageURL
	}
	if sc.ContainerImageSize != nil {
		sizeStr := strconv.FormatInt(*sc.ContainerImageSize, 10)
		resp.ContainerImageSize = &sizeStr
	}

	// 上传者名称
	if sc.UploadedBy != nil {
		idStr := strconv.FormatInt(*sc.UploadedBy, 10)
		resp.UploadedBy = &idStr
		name := s.userNameQuerier.GetUserName(ctx, *sc.UploadedBy)
		if name != "" {
			resp.UploaderName = &name
		}
	}

	return resp
}

// ptrIfNotEmpty 非空字符串转指针
func ptrIfNotEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
