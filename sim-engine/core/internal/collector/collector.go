package collector

import (
	"encoding/json"
	"errors"
	"strings"
)

// RenderPatch 是注入 SimEngine 渲染状态的采集数据补丁。
type RenderPatch struct {
	Source      string
	DataType    string
	TimestampMS int64
	PayloadJSON []byte
	PatchJSON   []byte
}

// Normalize 将 Collector Agent 事件转换为渲染状态补丁。
func Normalize(event Event) (RenderPatch, error) {
	if strings.TrimSpace(event.Source) == "" {
		return RenderPatch{}, errors.New("collector source is required")
	}
	if strings.TrimSpace(event.DataType) == "" {
		return RenderPatch{}, errors.New("collector data_type is required")
	}

	patchJSON, err := json.Marshal(map[string]any{
		"collection": map[string]any{
			event.Source: map[string]any{
				event.DataType: map[string]any{
					"timestamp_ms": event.TimestampMS,
					"payload":      json.RawMessage(event.PayloadJSON),
				},
			},
		},
	})
	if err != nil {
		return RenderPatch{}, err
	}

	return RenderPatch{
		Source:      event.Source,
		DataType:    event.DataType,
		TimestampMS: event.TimestampMS,
		PayloadJSON: cloneBytes(event.PayloadJSON),
		PatchJSON:   patchJSON,
	}, nil
}
