// template_rules.go
// 模块04 — 实验环境：模板规则与运行配置组装辅助
// 负责在 service 层统一承载模板校验汇总、容器配置合并、服务发现变量生成等业务规则
// 避免上层 service 重复实现规则拼装，同时不下沉到 repository 或基础能力层

package experiment

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"k8s.io/apimachinery/pkg/api/resource"

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

// toolEcosystemRule 定义专属工具与目标生态的映射关系。
type toolEcosystemRule struct {
	ImageName           string
	ExpectedEcosystem   string
	ExpectedDisplayText string
}

var nonEnvCharPattern = regexp.MustCompile(`[^A-Z0-9]+`)
var serviceRefPattern = regexp.MustCompile(`\$\{([A-Z0-9_]+)}`)

var specializedToolRules = []toolEcosystemRule{
	{ImageName: "remix-ide", ExpectedEcosystem: enum.EcosystemEthereum, ExpectedDisplayText: "EVM"},
	{ImageName: "blockscout", ExpectedEcosystem: enum.EcosystemEthereum, ExpectedDisplayText: "EVM"},
	{ImageName: "slither", ExpectedEcosystem: enum.EcosystemEthereum, ExpectedDisplayText: "EVM"},
	{ImageName: "mythril", ExpectedEcosystem: enum.EcosystemEthereum, ExpectedDisplayText: "EVM"},
	{ImageName: "echidna", ExpectedEcosystem: enum.EcosystemEthereum, ExpectedDisplayText: "EVM"},
	{ImageName: "fabric-explorer", ExpectedEcosystem: enum.EcosystemFabric, ExpectedDisplayText: "Fabric"},
	{ImageName: "fabric-tools", ExpectedEcosystem: enum.EcosystemFabric, ExpectedDisplayText: "Fabric"},
	{ImageName: "webase-front", ExpectedEcosystem: enum.EcosystemFisco, ExpectedDisplayText: "FISCO BCOS"},
	{ImageName: "webase-web", ExpectedEcosystem: enum.EcosystemFisco, ExpectedDisplayText: "FISCO BCOS"},
	{ImageName: "chainmaker-explorer", ExpectedEcosystem: enum.EcosystemChainmaker, ExpectedDisplayText: "ChainMaker"},
}

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

// buildConditionalEnvVarExamples 从镜像环境变量定义中提取条件变量示例。
// 仅返回带条件分支的变量，供镜像配置模板接口按文档展示。
func buildConditionalEnvVarExamples(raw json.RawMessage) []dto.ConditionalEnvVarExample {
	if len(raw) == 0 {
		return []dto.ConditionalEnvVarExample{}
	}

	var items []dto.ImageEnvVarItem
	if err := json.Unmarshal(raw, &items); err != nil {
		return []dto.ConditionalEnvVarExample{}
	}

	result := make([]dto.ConditionalEnvVarExample, 0, len(items))
	for _, item := range items {
		if len(item.Conditions) == 0 {
			continue
		}
		description := "条件环境变量示例"
		if item.Desc != nil && strings.TrimSpace(*item.Desc) != "" {
			description = strings.TrimSpace(*item.Desc)
		}
		result = append(result, dto.ConditionalEnvVarExample{
			Key:          item.Key,
			DefaultValue: item.Value,
			Conditions:   item.Conditions,
			Description:  description,
		})
	}
	return result
}

// buildMissingDependencyIssue 构建缺失依赖的结构化验证问题。
func buildMissingDependencyIssue(containerName, dependency string) dto.ValidationIssue {
	sourceContainer := containerName
	missingDependency := dependency
	return dto.ValidationIssue{
		Code:              "L1_MISSING_DEPENDENCY",
		Message:           fmt.Sprintf("依赖完整性检查未通过 — %s 需要 %s 作为依赖", containerName, dependency),
		SourceContainer:   &sourceContainer,
		MissingDependency: &missingDependency,
		Suggestion: &dto.ValidationSuggestion{
			Action: "add_container",
			Image:  dependency,
			Reason: fmt.Sprintf("%s 缺少必要依赖 %s，请补充对应容器配置", containerName, dependency),
		},
	}
}

// decodeSimSceneConfig 解析模板场景配置，统一恢复 scene_params、initial_state 与 data_source_mode。
func decodeSimSceneConfig(raw json.RawMessage) simSceneConfig {
	var cfg simSceneConfig
	if len(raw) == 0 {
		return cfg
	}
	_ = json.Unmarshal(raw, &cfg)
	return cfg
}

// buildResourceReasonablenessIssues 构建 L3 资源合理性问题。
func buildResourceReasonablenessIssues(
	template *entity.ExperimentTemplate,
	containers []*entity.TemplateContainer,
	versions map[int64]*entity.ImageVersion,
	images map[int64]*entity.Image,
) []dto.ValidationIssue {
	issues := make([]dto.ValidationIssue, 0)
	totalCPUMilli := int64(0)
	quotaCPUMilli, hasQuotaCPU := parseCPUQuantity(trimmedStringPtr(template.CPULimit))

	for _, container := range containers {
		if container == nil {
			continue
		}
		version := versions[container.ImageVersionID]
		if version == nil {
			continue
		}
		image := images[version.ImageID]

		effectiveCPU := firstNonEmptyString(container.CPULimit, version.MinCPU)
		if milli, ok := parseCPUQuantity(effectiveCPU); ok {
			totalCPUMilli += milli
		}

		effectiveMemory := firstNonEmptyString(container.MemoryLimit, version.MinMemory)
		if minCPU := trimmedStringPtr(version.MinCPU); minCPU != "" && trimmedStringPtr(container.CPULimit) != "" {
			if currentMilli, ok := parseCPUQuantity(trimmedStringPtr(container.CPULimit)); ok {
				if minMilli, minOK := parseCPUQuantity(minCPU); minOK && currentMilli < minMilli {
					issues = append(issues, dto.ValidationIssue{
						Code:    "L3_IMAGE_RESOURCE_BELOW_MINIMUM",
						Message: fmt.Sprintf("%s 配置的资源低于最低要求（最低 %s CPU / %s 内存）", resolveImageDisplayName(image), minCPU, trimmedStringPtr(version.MinMemory)),
					})
				}
			}
		}
		if minMemory := trimmedStringPtr(version.MinMemory); minMemory != "" && trimmedStringPtr(container.MemoryLimit) != "" {
			if currentBytes, ok := parseMemoryQuantity(trimmedStringPtr(container.MemoryLimit)); ok {
				if minBytes, minOK := parseMemoryQuantity(minMemory); minOK && currentBytes < minBytes {
					issues = append(issues, dto.ValidationIssue{
						Code:    "L3_IMAGE_RESOURCE_BELOW_MINIMUM",
						Message: fmt.Sprintf("%s 配置的资源低于最低要求（最低 %s CPU / %s 内存）", resolveImageDisplayName(image), trimmedStringPtr(version.MinCPU), minMemory),
					})
				}
			}
		}

		_ = effectiveMemory
	}

	if hasQuotaCPU && quotaCPUMilli > 0 && totalCPUMilli > quotaCPUMilli {
		currentTotalCPU := formatCPUQuantity(totalCPUMilli)
		quotaLimitCPU := trimmedStringPtr(template.CPULimit)
		issues = append(issues, dto.ValidationIssue{
			Code:            "L3_TOTAL_CPU_EXCEEDS_QUOTA",
			Message:         fmt.Sprintf("当前配置总 CPU 需求 %s 核，超过单实验资源上限 %s 核，可能导致启动失败", currentTotalCPU, quotaLimitCPU),
			CurrentTotalCPU: &currentTotalCPU,
			QuotaLimitCPU:   &quotaLimitCPU,
		})
	}

	return issues
}

// buildEcosystemConsistencyIssues 构建 L4 工具生态一致性提示。
func buildEcosystemConsistencyIssues(
	containers []*entity.TemplateContainer,
	versions map[int64]*entity.ImageVersion,
	images map[int64]*entity.Image,
) []dto.ValidationIssue {
	currentSet := make(map[string]struct{})
	toolSet := make(map[string]string)

	for _, container := range containers {
		if container == nil {
			continue
		}
		version := versions[container.ImageVersionID]
		if version == nil {
			continue
		}
		image := images[version.ImageID]
		if image == nil {
			continue
		}

		imageName := strings.ToLower(strings.TrimSpace(image.Name))
		if expected, ok := specializedToolExpectedEcosystem(imageName); ok {
			toolSet[imageName] = expected
			continue
		}

		ecosystem := strings.ToLower(strings.TrimSpace(trimmedStringPtr(image.Ecosystem)))
		if ecosystem != "" && ecosystem != enum.EcosystemGeneral {
			currentSet[ecosystem] = struct{}{}
		}
	}

	currentEcosystems := make([]string, 0, len(currentSet))
	for ecosystem := range currentSet {
		currentEcosystems = append(currentEcosystems, ecosystem)
	}
	sort.Strings(currentEcosystems)

	issues := make([]dto.ValidationIssue, 0)
	for toolName, expected := range toolSet {
		if _, ok := currentSet[expected]; ok {
			continue
		}
		toolImage := toolName
		expectedEcosystem := expected
		displayText := expectedToolDisplayText(toolName, expected)
		issues = append(issues, dto.ValidationIssue{
			Code:              "L4_TOOL_ECOSYSTEM_MISMATCH",
			Message:           fmt.Sprintf("%s 适用于 %s 生态，当前实验未包含 %s 链节点", toolName, displayText, displayText),
			ToolImage:         &toolImage,
			ExpectedEcosystem: &expectedEcosystem,
			CurrentEcosystems: append([]string(nil), currentEcosystems...),
		})
	}

	sort.Slice(issues, func(i, j int) bool {
		return trimmedStringPtr(issues[i].ToolImage) < trimmedStringPtr(issues[j].ToolImage)
	})
	return issues
}

// buildServiceDiscoveryEnvVars 为模板内所有容器生成统一的服务发现变量。
func buildServiceDiscoveryEnvVars(containers []entity.TemplateContainer) map[string]string {
	return buildServiceDiscoveryEnvVarsWithHostResolver(containers, func(container entity.TemplateContainer) string {
		return container.ContainerName
	})
}

// buildServiceDiscoveryEnvVarsWithHostResolver 按指定主机解析规则生成服务发现变量。
func buildServiceDiscoveryEnvVarsWithHostResolver(
	containers []entity.TemplateContainer,
	resolveHost func(container entity.TemplateContainer) string,
) map[string]string {
	envs := make(map[string]string)
	usedNames := make(map[string]int)

	for _, container := range containers {
		baseKey := normalizeContainerEnvPrefix(container.ContainerName)
		if baseKey == "" {
			continue
		}
		nameIndex := usedNames[baseKey] + 1
		usedNames[baseKey] = nameIndex
		envKeyPrefix := baseKey
		if nameIndex > 1 {
			envKeyPrefix = fmt.Sprintf("%s_%d", baseKey, nameIndex)
		}
		host := container.ContainerName
		if resolveHost != nil {
			host = resolveHost(container)
		}
		envs[envKeyPrefix+"_HOST"] = host

		for _, port := range parseServiceDiscoveryPorts(json.RawMessage(container.Ports)) {
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
			envs[fmt.Sprintf("%s_PORT_%d", envKeyPrefix, servicePort)] = strconv.Itoa(servicePort)
		}
	}

	return envs
}

// mergeContainerConfigWithServiceDiscovery 合并镜像默认配置、条件变量和运行期服务发现变量。
func mergeContainerConfigWithServiceDiscovery(
	container entity.TemplateContainer,
	image *entity.Image,
	version *entity.ImageVersion,
	allContainers []entity.TemplateContainer,
	serviceDiscoveryEnv map[string]string,
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
	for key, value := range serviceDiscoveryEnv {
		envVars[key] = value
	}
	mergeDefaultEnvVars(envVars, image)
	if err := mergeConditionalEnvVars(envVars, image, allContainers); err != nil {
		return ContainerSpec{}, err
	}
	if explicitVars := parseExplicitEnvVars(json.RawMessage(container.EnvVars)); len(explicitVars) > 0 {
		for key, value := range explicitVars {
			envVars[key] = value
		}
	}
	spec.EnvVars = envVars

	spec.Ports = mergePorts(image, json.RawMessage(container.Ports))
	spec.Volumes = mergeVolumes(image, json.RawMessage(container.Volumes))
	return spec, nil
}

// calculateRemainingConcurrency 计算学校剩余可分配并发额度。
func calculateRemainingConcurrency(maxConcurrency *int, quotas []*entity.ResourceQuota, currentCourseID *int64) int {
	allocated := 0
	for _, quota := range quotas {
		if quota == nil || quota.CourseID == nil {
			continue
		}
		if currentCourseID != nil && *quota.CourseID == *currentCourseID {
			continue
		}
		allocated += quotaMaxConcurrency(quota.MaxConcurrency)
	}
	remaining := quotaMaxConcurrency(maxConcurrency) - allocated
	if remaining < 0 {
		return 0
	}
	return remaining
}

// buildMixedDataCollectionConfig 构建混合实验的数据采集配置。
func buildMixedDataCollectionConfig(template *TemplateAggregate) json.RawMessage {
	if template == nil || template.Template == nil || template.Template.ExperimentType != enum.ExperimentTypeMixed {
		return nil
	}

	sources := make([]json.RawMessage, 0)
	for _, scene := range template.SimScenes {
		cfg := decodeSimSceneConfig(json.RawMessage(scene.Config))
		if cfg.DataSourceMode != enum.DataSourceModeCollect && cfg.DataSourceMode != enum.DataSourceModeDual {
			continue
		}
		if len(scene.DataSourceConfig) == 0 {
			continue
		}
		sources = append(sources, json.RawMessage(scene.DataSourceConfig))
	}
	if len(sources) == 0 {
		return nil
	}

	payload, _ := json.Marshal(map[string]any{
		"sources": sources,
	})
	return payload
}

// parseCPUQuantity 解析 CPU 字符串为毫核数。
func parseCPUQuantity(value string) (int64, bool) {
	value = strings.TrimSpace(value)
	if value == "" || value == "0" {
		return 0, false
	}
	quantity, err := resource.ParseQuantity(value)
	if err != nil {
		return 0, false
	}
	return quantity.MilliValue(), true
}

// parseMemoryQuantity 解析内存字符串为字节数。
func parseMemoryQuantity(value string) (int64, bool) {
	value = strings.TrimSpace(value)
	if value == "" || value == "0" {
		return 0, false
	}
	quantity, err := resource.ParseQuantity(value)
	if err != nil {
		return 0, false
	}
	return quantity.Value(), true
}

// formatCPUQuantity 将毫核转换为文档示例使用的核数字符串。
func formatCPUQuantity(milli int64) string {
	if milli%1000 == 0 {
		return strconv.FormatInt(milli/1000, 10)
	}
	return strconv.FormatFloat(float64(milli)/1000, 'f', -1, 64)
}

// resolveImageDisplayName 返回镜像展示名称，优先使用 display_name。
func resolveImageDisplayName(image *entity.Image) string {
	if image == nil {
		return "镜像"
	}
	if strings.TrimSpace(image.DisplayName) != "" {
		return image.DisplayName
	}
	if strings.TrimSpace(image.Name) != "" {
		return image.Name
	}
	return "镜像"
}

// firstNonEmptyString 读取第一个非空字符串指针。
func firstNonEmptyString(values ...*string) string {
	for _, value := range values {
		if strings.TrimSpace(trimmedStringPtr(value)) != "" {
			return strings.TrimSpace(trimmedStringPtr(value))
		}
	}
	return ""
}

// trimmedStringPtr 将字符串指针安全解引用并去除首尾空白。
func trimmedStringPtr(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

// specializedToolExpectedEcosystem 返回专属工具要求的目标生态。
func specializedToolExpectedEcosystem(imageName string) (string, bool) {
	for _, rule := range specializedToolRules {
		if rule.ImageName == imageName {
			return rule.ExpectedEcosystem, true
		}
	}
	return "", false
}

// expectedToolDisplayText 返回工具提示使用的生态展示文本。
func expectedToolDisplayText(imageName, defaultText string) string {
	for _, rule := range specializedToolRules {
		if rule.ImageName == imageName && strings.TrimSpace(rule.ExpectedDisplayText) != "" {
			return rule.ExpectedDisplayText
		}
	}
	return defaultText
}

// buildTemplateK8sConfig 生成模板的 K8s 编排预览配置。
func buildTemplateK8sConfig(template *TemplateAggregate) json.RawMessage {
	if template == nil || template.Template == nil {
		return nil
	}
	if template.Template.ExperimentType != enum.ExperimentTypeReal && template.Template.ExperimentType != enum.ExperimentTypeMixed {
		return nil
	}

	containers := make([]map[string]any, 0, len(template.Containers))
	for _, container := range template.Containers {
		if container == nil {
			continue
		}
		item := map[string]any{
			"container_name":   container.ContainerName,
			"image_version_id": strconv.FormatInt(container.ImageVersionID, 10),
			"deployment_scope": container.DeploymentScope,
			"is_primary":       container.IsPrimary,
			"startup_order":    container.StartupOrder,
		}
		if container.RoleID != nil {
			item["role_id"] = strconv.FormatInt(*container.RoleID, 10)
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

	topologyMode := int16(0)
	if template.Template.TopologyMode != nil {
		topologyMode = *template.Template.TopologyMode
	}
	if topologyMode == enum.TopologyModeShared {
		payload, _ := json.Marshal(map[string]any{
			"experiment_type":           template.Template.ExperimentType,
			"topology_mode":             topologyMode,
			"shared_namespace_pattern":  "exp-shared-{template_id}-{lesson_id}",
			"student_namespace_pattern": "exp-{instance_id}",
			"containers":                containers,
		})
		return payload
	}
	payload, _ := json.Marshal(map[string]any{
		"experiment_type": template.Template.ExperimentType,
		"topology_mode":   topologyMode,
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

// mergeConditionalEnvVars 根据镜像默认环境变量中的条件分支触发附加变量。
// AC-48 明确要求条件规则来自镜像配置模板本身，而不是教师填写的容器显式环境变量。
func mergeConditionalEnvVars(target map[string]string, image *entity.Image, allContainers []entity.TemplateContainer) error {
	if image == nil || len(image.DefaultEnvVars) == 0 {
		return nil
	}

	var items []dto.ImageEnvVarItem
	if err := json.Unmarshal(image.DefaultEnvVars, &items); err != nil {
		return nil
	}
	containerSet := make(map[string]struct{}, len(allContainers))
	for _, container := range allContainers {
		containerSet[container.ContainerName] = struct{}{}
	}

	for _, item := range items {
		if len(item.Conditions) == 0 {
			continue
		}
		for _, condition := range item.Conditions {
			if !strings.HasPrefix(condition.When, "container_exists:") {
				continue
			}
			containerName := strings.TrimPrefix(condition.When, "container_exists:")
			if _, ok := containerSet[containerName]; !ok {
				continue
			}
			if item.Key != "" && condition.Value != "" {
				target[item.Key] = condition.Value
			}
			for _, inject := range condition.InjectVars {
				key := inject.Key
				value := inject.Value
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
		appendPorts(json.RawMessage(image.DefaultPorts))
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

// detectDependencyCycles 检测容器依赖图是否存在环。
func detectDependencyCycles(containers []*entity.TemplateContainer) []string {
	graph := make(map[string][]string)
	for _, container := range containers {
		if container == nil || container.ContainerName == "" {
			continue
		}
		graph[container.ContainerName] = parseDependsOn(json.RawMessage(container.DependsOn))
	}

	visited := make(map[string]int)
	stack := make([]string, 0, len(graph))
	cycles := make([]string, 0)
	seen := make(map[string]struct{})

	var dfs func(string)
	dfs = func(node string) {
		visited[node] = 1
		stack = append(stack, node)
		for _, next := range graph[node] {
			if _, ok := graph[next]; !ok {
				continue
			}
			if visited[next] == 0 {
				dfs(next)
				continue
			}
			if visited[next] != 1 {
				continue
			}
			index := -1
			for i, item := range stack {
				if item == next {
					index = i
					break
				}
			}
			if index < 0 {
				continue
			}
			cycle := append(append([]string{}, stack[index:]...), next)
			key := strings.Join(cycle, "->")
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			cycles = append(cycles, key)
		}
		stack = stack[:len(stack)-1]
		visited[node] = 2
	}

	for node := range graph {
		if visited[node] == 0 {
			dfs(node)
		}
	}
	return cycles
}

// findInvalidStartupOrders 返回依赖容器未先启动的配置问题。
func findInvalidStartupOrders(containers []*entity.TemplateContainer) []string {
	orderByName := make(map[string]int, len(containers))
	for _, container := range containers {
		if container == nil || container.ContainerName == "" {
			continue
		}
		orderByName[container.ContainerName] = container.StartupOrder
	}

	issues := make([]string, 0)
	for _, container := range containers {
		if container == nil || container.ContainerName == "" {
			continue
		}
		for _, dep := range parseDependsOn(json.RawMessage(container.DependsOn)) {
			depOrder, ok := orderByName[dep]
			if !ok {
				continue
			}
			if depOrder >= container.StartupOrder {
				issues = append(issues, fmt.Sprintf("%s 依赖 %s，但启动顺序未晚于依赖容器", container.ContainerName, dep))
			}
		}
	}
	return issues
}

// findMissingServiceReferences 返回环境变量中引用但不存在的服务发现变量。
func findMissingServiceReferences(containers []*entity.TemplateContainer) []string {
	all := make([]entity.TemplateContainer, 0, len(containers))
	for _, container := range containers {
		if container == nil {
			continue
		}
		all = append(all, *container)
	}
	existingRefs := buildServiceDiscoveryEnvVars(all)

	issues := make([]string, 0)
	for _, container := range containers {
		if container == nil || container.ContainerName == "" {
			continue
		}
		explicitVars := parseExplicitEnvVars(json.RawMessage(container.EnvVars))
		for key, value := range explicitVars {
			matches := serviceRefPattern.FindAllStringSubmatch(value, -1)
			for _, match := range matches {
				if len(match) < 2 {
					continue
				}
				ref := match[1]
				if _, ok := existingRefs[ref]; ok {
					continue
				}
				issues = append(issues, fmt.Sprintf("%s 的环境变量 %s 引用了不存在的服务 %s", container.ContainerName, key, ref))
			}
		}
	}
	return issues
}

// parseDependsOn 解析容器 depends_on 配置。
func parseDependsOn(raw json.RawMessage) []string {
	if len(raw) == 0 {
		return nil
	}
	var values []string
	if err := json.Unmarshal(raw, &values); err != nil {
		return nil
	}
	return values
}

// mergeStringMap 合并两个字符串 map，后者覆盖前者。
func mergeStringMap(base map[string]string, extra map[string]string) map[string]string {
	if len(base) == 0 && len(extra) == 0 {
		return map[string]string{}
	}
	merged := make(map[string]string, len(base)+len(extra))
	for key, value := range base {
		merged[key] = value
	}
	for key, value := range extra {
		merged[key] = value
	}
	return merged
}
