// timeutil.go
// 提供后端统一的时间解析与格式化基础能力。
// 当前主要服务于 service 层处理 API DTO 中的 ISO 8601/RFC3339 时间字符串，
// 避免把运行时解析逻辑放进 model/dto 等纯数据表达层。

package timeutil

import "time"

// ParseRFC3339 解析 API 请求中使用的 RFC3339 时间字符串。
// 当传入空字符串时返回 nil，便于上层直接表达“未填写时间”。
func ParseRFC3339(value string) (*time.Time, error) {
	if value == "" {
		return nil, nil
	}

	parsedAt, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return nil, err
	}
	return &parsedAt, nil
}
