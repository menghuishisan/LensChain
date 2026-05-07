// template_service_student.go
// 模块04 — 实验环境：学生端模板只读查询
// 学生只能查看同校已发布模板的摘要信息，用于 P-42 启动页和 P-40 模板选择器。

package experiment

import (
	"context"
	"strconv"

	"github.com/lenschain/backend/internal/model/dto"
	"github.com/lenschain/backend/internal/model/enum"
	svcctx "github.com/lenschain/backend/internal/pkg/context"
	"github.com/lenschain/backend/internal/pkg/errcode"
	experimentrepo "github.com/lenschain/backend/internal/repository/experiment"
)

// StudentList 学生端已发布模板列表。
// 仅返回同校已发布模板，复用 TemplateListItem 结构。
func (s *templateService) StudentList(ctx context.Context, sc *svcctx.ServiceContext, req *dto.StudentTemplateListReq) ([]*dto.TemplateListItem, int64, error) {
	params := &experimentrepo.TemplateListParams{
		SchoolID: sc.SchoolID,
		Status:   int16(enum.TemplateStatusPublished),
		Page:     req.Page,
		PageSize: req.PageSize,
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

// StudentGetSummary 学生端模板摘要详情。
// 仅返回同校已发布模板的展示信息，不暴露 K8s 配置、初始化脚本等。
func (s *templateService) StudentGetSummary(ctx context.Context, sc *svcctx.ServiceContext, id int64) (*dto.StudentTemplateSummaryResp, error) {
	template, err := s.templateRepo.GetByID(ctx, id)
	if err != nil {
		return nil, errcode.ErrTemplateNotFound
	}
	// 学生只能访问同校已发布模板
	if template.SchoolID != sc.SchoolID {
		return nil, errcode.ErrTemplateNotFound
	}
	if template.Status != enum.TemplateStatusPublished {
		return nil, errcode.ErrTemplateNotFound
	}

	// 加载容器和检查点
	containers, err := s.containerRepo.ListByTemplateID(ctx, id)
	if err != nil {
		return nil, err
	}
	checkpoints, err := s.checkpointRepo.ListByTemplateID(ctx, id)
	if err != nil {
		return nil, err
	}

	// 加载标签
	templateTags, _ := s.templateTagRepo.ListByTemplateID(ctx, id)
	var tags []dto.TagResp
	for _, tt := range templateTags {
		tag, tagErr := s.tagRepo.GetByID(ctx, tt.TagID)
		if tagErr == nil {
			tags = append(tags, dto.TagResp{
				ID:       strconv.FormatInt(tag.ID, 10),
				Name:     tag.Name,
				Category: tag.Category,
			})
		}
	}

	// 加载仿真场景（仅 ID 和 time_control_mode）
	simScenes, _ := s.simSceneRepo.ListByTemplateID(ctx, id)
	var simSceneSummaries []dto.StudentSimSceneSummary
	for _, scene := range simScenes {
		ss := dto.StudentSimSceneSummary{
			ID: strconv.FormatInt(scene.ID, 10),
		}
		if scene.ScenarioID != 0 {
			scenario, scErr := s.scenarioRepo.GetByID(ctx, scene.ScenarioID)
			if scErr == nil {
				ss.Scenario = &dto.StudentSimScenarioMinimal{
					Code:            scenario.Code,
					Name:            scenario.Name,
					Category:        scenario.Category,
					TimeControlMode: scenario.TimeControlMode,
				}
			}
		}
		simSceneSummaries = append(simSceneSummaries, ss)
	}

	// 构建容器摘要（仅名称和镜像信息）
	var containerSummaries []dto.StudentContainerSummary
	for _, c := range containers {
		summary := dto.StudentContainerSummary{
			ContainerName: c.ContainerName,
		}
		if c.ImageVersionID != 0 {
			iv, ivErr := s.imageVersionRepo.GetByID(ctx, c.ImageVersionID)
			if ivErr == nil {
				img, imgErr := s.imageRepo.GetByID(ctx, iv.ImageID)
				imgName := ""
				imgDisplayName := ""
				var iconURL *string
				if imgErr == nil {
					imgName = img.Name
					imgDisplayName = img.DisplayName
					iconURL = img.IconURL
				}
				summary.ImageVersion = &dto.ContainerImageVersionResp{
					ID:               strconv.FormatInt(iv.ID, 10),
					ImageName:        imgName,
					ImageDisplayName: imgDisplayName,
					Version:          iv.Version,
					IconURL:          iconURL,
				}
			}
		}
		containerSummaries = append(containerSummaries, summary)
	}

	// 处理可空字段
	var topologyMode int16
	if template.TopologyMode != nil {
		topologyMode = *template.TopologyMode
	}
	var maxDuration int
	if template.MaxDuration != nil {
		maxDuration = *template.MaxDuration
	}

	resp := &dto.StudentTemplateSummaryResp{
		ID:                 strconv.FormatInt(template.ID, 10),
		Title:              template.Title,
		Description:        template.Description,
		Objectives:         template.Objectives,
		Instructions:       template.Instructions,
		ExperimentType:     template.ExperimentType,
		ExperimentTypeText: enum.GetExperimentTypeText(template.ExperimentType),
		TopologyMode:       topologyMode,
		TopologyModeText:   enum.GetTopologyModeText(topologyMode),
		TotalScore:         int(template.TotalScore),
		MaxDuration:        maxDuration,
		CPULimit:           template.CPULimit,
		MemoryLimit:        template.MemoryLimit,
		DiskLimit:          template.DiskLimit,
		Containers:         containerSummaries,
		CheckpointCount:    len(checkpoints),
		Tags:               tags,
		SimScenes:          simSceneSummaries,
	}
	return resp, nil
}
