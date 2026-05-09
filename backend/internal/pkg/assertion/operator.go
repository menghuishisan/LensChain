package assertion

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Operator 平台统一的状态断言算子集合。
// 与 docs/modules/05-CTF竞赛/02-数据库设计.md §challenge_assertions.operator
// 以及 docs/modules/04-实验环境/02-数据库设计.md §template_checkpoints.assertion_config 完全一致。
type Operator string

const (
	OpEq       Operator = "eq"
	OpNe       Operator = "ne"
	OpGt       Operator = "gt"
	OpGte      Operator = "gte"
	OpLt       Operator = "lt"
	OpLte      Operator = "lte"
	OpContains Operator = "contains"
)

// IsValid 判定字符串是否为支持的算子。
func IsValidOperator(op string) bool {
	switch Operator(strings.ToLower(strings.TrimSpace(op))) {
	case OpEq, OpNe, OpGt, OpGte, OpLt, OpLte, OpContains:
		return true
	}
	return false
}

// Compare 按算子比较 actual 与 expected，返回 (passed, reason)。失败时 reason 为可读中文。
//
// 严格类型规则（不做隐式转换）：
//   - eq / ne：通过 JSON 编码后字符串比较，能正确处理 number/string/bool/array/object；
//   - gt/gte/lt/lte：仅当两侧都是 JSON 数字（float64 / json.Number / int64）时才比较；
//     字符串数字（如 "42"）不视为合法数字，立即报错；
//   - contains：actual 为字符串时按子串匹配；actual 为数组时按 JSON 相等找元素；
//     其他类型立即报错。
func Compare(actual any, op string, expected any) (bool, string) {
	switch Operator(strings.ToLower(strings.TrimSpace(op))) {
	case OpEq:
		if jsonEqual(actual, expected) {
			return true, ""
		}
		return false, fmt.Sprintf("期望 %s == %s", brief(actual), brief(expected))
	case OpNe:
		if !jsonEqual(actual, expected) {
			return true, ""
		}
		return false, fmt.Sprintf("期望 %s != %s", brief(actual), brief(expected))
	case OpGt, OpGte, OpLt, OpLte:
		af, aok := numericValue(actual)
		ef, eok := numericValue(expected)
		if !aok || !eok {
			return false, fmt.Sprintf("数值比较要求两侧均为数字，实际 actual=%s expected=%s", brief(actual), brief(expected))
		}
		switch Operator(op) {
		case OpGt:
			if af > ef {
				return true, ""
			}
			return false, fmt.Sprintf("期望 %v > %v", af, ef)
		case OpGte:
			if af >= ef {
				return true, ""
			}
			return false, fmt.Sprintf("期望 %v >= %v", af, ef)
		case OpLt:
			if af < ef {
				return true, ""
			}
			return false, fmt.Sprintf("期望 %v < %v", af, ef)
		case OpLte:
			if af <= ef {
				return true, ""
			}
			return false, fmt.Sprintf("期望 %v <= %v", af, ef)
		}
		return false, "比较失败"
	case OpContains:
		return evalContains(actual, expected)
	default:
		return false, fmt.Sprintf("不支持的算子 %q", op)
	}
}

// evalContains 实现 contains 语义。
func evalContains(actual any, expected any) (bool, string) {
	switch a := actual.(type) {
	case string:
		es, ok := expected.(string)
		if !ok {
			return false, "contains 在字符串场景下要求 expected 为字符串"
		}
		if strings.Contains(a, es) {
			return true, ""
		}
		return false, fmt.Sprintf("期望字符串 %q 包含 %q", a, es)
	case []any:
		for _, item := range a {
			if jsonEqual(item, expected) {
				return true, ""
			}
		}
		return false, fmt.Sprintf("数组未包含期望值 %s", brief(expected))
	default:
		return false, fmt.Sprintf("contains 仅适用于字符串或数组，实际为 %T", actual)
	}
}

// jsonEqual 用 JSON 编码后字符串比较实现稳定的语义相等。
// 直接 reflect.DeepEqual 在 number 类型上会有歧义（int vs float64），用 JSON 序列化兜底。
func jsonEqual(left any, right any) bool {
	l, lerr := json.Marshal(left)
	r, rerr := json.Marshal(right)
	if lerr != nil || rerr != nil {
		return false
	}
	return string(l) == string(r)
}

// numericValue 把 any 转为 float64，仅接受真正的数字类型，不接受字符串数字。
// JSON 解码默认数字是 float64；调用方使用 json.Decoder.UseNumber() 时会得到 json.Number。
func numericValue(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int32:
		return float64(n), true
	case int64:
		return float64(n), true
	case json.Number:
		f, err := n.Float64()
		if err != nil {
			return 0, false
		}
		return f, true
	default:
		return 0, false
	}
}

// brief 把任意值转成短字符串用于错误描述，过长则截断。
func brief(v any) string {
	if v == nil {
		return "null"
	}
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("%v", v)
	}
	s := string(data)
	if len(s) > 120 {
		return s[:117] + "..."
	}
	return s
}
