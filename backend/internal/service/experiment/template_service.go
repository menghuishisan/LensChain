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
	scenarioRepo     experimentrepo.ScenarioRepository
	linkGroupRepo    experimentrepo.LinkGroupRepository
	tagRepo          experimentrepo.TagRepository
	templateTagRepo  experimentrepo.TemplateTagRepository
	roleRepo         experimentrepo.RoleRepository
	userNameQuerier  UserNameQuerier
	eventDispatcher  NotificationEventDispatcher
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
	scenarioRepo experimentrepo.ScenarioRepository,
	linkGroupRepo experimentrepo.LinkGroupRepository,
	tagRepo experimentrepo.TagRepository,
	templateTagRepo experimentrepo.TemplateTagRepository,
	roleRepo experimentrepo.RoleRepository,
	userNameQuerier UserNameQuerier,
	eventDispatcher NotificationEventDispatcher,
) TemplateService {
	return &templateService{
		db: db, templateRepo: templateRepo,
		containerRepo: containerRepo, checkpointRepo: checkpointRepo,
		initScriptRepo: initScriptRepo, simSceneRepo: simSceneRepo,
		imageRepo: imageRepo, imageVersionRepo: imageVersionRepo,
		scenarioRepo: scenarioRepo, linkGroupRepo: linkGroupRepo,
		tagRepo: tagRepo, templateTagRepo: templateTagRepo,
		roleRepo: roleRepo, userNameQuerier: userNameQuerier,
		eventDispatcher: eventDispatcher,
	}
}

// ---------------------------------------------------------------------------
// 模板 CRUD
// ---------------------------------------------------------------------------

// Create 创建实验模板
func (s *templateService) Create(ctx context.Context, sc *svcctx.ServiceContext, req *dto.CreateTemplateReq) (*dto.CreateTemplateResp, error) {
	topologyMode := req.TopologyMode
	template := &entity.ExperimentTemplate{
		ID:                 snowflake.Generate(),
		SchoolID:           sc.SchoolID,
		TeacherID:          sc.UserID,
		Title:              req.Title,
		Description:        req.Description,
		Objectives:         req.Objectives,
		Instructions:       req.Instructions,
		ReferenceMaterials: req.ReferenceMaterials,
		ExperimentType:     req.ExperimentType,
		TopologyMode:       &topologyMode,
		JudgeMode:          req.JudgeMode,
		AutoWeight:         req.AutoWeight,
		ManualWeight:       req.ManualWeight,
		TotalScore:         float64(req.TotalScore),
		MaxDuration:        &req.MaxDuration,
		ScoreStrategy:      req.ScoreStrategy,
		Status:             enum.TemplateStatusDraft,
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
		ExperimentType:     template.ExperimentType,
		ExperimentTypeText: enum.GetExperimentTypeText(template.ExperimentType),
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
	if _, err := ensureTemplateReadAccess(ctx, s.templateRepo, sc, id); err != nil {
		return nil, err
	}
	template, err := loadTemplateAggregate(
		ctx,
		s.templateRepo,
		s.containerRepo,
		s.checkpointRepo,
		s.initScriptRepo,
		s.simSceneRepo,
		s.templateTagRepo,
		s.tagRepo,
		s.roleRepo,
		id,
	)
	if err != nil {
		return nil, errcode.ErrTemplateNotFound
	}
	return s.toTemplateResp(ctx, template), nil
}

// Update 更新实验模板
func (s *templateService) Update(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.UpdateTemplateReq) error {
	template, err := ensureTemplateOwnerAccess(ctx, s.templateRepo, sc, id)
	if err != nil {
		return err
	}
	allowStructureEdit, err := s.canEditTemplateStructure(ctx, template)
	if err != nil {
		return err
	}

	fields := make(map[string]interface{})
	basicOnly := !allowStructureEdit
	targetExperimentType := template.ExperimentType
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
	if req.ReferenceMaterials != nil {
		fields["reference_materials"] = *req.ReferenceMaterials
	}
	if !basicOnly && req.ExperimentType != nil {
		targetExperimentType = *req.ExperimentType
		fields["experiment_type"] = *req.ExperimentType
	}
	if !basicOnly && req.TopologyMode != nil {
		fields["topology_mode"] = *req.TopologyMode
	}
	if !basicOnly && req.JudgeMode != nil {
		fields["judge_mode"] = *req.JudgeMode
	}
	if req.AutoWeight != nil && (!basicOnly || template.JudgeMode == enum.JudgeModeMixed) {
		fields["auto_weight"] = *req.AutoWeight
	}
	if req.ManualWeight != nil && (!basicOnly || template.JudgeMode == enum.JudgeModeMixed) {
		fields["manual_weight"] = *req.ManualWeight
	}
	if req.TotalScore != nil {
		fields["total_score"] = float64(*req.TotalScore)
	}
	if !basicOnly && req.MaxDuration != nil {
		fields["max_duration"] = *req.MaxDuration
	}
	if !basicOnly && req.IdleTimeout != nil {
		fields["idle_timeout"] = *req.IdleTimeout
	}
	if !basicOnly && req.CPULimit != nil {
		fields["cpu_limit"] = *req.CPULimit
	}
	if !basicOnly && req.MemoryLimit != nil {
		fields["memory_limit"] = *req.MemoryLimit
	}
	if !basicOnly && req.DiskLimit != nil {
		fields["disk_limit"] = *req.DiskLimit
	}
	if !basicOnly && req.ScoreStrategy != nil {
		fields["score_strategy"] = *req.ScoreStrategy
	}
	if basicOnly && hasTemplateStructureChanges(req) {
		return errcode.ErrInvalidParams.WithMessage("该模板已被课时引用，仅允许修改基本信息、实验说明和检查点")
	}
	if len(fields) == 0 {
		return nil
	}
	fields["updated_at"] = time.Now()
	if !basicOnly && targetExperimentType != template.ExperimentType {
		return s.updateTemplateWithTypeSwitchCleanup(ctx, id, fields, targetExperimentType)
	}
	return s.templateRepo.UpdateFields(ctx, id, fields)
}

// updateTemplateWithTypeSwitchCleanup 在实验类型切换时统一清理不再适用的配置。
// 文档要求切换实验类型后移除不适用的容器、仿真场景及其衍生配置，避免模板同时保留两套互斥实现。
func (s *templateService) updateTemplateWithTypeSwitchCleanup(
	ctx context.Context,
	templateID int64,
	fields map[string]interface{},
	targetExperimentType int16,
) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&entity.ExperimentTemplate{}).
			Where("id = ?", templateID).
			Updates(fields).Error; err != nil {
			return err
		}

		switch targetExperimentType {
		case enum.ExperimentTypeSimulation:
			if err := tx.Where("template_id = ?", templateID).Delete(&entity.TemplateContainer{}).Error; err != nil {
				return err
			}
			if err := tx.Where("template_id = ?", templateID).Delete(&entity.TemplateInitScript{}).Error; err != nil {
				return err
			}
			if err := tx.Where("template_id = ?", templateID).Delete(&entity.TemplateRole{}).Error; err != nil {
				return err
			}
			if err := tx.Where("template_id = ? AND check_type = ?", templateID, enum.CheckTypeScript).
				Delete(&entity.TemplateCheckpoint{}).Error; err != nil {
				return err
			}
			return tx.Model(&entity.ExperimentTemplate{}).
				Where("id = ?", templateID).
				Updates(map[string]interface{}{
					"topology_mode":  nil,
					"cpu_limit":      nil,
					"memory_limit":   nil,
					"disk_limit":     nil,
					"k8s_config":     nil,
					"network_config": nil,
					"updated_at":     time.Now(),
				}).Error
		case enum.ExperimentTypeReal:
			if err := tx.Where("template_id = ?", templateID).Delete(&entity.TemplateSimScene{}).Error; err != nil {
				return err
			}
			if err := tx.Where("template_id = ? AND check_type = ?", templateID, enum.CheckTypeSimAssert).
				Delete(&entity.TemplateCheckpoint{}).Error; err != nil {
				return err
			}
			return tx.Model(&entity.ExperimentTemplate{}).
				Where("id = ?", templateID).
				Updates(map[string]interface{}{
					"sim_layout": nil,
					"updated_at": time.Now(),
				}).Error
		default:
			return nil
		}
	})
}

// Delete 删除实验模板
func (s *templateService) Delete(ctx context.Context, sc *svcctx.ServiceContext, id int64) error {
	_, err := ensureTemplateOwnerAccess(ctx, s.templateRepo, sc, id)
	if err != nil {
		return err
	}
	hasCourseReferences, err := s.templateRepo.HasCourseReferences(ctx, id)
	if err != nil {
		return err
	}
	if hasCourseReferences {
		return errcode.ErrInvalidParams.WithMessage("该模板已被课时引用，不可删除")
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
	items, err := s.buildTemplateListItems(ctx, templates)
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// ---------------------------------------------------------------------------
// 发布 / 下架
// ---------------------------------------------------------------------------

// Publish 发布实验模板
func (s *templateService) Publish(ctx context.Context, sc *svcctx.ServiceContext, id int64) error {
	if _, err := ensureTemplateOwnerAccess(ctx, s.templateRepo, sc, id); err != nil {
		return err
	}
	template, err := loadTemplateAggregate(
		ctx,
		s.templateRepo,
		s.containerRepo,
		s.checkpointRepo,
		s.initScriptRepo,
		s.simSceneRepo,
		s.templateTagRepo,
		s.tagRepo,
		s.roleRepo,
		id,
	)
	if err != nil {
		return errcode.ErrTemplateNotFound
	}
	if template.Template.Status == enum.TemplateStatusPublished {
		return errcode.ErrTemplateAlreadyPublished
	}
	if template.Template.Status != enum.TemplateStatusDraft {
		return errcode.ErrTemplateNotDraft
	}
	if len(template.Containers) == 0 && len(template.SimScenes) == 0 {
		return errcode.ErrInvalidParams.WithMessage("请至少配置一个容器或一个仿真场景")
	}
	if len(template.Checkpoints) == 0 {
		return errcode.ErrInvalidParams.WithMessage("请至少配置一个检查点")
	}
	if publishErr := validateTemplatePublishReadiness(template); publishErr != nil {
		return publishErr
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

// validateTemplatePublishReadiness 校验模板是否满足拓扑模式对应的发布前置条件。
func validateTemplatePublishReadiness(template *TemplateAggregate) error {
	if template == nil || template.Template == nil {
		return errcode.ErrTemplateNotFound
	}
	topologyMode := int16(0)
	if template.Template.TopologyMode != nil {
		topologyMode = *template.Template.TopologyMode
	}
	if topologyMode != enum.TopologyModeShared {
		return nil
	}

	sharedCount := 0
	instanceCount := 0
	for _, container := range template.Containers {
		if container == nil {
			continue
		}
		if container.DeploymentScope == enum.ContainerDeploymentScopeShared {
			sharedCount++
			continue
		}
		instanceCount++
	}
	if sharedCount == 0 {
		return errcode.ErrInvalidParams.WithMessage("共享基础设施模板至少需要一个共享容器")
	}
	if instanceCount == 0 {
		return errcode.ErrInvalidParams.WithMessage("共享基础设施模板至少需要一个学生实例容器")
	}
	return nil
}

// Unpublish 下架实验模板
func (s *templateService) Unpublish(ctx context.Context, sc *svcctx.ServiceContext, id int64) error {
	template, err := ensureTemplateOwnerAccess(ctx, s.templateRepo, sc, id)
	if err != nil {
		return err
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
	if _, err := ensureTemplateCloneAccess(ctx, s.templateRepo, sc, id); err != nil {
		return nil, err
	}
	source, err := loadTemplateAggregate(
		ctx,
		s.templateRepo,
		s.containerRepo,
		s.checkpointRepo,
		s.initScriptRepo,
		s.simSceneRepo,
		s.templateTagRepo,
		s.tagRepo,
		s.roleRepo,
		id,
	)
	if err != nil {
		return nil, errcode.ErrTemplateNotFound
	}

	var newTemplate *entity.ExperimentTemplate
	txErr := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		roleIDMap := make(map[int64]int64, len(source.Roles))

		// 克隆主表
		newID := snowflake.Generate()
		clonedFromID := source.Template.ID
		newTemplate = &entity.ExperimentTemplate{
			ID:                 newID,
			SchoolID:           sc.SchoolID,
			TeacherID:          sc.UserID,
			Title:              source.Template.Title + "（副本）",
			Description:        source.Template.Description,
			Objectives:         source.Template.Objectives,
			Instructions:       source.Template.Instructions,
			ReferenceMaterials: source.Template.ReferenceMaterials,
			ExperimentType:     source.Template.ExperimentType,
			TopologyMode:       source.Template.TopologyMode,
			JudgeMode:          source.Template.JudgeMode,
			AutoWeight:         source.Template.AutoWeight,
			ManualWeight:       source.Template.ManualWeight,
			TotalScore:         source.Template.TotalScore,
			MaxDuration:        source.Template.MaxDuration,
			IdleTimeout:        source.Template.IdleTimeout,
			CPULimit:           source.Template.CPULimit,
			MemoryLimit:        source.Template.MemoryLimit,
			DiskLimit:          source.Template.DiskLimit,
			ScoreStrategy:      source.Template.ScoreStrategy,
			ClonedFromID:       &clonedFromID,
			Status:             enum.TemplateStatusDraft,
			SimLayout:          source.Template.SimLayout,
			K8sConfig:          source.Template.K8sConfig,
			NetworkConfig:      source.Template.NetworkConfig,
		}
		if err := tx.Create(newTemplate).Error; err != nil {
			return err
		}

		// 克隆角色。
		// 多人协作模板中的容器会引用角色ID，必须先生成新角色并建立映射，避免副本仍指向原模板角色。
		if len(source.Roles) > 0 {
			roles := make([]*entity.TemplateRole, 0, len(source.Roles))
			for _, r := range source.Roles {
				if r == nil {
					continue
				}
				newRoleID := snowflake.Generate()
				roleIDMap[r.ID] = newRoleID
				roles = append(roles, &entity.TemplateRole{
					ID:          newRoleID,
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

		// 克隆容器配置
		if len(source.Containers) > 0 {
			containers := make([]*entity.TemplateContainer, 0, len(source.Containers))
			for _, c := range source.Containers {
				if c == nil {
					continue
				}
				var clonedRoleID *int64
				if c.RoleID != nil {
					if mappedRoleID, ok := roleIDMap[*c.RoleID]; ok {
						clonedRoleID = &mappedRoleID
					}
				}
				containers = append(containers, &entity.TemplateContainer{
					ID:             snowflake.Generate(),
					TemplateID:     newID,
					ImageVersionID: c.ImageVersionID,
					ContainerName:  c.ContainerName,
					RoleID:         clonedRoleID,
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
				if cp == nil {
					continue
				}
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
				if is == nil {
					continue
				}
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
				if ss == nil {
					continue
				}
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
				if tt == nil {
					continue
				}
				tags = append(tags, &entity.TemplateTag{
					ID:         snowflake.Generate(),
					TemplateID: newID,
					TagID:      tt.ID,
				})
			}
			if err := tx.CreateInBatches(tags, 50).Error; err != nil {
				return err
			}
		}

		return nil
	})
	if txErr != nil {
		return nil, txErr
	}

	topologyMode := int16(0)
	if newTemplate.TopologyMode != nil {
		topologyMode = *newTemplate.TopologyMode
	}
	return &dto.CreateTemplateResp{
		ID:                 strconv.FormatInt(newTemplate.ID, 10),
		Title:              newTemplate.Title,
		ExperimentType:     newTemplate.ExperimentType,
		ExperimentTypeText: enum.GetExperimentTypeText(newTemplate.ExperimentType),
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
	template, err := ensureTemplateOwnerAccess(ctx, s.templateRepo, sc, id)
	if err != nil {
		return err
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
	template, err := ensureTemplateOwnerAccess(ctx, s.templateRepo, sc, id)
	if err != nil {
		return nil, err
	}
	return &dto.K8sConfigResp{
		TemplateID: strconv.FormatInt(template.ID, 10),
		K8sConfig:  json.RawMessage(template.K8sConfig),
	}, nil
}

// SetK8sConfig 设置模板K8s编排配置
func (s *templateService) SetK8sConfig(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.K8sConfigReq) error {
	template, err := ensureTemplateOwnerAccess(ctx, s.templateRepo, sc, id)
	if err != nil {
		return err
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
	if _, err := ensureTemplateOwnerAccess(ctx, s.templateRepo, sc, id); err != nil {
		return nil, err
	}
	template, err := loadTemplateAggregate(
		ctx,
		s.templateRepo,
		s.containerRepo,
		s.checkpointRepo,
		s.initScriptRepo,
		s.simSceneRepo,
		s.templateTagRepo,
		s.tagRepo,
		s.roleRepo,
		id,
	)
	if err != nil {
		return nil, errcode.ErrTemplateNotFound
	}

	levels := req.Levels
	if len(levels) == 0 {
		levels = []int{1, 2, 3, 4, 5}
	}

	resp := &dto.ValidateTemplateResp{
		TemplateID: strconv.FormatInt(template.Template.ID, 10),
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

// canEditTemplateStructure 判断模板当前是否允许修改结构性配置。
// 文档要求：
// 1. 草稿模板可自由修改；
// 2. 已发布但未被课时引用的模板仍可继续调整完整编排；
// 3. 已发布且已被课时引用时，只允许修改基本信息、实验说明和检查点，不允许调整结构。
func (s *templateService) canEditTemplateStructure(ctx context.Context, template *entity.ExperimentTemplate) (bool, error) {
	if template == nil {
		return false, errcode.ErrTemplateNotFound
	}
	switch template.Status {
	case enum.TemplateStatusDraft:
		return true, nil
	case enum.TemplateStatusPublished:
		hasCourseReferences, err := s.templateRepo.HasCourseReferences(ctx, template.ID)
		if err != nil {
			return false, err
		}
		return !hasCourseReferences, nil
	default:
		return false, errcode.ErrTemplateNotDraft
	}
}

// hasTemplateStructureChanges 判断更新请求中是否包含结构性字段。
// 已被课时引用的已发布模板只能修改基础信息、说明和检查点评分相关配置，
// 不允许再调整实验类型、资源、判题模式、成绩策略和拓扑等结构性配置。
func hasTemplateStructureChanges(req *dto.UpdateTemplateReq) bool {
	if req == nil {
		return false
	}
	return req.ExperimentType != nil ||
		req.TopologyMode != nil ||
		req.JudgeMode != nil ||
		req.MaxDuration != nil ||
		req.IdleTimeout != nil ||
		req.CPULimit != nil ||
		req.MemoryLimit != nil ||
		req.DiskLimit != nil ||
		req.ScoreStrategy != nil
}

// validateLevel 执行单层验证
func (s *templateService) validateLevel(ctx context.Context, template *TemplateAggregate, level int) dto.ValidationLevelResult {
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
		s.validateResourceReasonableness(ctx, template, &result)
	case 4: // 生态一致性
		result.LevelName = "生态一致性检查"
		result.Severity = "hint"
		s.validateEcosystemConsistency(ctx, template, &result)
	case 5: // 连通性预检
		result.LevelName = "连通性预检"
		result.Severity = "info"
		s.validateConnectivityPrecheck(template, &result)
	}

	return result
}

// validateDependencyIntegrity 验证依赖完整性
func (s *templateService) validateDependencyIntegrity(ctx context.Context, template *TemplateAggregate, result *dto.ValidationLevelResult) {
	if len(template.Containers) == 0 && len(template.SimScenes) == 0 {
		result.Passed = false
		result.Issues = append(result.Issues, dto.ValidationIssue{
			Code:    "L1_EMPTY_TEMPLATE",
			Message: "请至少配置一个容器或一个仿真场景",
		})
	}

	// 检查是否有容器配置
	if len(template.Containers) == 0 && (template.Template.ExperimentType == enum.ExperimentTypeReal || template.Template.ExperimentType == enum.ExperimentTypeMixed) {
		result.Passed = false
		result.Issues = append(result.Issues, dto.ValidationIssue{
			Code:    "L1_MISSING_CONTAINER",
			Message: "真实环境/混合实验必须配置至少一个容器",
		})
	}

	// 检查仿真场景配置
	if len(template.SimScenes) == 0 && (template.Template.ExperimentType == enum.ExperimentTypeSimulation || template.Template.ExperimentType == enum.ExperimentTypeMixed) {
		result.Passed = false
		result.Issues = append(result.Issues, dto.ValidationIssue{
			Code:    "L1_MISSING_SIM_SCENE",
			Message: "纯仿真/混合实验必须配置至少一个仿真场景",
		})
	}
	if len(template.Checkpoints) == 0 {
		result.Passed = false
		result.Issues = append(result.Issues, dto.ValidationIssue{
			Code:    "L1_MISSING_CHECKPOINT",
			Message: "请至少配置一个检查点",
		})
	}

	roleByID := make(map[int64]*entity.TemplateRole, len(template.Roles))
	for _, role := range template.Roles {
		if role == nil {
			continue
		}
		roleByID[role.ID] = role
	}
	hasRoleScopedContainer := false
	for _, container := range template.Containers {
		if container == nil || container.RoleID == nil {
			continue
		}
		hasRoleScopedContainer = true
		if _, ok := roleByID[*container.RoleID]; ok {
			continue
		}
		result.Passed = false
		result.Issues = append(result.Issues, dto.ValidationIssue{
			Code:    "L1_CONTAINER_ROLE_INVALID",
			Message: fmt.Sprintf("容器 %s 绑定的角色不存在或不属于当前实验模板", container.ContainerName),
		})
	}

	topologyMode := int16(0)
	if template.Template.TopologyMode != nil {
		topologyMode = *template.Template.TopologyMode
	}
	if topologyMode == enum.TopologyModeCollaborate {
		if len(template.Roles) == 0 {
			result.Passed = false
			result.Issues = append(result.Issues, dto.ValidationIssue{
				Code:    "L1_GROUP_TOPOLOGY_ROLE_MISSING",
				Message: "多人协作组网模板必须先定义角色",
			})
		}
	} else {
		if len(template.Roles) > 0 {
			result.Passed = false
			result.Issues = append(result.Issues, dto.ValidationIssue{
				Code:    "L1_ROLE_TOPOLOGY_INVALID",
				Message: "仅多人协作组网拓扑允许定义角色",
			})
		}
		if hasRoleScopedContainer {
			result.Passed = false
			result.Issues = append(result.Issues, dto.ValidationIssue{
				Code:    "L1_ROLE_CONTAINER_TOPOLOGY_INVALID",
				Message: "仅多人协作组网拓扑允许配置角色专属容器",
			})
		}
	}
	if topologyMode == enum.TopologyModeShared {
		sharedCount := 0
		instanceCount := 0
		for _, container := range template.Containers {
			if container == nil {
				continue
			}
			if container.DeploymentScope == enum.ContainerDeploymentScopeShared {
				sharedCount++
				if container.RoleID != nil {
					result.Passed = false
					result.Issues = append(result.Issues, dto.ValidationIssue{
						Code:    "L1_SHARED_CONTAINER_ROLE_INVALID",
						Message: fmt.Sprintf("共享容器 %s 不允许绑定协作角色", container.ContainerName),
					})
				}
				continue
			}
			instanceCount++
		}
		if sharedCount == 0 {
			result.Passed = false
			result.Issues = append(result.Issues, dto.ValidationIssue{
				Code:    "L1_SHARED_TOPOLOGY_SHARED_CONTAINER_MISSING",
				Message: "共享基础设施模板必须至少配置一个共享容器",
			})
		}
		if instanceCount == 0 {
			result.Passed = false
			result.Issues = append(result.Issues, dto.ValidationIssue{
				Code:    "L1_SHARED_TOPOLOGY_INSTANCE_CONTAINER_MISSING",
				Message: "共享基础设施模板必须至少配置一个学生实例容器",
			})
		}
	}

	// 检查检查点总分是否匹配
	if len(template.Checkpoints) > 0 {
		var totalScore float64
		for _, cp := range template.Checkpoints {
			if cp != nil {
				totalScore += cp.Score
			}
		}
		if totalScore != template.Template.TotalScore {
			result.Issues = append(result.Issues, dto.ValidationIssue{
				Code:    "L3_SCORE_MISMATCH",
				Message: fmt.Sprintf("检查点总分(%.2f)与模板总分(%.2f)不一致", totalScore, template.Template.TotalScore),
			})
		}
	}

	// 检查容器依赖引用
	containerNames := make(map[string]bool)
	for _, c := range template.Containers {
		if c == nil {
			continue
		}
		containerNames[c.ContainerName] = true
	}
	for _, c := range template.Containers {
		if c == nil {
			continue
		}
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
		if c == nil {
			continue
		}
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
		if image.Status != enum.ImageStatusNormal {
			result.Passed = false
			result.Issues = append(result.Issues, dto.ValidationIssue{
				Code:    "L1_IMAGE_NOT_AVAILABLE",
				Message: fmt.Sprintf("容器 %s 关联的镜像 %s 当前不可用", c.ContainerName, image.Name),
			})
			continue
		}

		for _, dependency := range parseRequiredDependencies(json.RawMessage(image.RequiredDependencies)) {
			if containerNames[dependency] {
				continue
			}
			result.Passed = false
			result.Issues = append(result.Issues, buildMissingDependencyIssue(c.ContainerName, dependency))
		}
	}
}

// validatePortConflicts 验证端口冲突
func (s *templateService) validatePortConflicts(template *TemplateAggregate, result *dto.ValidationLevelResult) {
	type portEntry struct {
		Port      int
		Container string
	}
	usedPorts := make(map[int]string)

	for _, c := range template.Containers {
		if c == nil {
			continue
		}
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

// validateResourceReasonableness 验证资源合理性。
func (s *templateService) validateResourceReasonableness(ctx context.Context, template *TemplateAggregate, result *dto.ValidationLevelResult) {
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
		if c == nil {
			continue
		}
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

	versionMap := make(map[int64]*entity.ImageVersion, len(template.Containers))
	imageMap := make(map[int64]*entity.Image, len(template.Containers))
	for _, container := range template.Containers {
		if container == nil {
			continue
		}
		if _, exists := versionMap[container.ImageVersionID]; !exists {
			version, err := s.imageVersionRepo.GetByID(ctx, container.ImageVersionID)
			if err == nil && version != nil {
				versionMap[container.ImageVersionID] = version
			}
		}
		version := versionMap[container.ImageVersionID]
		if version == nil {
			continue
		}
		if _, exists := imageMap[version.ImageID]; !exists {
			image, err := s.imageRepo.GetByID(ctx, version.ImageID)
			if err == nil && image != nil {
				imageMap[version.ImageID] = image
			}
		}
	}
	result.Issues = append(result.Issues, buildResourceReasonablenessIssues(template.Template, template.Containers, versionMap, imageMap)...)
}

// validateEcosystemConsistency 验证工具镜像与当前实验生态的一致性。
func (s *templateService) validateEcosystemConsistency(ctx context.Context, template *TemplateAggregate, result *dto.ValidationLevelResult) {
	versionMap := make(map[int64]*entity.ImageVersion, len(template.Containers))
	imageMap := make(map[int64]*entity.Image, len(template.Containers))
	for _, container := range template.Containers {
		if container == nil {
			continue
		}
		if _, exists := versionMap[container.ImageVersionID]; !exists {
			version, err := s.imageVersionRepo.GetByID(ctx, container.ImageVersionID)
			if err == nil && version != nil {
				versionMap[container.ImageVersionID] = version
			}
		}
		version := versionMap[container.ImageVersionID]
		if version == nil {
			continue
		}
		if _, exists := imageMap[version.ImageID]; !exists {
			image, err := s.imageRepo.GetByID(ctx, version.ImageID)
			if err == nil && image != nil {
				imageMap[version.ImageID] = image
			}
		}
	}
	result.Issues = append(result.Issues, buildEcosystemConsistencyIssues(template.Containers, versionMap, imageMap)...)
}

// validateConnectivityPrecheck 连通性预检
func (s *templateService) validateConnectivityPrecheck(template *TemplateAggregate, result *dto.ValidationLevelResult) {
	// 检查初始化脚本引用的容器是否存在
	containerNames := make(map[string]bool)
	for _, c := range template.Containers {
		if c == nil {
			continue
		}
		containerNames[c.ContainerName] = true
	}
	for _, is := range template.InitScripts {
		if is == nil {
			continue
		}
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
		if cp == nil {
			continue
		}
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

	for _, cycle := range detectDependencyCycles(template.Containers) {
		result.Passed = false
		result.Issues = append(result.Issues, dto.ValidationIssue{
			Code:    "L5_DEPENDENCY_CYCLE",
			Message: fmt.Sprintf("容器依赖关系存在环：%s", cycle),
		})
	}

	for _, issue := range findInvalidStartupOrders(template.Containers) {
		result.Passed = false
		result.Issues = append(result.Issues, dto.ValidationIssue{
			Code:    "L5_STARTUP_ORDER_INVALID",
			Message: issue,
		})
	}

	for _, issue := range findMissingServiceReferences(template.Containers) {
		result.Passed = false
		result.Issues = append(result.Issues, dto.ValidationIssue{
			Code:    "L5_SERVICE_REF_MISSING",
			Message: issue,
		})
	}
}

// ---------------------------------------------------------------------------
// 转换方法
// ---------------------------------------------------------------------------

// toTemplateResp 转换模板为详情响应
func (s *templateService) toTemplateResp(ctx context.Context, t *TemplateAggregate) *dto.TemplateResp {
	if t == nil || t.Template == nil {
		return nil
	}
	template := t.Template
	topologyMode := int16(0)
	if template.TopologyMode != nil {
		topologyMode = *template.TopologyMode
	}
	maxDuration := 0
	if template.MaxDuration != nil {
		maxDuration = *template.MaxDuration
	}

	resp := &dto.TemplateResp{
		ID:                 strconv.FormatInt(template.ID, 10),
		Title:              template.Title,
		Description:        template.Description,
		Objectives:         template.Objectives,
		Instructions:       template.Instructions,
		ReferenceMaterials: template.ReferenceMaterials,
		ExperimentType:     template.ExperimentType,
		ExperimentTypeText: enum.GetExperimentTypeText(template.ExperimentType),
		TopologyMode:       topologyMode,
		TopologyModeText:   enum.GetTopologyModeText(topologyMode),
		JudgeMode:          template.JudgeMode,
		JudgeModeText:      enum.GetJudgeModeText(template.JudgeMode),
		AutoWeight:         template.AutoWeight,
		ManualWeight:       template.ManualWeight,
		TotalScore:         int(template.TotalScore),
		MaxDuration:        maxDuration,
		CPULimit:           template.CPULimit,
		MemoryLimit:        template.MemoryLimit,
		DiskLimit:          template.DiskLimit,
		ScoreStrategy:      template.ScoreStrategy,
		IsShared:           template.IsShared,
		Status:             template.Status,
		StatusText:         enum.GetTemplateStatusText(template.Status),
		K8sConfig:          json.RawMessage(template.K8sConfig),
		CreatedAt:          template.CreatedAt.Format(time.RFC3339),
		UpdatedAt:          template.UpdatedAt.Format(time.RFC3339),
	}

	if template.IdleTimeout > 0 {
		idleTimeout := template.IdleTimeout
		resp.IdleTimeout = &idleTimeout
	}

	// 教师信息
	teacherName := s.userNameQuerier.GetUserName(ctx, template.TeacherID)
	resp.Teacher = &dto.SimpleUserResp{
		ID:   strconv.FormatInt(template.TeacherID, 10),
		Name: teacherName,
	}

	// 容器列表
	resp.Containers = make([]dto.ContainerResp, 0, len(t.Containers))
	for _, c := range t.Containers {
		if c == nil {
			continue
		}
		resp.Containers = append(resp.Containers, s.toContainerResp(ctx, c))
	}

	// 检查点列表
	resp.Checkpoints = make([]dto.CheckpointResp, 0, len(t.Checkpoints))
	for _, cp := range t.Checkpoints {
		if cp == nil {
			continue
		}
		resp.Checkpoints = append(resp.Checkpoints, s.toCheckpointResp(cp))
	}

	// 初始化脚本列表
	resp.InitScripts = make([]dto.InitScriptResp, 0, len(t.InitScripts))
	for _, is := range t.InitScripts {
		if is == nil {
			continue
		}
		resp.InitScripts = append(resp.InitScripts, s.toInitScriptResp(is))
	}

	// 仿真场景列表
	resp.SimScenes = make([]dto.TemplateSimSceneResp, 0, len(t.SimScenes))
	for _, ss := range t.SimScenes {
		if ss == nil {
			continue
		}
		resp.SimScenes = append(resp.SimScenes, s.toSimSceneResp(ctx, ss))
	}

	// 标签列表
	resp.Tags = s.buildTagResponses(t.Tags)

	// 角色列表
	resp.Roles = make([]dto.RoleResp, 0, len(t.Roles))
	for _, r := range t.Roles {
		if r == nil {
			continue
		}
		resp.Roles = append(resp.Roles, s.toRoleResp(r))
	}

	return resp
}

// buildTagResponses 构建模板标签响应列表。
func (s *templateService) buildTagResponses(tags []*entity.Tag) []dto.TagResp {
	resp := make([]dto.TagResp, 0, len(tags))
	for _, tag := range tags {
		if tag == nil {
			continue
		}
		resp = append(resp, *buildTagResp(tag))
	}
	return resp
}

// buildTemplateListItems 批量聚合模板列表所需的容器数、检查点数和标签信息。
func (s *templateService) buildTemplateListItems(ctx context.Context, templates []*entity.ExperimentTemplate) ([]*dto.TemplateListItem, error) {
	templateIDs := make([]int64, 0, len(templates))
	for _, template := range templates {
		if template == nil {
			continue
		}
		templateIDs = append(templateIDs, template.ID)
	}

	containers, err := s.containerRepo.ListByTemplateIDs(ctx, templateIDs)
	if err != nil {
		return nil, err
	}
	checkpoints, err := s.checkpointRepo.ListByTemplateIDs(ctx, templateIDs)
	if err != nil {
		return nil, err
	}
	templateTags, err := s.templateTagRepo.ListByTemplateIDs(ctx, templateIDs)
	if err != nil {
		return nil, err
	}

	tagIDs := make([]int64, 0, len(templateTags))
	seenTagIDs := make(map[int64]struct{}, len(templateTags))
	for _, relation := range templateTags {
		if relation == nil {
			continue
		}
		if _, exists := seenTagIDs[relation.TagID]; exists {
			continue
		}
		seenTagIDs[relation.TagID] = struct{}{}
		tagIDs = append(tagIDs, relation.TagID)
	}

	var tags []*entity.Tag
	if len(tagIDs) > 0 {
		tags, err = s.tagRepo.ListByIDs(ctx, tagIDs)
		if err != nil {
			return nil, err
		}
	}

	return buildTemplateListItemsWithMetadata(templates, containers, checkpoints, templateTags, tags), nil
}

// toContainerResp 转换容器为响应
func (s *templateService) toContainerResp(ctx context.Context, c *entity.TemplateContainer) dto.ContainerResp {
	var version *entity.ImageVersion
	if s.imageVersionRepo != nil {
		if loadedVersion, err := s.imageVersionRepo.GetByID(ctx, c.ImageVersionID); err == nil {
			version = loadedVersion
		}
	}
	var image *entity.Image
	if version != nil && s.imageRepo != nil {
		if loadedImage, err := s.imageRepo.GetByID(ctx, version.ImageID); err == nil {
			image = loadedImage
		}
	}
	allContainers := make([]entity.TemplateContainer, 0, 1)
	if s.containerRepo != nil {
		if containerList, err := s.containerRepo.ListByTemplateID(ctx, c.TemplateID); err == nil {
			allContainers = make([]entity.TemplateContainer, 0, len(containerList))
			for _, item := range containerList {
				if item == nil {
					continue
				}
				allContainers = append(allContainers, *item)
			}
		}
	}
	if len(allContainers) == 0 {
		allContainers = append(allContainers, *c)
	}
	return *buildTemplateContainerRespWithImage(c, version, image, allContainers)
}

// toCheckpointResp 转换检查点为响应
func (s *templateService) toCheckpointResp(cp *entity.TemplateCheckpoint) dto.CheckpointResp {
	return *buildTemplateCheckpointResp(cp)
}

// toInitScriptResp 转换初始化脚本为响应
func (s *templateService) toInitScriptResp(is *entity.TemplateInitScript) dto.InitScriptResp {
	return *buildTemplateInitScriptResp(is)
}

// toSimSceneResp 转换仿真场景配置为响应
func (s *templateService) toSimSceneResp(ctx context.Context, ss *entity.TemplateSimScene) dto.TemplateSimSceneResp {
	var scenario *entity.SimScenario
	if s.scenarioRepo != nil {
		if loadedScenario, err := s.scenarioRepo.GetByID(ctx, ss.ScenarioID); err == nil {
			scenario = loadedScenario
		}
	}
	var linkGroup *entity.SimLinkGroup
	if ss.LinkGroupID != nil && s.linkGroupRepo != nil {
		if loadedLinkGroup, err := s.linkGroupRepo.GetByID(ctx, *ss.LinkGroupID); err == nil {
			linkGroup = loadedLinkGroup
		}
	}
	return *buildTemplateSimSceneRespWithRelations(ss, scenario, linkGroup)
}

// toRoleResp 转换角色为响应
func (s *templateService) toRoleResp(r *entity.TemplateRole) dto.RoleResp {
	return *buildTemplateRoleResp(r)
}
