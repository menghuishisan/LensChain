// group_runtime_helper.go
// 模块04 — 实验环境：多人实验运行时编排辅助
// 负责在 service 层统一解析分组成员运行上下文，生成角色容器部署计划与组内服务发现变量

package experiment

import (
	"context"
	"fmt"

	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/model/enum"
	"github.com/lenschain/backend/internal/pkg/errcode"
)

// runtimeContainerPlan 表示实例启动时的容器部署计划。
type runtimeContainerPlan struct {
	Containers                []entity.TemplateContainer
	ContainerNameSet          map[string]struct{}
	ServiceDiscoveryEnvs      map[string]string
	SharedContainers          []entity.TemplateContainer
	SharedContainerNameSet    map[string]struct{}
	SharedServiceDiscoveryEnvs map[string]string
}

// resolveRuntimeContainerPlan 根据实例与分组上下文生成本次实际部署的容器计划。
func (s *instanceService) resolveRuntimeContainerPlan(
	ctx context.Context,
	instance *entity.ExperimentInstance,
	template *TemplateAggregate,
) (*runtimeContainerPlan, error) {
	allContainers := flattenTemplateContainers(template)
	sharedContainers := filterContainersByDeploymentScope(allContainers, enum.ContainerDeploymentScopeShared)
	plan := &runtimeContainerPlan{
		Containers:                 allContainers,
		ContainerNameSet:           buildContainerNameSet(allContainers),
		ServiceDiscoveryEnvs:       buildServiceDiscoveryEnvVars(allContainers),
		SharedContainers:           sharedContainers,
		SharedContainerNameSet:     buildContainerNameSet(sharedContainers),
		SharedServiceDiscoveryEnvs: buildServiceDiscoveryEnvVars(sharedContainers),
	}
	if template != nil && template.Template != nil && template.Template.TopologyMode != nil && *template.Template.TopologyMode == enum.TopologyModeShared {
		instanceContainers := filterContainersByDeploymentScope(allContainers, enum.ContainerDeploymentScopeInstance)
		plan.Containers = instanceContainers
		plan.ContainerNameSet = buildContainerNameSet(instanceContainers)
		plan.ServiceDiscoveryEnvs = buildServiceDiscoveryEnvVars(instanceContainers)
		return plan, nil
	}
	if instance == nil || instance.GroupID == nil {
		return plan, nil
	}

	members, err := s.groupMemberRepo.ListByGroupID(ctx, *instance.GroupID)
	if err != nil {
		return nil, err
	}
	currentMember := findGroupMemberByStudent(members, instance.StudentID)
	if currentMember == nil {
		return nil, errcode.ErrForbidden.WithMessage("当前学生不在实验分组中")
	}
	if currentMember.RoleID == nil {
		return nil, errcode.ErrInvalidParams.WithMessage("分组成员尚未分配角色")
	}

	deployable := filterContainersByRole(allContainers, *currentMember.RoleID)
	plan.Containers = deployable
	plan.ContainerNameSet = buildContainerNameSet(deployable)

	instances, err := s.instanceRepo.ListByGroupID(ctx, *instance.GroupID)
	if err != nil {
		return nil, err
	}
	roleNamespaces := buildRoleNamespaceIndex(members, buildLatestInstanceByStudent(instances))
	plan.ServiceDiscoveryEnvs = buildServiceDiscoveryEnvVarsWithHostResolver(
		allContainers,
		func(container entity.TemplateContainer) string {
			if container.RoleID == nil || *container.RoleID == *currentMember.RoleID {
				return container.ContainerName
			}
			namespace := roleNamespaces[*container.RoleID]
			if namespace == "" {
				return ""
			}
			return fmt.Sprintf("%s.%s.svc.cluster.local", container.ContainerName, namespace)
		},
	)

	return plan, nil
}

// filterContainersByDeploymentScope 根据部署范围过滤容器。
func filterContainersByDeploymentScope(containers []entity.TemplateContainer, scope int16) []entity.TemplateContainer {
	filtered := make([]entity.TemplateContainer, 0, len(containers))
	for _, container := range containers {
		currentScope := container.DeploymentScope
		if currentScope == 0 {
			currentScope = enum.ContainerDeploymentScopeInstance
		}
		if currentScope != scope {
			continue
		}
		filtered = append(filtered, container)
	}
	return filtered
}

// flattenTemplateContainers 将模板容器切片转换为值类型切片，方便运行时规划复用。
func flattenTemplateContainers(template *TemplateAggregate) []entity.TemplateContainer {
	if template == nil || len(template.Containers) == 0 {
		return []entity.TemplateContainer{}
	}
	items := make([]entity.TemplateContainer, 0, len(template.Containers))
	for _, container := range template.Containers {
		if container == nil {
			continue
		}
		items = append(items, *container)
	}
	return items
}

// buildContainerNameSet 构建容器名集合，用于初始化脚本与其他运行时目标过滤。
func buildContainerNameSet(containers []entity.TemplateContainer) map[string]struct{} {
	set := make(map[string]struct{}, len(containers))
	for _, container := range containers {
		if container.ContainerName == "" {
			continue
		}
		set[container.ContainerName] = struct{}{}
	}
	return set
}

// filterContainersByRole 过滤出当前角色应部署的容器。
func filterContainersByRole(containers []entity.TemplateContainer, roleID int64) []entity.TemplateContainer {
	filtered := make([]entity.TemplateContainer, 0, len(containers))
	for _, container := range containers {
		if container.RoleID == nil || *container.RoleID == roleID {
			filtered = append(filtered, container)
		}
	}
	return filtered
}

// findGroupMemberByStudent 按学生 ID 查找分组成员。
func findGroupMemberByStudent(members []*entity.GroupMember, studentID int64) *entity.GroupMember {
	for _, member := range members {
		if member == nil {
			continue
		}
		if member.StudentID == studentID {
			return member
		}
	}
	return nil
}

// buildRoleNamespaceIndex 为每个角色选择一个已启动实例的命名空间，供组内服务发现引用。
func buildRoleNamespaceIndex(
	members []*entity.GroupMember,
	latestByStudent map[int64]*entity.ExperimentInstance,
) map[int64]string {
	namespaces := make(map[int64]string)
	for _, member := range members {
		if member == nil || member.RoleID == nil {
			continue
		}
		if _, exists := namespaces[*member.RoleID]; exists {
			continue
		}
		namespace := resolveInstanceNamespace(latestByStudent[member.StudentID])
		if namespace == "" {
			continue
		}
		namespaces[*member.RoleID] = namespace
	}
	return namespaces
}

// resolveInstanceNamespace 返回实例在 K8s 中应使用的命名空间名。
func resolveInstanceNamespace(instance *entity.ExperimentInstance) string {
	if instance == nil {
		return ""
	}
	if instance.Namespace != nil && *instance.Namespace != "" {
		return *instance.Namespace
	}
	return fmt.Sprintf("exp-%d", instance.ID)
}
