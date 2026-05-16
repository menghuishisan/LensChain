'use client';

/**
 * SimSceneCanvas — 单个场景的画布容器（06.2 §一 主画布区）。
 *
 * 职责很薄：mount 时把 canvas 交给 SimPanel.attachScene；unmount 时 detach。
 * canvas 自适应通过 SceneView 内置 ResizeObserver 处理，本组件不参与尺寸计算。
 *
 * 双击触发 onDoubleClick（用于 grid → focus 切换）。
 */

import { useEffect, useRef } from 'react';
import { cn } from '@/lib/utils';

export interface SimSceneCanvasProps {
  sceneCode: string;
  attachScene: (sceneCode: string, canvas: HTMLCanvasElement) => void;
  detachScene: (sceneCode: string) => void;
  /** 双击进入 focus 视图。 */
  onDoubleClick?: (sceneCode: string) => void;
  className?: string;
}

export function SimSceneCanvas(props: SimSceneCanvasProps) {
  const { sceneCode, attachScene, detachScene, onDoubleClick, className } = props;
  const canvasRef = useRef<HTMLCanvasElement | null>(null);

  useEffect(() => {
    const el = canvasRef.current;
    if (!el) return;
    attachScene(sceneCode, el);
    return () => { detachScene(sceneCode); };
  }, [sceneCode, attachScene, detachScene]);

  return (
    <canvas
      ref={canvasRef}
      data-scene-code={sceneCode}
      onDoubleClick={() => onDoubleClick?.(sceneCode)}
      className={cn(
        'block h-full w-full rounded-md border border-border bg-card outline-none',
        'select-none',
        onDoubleClick && 'cursor-pointer',
        className,
      )}
    />
  );
}
