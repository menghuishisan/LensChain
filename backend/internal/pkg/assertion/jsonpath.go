// Package assertion 提供平台范围内通用的状态断言基础能力：JSONPath 解析与算子比较。
//
// 服务层（实验检查点 / CTF 挑战断言 / 后续模块）共享同一个 DSL 语法和算子集合，
// 所有可观测状态的"路径 + 算子 + 期望值"判定都应通过本包实现，避免各处重新发明轮子。
//
// 设计原则：
//   - 纯函数，无任何业务依赖；
//   - 严格类型，不做隐式类型转换（如字符串 → 数字），所有类型不匹配立即返回错误；
//   - 错误消息中文，直接面向学生 / 教师展示。
package assertion

import (
	"fmt"
	"strconv"
	"strings"
)

// JSONPath 表示编译后的 JSONPath。
// 不导出字段：调用方只能通过 Compile 创建并通过 Lookup 求值，确保不可变。
type JSONPath struct {
	raw      string
	segments []pathSegment
}

// pathSegKind 标识单个路径段的语义。
type pathSegKind int

const (
	segField    pathSegKind = 1 // .field
	segIndex    pathSegKind = 2 // [N]
	segWildcard pathSegKind = 3 // [*]
	segLength   pathSegKind = 4 // .length（仅末尾，作用于数组）
)

type pathSegment struct {
	kind  pathSegKind
	field string
	index int
}

// Compile 解析路径字符串为可重复求值的 JSONPath。
//
// 支持的语法子集（足够覆盖所有内置场景断言；如未来需要过滤表达式可在此扩展）：
//
//	$                  根
//	$.field            对象字段
//	$.a.b.c            嵌套字段
//	$.field[N]         数组下标（非负整数）
//	$.field[*]         数组通配（聚合后续段，返回 []any）
//	$.field.length     数组长度（必须位于末尾，返回 int64）
func Compile(path string) (JSONPath, error) {
	if !strings.HasPrefix(path, "$") {
		return JSONPath{}, fmt.Errorf("路径 %q 必须以 $ 开始", path)
	}
	rest := path[1:]
	segments := make([]pathSegment, 0, 4)
	for len(rest) > 0 {
		switch rest[0] {
		case '.':
			rest = rest[1:]
			j := 0
			for j < len(rest) && rest[j] != '.' && rest[j] != '[' {
				j++
			}
			ident := rest[:j]
			if ident == "" {
				return JSONPath{}, fmt.Errorf("路径 %q 包含空字段名", path)
			}
			rest = rest[j:]
			if ident == "length" {
				if len(rest) > 0 {
					return JSONPath{}, fmt.Errorf("路径 %q: length 必须位于末尾", path)
				}
				segments = append(segments, pathSegment{kind: segLength})
				return JSONPath{raw: path, segments: segments}, nil
			}
			segments = append(segments, pathSegment{kind: segField, field: ident})
		case '[':
			end := strings.IndexByte(rest, ']')
			if end < 0 {
				return JSONPath{}, fmt.Errorf("路径 %q: 缺少 ]", path)
			}
			body := rest[1:end]
			rest = rest[end+1:]
			if body == "*" {
				segments = append(segments, pathSegment{kind: segWildcard})
				continue
			}
			idx, err := strconv.Atoi(body)
			if err != nil || idx < 0 {
				return JSONPath{}, fmt.Errorf("路径 %q: 非法下标 [%s]", path, body)
			}
			segments = append(segments, pathSegment{kind: segIndex, index: idx})
		default:
			return JSONPath{}, fmt.Errorf("路径 %q: 非法字符 %q", path, rest[0])
		}
	}
	return JSONPath{raw: path, segments: segments}, nil
}

// Raw 返回原始路径字符串，便于错误信息定位。
func (p JSONPath) Raw() string { return p.raw }

// Lookup 在 root 上按编译后的路径求值，返回命中值。
//
// root 必须是 encoding/json 解码出的 map[string]any（或在通配段内为 []any）。
// 任一段失败都返回带路径前缀的错误，方便排错。
func (p JSONPath) Lookup(root any) (any, error) {
	return walk(root, p.segments, p.raw)
}

// walk 顺序消费段序列。处理通配段时把剩余段下沉到每个数组元素。
func walk(current any, segments []pathSegment, raw string) (any, error) {
	for i, seg := range segments {
		switch seg.kind {
		case segLength:
			arr, ok := current.([]any)
			if !ok {
				return nil, fmt.Errorf("路径 %q: length 仅适用于数组", raw)
			}
			return int64(len(arr)), nil
		case segWildcard:
			arr, ok := current.([]any)
			if !ok {
				return nil, fmt.Errorf("路径 %q: [*] 仅适用于数组", raw)
			}
			rest := segments[i+1:]
			if len(rest) == 0 {
				clone := make([]any, len(arr))
				copy(clone, arr)
				return clone, nil
			}
			collected := make([]any, 0, len(arr))
			for _, item := range arr {
				sub, err := walk(item, rest, raw)
				if err != nil {
					return nil, err
				}
				collected = append(collected, sub)
			}
			return collected, nil
		case segField:
			obj, ok := current.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("路径 %q: 段 .%s 期望对象", raw, seg.field)
			}
			v, exists := obj[seg.field]
			if !exists {
				return nil, fmt.Errorf("路径 %q: 字段 %q 不存在", raw, seg.field)
			}
			current = v
		case segIndex:
			arr, ok := current.([]any)
			if !ok {
				return nil, fmt.Errorf("路径 %q: 下标 [%d] 期望数组", raw, seg.index)
			}
			if seg.index >= len(arr) {
				return nil, fmt.Errorf("路径 %q: 下标 [%d] 越界（长度=%d）", raw, seg.index, len(arr))
			}
			current = arr[seg.index]
		default:
			return nil, fmt.Errorf("路径 %q: 未知段类型", raw)
		}
	}
	return current, nil
}
