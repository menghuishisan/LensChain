// Package collectoragent 提供混合实验 Collector sidecar 的配置解析能力。
package collectoragent

import (
	"encoding/json"
	"fmt"
	"net/url"
	"slices"
	"strings"
)

// Config 表示 Collector Agent 的运行配置。
type Config struct {
	TargetContainer  string
	Ecosystem        string
	SessionID        string
	CoreWebSocketURL string
	PollIntervalMS   int
	CollectEvents    []string
	RPCEndpoint      string
	WSEndpoint       string
}

// sourceConfigEnvelope 表示包含多条采集源配置的 sidecar 聚合配置。
type sourceConfigEnvelope struct {
	Sources []json.RawMessage `json:"sources"`
}

// sourceConfig 表示单条采集源配置。
type sourceConfig struct {
	ContainerName   string               `json:"container_name"`
	TargetContainer string               `json:"target_container"`
	CollectorAgent  collectorAgentConfig `json:"collector_agent"`
}

// collectorAgentConfig 表示单个采集源对应的 Collector 适配器配置。
type collectorAgentConfig struct {
	Adapter        string   `json:"adapter"`
	RPCPort        int      `json:"rpc_port"`
	WSPort         int      `json:"ws_port"`
	CollectEvents  []string `json:"collect_events"`
	PollIntervalMS int      `json:"poll_interval_ms"`
}

// ParseConfig 从环境变量与数据源配置中构造 Collector Agent 配置。
func ParseConfig(targetContainer string, ecosystem string, sessionID string, coreWebSocketURL string, rawConfig string) (*Config, error) {
	targetContainer = strings.TrimSpace(targetContainer)
	ecosystem = strings.TrimSpace(ecosystem)
	sessionID = strings.TrimSpace(sessionID)
	coreWebSocketURL = strings.TrimSpace(coreWebSocketURL)
	if targetContainer == "" {
		return nil, fmt.Errorf("collector target container is required")
	}
	if ecosystem == "" {
		return nil, fmt.Errorf("collector ecosystem is required")
	}
	if sessionID == "" {
		return nil, fmt.Errorf("collector session id is required")
	}
	if coreWebSocketURL == "" {
		return nil, fmt.Errorf("collector websocket url is required")
	}

	sources, err := parseSourceConfigs(rawConfig)
	if err != nil {
		return nil, err
	}
	selected, err := selectSourceConfig(sources, targetContainer, ecosystem)
	if err != nil {
		return nil, err
	}

	pollInterval := selected.CollectorAgent.PollIntervalMS
	if pollInterval <= 0 {
		pollInterval = 1000
	}

	cfg := &Config{
		TargetContainer:  targetContainer,
		Ecosystem:        ecosystem,
		SessionID:        sessionID,
		CoreWebSocketURL: coreWebSocketURL,
		PollIntervalMS:   pollInterval,
		CollectEvents:    normalizeCollectEvents(selected.CollectorAgent.CollectEvents),
	}
	if selected.CollectorAgent.RPCPort > 0 {
		cfg.RPCEndpoint = fmt.Sprintf("http://%s:%d", targetContainer, selected.CollectorAgent.RPCPort)
	}
	if selected.CollectorAgent.WSPort > 0 {
		cfg.WSEndpoint = fmt.Sprintf("ws://%s:%d", targetContainer, selected.CollectorAgent.WSPort)
	}
	if _, err := url.Parse(cfg.CoreWebSocketURL); err != nil {
		return nil, fmt.Errorf("invalid collector websocket url: %w", err)
	}
	return cfg, nil
}

// parseSourceConfigs 将单条或聚合后的数据源配置统一解析为采集源列表。
func parseSourceConfigs(rawConfig string) ([]sourceConfig, error) {
	if strings.TrimSpace(rawConfig) == "" {
		return nil, fmt.Errorf("collector config is required")
	}

	var envelope sourceConfigEnvelope
	if err := json.Unmarshal([]byte(rawConfig), &envelope); err == nil && len(envelope.Sources) > 0 {
		sources := make([]sourceConfig, 0, len(envelope.Sources))
		for _, raw := range envelope.Sources {
			var item sourceConfig
			if err := json.Unmarshal(raw, &item); err != nil {
				return nil, fmt.Errorf("parse collector source config: %w", err)
			}
			sources = append(sources, item)
		}
		return sources, nil
	}

	var single sourceConfig
	if err := json.Unmarshal([]byte(rawConfig), &single); err != nil {
		return nil, fmt.Errorf("parse collector config: %w", err)
	}
	return []sourceConfig{single}, nil
}

// selectSourceConfig 选择与目标容器和生态编码完全匹配的采集源配置。
func selectSourceConfig(sources []sourceConfig, targetContainer string, ecosystem string) (*sourceConfig, error) {
	for i := range sources {
		source := &sources[i]
		containerName := strings.TrimSpace(source.TargetContainer)
		if containerName == "" {
			containerName = strings.TrimSpace(source.ContainerName)
		}
		if containerName != targetContainer {
			continue
		}
		if strings.TrimSpace(source.CollectorAgent.Adapter) != ecosystem {
			continue
		}
		return source, nil
	}
	return nil, fmt.Errorf("collector source config not found for container %s and ecosystem %s", targetContainer, ecosystem)
}

// normalizeCollectEvents 归一化采集事件列表，去除空值与重复项。
func normalizeCollectEvents(events []string) []string {
	if len(events) == 0 {
		return nil
	}
	normalized := make([]string, 0, len(events))
	seen := make(map[string]struct{}, len(events))
	for _, item := range events {
		value := strings.ToLower(strings.TrimSpace(item))
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		normalized = append(normalized, value)
	}
	slices.Sort(normalized)
	return normalized
}
