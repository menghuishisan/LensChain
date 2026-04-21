// instance_runtime_policy.go
// 模块04 — 实验环境：实例运行时网络与采集策略辅助
// 负责在 service 层生成命名空间标签、网络互通策略和混合实验 Collector 注入计划
// 该文件只承载实例编排相关业务规则，不直接执行 K8s 调用或持久化操作

package experiment

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"gorm.io/datatypes"

	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/model/enum"
)

const groupNamespaceLabelKey = "lenschain/group-id"

// buildInstanceNamespaceLabels 构建实例命名空间标签，用于资源隔离与同组互通。
func buildInstanceNamespaceLabels(instance *entity.ExperimentInstance) map[string]string {
	labels := map[string]string{
		"app":         "lenschain-experiment",
		"instance-id": fmt.Sprintf("%d", instance.ID),
		"school-id":   fmt.Sprintf("%d", instance.SchoolID),
	}
	if instance.GroupID != nil {
		labels[groupNamespaceLabelKey] = fmt.Sprintf("%d", *instance.GroupID)
	}
	return labels
}

// buildInstanceNetworkPolicy 为实验实例构建统一网络策略。
// 默认允许同实例命名空间互通；分组实验额外放开同组命名空间之间的通信。
func buildInstanceNetworkPolicy(instance *entity.ExperimentInstance) *NetworkPolicySpec {
	if instance == nil {
		return nil
	}
	policy := &NetworkPolicySpec{
		AllowSameNamespace: true,
	}
	if instance.GroupID != nil {
		policy.AllowNamespaceLabelSelectors = append(policy.AllowNamespaceLabelSelectors, map[string]string{
			groupNamespaceLabelKey: fmt.Sprintf("%d", *instance.GroupID),
		})
	}
	return policy
}

// buildCollectorInjectionPlan 根据混合实验模板和容器生态确定是否需要注入 Collector sidecar。
func buildCollectorInjectionPlan(template *TemplateAggregate, container *entity.TemplateContainer, image *entity.Image) *CollectorSidecarSpec {
	if template == nil || template.Template == nil || container == nil || image == nil || image.Ecosystem == nil {
		return nil
	}
	if template.Template.ExperimentType != enum.ExperimentTypeMixed {
		return nil
	}
	ecosystem := strings.TrimSpace(*image.Ecosystem)
	if ecosystem == "" || !enum.IsValidCollectorEcosystem(ecosystem) {
		return nil
	}

	matchedConfigs := make([]datatypes.JSON, 0)
	for _, scene := range template.SimScenes {
		targets := collectDataSourceTargetContainers(scene.DataSourceConfig)
		for _, target := range targets {
			if target != container.ContainerName {
				continue
			}
			matchedConfigs = append(matchedConfigs, scene.DataSourceConfig)
			break
		}
	}
	if len(matchedConfigs) == 0 {
		return nil
	}
	return &CollectorSidecarSpec{
		TargetContainer:  container.ContainerName,
		Ecosystem:        ecosystem,
		DataSourceConfig: mergeCollectorDataSourceConfig(matchedConfigs),
	}
}

// collectDataSourceTargetContainers 从混合实验数据采集配置中提取目标容器名。
func collectDataSourceTargetContainers(raw datatypes.JSON) []string {
	if len(raw) == 0 {
		return nil
	}

	var payload any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil
	}

	set := make(map[string]struct{})
	collectTargetContainers(payload, set)

	result := make([]string, 0, len(set))
	for item := range set {
		result = append(result, item)
	}
	sort.Strings(result)
	return result
}

// mergeCollectorDataSourceConfig 将同一容器关联的多条场景采集配置合并为统一 sidecar 配置。
func mergeCollectorDataSourceConfig(configs []datatypes.JSON) []byte {
	if len(configs) == 0 {
		return nil
	}
	if len(configs) == 1 {
		return []byte(configs[0])
	}
	payload, err := json.Marshal(map[string]any{
		"sources": configs,
	})
	if err != nil {
		return []byte(configs[0])
	}
	return payload
}

// collectTargetContainers 递归收集数据采集配置中的目标容器字段。
func collectTargetContainers(node any, targets map[string]struct{}) {
	switch typed := node.(type) {
	case map[string]any:
		for _, key := range []string{"target_container", "container_name", "source_container"} {
			if value, ok := typed[key].(string); ok {
				value = strings.TrimSpace(value)
				if value != "" {
					targets[value] = struct{}{}
				}
			}
		}
		for _, value := range typed {
			collectTargetContainers(value, targets)
		}
	case []any:
		for _, item := range typed {
			collectTargetContainers(item, targets)
		}
	}
}
