'use client';

// ToolIframe.tsx
// 轻量工具 iframe 包装：调用 useToolProxyCookie 预热反代 cookie 后再渲染 iframe。
// 用于不需要全屏 / 工具栏装饰的简单嵌入场景（如区块链浏览器、监控仪表盘）。
// IDE / VNC 因为有自己的 UI 装饰，不复用本组件，但鉴权逻辑相同（都走 useToolProxyCookie）。

import { LoadingState } from '@/components/ui/LoadingState';
import { useToolProxyCookie } from '@/hooks/useToolProxyCookie';
import { resolveToolProxyURL } from '@/services/experimentToolProxy';
import { cn } from '@/lib/utils';
import type { ID } from '@/types/api';

export interface ToolIframeProps {
  /** 后端签发的相对路径，形如 /instance/<id>/<kind>/ */
  src: string;
  /** 实例 ID */
  instanceID: ID;
  /** 工具类型，与 instance_containers.tool_kind 完全一致 */
  toolKind: string;
  /** iframe title，可访问性必填 */
  title: string;
  className?: string;
}

export function ToolIframe({ src, instanceID, toolKind, title, className }: ToolIframeProps) {
  const cookieQuery = useToolProxyCookie(instanceID, toolKind, !!src);

  if (!src) {
    return null;
  }
  if (cookieQuery.isError) {
    return (
      <div className={cn('flex items-center justify-center rounded border bg-destructive/5 p-8', className)}>
        <p className="text-sm text-destructive">{title} 鉴权失败：{(cookieQuery.error as Error)?.message ?? '未知错误'}</p>
      </div>
    );
  }
  if (!cookieQuery.isSuccess) {
    return <LoadingState variant="spinner" />;
  }
  return (
    <iframe
      src={resolveToolProxyURL(src)}
      className={cn('h-full w-full rounded border', className)}
      sandbox="allow-same-origin allow-scripts allow-popups allow-forms"
      title={title}
    />
  );
}
