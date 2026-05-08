// content-safety.ts
// 模块03内容安全工具：Markdown/富文本安全渲染和附件类型、大小校验。

const VIDEO_MAX_SIZE = 500 * 1024 * 1024;
const DOCUMENT_MAX_SIZE = 50 * 1024 * 1024;

/**
 * 附件校验结果。
 */
export interface AttachmentValidationResult {
  isValid: boolean;
  error?: string;
}

/**
 * validateCourseAttachment 校验课程附件文件类型和大小。
 */
export function validateCourseAttachment(file: File, kind: "video" | "document" = "document"): AttachmentValidationResult {
  if (kind === "video") {
    if (!file.type.startsWith("video/")) {
      return { isValid: false, error: "仅支持视频文件" };
    }
    if (file.size > VIDEO_MAX_SIZE) {
      return { isValid: false, error: "视频文件不能超过500MB" };
    }
    return { isValid: true };
  }

  const allowedDocumentTypes = [
    "application/pdf",
    "application/msword",
    "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
    "application/vnd.ms-powerpoint",
    "application/vnd.openxmlformats-officedocument.presentationml.presentation",
  ];

  if (!allowedDocumentTypes.includes(file.type)) {
    return { isValid: false, error: "仅支持 PDF/Word/PPT 文档" };
  }
  if (file.size > DOCUMENT_MAX_SIZE) {
    return { isValid: false, error: "文档文件不能超过50MB" };
  }
  return { isValid: true };
}
