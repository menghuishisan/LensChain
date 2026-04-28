"use client";

// ImageDocSidebar.tsx
// 镜像文档侧边栏组件
// 360px 宽侧边栏，加载镜像结构化文档（概述、端口、环境变量、搭配建议等）

import { BookOpen, X } from "lucide-react";

import { Badge } from "@/components/ui/Badge";
import { Button } from "@/components/ui/Button";
import { LoadingState } from "@/components/ui/LoadingState";
import { useImageDocumentation } from "@/hooks/useExperimentTemplates";
import type { ID } from "@/types/api";

/**
 * ImageDocSidebar 属性。
 */
interface ImageDocSidebarProps {
  imageID: ID;
  onClose: () => void;
}

/**
 * ImageDocSidebar 镜像文档侧边栏组件。
 * 从 API 加载镜像结构化文档并分段渲染。
 */
export function ImageDocSidebar({ imageID, onClose }: ImageDocSidebarProps) {
  const docQuery = useImageDocumentation(imageID);

  return (
    <div className="fixed inset-y-0 right-0 z-40 flex w-[360px] flex-col border-l border-border bg-background shadow-xl">
      {/* 顶部标题栏 */}
      <div className="flex items-center justify-between border-b border-border px-4 py-3">
        <div className="flex items-center gap-2">
          <BookOpen className="h-4 w-4 text-primary" />
          <span className="font-semibold text-sm">镜像文档</span>
          {docQuery.data ? (
            <Badge variant="outline" className="text-xs">{docQuery.data.display_name}</Badge>
          ) : null}
        </div>
        <Button variant="ghost" size="sm" onClick={onClose}>
          <X className="h-4 w-4" />
        </Button>
      </div>

      {/* 内容区域 */}
      <div className="flex-1 overflow-y-auto p-4 space-y-5">
        {docQuery.isLoading ? (
          <LoadingState title="加载文档" description="正在读取镜像文档内容。" />
        ) : docQuery.isError ? (
          <p className="text-sm text-destructive">文档加载失败，请稍后重试。</p>
        ) : docQuery.data ? (
          <>
            <div className="space-y-1">
              <p className="text-lg font-semibold">{docQuery.data.display_name}</p>
              <p className="text-xs text-muted-foreground">{docQuery.data.name}</p>
            </div>
            {Object.entries(docQuery.data.sections).map(([sectionKey, sectionContent]) => (
              <div key={sectionKey} className="space-y-2">
                <p className="font-semibold text-sm capitalize">{formatSectionTitle(sectionKey)}</p>
                <div className="whitespace-pre-wrap rounded-xl border border-border bg-muted/25 p-3 text-sm text-muted-foreground leading-6">
                  {sectionContent}
                </div>
              </div>
            ))}
          </>
        ) : (
          <p className="text-sm text-muted-foreground">无可用文档。</p>
        )}
      </div>
    </div>
  );
}

/**
 * formatSectionTitle 将文档节键名转为可读标题。
 */
function formatSectionTitle(key: string) {
  const titles: Record<string, string> = {
    overview: "概述",
    ports: "端口说明",
    env_vars: "环境变量",
    volumes: "挂载卷",
    companions: "搭配建议",
    resources: "资源要求",
    quickstart: "快速开始",
    troubleshooting: "常见问题",
  };
  return titles[key] ?? key.replace(/_/g, " ");
}
