package experiment

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

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
func buildCollectorInjectionPlan(template *entity.ExperimentTemplate, container *entity.TemplateContainer, image *entity.Image) *CollectorSidecarSpec {
	if template == nil || container == nil || image == nil || image.Ecosystem == nil {
		return nil
	}
	if template.ExpType != enum.ExperimentTypeMixed {
		return nil
	}
	ecosystem := strings.TrimSpace(*image.Ecosystem)
	if ecosystem == "" || !enum.IsValidCollectorEcosystem(ecosystem) {
		return nil
	}

	matchedConfigs := make([]json.RawMessage, 0)
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
func collectDataSourceTargetContainers(raw json.RawMessage) []string {
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
func mergeCollectorDataSourceConfig(configs []json.RawMessage) []byte {
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
