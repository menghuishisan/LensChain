'use client';

// SimSceneCanvas.tsx
// 单场景 Canvas 渲染组件（06.2 §一 主画布区）。
// 管理 canvas drawingbuffer 与 SVG overlay 的实时尺寸同步，保证 HiDPI 清晰度。
// 由 SimSceneGrid 按布局模式管理多个 SimSceneCanvas 的排列。

import { useEffect, useRef } from 'react';
import { cn } from '@/lib/utils';

export interface SimSceneCanvasProps {
  sceneCode: string;
  category: string;
  attachScene: (code: string, cat: string, canvas: HTMLCanvasElement, overlay?: HTMLElement) => void;
  detachScene: (code: string) => void;
  redrawScene: (code: string) => void;
  hasState: boolean;
  /** 容器最小高度（px）。文档 §3.5：单/2/3 场景 480，2×2 网格 360，focus 主区 480。 */
  minHeight?: number;
  className?: string;
}

/**
 * SimSceneCanvas 渲染单个场景的 Canvas + SVG overlay。
 * 通过 ResizeObserver 实时同步 DOM 容器像素到 canvas drawingbuffer，
 * 避免渲染器取到浏览器默认 300x150（06.md 画布尺寸契约）。
 */
export function SimSceneCanvas({
  sceneCode,
  category,
  attachScene,
  detachScene,
  redrawScene,
  hasState,
  minHeight = 360,
  className,
}: SimSceneCanvasProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const svgRef = useRef<SVGSVGElement>(null);
  const attachedRef = useRef(false);

  useEffect(() => {
    const container = containerRef.current;
    const canvas = canvasRef.current;
    const svg = svgRef.current;
    if (!container || !canvas) return;

    if (!attachedRef.current) {
      attachScene(sceneCode, category, canvas, svg as unknown as HTMLElement ?? undefined);
      attachedRef.current = true;
    }

    const syncSize = () => {
      const rect = container.getBoundingClientRect();
      const dpr = window.devicePixelRatio || 1;
      const targetW = Math.max(1, Math.round(rect.width * dpr));
      const targetH = Math.max(1, Math.round(rect.height * dpr));
      let changed = false;
      if (canvas.width !== targetW) { canvas.width = targetW; changed = true; }
      if (canvas.height !== targetH) { canvas.height = targetH; changed = true; }
      if (svg) {
        const viewBox = `0 0 ${targetW} ${targetH}`;
        if (svg.getAttribute('viewBox') !== viewBox) { svg.setAttribute('viewBox', viewBox); changed = true; }
      }
      if (changed) redrawScene(sceneCode);
    };

    syncSize();
    const observer = new ResizeObserver(syncSize);
    observer.observe(container);

    return () => {
      observer.disconnect();
      if (attachedRef.current) {
        detachScene(sceneCode);
        attachedRef.current = false;
      }
    };
  }, [sceneCode, category, attachScene, detachScene, redrawScene]);

  return (
    <div
      ref={containerRef}
      className={cn('relative w-full h-full', className)}
      style={{ minHeight: `${minHeight}px` }}
    >
      <canvas ref={canvasRef} className="absolute inset-0 w-full h-full" />
      <svg ref={svgRef} className="absolute inset-0 w-full h-full pointer-events-none" xmlns="http://www.w3.org/2000/svg" />
      {!hasState && (
        <div className="absolute inset-0 flex items-center justify-center bg-background/50">
          <div className="text-center">
            <div className="mx-auto h-10 w-10 rounded-full border border-primary/30 bg-primary/10 animate-pulse" />
            <p className="mt-2 text-xs text-muted-foreground">等待仿真数据...</p>
          </div>
        </div>
      )}
    </div>
  );
}
