// document.go
// 文档文件安全校验工具
// 用于对 PDF、Word、PPT 等文档元数据做统一的类型和大小校验

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
