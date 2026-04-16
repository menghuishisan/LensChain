// Package app 提供 Collector 事件到仿真场景的路由与筛选逻辑。
package app

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/lenschain/sim-engine/core/internal/collector"
)

// collectionConfig 描述数据采集通道或场景的采集路由规则。
type collectionConfig struct {
	Mode           string   `json:"mode"`
	ContainerName  string   `json:"container_name"`
	Source         string   `json:"source"`
	SceneCodes     []string `json:"scene_codes"`
	CollectorAgent struct {
		Adapter        string   `json:"adapter"`
		CollectEvents  []string `json:"collect_events"`
		PollIntervalMS int64    `json:"poll_interval_ms"`
	} `json:"collector_agent"`
}

// collectionPayload 描述 Collector 事件负载中可能出现的路由字段。
type collectionPayload struct {
	ContainerName string `json:"container_name"`
	Container     string `json:"container"`
}

// resolveCollectionScenes 解析当前采集事件应该注入的目标场景。
func (e *Engine) resolveCollectionScenes(sessionID string, collectorConfigJSON []byte, event collector.Event) ([]string, error) {
	sessionConfig, err := parseCollectionConfig(collectorConfigJSON)
	if err != nil {
		return nil, err
	}
	if !matchesCollectionConfig(sessionConfig, "", event) {
		return nil, nil
	}

	e.mu.RLock()
	runtime, ok := e.runtimes[sessionID]
	e.mu.RUnlock()
	if !ok {
		return nil, errors.New("session runtime not found")
	}

	affectedScenes := make([]string, 0, len(runtime.activeSceneCodes))
	for _, sceneCode := range runtime.activeSceneCodes {
		sceneRuntime, sceneOK := e.scenes.Get(sessionID, sceneCode)
		if !sceneOK {
			continue
		}
		if !isCollectionMode(sceneRuntime.Meta.DataSourceMode) {
			continue
		}

		sceneConfig, configOK := runtime.sceneConfigs[sceneCode]
		if !configOK {
			continue
		}
		sceneMode := strings.TrimSpace(sceneConfig.DataSourceMode)
		if sceneMode == "" {
			sceneMode = sceneRuntime.Meta.DataSourceMode
		}
		if !isCollectionMode(sceneMode) {
			continue
		}
		config, configErr := parseCollectionConfig(sceneConfig.DataSourceConfigJSON)
		if configErr != nil {
			return nil, configErr
		}
		if !matchesCollectionConfig(config, sceneCode, event) {
			continue
		}
		affectedScenes = append(affectedScenes, sceneCode)
	}
	return affectedScenes, nil
}

// parseCollectionConfig 解析场景或会话的采集配置。
func parseCollectionConfig(configJSON []byte) (collectionConfig, error) {
	if len(configJSON) == 0 {
		return collectionConfig{}, nil
	}
	var config collectionConfig
	if err := json.Unmarshal(configJSON, &config); err != nil {
		return collectionConfig{}, errors.New("invalid data_source_config_json")
	}
	return config, nil
}

// matchesCollectionConfig 判断一条采集事件是否满足给定路由规则。
func matchesCollectionConfig(config collectionConfig, sceneCode string, event collector.Event) bool {
	if config.Mode != "" && !isCollectionMode(config.Mode) {
		return false
	}
	if len(config.SceneCodes) > 0 && sceneCode != "" && !containsText(config.SceneCodes, sceneCode) {
		return false
	}
	if config.Source != "" && !sameCollectionSource(config.Source, event.Source) {
		return false
	}
	if config.CollectorAgent.Adapter != "" && !sameCollectionSource(config.CollectorAgent.Adapter, event.Source) {
		return false
	}
	if len(config.CollectorAgent.CollectEvents) > 0 && !containsText(config.CollectorAgent.CollectEvents, event.DataType) {
		return false
	}

	if strings.TrimSpace(config.ContainerName) == "" {
		return true
	}
	payload, ok := parseCollectionPayload(event.PayloadJSON)
	if !ok {
		return false
	}
	containerName := strings.TrimSpace(payload.ContainerName)
	if containerName == "" {
		containerName = strings.TrimSpace(payload.Container)
	}
	return containerName == strings.TrimSpace(config.ContainerName)
}

// parseCollectionPayload 解析采集事件负载中的容器路由字段。
func parseCollectionPayload(payloadJSON []byte) (collectionPayload, bool) {
	if len(payloadJSON) == 0 {
		return collectionPayload{}, false
	}
	var payload collectionPayload
	if err := json.Unmarshal(payloadJSON, &payload); err != nil {
		return collectionPayload{}, false
	}
	return payload, true
}

// isCollectionMode 判断数据源模式是否允许由 Collector 驱动。
func isCollectionMode(mode string) bool {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "collection", "dual":
		return true
	default:
		return false
	}
}

// sameCollectionSource 判断场景配置的数据源与 Collector 事件来源是否等价。
func sameCollectionSource(expected string, actual string) bool {
	expected = normalizeCollectionSource(expected)
	actual = normalizeCollectionSource(actual)
	if expected == "" {
		return true
	}
	return actual != "" && expected == actual
}

// normalizeCollectionSource 统一不同生态适配器的等价命名。
func normalizeCollectionSource(value string) string {
	return collector.ResolveAdapterCode(value)
}

// containsText 判断字符串切片中是否包含目标值。
func containsText(values []string, target string) bool {
	target = strings.TrimSpace(target)
	for _, item := range values {
		if strings.TrimSpace(item) == target {
			return true
		}
	}
	return false
}
