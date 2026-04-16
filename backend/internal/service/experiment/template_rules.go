package experiment

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/lenschain/backend/internal/model/dto"
	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/model/enum"
)

// serviceDiscoveryPort 统一承载模板端口配置的最小解析结构。
type serviceDiscoveryPort struct {
	ContainerPort int    `json:"container_port"`
	ServicePort   int    `json:"service_port"`
	Port          int    `json:"port"`
	Protocol      string `json:"protocol"`
}

// conditionalEnvRule 承载条件环境变量规则。
type conditionalEnvRule struct {
	Key          string                   `json:"key"`
	Value        string                   `json:"value"`
	DefaultValue string                   `json:"default_value"`
	Conditions   []conditionalEnvDecision `json:"conditions"`
}

// conditionalEnvDecision 承载单个条件分支。
type conditionalEnvDecision struct {
	When       string              `json:"when"`
	Value      string              `json:"value"`
	InjectVars []map[string]string `json:"inject_vars"`
}

var nonEnvCharPattern = regexp.MustCompile(`[^A-Z0-9]+`)

// determinePublishable 判断验证结果是否允许发布。
func determinePublishable(results []dto.ValidationLevelResult) bool {
	for _, result := range results {
		for _, issue := range result.Issues {
			if strings.HasPrefix(issue.Code, "L1_") || strings.HasPrefix(issue.Code, "L2_") {
				return false
			}
		}
	}
	return true
}

// summarizeValidationIssues 根据问题编码前缀汇总错误等级。
func summarizeValidationIssues(results []dto.ValidationLevelResult) dto.ValidationSummary {
	summary := dto.ValidationSummary{}
	for _, result := range results {
		for _, issue := range result.Issues {
			switch {
			case strings.HasPrefix(issue.Code, "L1_"), strings.HasPrefix(issue.Code, "L2_"):
				summary.Errors++
			case strings.HasPrefix(issue.Code, "L3_"):
				summary.Warnings++
			case strings.HasPrefix(issue.Code, "L4_"):
				summary.Hints++
			default:
				summary.Infos++
			}
		}
	}
	return summary
}

// buildServiceDiscoveryEnvVars 为模板内所有容器生成统一的服务发现变量。
func buildServiceDiscoveryEnvVars(containers []entity.TemplateContainer) map[string]string {
	envs := make(map[string]string)
	usedNames := make(map[string]int)

	for _, container := range containers {
		baseKey := normalizeContainerEnvPrefix(container.ContainerName)
		if baseKey == "" {
			continue
		}
		if usedNames[baseKey] > 0 {
			baseKey = fmt.Sprintf("%s_%d", baseKey, usedNames[baseKey]+1)
		}
		usedNames[baseKey]++
		envs[baseKey+"_HOST"] = container.ContainerName

		for _, port := range parseServiceDiscoveryPorts(container.Ports) {
			servicePort := port.ServicePort
			if servicePort <= 0 {
				if port.Port > 0 {
					servicePort = port.Port
				} else {
					servicePort = port.ContainerPort
				}
			}
			if servicePort <= 0 {
				continue
			}
			envs[fmt.Sprintf("%s_PORT_%d", baseKey, servicePort)] = strconv.Itoa(servicePort)
		}
	}

	return envs
}

// mergeContainerConfig 合并镜像默认配置、条件环境变量和模板容器显式配置。
func mergeContainerConfig(
	container entity.TemplateContainer,
	image *entity.Image,
	version *entity.ImageVersion,
	allContainers []entity.TemplateContainer,
) (ContainerSpec, error) {
	spec := ContainerSpec{
		Name: container.ContainerName,
	}
	if version != nil {
		spec.Image = version.RegistryURL
		if container.CPULimit == nil && version.MinCPU != nil {
			spec.CPULimit = *version.MinCPU
		}
		if container.MemoryLimit == nil && version.MinMemory != nil {
			spec.MemoryLimit = *version.MinMemory
		}
	}
	if container.CPULimit != nil {
		spec.CPULimit = *container.CPULimit
	}
	if container.MemoryLimit != nil {
		spec.MemoryLimit = *container.MemoryLimit
	}

	envVars := make(map[string]string)
	for key, value := range buildServiceDiscoveryEnvVars(allContainers) {
		envVars[key] = value
	}
	mergeDefaultEnvVars(envVars, image)
	if err := mergeConditionalEnvVars(envVars, container.EnvVars, allContainers); err != nil {
		return ContainerSpec{}, err
	}
	if explicitVars := parseExplicitEnvVars(container.EnvVars); len(explicitVars) > 0 {
		for key, value := range explicitVars {
			envVars[key] = value
		}
	}
	spec.EnvVars = envVars

	spec.Ports = mergePorts(image, container.Ports)
	spec.Volumes = mergeVolumes(image, container.Volumes)
	return spec, nil
}

// calculateRemainingConcurrency 计算学校剩余可分配并发额度。
func calculateRemainingConcurrency(maxConcurrency int, quotas []*entity.ResourceQuota, currentCourseID *int64) int {
	allocated := 0
	for _, quota := range quotas {
		if quota == nil || quota.CourseID == nil {
			continue
		}
		if currentCourseID != nil && *quota.CourseID == *currentCourseID {
			continue
		}
		allocated += quota.MaxConcurrency
	}
	remaining := maxConcurrency - allocated
	if remaining < 0 {
		return 0
	}
	return remaining
}

// buildMixedDataCollectionConfig 构建混合实验的数据采集配置。
func buildMixedDataCollectionConfig(template *entity.ExperimentTemplate) json.RawMessage {
	if template == nil || template.ExpType != enum.ExperimentTypeMixed {
		return nil
	}

	sources := make([]json.RawMessage, 0)
	for _, scene := range template.SimScenes {
		var cfg simSceneConfig
		if len(scene.Config) > 0 {
			_ = json.Unmarshal(scene.Config, &cfg)
		}
		if cfg.DataSourceMode != enum.DataSourceModeReal && cfg.DataSourceMode != enum.DataSourceModeDual {
			continue
		}
		if len(scene.DataSourceConfig) == 0 {
			continue
		}
		sources = append(sources, scene.DataSourceConfig)
	}
	if len(sources) == 0 {
		return nil
	}

	payload, _ := json.Marshal(map[string]any{
		"sources": sources,
	})
	return payload
}

// buildTemplateK8sConfig 生成模板的 K8s 编排预览配置。
func buildTemplateK8sConfig(template *entity.ExperimentTemplate) json.RawMessage {
	if template == nil {
		return nil
	}
	if template.ExpType != enum.ExperimentTypeReal && template.ExpType != enum.ExperimentTypeMixed {
		return nil
	}

	containers := make([]map[string]any, 0, len(template.Containers))
	for _, container := range template.Containers {
		item := map[string]any{
			"container_name":   container.ContainerName,
			"image_version_id": strconv.FormatInt(container.ImageVersionID, 10),
			"is_primary":       container.IsPrimary,
			"startup_order":    container.StartupOrder,
		}
		if container.CPULimit != nil {
			item["cpu_limit"] = *container.CPULimit
		}
		if container.MemoryLimit != nil {
			item["memory_limit"] = *container.MemoryLimit
		}
		if len(container.Ports) > 0 {
			item["ports"] = json.RawMessage(container.Ports)
		}
		if len(container.Volumes) > 0 {
			item["volumes"] = json.RawMessage(container.Volumes)
		}
		containers = append(containers, item)
	}

	payload, _ := json.Marshal(map[string]any{
		"experiment_type": template.ExpType,
		"topology_mode":   template.TopologyMode,
		"containers":      containers,
	})
	return payload
}

// normalizeContainerEnvPrefix 将容器名转换为环境变量前缀。
func normalizeContainerEnvPrefix(name string) string {
	upper := strings.ToUpper(strings.TrimSpace(name))
	upper = nonEnvCharPattern.ReplaceAllString(upper, "_")
	upper = strings.Trim(upper, "_")
	return upper
}

// parseServiceDiscoveryPorts 解析容器端口配置。
func parseServiceDiscoveryPorts(raw json.RawMessage) []serviceDiscoveryPort {
	if len(raw) == 0 {
		return nil
	}

	var ports []serviceDiscoveryPort
	if err := json.Unmarshal(raw, &ports); err == nil {
		return ports
	}
	return nil
}

// mergeDefaultEnvVars 合并镜像默认环境变量。
func mergeDefaultEnvVars(target map[string]string, image *entity.Image) {
	if image == nil || len(image.DefaultEnvVars) == 0 {
		return
	}
	var items []map[string]any
	if err := json.Unmarshal(image.DefaultEnvVars, &items); err != nil {
		return
	}
	for _, item := range items {
		key, _ := item["key"].(string)
		value, _ := item["value"].(string)
		if key == "" {
			continue
		}
		target[key] = value
	}
}

// mergeConditionalEnvVars 根据已存在容器触发条件环境变量。
func mergeConditionalEnvVars(target map[string]string, raw json.RawMessage, allContainers []entity.TemplateContainer) error {
	if len(raw) == 0 {
		return nil
	}

	var rules []conditionalEnvRule
	if err := json.Unmarshal(raw, &rules); err != nil {
		return nil
	}
	containerSet := make(map[string]struct{}, len(allContainers))
	for _, container := range allContainers {
		containerSet[container.ContainerName] = struct{}{}
	}

	for _, rule := range rules {
		if len(rule.Conditions) == 0 {
			continue
		}
		if rule.Key != "" && rule.DefaultValue != "" {
			target[rule.Key] = rule.DefaultValue
		}
		for _, condition := range rule.Conditions {
			if !strings.HasPrefix(condition.When, "container_exists:") {
				continue
			}
			containerName := strings.TrimPrefix(condition.When, "container_exists:")
			if _, ok := containerSet[containerName]; !ok {
				continue
			}
			if rule.Key != "" && condition.Value != "" {
				target[rule.Key] = condition.Value
			}
			for _, inject := range condition.InjectVars {
				key := inject["key"]
				value := inject["value"]
				if key == "" {
					continue
				}
				target[key] = value
			}
		}
	}
	return nil
}

// parseExplicitEnvVars 解析模板显式配置的环境变量。
func parseExplicitEnvVars(raw json.RawMessage) map[string]string {
	if len(raw) == 0 {
		return nil
	}

	result := make(map[string]string)
	var object map[string]string
	if err := json.Unmarshal(raw, &object); err == nil {
		return object
	}

	var items []map[string]any
	if err := json.Unmarshal(raw, &items); err != nil {
		return result
	}
	for _, item := range items {
		key, _ := item["key"].(string)
		value, ok := item["value"].(string)
		if key == "" || !ok {
			continue
		}
		result[key] = value
	}
	return result
}

// mergePorts 合并镜像默认端口和模板显式端口。
func mergePorts(image *entity.Image, explicit json.RawMessage) []PortSpec {
	ports := make([]PortSpec, 0)
	seen := make(map[string]struct{})

	appendPorts := func(raw json.RawMessage) {
		for _, port := range parseServiceDiscoveryPorts(raw) {
			containerPort := port.ContainerPort
			if containerPort <= 0 {
				containerPort = port.Port
			}
			if containerPort <= 0 {
				continue
			}
			servicePort := port.ServicePort
			if servicePort <= 0 {
				if port.Port > 0 {
					servicePort = port.Port
				} else {
					servicePort = containerPort
				}
			}
			key := fmt.Sprintf("%d/%s/%d", containerPort, strings.ToLower(port.Protocol), servicePort)
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			protocol := port.Protocol
			if protocol == "" {
				protocol = "tcp"
			}
			ports = append(ports, PortSpec{
				ContainerPort: containerPort,
				ServicePort:   servicePort,
				Protocol:      protocol,
			})
		}
	}

	if image != nil {
		appendPorts(image.DefaultPorts)
	}
	appendPorts(explicit)
	sort.SliceStable(ports, func(i, j int) bool {
		return ports[i].ContainerPort < ports[j].ContainerPort
	})
	return ports
}

// mergeVolumes 合并镜像默认卷和模板显式卷。
func mergeVolumes(image *entity.Image, explicit json.RawMessage) []VolumeSpec {
	volumes := make([]VolumeSpec, 0)
	seen := make(map[string]struct{})

	appendVolume := func(spec VolumeSpec) {
		if spec.Name == "" {
			spec.Name = normalizeVolumeName(spec.MountPath)
		}
		if spec.Name == "" || spec.MountPath == "" {
			return
		}
		key := spec.Name + ":" + spec.MountPath
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		volumes = append(volumes, spec)
	}

	if image != nil && len(image.DefaultVolumes) > 0 {
		var defaults []map[string]any
		if err := json.Unmarshal(image.DefaultVolumes, &defaults); err == nil {
			for _, item := range defaults {
				path, _ := item["path"].(string)
				appendVolume(VolumeSpec{MountPath: path})
			}
		}
	}
	if len(explicit) > 0 {
		var items []VolumeSpec
		if err := json.Unmarshal(explicit, &items); err == nil {
			for _, item := range items {
				appendVolume(item)
			}
		}
	}

	return volumes
}

// parseRequiredDependencies 解析镜像必需依赖容器名称集合。
func parseRequiredDependencies(raw json.RawMessage) []string {
	if len(raw) == 0 {
		return nil
	}

	var names []string
	if err := json.Unmarshal(raw, &names); err == nil {
		return names
	}

	var objects []map[string]any
	if err := json.Unmarshal(raw, &objects); err != nil {
		return nil
	}

	result := make([]string, 0, len(objects))
	for _, item := range objects {
		required, _ := item["required"].(bool)
		if !required && item["required"] != nil {
			continue
		}
		name, _ := item["name"].(string)
		if name == "" {
			name, _ = item["container_name"].(string)
		}
		if name == "" {
			continue
		}
		result = append(result, name)
	}
	return result
}

// normalizeVolumeName 根据挂载路径生成稳定卷名。
func normalizeVolumeName(path string) string {
	name := strings.Trim(path, "/")
	name = strings.ReplaceAll(name, "/", "-")
	name = strings.ReplaceAll(name, ".", "-")
	name = strings.Trim(name, "-")
	if name == "" {
		return ""
	}
	return name
}
