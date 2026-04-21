// input_helper.go
// 模块03 — 课程与教学：输入解析辅助方法
// 统一处理模块内重复使用的可选雪花 ID 解析，避免参数格式错误被静默忽略

package course

import (
	"time"

	"github.com/lenschain/backend/internal/pkg/errcode"
	"github.com/lenschain/backend/internal/pkg/snowflake"
)

// parseOptionalSnowflakeID 解析可选雪花 ID。
// 传空值时返回 nil；传入非空但格式非法时返回明确业务错误。
func parseOptionalSnowflakeID(raw *string, invalidMessage string) (*int64, error) {
	if raw == nil {
		return nil, nil
	}
	if *raw == "" {
		return nil, nil
	}
	parsedID, err := snowflake.ParseString(*raw)
	if err != nil {
		return nil, errcode.ErrInvalidParams.WithMessage(invalidMessage)
	}
	return &parsedID, nil
}

// validateCourseTimeRange 校验课程开始/结束时间范围。
// 课程时间属于主表核心数据，服务层需要保证结束时间晚于开始时间。
func validateCourseTimeRange(startAt, endAt *time.Time) error {
	if startAt == nil || endAt == nil {
		return nil
	}
	if !endAt.After(*startAt) {
		return errcode.ErrInvalidParams.WithMessage("课程结束时间必须晚于开始时间")
	}
	return nil
}
