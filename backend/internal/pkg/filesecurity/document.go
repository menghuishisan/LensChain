// document.go
// 该文件提供文档类附件的统一安全校验能力，负责按业务场景校验文件扩展名、MIME 类型和
// 大小限制。课程附件、实验报告、成绩单上传等涉及文档文件的入口，都应通过这里做基础
// 安全兜底，而不是在各模块里零散写一套校验规则。

package filesecurity

import (
	"path/filepath"
	"strings"

	"github.com/lenschain/backend/internal/pkg/errcode"
)

// DocumentRule 文档文件校验规则。
type DocumentRule struct {
	MaxSize        int64
	AllowedMIMEs   map[string]bool
	AllowedExts    map[string]bool
	TooLargeMsg    string
	InvalidTypeMsg string
}

// ValidateDocument 校验文档文件类型和大小。
func ValidateDocument(fileName, fileType string, fileSize int64, rule DocumentRule) error {
	if rule.MaxSize > 0 && fileSize > rule.MaxSize {
		if strings.TrimSpace(rule.TooLargeMsg) != "" {
			return errcode.ErrInvalidParams.WithMessage(rule.TooLargeMsg)
		}
		return errcode.ErrInvalidParams.WithMessage("文件大小超过限制")
	}

	normalizedType := strings.ToLower(strings.TrimSpace(fileType))
	if normalizedType != "" && rule.AllowedMIMEs[normalizedType] {
		return nil
	}

	ext := strings.ToLower(filepath.Ext(strings.TrimSpace(fileName)))
	if ext != "" && rule.AllowedExts[ext] {
		return nil
	}

	if strings.TrimSpace(rule.InvalidTypeMsg) != "" {
		return errcode.ErrInvalidFileType.WithMessage(rule.InvalidTypeMsg)
	}
	return errcode.ErrInvalidFileType
}
