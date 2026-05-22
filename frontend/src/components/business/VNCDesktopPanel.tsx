'use client';

// VNCDesktopPanel.tsx
// 图形化桌面面板
// 通过 iframe 加载 noVNC Web 客户端，支持全屏切换
//
// iframe 鉴权：必须先调 useToolProxyCookie(instanceID, "desktop") 签发 HttpOnly cookie。
// 详见 services/experimentToolProxy.ts 与 backend handler/experiment/tool_proxy.go。

import { useState } from 'react';
import { Monitor, Maximize2, Minimize2, ExternalLink } from 'lucide-react';
import { Button } from '@/components/ui/Button';
import { Badge } from '@/components/ui/Badge';
import { LoadingState } from '@/components/ui/LoadingState';
import { useToolProxyCookie } from '@/hooks/useToolProxyCookie';
import { resolveToolProxyURL } from '@/services/experimentToolProxy';
import { cn } from '@/lib/utils';
import type { ID } from '@/types/api';

interface VNCDesktopPanelProps {
  /** 后端签发的相对路径，形如 /instance/<id>/desktop/ */
  accessUrl: string;
  /** 实例 ID，签 cookie 时使用 */
  instanceID: ID;
  /** 工具类型，默认 "desktop" */
  toolKind?: string;
  className?: string;
}

/**
 * 图形桌面面板
 * 通过 iframe 嵌入 noVNC，提供图形化桌面交互
 */
export function VNCDesktopPanel({ accessUrl, instanceID, toolKind = 'desktop', className }: VNCDesktopPanelProps) {
  const [isFullscreen, setIsFullscreen] = useState(false);
  const cookieQuery = useToolProxyCookie(instanceID, toolKind, !!accessUrl);
  const iframeURL = resolveToolProxyURL(accessUrl);

  if (!accessUrl) {
    return (
      <div className={cn('flex items-center justify-center rounded-md border bg-muted/20 p-8', className)}>
        <p className="text-sm text-muted-foreground">此实验未配置图形桌面环境</p>
      </div>
    );
  }
  if (cookieQuery.isError) {
    return (
      <div className={cn('flex items-center justify-center rounded-md border bg-destructive/5 p-8', className)}>
        <p className="text-sm text-destructive">图形桌面鉴权失败：{(cookieQuery.error as Error)?.message ?? '未知错误'}</p>
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
          <Monitor className="h-4 w-4" />
          <Badge variant="outline">图形桌面</Badge>
        </div>
        <div className="flex gap-1">
          <Button variant="ghost" size="sm" onClick={() => setIsFullscreen(!isFullscreen)}>
            {isFullscreen ? <Minimize2 className="h-4 w-4" /> : <Maximize2 className="h-4 w-4" />}
          </Button>
          <Button variant="ghost" size="sm" onClick={() => window.open(iframeURL, '_blank')}>
            <ExternalLink className="h-4 w-4" />
          </Button>
        </div>
      </div>
      <iframe
        src={iframeURL}
        className="flex-1 min-h-[500px] rounded-md border"
        sandbox="allow-same-origin allow-scripts allow-popups allow-forms"
        title="VNC Desktop"
      />
    </div>
  );
}
