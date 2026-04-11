// file_security_helper.go
// 模块03 — 课程与教学：课时附件安全校验
// 对照验收标准，对附件类型和大小做服务端兜底校验。

package course

import (
	"path/filepath"
	"strings"

	"github.com/lenschain/backend/internal/pkg/errcode"
)

const (
	maxDocumentAttachmentSize = 50 * 1024 * 1024
)

var allowedDocumentExtensions = map[string]bool{
	".pdf":  true,
	".doc":  true,
	".docx": true,
	".ppt":  true,
	".pptx": true,
}

var allowedDocumentMIMEs = map[string]bool{
	"application/pdf":    true,
	"application/msword": true,
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document":   true,
	"application/vnd.ms-powerpoint":                                             true,
	"application/vnd.openxmlformats-officedocument.presentationml.presentation": true,
}

// validateLessonAttachment 校验课时附件类型和大小
func validateLessonAttachment(fileName, fileType string, fileSize int64) error {
	if fileSize > maxDocumentAttachmentSize {
		return errcode.ErrInvalidParams.WithMessage("文档文件不能超过50MB")
	}

	normalizedType := strings.ToLower(strings.TrimSpace(fileType))
	if allowedDocumentMIMEs[normalizedType] {
		return nil
	}

	ext := strings.ToLower(filepath.Ext(fileName))
	if allowedDocumentExtensions[ext] {
		return nil
	}

	return errcode.ErrInvalidParams.WithMessage("仅支持PDF、Word、PPT文档附件")
}
