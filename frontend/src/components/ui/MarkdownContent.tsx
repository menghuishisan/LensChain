"use client";

// MarkdownContent.tsx
// 统一的 Markdown 文本渲染组件，用于公告、讨论、课时内容、实验说明等长文本。
// 解析 GFM（表格、任务列表、删除线等）并把单个换行视为软换行 <br>，更贴近用户直觉。
// React 默认转义 HTML，不解析原始 <script> 等标签，避免 XSS。

import ReactMarkdown from "react-markdown";
import remarkBreaks from "remark-breaks";
import remarkGfm from "remark-gfm";

import { cn } from "@/lib/utils";

export interface MarkdownContentProps {
  /** 待渲染的 Markdown 源文本。 */
  content: string | null | undefined;
  /** 空内容时展示的占位文本。 */
  empty?: string;
  /** 追加的容器类名。 */
  className?: string;
  /** 是否使用紧凑尺寸（prose-sm）。 */
  compact?: boolean;
}

/**
 * MarkdownContent 将 Markdown 文本渲染为富文本。
 * 统一处理样式（tailwind typography）、GFM 扩展与软换行。
 */
export function MarkdownContent({ content, empty = "暂无内容", className, compact = true }: MarkdownContentProps) {
  const text = content?.trim() ?? "";
  if (text.length === 0) {
    return <p className={cn("text-sm text-muted-foreground", className)}>{empty}</p>;
  }
  return (
    <div className={cn("prose max-w-none dark:prose-invert", compact && "prose-sm", className)}>
      <ReactMarkdown remarkPlugins={[remarkGfm, remarkBreaks]}>{text}</ReactMarkdown>
    </div>
  );
}
