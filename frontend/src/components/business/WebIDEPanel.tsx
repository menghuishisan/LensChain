'use client';

// WebIDEPanel.tsx
// Web IDE 嵌入面板
// 通过 iframe 加载 code-server 实例，支持全屏切换和外部打开

import { useState } from 'react';
import { Code, Maximize2, Minimize2, ExternalLink } from 'lucide-react';
import { Button } from '@/components/ui/Button';
import { Badge } from '@/components/ui/Badge';
import { cn } from '@/lib/utils';

interface WebIDEPanelProps {
  accessUrl: string;
  className?: string;
}

/**
 * Web IDE 面板
 * 通过 iframe 嵌入 code-server，提供浏览器内代码编辑能力
 */
export function WebIDEPanel({ accessUrl, className }: WebIDEPanelProps) {
  const [isFullscreen, setIsFullscreen] = useState(false);

  if (!accessUrl) {
    return (
      <div className={cn('flex items-center justify-center rounded-md border bg-muted/20 p-8', className)}>
        <p className="text-sm text-muted-foreground">此实验未配置 Web IDE 环境</p>
      </div>
    );
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
