// runtime_loader.go
// 模块04 — 实验环境：service 层运行时聚合加载辅助
// 负责在 service 层基于多个 repository 组装模板与实例的聚合视图，避免上层继续依赖旧版仓储聚合接口。

package experiment

import (
	"context"

	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/pkg/errcode"
	experimentrepo "github.com/lenschain/backend/internal/repository/experiment"
)

// TemplateAggregate 表示 service 层使用的模板聚合视图。
// repository 仅返回各自表实体，聚合关系由 service 在这里统一组装。
type TemplateAggregate struct {
	Template    *entity.ExperimentTemplate
	Containers  []*entity.TemplateContainer
	Checkpoints []*entity.TemplateCheckpoint
	InitScripts []*entity.TemplateInitScript
	SimScenes   []*entity.TemplateSimScene
	Tags        []*entity.Tag
	Roles       []*entity.TemplateRole
}

// InstanceAggregate 表示 service 层使用的实例聚合视图。
type InstanceAggregate struct {
	Instance          *entity.ExperimentInstance
	Containers        []*entity.InstanceContainer
	CheckpointResults []*entity.CheckpointResult
}

// loadTemplateAggregate 组装模板聚合视图。
func loadTemplateAggregate(
	ctx context.Context,
	templateRepo experimentrepo.TemplateRepository,
	containerRepo experimentrepo.ContainerRepository,
	checkpointRepo experimentrepo.CheckpointRepository,
	initScriptRepo experimentrepo.InitScriptRepository,
	simSceneRepo experimentrepo.SimSceneRepository,
	templateTagRepo experimentrepo.TemplateTagRepository,
	tagRepo experimentrepo.TagRepository,
	roleRepo experimentrepo.RoleRepository,
	templateID int64,
) (*TemplateAggregate, error) {
	template, err := templateRepo.GetByID(ctx, templateID)
	if err != nil {
		return nil, errcode.ErrTemplateNotFound
	}

	aggregate := &TemplateAggregate{Template: template}
	if containerRepo != nil {
		if aggregate.Containers, err = containerRepo.ListByTemplateID(ctx, templateID); err != nil {
			return nil, err
		}
	}
	if checkpointRepo != nil {
		if aggregate.Checkpoints, err = checkpointRepo.ListByTemplateID(ctx, templateID); err != nil {
			return nil, err
		}
	}
	if initScriptRepo != nil {
		if aggregate.InitScripts, err = initScriptRepo.ListByTemplateID(ctx, templateID); err != nil {
			return nil, err
		}
	}
	if simSceneRepo != nil {
		if aggregate.SimScenes, err = simSceneRepo.ListByTemplateID(ctx, templateID); err != nil {
			return nil, err
		}
	}
	if roleRepo != nil {
		if aggregate.Roles, err = roleRepo.ListByTemplateID(ctx, templateID); err != nil {
			return nil, err
		}
	}
	if templateTagRepo != nil && tagRepo != nil {
		templateTags, listErr := templateTagRepo.ListByTemplateID(ctx, templateID)
		if listErr != nil {
			return nil, listErr
		}
		tagIDs := make([]int64, 0, len(templateTags))
		for _, templateTag := range templateTags {
			tagIDs = append(tagIDs, templateTag.TagID)
		}
		if len(tagIDs) > 0 {
			if aggregate.Tags, err = tagRepo.ListByIDs(ctx, tagIDs); err != nil {
				return nil, err
			}
		}
	}

	return aggregate, nil
}

// loadInstanceAggregate 组装实例聚合视图。
func loadInstanceAggregate(
	ctx context.Context,
	instanceRepo experimentrepo.InstanceRepository,
	containerRepo experimentrepo.InstanceContainerRepository,
	checkResultRepo experimentrepo.CheckpointResultRepository,
	instanceID int64,
) (*InstanceAggregate, error) {
	instance, err := instanceRepo.GetByID(ctx, instanceID)
	if err != nil {
		return nil, errcode.ErrInstanceNotFound
	}

	aggregate := &InstanceAggregate{Instance: instance}
	if containerRepo != nil {
		if aggregate.Containers, err = containerRepo.ListByInstanceID(ctx, instanceID); err != nil {
			return nil, err
		}
	}
	if checkResultRepo != nil {
		if aggregate.CheckpointResults, err = checkResultRepo.ListByInstanceID(ctx, instanceID); err != nil {
			return nil, err
		}
	}
	return aggregate, nil
}
