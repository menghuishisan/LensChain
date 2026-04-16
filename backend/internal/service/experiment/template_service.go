// template_service.go
// 模块04 — 实验环境：实验模板业务逻辑
// 负责模板CRUD、发布/下架、克隆、共享、K8s配置、验证
// 对照 docs/modules/04-实验环境/03-API接口设计.md

package experiment

import (
	"context"
	"encoding/json"
	"fmt"
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

// TemplateService 实验模板服务接口
type TemplateService interface {
	Create(ctx context.Context, sc *svcctx.ServiceContext, req *dto.CreateTemplateReq) (*dto.CreateTemplateResp, error)
	GetByID(ctx context.Context, sc *svcctx.ServiceContext, id int64) (*dto.TemplateResp, error)
	Update(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.UpdateTemplateReq) error
	Delete(ctx context.Context, sc *svcctx.ServiceContext, id int64) error
	List(ctx context.Context, sc *svcctx.ServiceContext, req *dto.TemplateListReq) ([]*dto.TemplateListItem, int64, error)
	ListShared(ctx context.Context, sc *svcctx.ServiceContext, req *dto.SharedTemplateListReq) ([]*dto.TemplateListItem, int64, error)
	GetSharedByID(ctx context.Context, sc *svcctx.ServiceContext, id int64) (*dto.TemplateResp, error)
	Publish(ctx context.Context, sc *svcctx.ServiceContext, id int64) error
	Unpublish(ctx context.Context, sc *svcctx.ServiceContext, id int64) error
	Clone(ctx context.Context, sc *svcctx.ServiceContext, id int64) (*dto.CreateTemplateResp, error)
	Share(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.ShareTemplateReq) error
	GetK8sConfig(ctx context.Context, sc *svcctx.ServiceContext, id int64) (*dto.K8sConfigResp, error)
	SetK8sConfig(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.K8sConfigReq) error
	Validate(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.ValidateTemplateReq) (*dto.ValidateTemplateResp, error)
}

// templateService 实验模板服务实现
type templateService struct {
	db               *gorm.DB
	templateRepo     experimentrepo.TemplateRepository
	containerRepo    experimentrepo.ContainerRepository
	checkpointRepo   experimentrepo.CheckpointRepository
	initScriptRepo   experimentrepo.InitScriptRepository
	simSceneRepo     experimentrepo.SimSceneRepository
	imageRepo        experimentrepo.ImageRepository
	imageVersionRepo experimentrepo.ImageVersionRepository
	tagRepo          experimentrepo.TagRepository
	templateTagRepo  experimentrepo.TemplateTagRepository
	roleRepo         experimentrepo.RoleRepository
	userNameQuerier  UserNameQuerier
}

// NewTemplateService 创建实验模板服务实例
func NewTemplateService(
	db *gorm.DB,
	templateRepo experimentrepo.TemplateRepository,
	containerRepo experimentrepo.ContainerRepository,
	checkpointRepo experimentrepo.CheckpointRepository,
	initScriptRepo experimentrepo.InitScriptRepository,
	simSceneRepo experimentrepo.SimSceneRepository,
	imageRepo experimentrepo.ImageRepository,
	imageVersionRepo experimentrepo.ImageVersionRepository,
	tagRepo experimentrepo.TagRepository,
	templateTagRepo experimentrepo.TemplateTagRepository,
	roleRepo experimentrepo.RoleRepository,
	userNameQuerier UserNameQuerier,
) TemplateService {
	return &templateService{
		db: db, templateRepo: templateRepo,
		containerRepo: containerRepo, checkpointRepo: checkpointRepo,
		initScriptRepo: initScriptRepo, simSceneRepo: simSceneRepo,
		imageRepo: imageRepo, imageVersionRepo: imageVersionRepo,
		tagRepo: tagRepo, templateTagRepo: templateTagRepo,
		roleRepo: roleRepo, userNameQuerier: userNameQuerier,
	}
}

// ---------------------------------------------------------------------------
// 模板 CRUD
// ---------------------------------------------------------------------------

// Create 创建实验模板
func (s *templateService) Create(ctx context.Context, sc *svcctx.ServiceContext, req *dto.CreateTemplateReq) (*dto.CreateTemplateResp, error) {
	topologyMode := req.TopologyMode
	template := &entity.ExperimentTemplate{
		ID:            snowflake.Generate(),
		SchoolID:      sc.SchoolID,
		TeacherID:     sc.UserID,
		Title:         req.Title,
		Description:   req.Description,
		Objectives:    req.Objectives,
		Instructions:  req.Instructions,
		References:    req.References,
		ExpType:       req.ExperimentType,
		TopologyMode:  &topologyMode,
		JudgeMode:     req.JudgeMode,
		AutoWeight:    req.AutoWeight,
		ManualWeight:  req.ManualWeight,
		TotalScore:    float64(req.TotalScore),
		MaxDuration:   &req.MaxDuration,
		ScoreStrategy: req.ScoreStrategy,
		Status:        enum.TemplateStatusDraft,
	}
	if req.IdleTimeout != nil {
		template.IdleTimeout = *req.IdleTimeout
	}
	if req.CPULimit != nil {
		template.CPULimit = req.CPULimit
	}
	if req.MemoryLimit != nil {
		template.MemoryLimit = req.MemoryLimit
	}
	if req.DiskLimit != nil {
		template.DiskLimit = req.DiskLimit
	}

	if err := s.templateRepo.Create(ctx, template); err != nil {
		return nil, err
	}

	return &dto.CreateTemplateResp{
		ID:                 strconv.FormatInt(template.ID, 10),
		Title:              template.Title,
		ExperimentType:     template.ExpType,
		ExperimentTypeText: enum.GetExperimentTypeText(template.ExpType),
		Status:             template.Status,
		StatusText:         enum.GetTemplateStatusText(template.Status),
		TopologyMode:       topologyMode,
		TopologyModeText:   enum.GetTopologyModeText(topologyMode),
		JudgeMode:          template.JudgeMode,
		JudgeModeText:      enum.GetJudgeModeText(template.JudgeMode),
	}, nil
}

// GetByID 获取实验模板详情
func (s *templateService) GetByID(ctx context.Context, sc *svcctx.ServiceContext, id int64) (*dto.TemplateResp, error) {
	template, err := s.templateRepo.GetByIDWithAll(ctx, id)
	if err != nil {
		return nil, errcode.ErrTemplateNotFound
	}
	return s.toTemplateResp(ctx, template), nil
}

// Update 更新实验模板
func (s *templateService) Update(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.UpdateTemplateReq) error {
	template, err := s.templateRepo.GetByID(ctx, id)
	if err != nil {
		return errcode.ErrTemplateNotFound
	}
	if template.Status != enum.TemplateStatusDraft {
		return errcode.ErrTemplateNotDraft
	}

	fields := make(map[string]interface{})
	if req.Title != nil {
		fields["title"] = *req.Title
	}
	if req.Description != nil {
		fields["description"] = *req.Description
	}
	if req.Objectives != nil {
		fields["objectives"] = *req.Objectives
	}
	if req.Instructions != nil {
		fields["instructions"] = *req.Instructions
	}
	if req.References != nil {
		fields["references"] = *req.References
	}
	if req.ExperimentType != nil {
		fields["experiment_type"] = *req.ExperimentType
	}
	if req.TopologyMode != nil {
		fields["topology_mode"] = *req.TopologyMode
	}
	if req.JudgeMode != nil {
		fields["judge_mode"] = *req.JudgeMode
	}
	if req.AutoWeight != nil {
		fields["auto_weight"] = *req.AutoWeight
	}
	if req.ManualWeight != nil {
		fields["manual_weight"] = *req.ManualWeight
	}
	if req.TotalScore != nil {
		fields["total_score"] = float64(*req.TotalScore)
	}
	if req.MaxDuration != nil {
		fields["max_duration"] = *req.MaxDuration
	}
	if req.IdleTimeout != nil {
		fields["idle_timeout"] = *req.IdleTimeout
	}
	if req.CPULimit != nil {
		fields["cpu_limit"] = *req.CPULimit
	}
	if req.MemoryLimit != nil {
		fields["memory_limit"] = *req.MemoryLimit
	}
	if req.DiskLimit != nil {
		fields["disk_limit"] = *req.DiskLimit
	}
	if req.ScoreStrategy != nil {
		fields["score_strategy"] = *req.ScoreStrategy
	}
	if len(fields) == 0 {
		return nil
	}
	fields["updated_at"] = time.Now()
	return s.templateRepo.UpdateFields(ctx, id, fields)
}

// Delete 删除实验模板
func (s *templateService) Delete(ctx context.Context, sc *svcctx.ServiceContext, id int64) error {
	_, err := s.templateRepo.GetByID(ctx, id)
	if err != nil {
		return errcode.ErrTemplateNotFound
	}
	hasInstances, err := s.templateRepo.HasInstances(ctx, id)
	if err != nil {
		return err
	}
	if hasInstances {
		return errcode.ErrTemplateHasInstances
	}
	return s.templateRepo.SoftDelete(ctx, id)
}

// List 实验模板列表
func (s *templateService) List(ctx context.Context, sc *svcctx.ServiceContext, req *dto.TemplateListReq) ([]*dto.TemplateListItem, int64, error) {
	tagID, _ := snowflake.ParseString(req.TagID)
	params := &experimentrepo.TemplateListParams{
		SchoolID:       sc.SchoolID,
		TeacherID:      sc.UserID,
		Keyword:        req.Keyword,
		Status:         req.Status,
		ExperimentType: req.ExperimentType,
		TagID:          tagID,
		SortBy:         req.SortBy,
		SortOrder:      req.SortOrder,
		Page:           req.Page,
		PageSize:       req.PageSize,
	}
	// 超管可查看所有模板
	if sc.IsSuperAdmin() {
		params.TeacherID = 0
		params.SchoolID = 0
	}

	templates, total, err := s.templateRepo.List(ctx, params)
	if err != nil {
		return nil, 0, err
	}
	items := make([]*dto.TemplateListItem, 0, len(templates))
	for _, t := range templates {
		items = append(items, s.toTemplateListItem(ctx, t))
	}
	return items, total, nil
}

// ---------------------------------------------------------------------------
// 发布 / 下架
// ---------------------------------------------------------------------------

// Publish 发布实验模板
func (s *templateService) Publish(ctx context.Context, sc *svcctx.ServiceContext, id int64) error {
	template, err := s.templateRepo.GetByIDWithAll(ctx, id)
	if err != nil {
		return errcode.ErrTemplateNotFound
	}
	if template.Status == enum.TemplateStatusPublished {
		return errcode.ErrTemplateAlreadyPublished
	}
	if template.Status != enum.TemplateStatusDraft {
		return errcode.ErrTemplateNotDraft
	}
	if len(template.Containers) == 0 && len(template.SimScenes) == 0 {
		return errcode.ErrInvalidParams.WithMessage("请至少配置一个容器或一个仿真场景")
	}

	validation, err := s.Validate(ctx, sc, id, &dto.ValidateTemplateReq{})
	if err != nil {
		return err
	}
	if !validation.IsPublishable {
		return errcode.ErrInvalidParams.WithMessage("模板配置验证未通过，存在阻断发布的 L1/L2 问题")
	}

	fields := map[string]interface{}{
		"status":     enum.TemplateStatusPublished,
		"updated_at": time.Now(),
	}
	if generated := buildTemplateK8sConfig(template); len(generated) > 0 {
		fields["k8s_config"] = generated
	}
	return s.templateRepo.UpdateFields(ctx, id, fields)
}

// Unpublish 下架实验模板
func (s *templateService) Unpublish(ctx context.Context, sc *svcctx.ServiceContext, id int64) error {
	template, err := s.templateRepo.GetByID(ctx, id)
	if err != nil {
		return errcode.ErrTemplateNotFound
	}
	if template.Status != enum.TemplateStatusPublished {
		return errcode.ErrTemplateNotPublished
	}
	return s.templateRepo.UpdateFields(ctx, id, map[string]interface{}{
		"status":     enum.TemplateStatusOffShelf,
		"updated_at": time.Now(),
	})
}

// ---------------------------------------------------------------------------
// 克隆
// ---------------------------------------------------------------------------

// Clone 克隆实验模板（含所有子资源，事务操作）
func (s *templateService) Clone(ctx context.Context, sc *svcctx.ServiceContext, id int64) (*dto.CreateTemplateResp, error) {
	source, err := s.templateRepo.GetByIDWithAll(ctx, id)
	if err != nil {
		return nil, errcode.ErrTemplateNotFound
	}

	var newTemplate *entity.ExperimentTemplate
	txErr := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 克隆主表
		newID := snowflake.Generate()
		clonedFromID := source.ID
		newTemplate = &entity.ExperimentTemplate{
			ID:            newID,
			SchoolID:      sc.SchoolID,
			TeacherID:     sc.UserID,
			Title:         source.Title + "（副本）",
			Description:   source.Description,
			Objectives:    source.Objectives,
			Instructions:  source.Instructions,
			References:    source.References,
			ExpType:       source.ExpType,
			TopologyMode:  source.TopologyMode,
			JudgeMode:     source.JudgeMode,
			AutoWeight:    source.AutoWeight,
			ManualWeight:  source.ManualWeight,
			TotalScore:    source.TotalScore,
			MaxDuration:   source.MaxDuration,
			IdleTimeout:   source.IdleTimeout,
			CPULimit:      source.CPULimit,
			MemoryLimit:   source.MemoryLimit,
			DiskLimit:     source.DiskLimit,
			ScoreStrategy: source.ScoreStrategy,
			ClonedFromID:  &clonedFromID,
			Status:        enum.TemplateStatusDraft,
			SimLayout:     source.SimLayout,
			K8sConfig:     source.K8sConfig,
			NetworkConfig: source.NetworkConfig,
		}
		if err := tx.Create(newTemplate).Error; err != nil {
			return err
		}

		// 克隆容器配置
		if len(source.Containers) > 0 {
			containers := make([]*entity.TemplateContainer, 0, len(source.Containers))
			for _, c := range source.Containers {
				containers = append(containers, &entity.TemplateContainer{
					ID:             snowflake.Generate(),
					TemplateID:     newID,
					ImageVersionID: c.ImageVersionID,
					ContainerName:  c.ContainerName,
					RoleID:         c.RoleID,
					EnvVars:        c.EnvVars,
					Ports:          c.Ports,
					Volumes:        c.Volumes,
					CPULimit:       c.CPULimit,
					MemoryLimit:    c.MemoryLimit,
					DependsOn:      c.DependsOn,
					StartupOrder:   c.StartupOrder,
					IsPrimary:      c.IsPrimary,
					SortOrder:      c.SortOrder,
				})
			}
			if err := tx.CreateInBatches(containers, 50).Error; err != nil {
				return err
			}
		}

		// 克隆检查点
		if len(source.Checkpoints) > 0 {
			checkpoints := make([]*entity.TemplateCheckpoint, 0, len(source.Checkpoints))
			for _, cp := range source.Checkpoints {
				checkpoints = append(checkpoints, &entity.TemplateCheckpoint{
					ID:              snowflake.Generate(),
					TemplateID:      newID,
					Title:           cp.Title,
					Description:     cp.Description,
					CheckType:       cp.CheckType,
					ScriptContent:   cp.ScriptContent,
					ScriptLanguage:  cp.ScriptLanguage,
					TargetContainer: cp.TargetContainer,
					AssertionConfig: cp.AssertionConfig,
					Score:           cp.Score,
					Scope:           cp.Scope,
					SortOrder:       cp.SortOrder,
				})
			}
			if err := tx.CreateInBatches(checkpoints, 50).Error; err != nil {
				return err
			}
		}

		// 克隆初始化脚本
		if len(source.InitScripts) > 0 {
			scripts := make([]*entity.TemplateInitScript, 0, len(source.InitScripts))
			for _, is := range source.InitScripts {
				scripts = append(scripts, &entity.TemplateInitScript{
					ID:              snowflake.Generate(),
					TemplateID:      newID,
					TargetContainer: is.TargetContainer,
					ScriptContent:   is.ScriptContent,
					ScriptLanguage:  is.ScriptLanguage,
					ExecutionOrder:  is.ExecutionOrder,
					Timeout:         is.Timeout,
				})
			}
			if err := tx.CreateInBatches(scripts, 50).Error; err != nil {
				return err
			}
		}

		// 克隆仿真场景配置
		if len(source.SimScenes) > 0 {
			scenes := make([]*entity.TemplateSimScene, 0, len(source.SimScenes))
			for _, ss := range source.SimScenes {
				scenes = append(scenes, &entity.TemplateSimScene{
					ID:               snowflake.Generate(),
					TemplateID:       newID,
					ScenarioID:       ss.ScenarioID,
					LinkGroupID:      ss.LinkGroupID,
					Config:           ss.Config,
					LayoutPosition:   ss.LayoutPosition,
					DataSourceConfig: ss.DataSourceConfig,
					SortOrder:        ss.SortOrder,
				})
			}
			if err := tx.CreateInBatches(scenes, 50).Error; err != nil {
				return err
			}
		}

		// 克隆标签关联
		if len(source.Tags) > 0 {
			tags := make([]*entity.TemplateTag, 0, len(source.Tags))
			for _, tt := range source.Tags {
				tags = append(tags, &entity.TemplateTag{
					ID:         snowflake.Generate(),
					TemplateID: newID,
					TagID:      tt.TagID,
				})
			}
			if err := tx.CreateInBatches(tags, 50).Error; err != nil {
				return err
			}
		}

		// 克隆角色
		if len(source.Roles) > 0 {
			roles := make([]*entity.TemplateRole, 0, len(source.Roles))
			for _, r := range source.Roles {
				roles = append(roles, &entity.TemplateRole{
					ID:          snowflake.Generate(),
					TemplateID:  newID,
					RoleName:    r.RoleName,
					Description: r.Description,
					MaxMembers:  r.MaxMembers,
					SortOrder:   r.SortOrder,
				})
			}
			if err := tx.CreateInBatches(roles, 50).Error; err != nil {
				return err
			}
		}

		return nil
	})
	if txErr != nil {
		return nil, txErr
	}

	topologyMode := 0
	if newTemplate.TopologyMode != nil {
		topologyMode = *newTemplate.TopologyMode
	}
	return &dto.CreateTemplateResp{
		ID:                 strconv.FormatInt(newTemplate.ID, 10),
		Title:              newTemplate.Title,
		ExperimentType:     newTemplate.ExpType,
		ExperimentTypeText: enum.GetExperimentTypeText(newTemplate.ExpType),
		Status:             newTemplate.Status,
		StatusText:         enum.GetTemplateStatusText(newTemplate.Status),
		TopologyMode:       topologyMode,
		TopologyModeText:   enum.GetTopologyModeText(topologyMode),
		JudgeMode:          newTemplate.JudgeMode,
		JudgeModeText:      enum.GetJudgeModeText(newTemplate.JudgeMode),
	}, nil
}

// ---------------------------------------------------------------------------
// 共享
// ---------------------------------------------------------------------------

// Share 设置模板共享状态
func (s *templateService) Share(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.ShareTemplateReq) error {
	template, err := s.templateRepo.GetByID(ctx, id)
	if err != nil {
		return errcode.ErrTemplateNotFound
	}
	if req.IsShared && template.Status != enum.TemplateStatusPublished {
		return errcode.ErrTemplateNotPublished
	}
	return s.templateRepo.UpdateFields(ctx, id, map[string]interface{}{
		"is_shared":  req.IsShared,
		"updated_at": time.Now(),
	})
}

// ---------------------------------------------------------------------------
// K8s 配置
// ---------------------------------------------------------------------------

// GetK8sConfig 获取模板K8s编排配置
func (s *templateService) GetK8sConfig(ctx context.Context, sc *svcctx.ServiceContext, id int64) (*dto.K8sConfigResp, error) {
	template, err := s.templateRepo.GetByID(ctx, id)
	if err != nil {
		return nil, errcode.ErrTemplateNotFound
	}
	return &dto.K8sConfigResp{
		TemplateID: strconv.FormatInt(template.ID, 10),
		K8sConfig:  template.K8sConfig,
	}, nil
}

// SetK8sConfig 设置模板K8s编排配置
func (s *templateService) SetK8sConfig(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.K8sConfigReq) error {
	template, err := s.templateRepo.GetByID(ctx, id)
	if err != nil {
		return errcode.ErrTemplateNotFound
	}
	if template.Status != enum.TemplateStatusDraft {
		return errcode.ErrTemplateNotDraft
	}
	return s.templateRepo.UpdateFields(ctx, id, map[string]interface{}{
		"k8s_config": req.K8sConfig,
		"updated_at": time.Now(),
	})
}

// ---------------------------------------------------------------------------
// 验证（5级验证）
// ---------------------------------------------------------------------------

// Validate 模板配置验证
func (s *templateService) Validate(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.ValidateTemplateReq) (*dto.ValidateTemplateResp, error) {
	template, err := s.templateRepo.GetByIDWithAll(ctx, id)
	if err != nil {
		return nil, errcode.ErrTemplateNotFound
	}

	levels := req.Levels
	if len(levels) == 0 {
		levels = []int{1, 2, 3, 4, 5}
	}

	resp := &dto.ValidateTemplateResp{
		TemplateID: strconv.FormatInt(template.ID, 10),
		Results:    make([]dto.ValidationLevelResult, 0, len(levels)),
	}

	for _, level := range levels {
		result := s.validateLevel(ctx, template, level)
		resp.Results = append(resp.Results, result)
	}
	resp.Summary = summarizeValidationIssues(resp.Results)
	resp.IsPublishable = determinePublishable(resp.Results)

	return resp, nil
}

// validateLevel 执行单层验证
func (s *templateService) validateLevel(ctx context.Context, template *entity.ExperimentTemplate, level int) dto.ValidationLevelResult {
	result := dto.ValidationLevelResult{
		Level:  level,
		Passed: true,
		Issues: make([]dto.ValidationIssue, 0),
	}

	switch level {
	case 1: // 依赖完整性
		result.LevelName = "依赖完整性检查"
		result.Severity = "error"
		s.validateDependencyIntegrity(ctx, template, &result)
	case 2: // 端口冲突
		result.LevelName = "端口冲突检查"
		result.Severity = "error"
		s.validatePortConflicts(template, &result)
	case 3: // 资源合理性
		result.LevelName = "资源合理性检查"
		result.Severity = "warning"
		s.validateResourceReasonableness(template, &result)
	case 4: // 生态一致性
		result.LevelName = "生态一致性检查"
		result.Severity = "warning"
		s.validateEcosystemConsistency(template, &result)
	case 5: // 连通性预检
		result.LevelName = "连通性预检"
		result.Severity = "hint"
		s.validateConnectivityPrecheck(template, &result)
	}

	return result
}

// validateDependencyIntegrity 验证依赖完整性
func (s *templateService) validateDependencyIntegrity(ctx context.Context, template *entity.ExperimentTemplate, result *dto.ValidationLevelResult) {
	if len(template.Containers) == 0 && len(template.SimScenes) == 0 {
		result.Passed = false
		result.Issues = append(result.Issues, dto.ValidationIssue{
			Code:    "L1_EMPTY_TEMPLATE",
			Message: "请至少配置一个容器或一个仿真场景",
		})
	}

	// 检查是否有容器配置
	if len(template.Containers) == 0 && (template.ExpType == enum.ExperimentTypeReal || template.ExpType == enum.ExperimentTypeMixed) {
		result.Passed = false
		result.Issues = append(result.Issues, dto.ValidationIssue{
			Code:    "L1_MISSING_CONTAINER",
			Message: "真实环境/混合实验必须配置至少一个容器",
		})
	}

	// 检查仿真场景配置
	if len(template.SimScenes) == 0 && (template.ExpType == enum.ExperimentTypeSimulation || template.ExpType == enum.ExperimentTypeMixed) {
		result.Passed = false
		result.Issues = append(result.Issues, dto.ValidationIssue{
			Code:    "L1_MISSING_SIM_SCENE",
			Message: "纯仿真/混合实验必须配置至少一个仿真场景",
		})
	}

	// 检查检查点总分是否匹配
	if len(template.Checkpoints) > 0 {
		var totalScore float64
		for _, cp := range template.Checkpoints {
			totalScore += cp.Score
		}
		if totalScore != template.TotalScore {
			result.Issues = append(result.Issues, dto.ValidationIssue{
				Code:    "L3_SCORE_MISMATCH",
				Message: fmt.Sprintf("检查点总分(%.2f)与模板总分(%.2f)不一致", totalScore, template.TotalScore),
			})
		}
	}

	// 检查容器依赖引用
	containerNames := make(map[string]bool)
	for _, c := range template.Containers {
		containerNames[c.ContainerName] = true
	}
	for _, c := range template.Containers {
		if len(c.DependsOn) > 0 {
			var deps []string
			if err := json.Unmarshal(c.DependsOn, &deps); err == nil {
				for _, dep := range deps {
					if !containerNames[dep] {
						result.Passed = false
						result.Issues = append(result.Issues, dto.ValidationIssue{
							Code:    "L1_DEPENDS_ON_NOT_FOUND",
							Message: fmt.Sprintf("容器 %s 依赖的容器 %s 不存在", c.ContainerName, dep),
						})
					}
				}
			}
		}
	}

	for _, c := range template.Containers {
		version, err := s.imageVersionRepo.GetByID(ctx, c.ImageVersionID)
		if err != nil {
			result.Passed = false
			result.Issues = append(result.Issues, dto.ValidationIssue{
				Code:    "L1_IMAGE_VERSION_NOT_FOUND",
				Message: fmt.Sprintf("容器 %s 关联的镜像版本不存在", c.ContainerName),
			})
			continue
		}

		image, err := s.imageRepo.GetByID(ctx, version.ImageID)
		if err != nil {
			result.Passed = false
			result.Issues = append(result.Issues, dto.ValidationIssue{
				Code:    "L1_IMAGE_NOT_FOUND",
				Message: fmt.Sprintf("容器 %s 关联的镜像不存在", c.ContainerName),
			})
			continue
		}

		for _, dependency := range parseRequiredDependencies(image.RequiredDependencies) {
			if containerNames[dependency] {
				continue
			}
			result.Passed = false
			result.Issues = append(result.Issues, dto.ValidationIssue{
				Code:    "L1_MISSING_DEPENDENCY",
				Message: fmt.Sprintf("依赖完整性检查未通过 — %s 需要 %s 作为依赖", c.ContainerName, dependency),
			})
		}
	}
}

// validatePortConflicts 验证端口冲突
func (s *templateService) validatePortConflicts(template *entity.ExperimentTemplate, result *dto.ValidationLevelResult) {
	type portEntry struct {
		Port      int
		Container string
	}
	usedPorts := make(map[int]string)

	for _, c := range template.Containers {
		if len(c.Ports) == 0 {
			continue
		}
		var ports []struct {
			ContainerPort int `json:"container_port"`
			ServicePort   int `json:"service_port"`
		}
		if err := json.Unmarshal(c.Ports, &ports); err != nil {
			continue
		}
		for _, p := range ports {
			if existing, ok := usedPorts[p.ServicePort]; ok && p.ServicePort > 0 {
				result.Passed = false
				result.Issues = append(result.Issues, dto.ValidationIssue{
					Code:    "L2_PORT_CONFLICT",
					Message: fmt.Sprintf("服务端口 %d 在容器 %s 和 %s 之间冲突", p.ServicePort, existing, c.ContainerName),
				})
			}
			if p.ServicePort > 0 {
				usedPorts[p.ServicePort] = c.ContainerName
			}
		}
	}
}

// validateResourceReasonableness 验证资源合理性
func (s *templateService) validateResourceReasonableness(template *entity.ExperimentTemplate, result *dto.ValidationLevelResult) {
	containerCount := len(template.Containers)
	if containerCount > 10 {
		result.Issues = append(result.Issues, dto.ValidationIssue{
			Code:    "L3_CONTAINER_COUNT_HIGH",
			Message: fmt.Sprintf("容器数量(%d)较多，可能影响启动速度和资源消耗", containerCount),
		})
	}

	// 检查是否有主容器
	hasPrimary := false
	for _, c := range template.Containers {
		if c.IsPrimary {
			hasPrimary = true
			break
		}
	}
	if containerCount > 0 && !hasPrimary {
		result.Issues = append(result.Issues, dto.ValidationIssue{
			Code:    "L3_PRIMARY_CONTAINER_MISSING",
			Message: "未设置主容器，建议指定一个主容器作为学生操作入口",
		})
	}
}

// validateEcosystemConsistency 验证模板标签与生态配置的基础一致性。
func (s *templateService) validateEcosystemConsistency(template *entity.ExperimentTemplate, result *dto.ValidationLevelResult) {
	if len(template.Tags) == 0 {
		result.Issues = append(result.Issues, dto.ValidationIssue{
			Code:    "L4_TAG_RECOMMENDED",
			Message: "建议为模板添加标签，便于分类和搜索",
		})
	}
}

// validateConnectivityPrecheck 连通性预检
func (s *templateService) validateConnectivityPrecheck(template *entity.ExperimentTemplate, result *dto.ValidationLevelResult) {
	// 检查初始化脚本引用的容器是否存在
	containerNames := make(map[string]bool)
	for _, c := range template.Containers {
		containerNames[c.ContainerName] = true
	}
	for _, is := range template.InitScripts {
		if !containerNames[is.TargetContainer] {
			result.Passed = false
			result.Issues = append(result.Issues, dto.ValidationIssue{
				Code:    "L5_INIT_SCRIPT_TARGET_MISSING",
				Message: fmt.Sprintf("初始化脚本目标容器 %s 不存在", is.TargetContainer),
			})
		}
	}

	// 检查检查点引用的容器是否存在
	for _, cp := range template.Checkpoints {
		if cp.TargetContainer != nil && *cp.TargetContainer != "" {
			if !containerNames[*cp.TargetContainer] {
				result.Passed = false
				result.Issues = append(result.Issues, dto.ValidationIssue{
					Code:    "L5_CHECKPOINT_TARGET_MISSING",
					Message: fmt.Sprintf("检查点 %s 目标容器 %s 不存在", cp.Title, *cp.TargetContainer),
				})
			}
		}
	}
}

// ---------------------------------------------------------------------------
// 转换方法
// ---------------------------------------------------------------------------

// toTemplateResp 转换模板为详情响应
func (s *templateService) toTemplateResp(ctx context.Context, t *entity.ExperimentTemplate) *dto.TemplateResp {
	topologyMode := 0
	if t.TopologyMode != nil {
		topologyMode = *t.TopologyMode
	}
	maxDuration := 0
	if t.MaxDuration != nil {
		maxDuration = *t.MaxDuration
	}

	resp := &dto.TemplateResp{
		ID:                 strconv.FormatInt(t.ID, 10),
		Title:              t.Title,
		Description:        t.Description,
		Objectives:         t.Objectives,
		Instructions:       t.Instructions,
		References:         t.References,
		ExperimentType:     t.ExpType,
		ExperimentTypeText: enum.GetExperimentTypeText(t.ExpType),
		TopologyMode:       topologyMode,
		TopologyModeText:   enum.GetTopologyModeText(topologyMode),
		JudgeMode:          t.JudgeMode,
		JudgeModeText:      enum.GetJudgeModeText(t.JudgeMode),
		AutoWeight:         t.AutoWeight,
		ManualWeight:       t.ManualWeight,
		TotalScore:         int(t.TotalScore),
		MaxDuration:        maxDuration,
		CPULimit:           t.CPULimit,
		MemoryLimit:        t.MemoryLimit,
		DiskLimit:          t.DiskLimit,
		ScoreStrategy:      t.ScoreStrategy,
		IsShared:           t.IsShared,
		Status:             t.Status,
		StatusText:         enum.GetTemplateStatusText(t.Status),
		K8sConfig:          t.K8sConfig,
		CreatedAt:          t.CreatedAt.Format(time.RFC3339),
		UpdatedAt:          t.UpdatedAt.Format(time.RFC3339),
	}

	if t.IdleTimeout > 0 {
		idleTimeout := t.IdleTimeout
		resp.IdleTimeout = &idleTimeout
	}

	// 教师信息
	teacherName := s.userNameQuerier.GetUserName(ctx, t.TeacherID)
	resp.Teacher = &dto.SimpleUserResp{
		ID:   strconv.FormatInt(t.TeacherID, 10),
		Name: teacherName,
	}

	// 容器列表
	resp.Containers = make([]dto.ContainerResp, 0, len(t.Containers))
	for _, c := range t.Containers {
		resp.Containers = append(resp.Containers, s.toContainerResp(&c))
	}

	// 检查点列表
	resp.Checkpoints = make([]dto.CheckpointResp, 0, len(t.Checkpoints))
	for _, cp := range t.Checkpoints {
		resp.Checkpoints = append(resp.Checkpoints, s.toCheckpointResp(&cp))
	}

	// 初始化脚本列表
	resp.InitScripts = make([]dto.InitScriptResp, 0, len(t.InitScripts))
	for _, is := range t.InitScripts {
		resp.InitScripts = append(resp.InitScripts, s.toInitScriptResp(&is))
	}

	// 仿真场景列表
	resp.SimScenes = make([]dto.TemplateSimSceneResp, 0, len(t.SimScenes))
	for _, ss := range t.SimScenes {
		resp.SimScenes = append(resp.SimScenes, s.toSimSceneResp(&ss))
	}

	// 标签列表
	resp.Tags = s.buildTemplateTagResponses(ctx, t.Tags)

	// 角色列表
	resp.Roles = make([]dto.RoleResp, 0, len(t.Roles))
	for _, r := range t.Roles {
		resp.Roles = append(resp.Roles, s.toRoleResp(&r))
	}

	return resp
}

// toTemplateListItem 转换模板为列表项
func (s *templateService) toTemplateListItem(ctx context.Context, t *entity.ExperimentTemplate) *dto.TemplateListItem {
	topologyMode := 0
	if t.TopologyMode != nil {
		topologyMode = *t.TopologyMode
	}
	maxDuration := 0
	if t.MaxDuration != nil {
		maxDuration = *t.MaxDuration
	}

	item := &dto.TemplateListItem{
		ID:                 strconv.FormatInt(t.ID, 10),
		Title:              t.Title,
		ExperimentType:     t.ExpType,
		ExperimentTypeText: enum.GetExperimentTypeText(t.ExpType),
		TopologyMode:       topologyMode,
		TopologyModeText:   enum.GetTopologyModeText(topologyMode),
		JudgeMode:          t.JudgeMode,
		JudgeModeText:      enum.GetJudgeModeText(t.JudgeMode),
		TotalScore:         int(t.TotalScore),
		MaxDuration:        maxDuration,
		IsShared:           t.IsShared,
		Status:             t.Status,
		StatusText:         enum.GetTemplateStatusText(t.Status),
		ContainerCount:     len(t.Containers),
		CheckpointCount:    len(t.Checkpoints),
		CreatedAt:          t.CreatedAt.Format(time.RFC3339),
		UpdatedAt:          t.UpdatedAt.Format(time.RFC3339),
	}

	item.Tags = s.buildTemplateTagResponses(ctx, t.Tags)

	return item
}

// buildTemplateTagResponses 补齐模板标签的名称和分类信息。
func (s *templateService) buildTemplateTagResponses(ctx context.Context, templateTags []entity.TemplateTag) []dto.TagResp {
	resp := make([]dto.TagResp, 0, len(templateTags))
	for _, tt := range templateTags {
		item := dto.TagResp{ID: strconv.FormatInt(tt.TagID, 10)}
		tag, err := s.tagRepo.GetByID(ctx, tt.TagID)
		if err == nil && tag != nil {
			item.Name = tag.Name
			item.Category = tag.Category
		}
		resp = append(resp, item)
	}
	return resp
}

// toContainerResp 转换容器为响应
func (s *templateService) toContainerResp(c *entity.TemplateContainer) dto.ContainerResp {
	resp := dto.ContainerResp{
		ID:             strconv.FormatInt(c.ID, 10),
		TemplateID:     strconv.FormatInt(c.TemplateID, 10),
		ImageVersionID: strconv.FormatInt(c.ImageVersionID, 10),
		ContainerName:  c.ContainerName,
		EnvVars:        c.EnvVars,
		Ports:          c.Ports,
		Volumes:        c.Volumes,
		CPULimit:       c.CPULimit,
		MemoryLimit:    c.MemoryLimit,
		DependsOn:      c.DependsOn,
		StartupOrder:   c.StartupOrder,
		IsPrimary:      c.IsPrimary,
	}
	if c.RoleID != nil {
		roleID := strconv.FormatInt(*c.RoleID, 10)
		resp.RoleID = &roleID
	}
	return resp
}

// toCheckpointResp 转换检查点为响应
func (s *templateService) toCheckpointResp(cp *entity.TemplateCheckpoint) dto.CheckpointResp {
	return dto.CheckpointResp{
		ID:              strconv.FormatInt(cp.ID, 10),
		TemplateID:      strconv.FormatInt(cp.TemplateID, 10),
		Title:           cp.Title,
		Description:     cp.Description,
		CheckType:       cp.CheckType,
		CheckTypeText:   enum.GetCheckTypeText(cp.CheckType),
		ScriptContent:   cp.ScriptContent,
		ScriptLanguage:  cp.ScriptLanguage,
		TargetContainer: cp.TargetContainer,
		AssertionConfig: cp.AssertionConfig,
		Score:           cp.Score,
		Scope:           cp.Scope,
		ScopeText:       enum.GetCheckpointScopeText(cp.Scope),
		SortOrder:       cp.SortOrder,
	}
}

// toInitScriptResp 转换初始化脚本为响应
func (s *templateService) toInitScriptResp(is *entity.TemplateInitScript) dto.InitScriptResp {
	return dto.InitScriptResp{
		ID:              strconv.FormatInt(is.ID, 10),
		TemplateID:      strconv.FormatInt(is.TemplateID, 10),
		TargetContainer: is.TargetContainer,
		ScriptContent:   is.ScriptContent,
		ScriptLanguage:  is.ScriptLanguage,
		ExecutionOrder:  is.ExecutionOrder,
		Timeout:         is.Timeout,
	}
}

// toSimSceneResp 转换仿真场景配置为响应
func (s *templateService) toSimSceneResp(ss *entity.TemplateSimScene) dto.TemplateSimSceneResp {
	resp := dto.TemplateSimSceneResp{
		ID:               strconv.FormatInt(ss.ID, 10),
		TemplateID:       strconv.FormatInt(ss.TemplateID, 10),
		SceneParams:      ss.Config,
		DataSourceConfig: ss.DataSourceConfig,
		LayoutPosition:   ss.LayoutPosition,
	}
	if ss.LinkGroupID != nil {
		lgID := strconv.FormatInt(*ss.LinkGroupID, 10)
		resp.LinkGroupID = &lgID
	}
	return resp
}

// toRoleResp 转换角色为响应
func (s *templateService) toRoleResp(r *entity.TemplateRole) dto.RoleResp {
	return dto.RoleResp{
		ID:          strconv.FormatInt(r.ID, 10),
		TemplateID:  strconv.FormatInt(r.TemplateID, 10),
		RoleName:    r.RoleName,
		Description: r.Description,
		MaxMembers:  r.MaxMembers,
	}
}
