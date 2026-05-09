'use client';

// WebIDEPanel.tsx
// Web IDE 嵌入面板
// 通过 iframe 加载 code-server 实例，支持全屏切换和外部打开
//
// iframe 鉴权：必须先调 useToolProxyCookie(instanceID, "ide") 签发 HttpOnly cookie，
// 再渲染 iframe 才能让 backend 反代 ToolProxyAuth 通过；详见 services/experimentToolProxy.ts
// 与 backend handler/experiment/tool_proxy.go。

import { useState } from 'react';
import { Code, Maximize2, Minimize2, ExternalLink } from 'lucide-react';
import { Button } from '@/components/ui/Button';
import { Badge } from '@/components/ui/Badge';
import { LoadingState } from '@/components/ui/LoadingState';
import { useToolProxyCookie } from '@/hooks/useToolProxyCookie';
import { cn } from '@/lib/utils';
import type { ID } from '@/types/api';

interface WebIDEPanelProps {
  /** 后端签发的相对路径，形如 /instance/<id>/ide/ */
  accessUrl: string;
  /** 实例 ID，签 cookie 时使用 */
  instanceID: ID;
  /** 工具类型，默认 "ide"，对应 instance_containers.tool_kind */
  toolKind?: string;
  className?: string;
}

/**
 * Web IDE 面板
 * 通过 iframe 嵌入 code-server，提供浏览器内代码编辑能力
 */
export function WebIDEPanel({ accessUrl, instanceID, toolKind = 'ide', className }: WebIDEPanelProps) {
  const [isFullscreen, setIsFullscreen] = useState(false);
  const cookieQuery = useToolProxyCookie(instanceID, toolKind, !!accessUrl);

  if (!accessUrl) {
    return (
      <div className={cn('flex items-center justify-center rounded-md border bg-muted/20 p-8', className)}>
        <p className="text-sm text-muted-foreground">此实验未配置 Web IDE 环境</p>
      </div>
    );
  }
  if (cookieQuery.isError) {
    return (
      <div className={cn('flex items-center justify-center rounded-md border bg-destructive/5 p-8', className)}>
        <p className="text-sm text-destructive">Web IDE 鉴权失败：{(cookieQuery.error as Error)?.message ?? '未知错误'}</p>
      </div>
    );
  }
  if (!cookieQuery.isSuccess) {
    return <LoadingState variant="spinner" />;
  }

  return (
    <div className={cn(
      'flex flex-col gap-2',
      isFullscreen && 'fixed inset-0 z-50 bg-background p-4',
      className,
    )}>
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Code className="h-4 w-4" />
          <Badge variant="outline">Web IDE</Badge>
        </div>
        <div className="flex gap-1">
          <Button variant="ghost" size="sm" onClick={() => setIsFullscreen(!isFullscreen)}>
            {isFullscreen ? <Minimize2 className="h-4 w-4" /> : <Maximize2 className="h-4 w-4" />}
          </Button>
          <Button variant="ghost" size="sm" onClick={() => window.open(accessUrl, '_blank')}>
            <ExternalLink className="h-4 w-4" />
          </Button>
        </div>
      </div>
      <iframe
        src={accessUrl}
        className="flex-1 min-h-[500px] rounded-md border"
        sandbox="allow-same-origin allow-scripts allow-popups allow-forms"
        title="Web IDE"
      />
    </div>
  );
}
