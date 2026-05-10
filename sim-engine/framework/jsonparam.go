// 模块：sim-engine/framework
// 文件职责：协议字段解码工具集 — 把 map[string]any 中的协议字段安全取值为 Go 强类型。
//
// 职责约束：
//   - 仅服务于"从 SharedState / Params / 协议消息 map 中按字段名取协议字段值"的解码场景；
//   - 不做业务展示（格式化文本、缩短 ID、本地化）；
//   - 不假定具体场景的业务模型（如不算"进度"、不构造 ID 规则）；
//   - 仅依赖 Go 标准库 encoding/json。

package framework

import "encoding/json"

// MapStr 从 map[string]any 中按 key 安全取字符串；缺失或类型不匹配时回退默认值。
//
// 本函数是 43 个场景共用的 map 取值快捷方式，避免每个场景重复定义 strOr。
func MapStr(m map[string]any, key, fallback string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return fallback
}

// MapInt 从 map[string]any 中按 key 安全取整数；兼容 JSON 解码后的 float64 / int / int64。
//
// 本函数是 43 个场景共用的 map 取值快捷方式，避免每个场景重复定义 intOr。
func MapInt(m map[string]any, key string, fallback int) int {
	if m == nil {
		return fallback
	}
	if v, ok := m[key]; ok {
		switch t := v.(type) {
		case float64:
			return int(t)
		case int:
			return t
		case int64:
			return int(t)
		}
	}
	return fallback
}

// MapBool 从 map[string]any 中按 key 安全取布尔；缺失或类型不匹配时回退默认值。
//
// 本函数是 43 个场景共用的 map 取值快捷方式，避免每个场景重复定义 boolOr。
func MapBool(m map[string]any, key string, fallback bool) bool {
	if v, ok := m[key].(bool); ok {
		return v
	}
	return fallback
}

// StringValue 从 any 中安全取字符串；类型不匹配或空白时回退默认值。
func StringValue(value any, fallback string) string {
	text, ok := value.(string)
	if !ok {
		return fallback
	}
	if isBlank(text) {
		return fallback
	}
	return text
}

// NumberValue 从 any 中安全取浮点数；兼容 JSON 解码后的常见整数与浮点类型。
func NumberValue(value any, fallback float64) float64 {
	switch typed := value.(type) {
	case float64:
		return typed
	case float32:
		return float64(typed)
	case int:
		return float64(typed)
	case int8:
		return float64(typed)
	case int16:
		return float64(typed)
	case int32:
		return float64(typed)
	case int64:
		return float64(typed)
	case uint:
		return float64(typed)
	case uint8:
		return float64(typed)
	case uint16:
		return float64(typed)
	case uint32:
		return float64(typed)
	case uint64:
		return float64(typed)
	}
	return fallback
}

// BoolValue 从 any 中安全取布尔；类型不匹配时回退默认值。
func BoolValue(value any, fallback bool) bool {
	flag, ok := value.(bool)
	if !ok {
		return fallback
	}
	return flag
}

// ToStringSliceOr 把 any 解码为字符串切片；类型不匹配或为空时回退默认值。
//
// 兼容输入：[]string / []any（含 string 元素）。
func ToStringSliceOr(value any, fallback []string) []string {
	if typed, ok := value.([]string); ok {
		return append([]string(nil), typed...)
	}
	raw, ok := value.([]any)
	if !ok {
		return append([]string(nil), fallback...)
	}
	result := make([]string, 0, len(raw))
	for _, item := range raw {
		if text, ok := item.(string); ok {
			result = append(result, text)
		}
	}
	if len(result) == 0 {
		return append([]string(nil), fallback...)
	}
	return result
}

// ToBoolSliceOr 把 any 解码为布尔切片；类型不匹配或为空时回退默认值。
func ToBoolSliceOr(value any, fallback []bool) []bool {
	if typed, ok := value.([]bool); ok {
		return append([]bool(nil), typed...)
	}
	raw, ok := value.([]any)
	if !ok {
		return append([]bool(nil), fallback...)
	}
	result := make([]bool, 0, len(raw))
	for _, item := range raw {
		if flag, ok := item.(bool); ok {
			result = append(result, flag)
		}
	}
	if len(result) == 0 {
		return append([]bool(nil), fallback...)
	}
	return result
}

// ToIntSliceOr 把 any 解码为整数切片；兼容 JSON 解码后的常见数值类型。
func ToIntSliceOr(value any, fallback []int) []int {
	if typed, ok := value.([]int); ok {
		return append([]int(nil), typed...)
	}
	raw, ok := value.([]any)
	if !ok {
		return append([]int(nil), fallback...)
	}
	result := make([]int, 0, len(raw))
	for _, item := range raw {
		result = append(result, int(NumberValue(item, 0)))
	}
	if len(result) == 0 {
		return append([]int(nil), fallback...)
	}
	return result
}

// ToIntMapOr 把 any 解码为字符串到整数的映射；类型不匹配或为空时回退默认值。
func ToIntMapOr(value any, fallback map[string]int) map[string]int {
	if typed, ok := value.(map[string]int); ok {
		return cloneIntMap(typed)
	}
	entry, ok := value.(map[string]any)
	if !ok {
		return cloneIntMap(fallback)
	}
	result := make(map[string]int, len(entry))
	for key, raw := range entry {
		result[key] = int(NumberValue(raw, 0))
	}
	if len(result) == 0 {
		return cloneIntMap(fallback)
	}
	return result
}

// CloneMap 深复制 map[string]any（通过 JSON 往返），避免与原 map 共享底层引用。
//
// 用于在 Init / Step / HandleAction 之间安全传递参数与共享状态视图。
func CloneMap(value map[string]any) map[string]any {
	if len(value) == 0 {
		return map[string]any{}
	}
	data, err := json.Marshal(value)
	if err != nil {
		return map[string]any{}
	}
	var copied map[string]any
	if err := json.Unmarshal(data, &copied); err != nil {
		return map[string]any{}
	}
	return copied
}

// MergeMap 把 patch 递归合并到 base 上；对象字段递归合并，非对象字段直接覆盖。
//
// 不修改入参；返回 base 的深拷贝合并结果。供联动共享状态与初始化参数使用。
func MergeMap(base map[string]any, patch map[string]any) map[string]any {
	result := CloneMap(base)
	for key, patchValue := range patch {
		baseChild, baseIsMap := result[key].(map[string]any)
		patchChild, patchIsMap := patchValue.(map[string]any)
		if baseIsMap && patchIsMap {
			result[key] = MergeMap(baseChild, patchChild)
			continue
		}
		result[key] = patchValue
	}
	return result
}

// cloneIntMap 复制整数映射；ToIntMapOr 内部使用。
func cloneIntMap(value map[string]int) map[string]int {
	if len(value) == 0 {
		return map[string]int{}
	}
	copied := make(map[string]int, len(value))
	for key, item := range value {
		copied[key] = item
	}
	return copied
}

// isBlank 判断字符串是否为空或全空白；不依赖 strings 包以避免 framework 依赖膨胀。
func isBlank(text string) bool {
	for i := 0; i < len(text); i++ {
		c := text[i]
		if c != ' ' && c != '\t' && c != '\n' && c != '\r' {
			return false
		}
	}
	return true
}
