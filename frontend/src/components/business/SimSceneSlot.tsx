'use client';

// SimSceneSlot.tsx
// 单场景插槽组件（06.2 §一 / §5.6）。
// 自包含：场景头（标题 + 截图 + 局部全屏）+ Canvas + 该场景独立的 InteractionForm。
// 由 SimSceneGrid 在 grid / focus / carousel 三种布局中复用。
// 设计要点：
//   - 每个 slot 自己调用 useSimInteraction，因此 schema/动作集互不干扰；
//   - InteractionForm 是否渲染由父级 showInteractionForm 控制
//     （文档 §5.6：grid 每场景一份；focus 仅聚焦场景；carousel 当前场景）；
//   - 局部全屏按钮仍保留场景级 Maximize（不同于 TopBar 的面板级全屏）。

import { useState } from 'react';
import { Camera, Maximize2, Minimize2 } from 'lucide-react';
import { Button } from '@/components/ui/Button';
import { cn } from '@/lib/utils';
import { SimSceneCanvas } from '@/components/business/SimSceneCanvas';
import { SimInteractionForm } from '@/components/business/SimInteractionForm';
import { useSimInteraction } from '@/hooks/useSimInteraction';
import type { JsonObject } from '@/types/experiment';

/** 场景配置项（与 SimSceneGrid 共用）。 */
export interface SceneSlotConfig {
  sceneCode: string;
  category: string;
  title: string;
  hasState: boolean;
}

export interface SimSceneSlotProps {
  scene: SceneSlotConfig;
  instanceID?: string;
  userRole?: string;
  /** WS 是否已连通（透传给 InteractionForm 决定按钮可用性）。 */
  connected: boolean;
  /** 容器最小高度（px）。父级按 §3.5 尺寸表传入。 */
  minHeight?: number;
  /** 是否在画布下方渲染 InteractionForm（§5.6 规则由父级决定）。 */
  showInteractionForm?: boolean;
  attachScene: (code: string, cat: string, canvas: HTMLCanvasElement, overlay?: HTMLElement) => void;
  detachScene: (code: string) => void;
  redrawScene: (code: string) => void;
  captureScene: (code: string) => string | null;
  onSubmitInteraction: (sceneCode: string, actionCode: string, params: JsonObject) => void;
  className?: string;
  /** 用于由父级（SimSceneGrid）传入动态尺寸约束（如 min-width）。 */
  style?: React.CSSProperties;
}

/**
 * SimSceneSlot 渲染一个场景插槽：场景头 + Canvas + 可选 InteractionForm。
 * 通过自带的 useSimInteraction 拉取本场景的 ActionDef 列表。
 */
export function SimSceneSlot({
  scene,
  instanceID,
  userRole,
  connected,
  minHeight = 360,
  showInteractionForm = false,
  attachScene,
  detachScene,
  redrawScene,
  captureScene,
  onSubmitInteraction,
  className,
  style,
}: SimSceneSlotProps) {
  const [isFullscreen, setIsFullscreen] = useState(false);

  const interaction = useSimInteraction({
    instanceID: instanceID ?? '',
    sceneCode: scene.sceneCode,
    enabled: !!instanceID && showInteractionForm,
    userRole,
  });

  const handleScreenshot = () => {
    const dataUrl = captureScene(scene.sceneCode);
    if (!dataUrl) return;
    const a = document.createElement('a');
    a.href = dataUrl;
    a.download = `${scene.sceneCode}-screenshot.png`;
    a.click();
  };

  return (
    <div
      className={cn(
        'rounded-xl border bg-card overflow-hidden flex flex-col',
        isFullscreen && 'fixed inset-4 z-50 bg-background shadow-2xl',
        className,
      )}
      style={isFullscreen ? undefined : style}
    >
      {/* 场景头 */}
      <div className="flex items-center justify-between px-3 py-2 border-b shrink-0">
        <div className="min-w-0">
          <p className="text-sm font-medium truncate">{scene.title}</p>
          <p className="text-xs text-muted-foreground">{scene.category}</p>
        </div>
        <div className="flex gap-1 shrink-0">
          <Button variant="ghost" size="sm" className="h-7 w-7 p-0" title="截图" onClick={handleScreenshot}>
            <Camera className="h-3.5 w-3.5" />
          </Button>
          <Button
            variant="ghost"
            size="sm"
            className="h-7 w-7 p-0"
            title={isFullscreen ? '退出场景全屏' : '场景全屏'}
            onClick={() => setIsFullscreen((v) => !v)}
          >
            {isFullscreen ? <Minimize2 className="h-3.5 w-3.5" /> : <Maximize2 className="h-3.5 w-3.5" />}
          </Button>
        </div>
      </div>

      {/* Canvas 区域 */}
      <SimSceneCanvas
        sceneCode={scene.sceneCode}
        category={scene.category}
        attachScene={attachScene}
        detachScene={detachScene}
        redrawScene={redrawScene}
        hasState={scene.hasState}
        minHeight={minHeight}
        className="flex-1"
      />

      {/* 该场景独立的 InteractionForm（§5.6） */}
      {showInteractionForm && interaction.actions.length > 0 && (
        <div className="shrink-0 max-h-[40%] overflow-y-auto border-t">
          <SimInteractionForm
            actions={interaction.actions}
            connected={connected}
            onSubmit={(actionCode, params) => onSubmitInteraction(scene.sceneCode, actionCode, params)}
          />
        </div>
      )}
    </div>
  );
}
