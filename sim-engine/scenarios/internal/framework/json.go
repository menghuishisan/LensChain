package framework

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
)

// DecodeMap 将 JSON 字节解码为通用对象映射。
func DecodeMap(raw []byte) (map[string]any, error) {
	if len(raw) == 0 {
		return map[string]any{}, nil
	}
	var value map[string]any
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, fmt.Errorf("解码 JSON 映射失败: %w", err)
	}
	return value, nil
}

// Encode 将任意可序列化对象编码为 JSON 字节。
func Encode(value any) ([]byte, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("编码 JSON 失败: %w", err)
	}
	return data, nil
}

// CloneMap 深复制对象映射，避免共享底层引用。
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

// MergeMap 递归合并对象映射，供联动共享状态和初始化补丁使用。
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

// StringValue 读取字符串值，并在空白文本或类型不匹配时回退默认值。
func StringValue(value any, fallback string) string {
	text, ok := value.(string)
	if !ok || strings.TrimSpace(text) == "" {
		return fallback
	}
	return text
}

// NumberValue 读取数值字段，兼容 JSON 解码后的常见整数和浮点类型。
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
	default:
		return fallback
	}
}

// BoolValue 读取布尔字段，并在缺失或类型不匹配时使用默认值。
func BoolValue(value any, fallback bool) bool {
	flag, ok := value.(bool)
	if !ok {
		return fallback
	}
	return flag
}

// BoolText 将布尔值格式化为统一中文文案。
func BoolText(flag bool) string {
	if flag {
		return "是"
	}
	return "否"
}

// BoolLabel 将布尔值格式化为自定义文案。
func BoolLabel(flag bool, trueText string, falseText string) string {
	if flag {
		return trueText
	}
	return falseText
}

// ToStringSlice 将通用 JSON 列表恢复为字符串切片。
func ToStringSlice(value any) []string {
	return ToStringSliceOr(value, []string{})
}

// ToStringSliceOr 将通用 JSON 列表恢复为字符串切片，并支持自定义默认值。
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
		text, ok := item.(string)
		if ok {
			result = append(result, text)
		}
	}
	if len(result) == 0 {
		return append([]string(nil), fallback...)
	}
	return result
}

// ToBoolSliceOr 将通用 JSON 列表恢复为布尔切片，并支持自定义默认值。
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
		flag, ok := item.(bool)
		if ok {
			result = append(result, flag)
		}
	}
	if len(result) == 0 {
		return append([]bool(nil), fallback...)
	}
	return result
}

// ToIntSlice 将通用 JSON 列表恢复为整数切片。
func ToIntSlice(value any) []int {
	return ToIntSliceOr(value, []int{})
}

// ToIntSliceOr 将通用 JSON 列表恢复为整数切片，并支持自定义默认值。
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
		switch typed := item.(type) {
		case float64:
			result = append(result, int(typed))
		case int:
			result = append(result, typed)
		case int8:
			result = append(result, int(typed))
		case int16:
			result = append(result, int(typed))
		case int32:
			result = append(result, int(typed))
		case int64:
			result = append(result, int(typed))
		case uint:
			result = append(result, int(typed))
		case uint8:
			result = append(result, int(typed))
		case uint16:
			result = append(result, int(typed))
		case uint32:
			result = append(result, int(typed))
		case uint64:
			result = append(result, int(typed))
		}
	}
	if len(result) == 0 {
		return append([]int(nil), fallback...)
	}
	return result
}

// ToIntMapOr 将通用 JSON 对象恢复为整数映射，并支持自定义默认值。
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

// HashText 计算输入文本的 SHA-256 十六进制摘要。
func HashText(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

// Abbreviate 按指定长度缩短长文本。
func Abbreviate(value string, limit int) string {
	return AbbreviateOr(value, limit, "")
}

// AbbreviateOr 按指定长度缩短长文本，并在空字符串时回退默认文案。
func AbbreviateOr(value string, limit int, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	if limit <= 0 || len(value) <= limit {
		return value
	}
	return value[:limit] + "..."
}

// NormalizeDashedID 将带固定前缀的标识规整为 `prefix-suffix` 形式。
func NormalizeDashedID(prefix string, value string, fallback string) string {
	prefix = strings.TrimSpace(strings.ToLower(prefix))
	if prefix == "" {
		return fallback
	}
	normalized := strings.TrimSpace(strings.ToLower(value))
	if normalized == "" {
		return fallback
	}
	normalized = strings.ReplaceAll(normalized, " ", "-")
	normalized = strings.ReplaceAll(normalized, "_", "-")
	normalized = strings.ReplaceAll(normalized, prefix+"-", prefix+"-")
	normalized = strings.ReplaceAll(normalized, prefix+"_", prefix+"-")
	if strings.HasPrefix(normalized, prefix) && !strings.HasPrefix(normalized, prefix+"-") {
		normalized = prefix + "-" + strings.TrimPrefix(normalized, prefix)
	}
	for strings.Contains(normalized, "--") {
		normalized = strings.ReplaceAll(normalized, "--", "-")
	}
	if strings.HasPrefix(normalized, prefix+"-") && len(normalized) > len(prefix)+1 {
		return normalized
	}
	return fallback
}

// NormalizeSlug 将展示标签规整为小写短横线标识。
func NormalizeSlug(value string, fallback string) string {
	normalized := strings.TrimSpace(strings.ToLower(value))
	if normalized == "" {
		return fallback
	}
	normalized = strings.ReplaceAll(normalized, " ", "-")
	normalized = strings.ReplaceAll(normalized, "_", "-")
	for strings.Contains(normalized, "--") {
		normalized = strings.ReplaceAll(normalized, "--", "-")
	}
	if normalized == "" {
		return fallback
	}
	return normalized
}

// cloneIntMap 复制整数映射，避免默认值被运行态修改。
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
