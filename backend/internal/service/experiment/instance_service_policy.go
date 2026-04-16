// instance_service_policy.go
// 模块04 — 实验环境：实例相关通用策略与校验
// 负责业务限流、实验报告文件校验等公共策略，避免在各 service 文件内重复实现

package experiment

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/lenschain/backend/internal/pkg/cache"
	"github.com/lenschain/backend/internal/pkg/errcode"
	"github.com/lenschain/backend/internal/pkg/filesecurity"
)

const experimentReportMaxSize = 50 * 1024 * 1024

var experimentReportRule = filesecurity.DocumentRule{
	MaxSize: experimentReportMaxSize,
	AllowedMIMEs: map[string]bool{
		"application/pdf":    true,
		"application/msword": true,
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document": true,
	},
	AllowedExts: map[string]bool{
		".pdf":  true,
		".doc":  true,
		".docx": true,
	},
	TooLargeMsg:    "报告文件不能超过50MB",
	InvalidTypeMsg: "仅支持PDF、Word报告文件",
}

// enforceHeartbeatRateLimit 执行心跳接口业务限流。
func (s *instanceService) enforceHeartbeatRateLimit(ctx context.Context, userID int64) error {
	key := cache.KeyExpHeartbeatRate + strconv.FormatInt(userID, 10)
	count, err := cache.IncrWithExpire(ctx, key, time.Minute)
	if err != nil {
		return err
	}
	if count > 2 {
		return errcode.ErrHeartbeatRateLimit
	}
	return nil
}

// enforceCheckpointRateLimit 执行检查点验证接口业务限流。
func (s *instanceService) enforceCheckpointRateLimit(ctx context.Context, userID int64) error {
	key := cache.KeyExpCheckpointRate + strconv.FormatInt(userID, 10)
	count, err := cache.IncrWithExpire(ctx, key, time.Minute)
	if err != nil {
		return err
	}
	if count > 10 {
		return errcode.ErrCheckpointRateLimit
	}
	return nil
}

// validateReportPayload 校验实验报告提交内容。
func validateReportPayload(content, fileURL, fileName *string, fileSize *int64) error {
	trimmedContent := ""
	if content != nil {
		trimmedContent = strings.TrimSpace(*content)
	}

	hasFileURL := fileURL != nil && strings.TrimSpace(*fileURL) != ""
	hasFileName := fileName != nil && strings.TrimSpace(*fileName) != ""
	hasFileSize := fileSize != nil && *fileSize > 0
	hasFile := hasFileURL || hasFileName || hasFileSize

	if trimmedContent == "" && !hasFile {
		return errcode.ErrInvalidParams.WithMessage("报告内容和报告文件至少提交一种")
	}

	if hasFile {
		if !hasFileURL || !hasFileName || !hasFileSize {
			return errcode.ErrInvalidParams.WithMessage("提交报告文件时必须同时提供 file_url、file_name 和 file_size")
		}
		if err := filesecurity.ValidateDocument(strings.TrimSpace(*fileName), "", *fileSize, experimentReportRule); err != nil {
			return err
		}
	}

	return nil
}

// normalizeOptionalText 归一化可选文本字段，空白字符串会被视为 nil。
func normalizeOptionalText(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}
