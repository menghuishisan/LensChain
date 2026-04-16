// file_security_helper.go
// 模块03 — 课程与教学：课时附件安全校验
// 对照验收标准，对附件类型和大小做服务端兜底校验。

package course

import (
	"github.com/lenschain/backend/internal/pkg/filesecurity"
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

var lessonAttachmentRule = filesecurity.DocumentRule{
	MaxSize:        maxDocumentAttachmentSize,
	AllowedMIMEs:   allowedDocumentMIMEs,
	AllowedExts:    allowedDocumentExtensions,
	TooLargeMsg:    "文档文件不能超过50MB",
	InvalidTypeMsg: "仅支持PDF、Word、PPT文档附件",
}

// validateLessonAttachment 校验课时附件类型和大小
func validateLessonAttachment(fileName, fileType string, fileSize int64) error {
	return filesecurity.ValidateDocument(fileName, fileType, fileSize, lessonAttachmentRule)
}
