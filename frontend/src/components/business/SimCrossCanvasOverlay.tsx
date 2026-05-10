'use client';

// SimCrossCanvasOverlay.tsx
// 联动模式跨画布因果弧线 SVG 叠加层（06.2 §6.2）。
// 全局叠加在 MultiSceneContainer 上，仅 C 联动 + 多场景同屏启用。
// WS 接收 link_triggers[]，从 source_anchor_id 几何中心到 target_anchor_id 几何中心绘制贝塞尔弧线。
// 配色按 LinkGroup 类目：attack=红 / network=蓝 / crypto=青 / economic=金 / consensus=紫 / blockchain-integrity=灰。
// 弧线持续 1.5-2s 后 fade out。

import { useEffect, useRef, useState, useCallback } from 'react';
import { cn } from '@/lib/utils';
import type { SimLinkGroupColorType } from '@/types/experiment';

/** 单条跨画布因果弧线。 */
export interface LinkArc {
  id: string;
  sourceX: number;
  sourceY: number;
  targetX: number;
  targetY: number;
  colorType: SimLinkGroupColorType;
  label?: string;
  /** 创建时间戳（ms），用于 fade out 计算。 */
  createdAt: number;
}

export interface SimCrossCanvasOverlayProps {
  /** 当前活跃的弧线列表。由 SimEnginePanel 从 WS link_triggers 推导。 */
  arcs: LinkArc[];
  /** 弧线持续时间（ms），默认 2000。 */
  fadeDuration?: number;
  className?: string;
}

/** LinkGroup 配色映射。 */
const COLOR_MAP: Record<SimLinkGroupColorType, string> = {
  attack: '#ef4444',
  network: '#3b82f6',
  crypto: '#06b6d4',
  economic: '#eab308',
  consensus: '#a855f7',
  'blockchain-integrity': '#6b7280',
};

/**
 * SimCrossCanvasOverlay 联动跨画布贝塞尔弧线叠加层（06.2 §6.2）。
 * 使用 position:absolute SVG 叠加于 scene grid 之上，pointer-events:none 不阻挡交互。
 */
export function SimCrossCanvasOverlay({ arcs, fadeDuration = 2000, className }: SimCrossCanvasOverlayProps) {
  const svgRef = useRef<SVGSVGElement>(null);
  const [visibleArcs, setVisibleArcs] = useState<LinkArc[]>([]);

  // 定时清理过期弧线
  useEffect(() => {
    setVisibleArcs(arcs);
    const timer = setInterval(() => {
      const now = Date.now();
      setVisibleArcs((prev) => prev.filter((a) => now - a.createdAt < fadeDuration));
    }, 200);
    return () => clearInterval(timer);
  }, [arcs, fadeDuration]);

  const getOpacity = useCallback(
    (createdAt: number) => {
      const elapsed = Date.now() - createdAt;
      if (elapsed >= fadeDuration) return 0;
      // 前 50% 不透明，后 50% 线性 fade
      const fadeStart = fadeDuration * 0.5;
      if (elapsed < fadeStart) return 0.85;
      return 0.85 * (1 - (elapsed - fadeStart) / (fadeDuration - fadeStart));
    },
    [fadeDuration],
  );

  if (visibleArcs.length === 0) return null;

  return (
    <svg
      ref={svgRef}
      className={cn('pointer-events-none absolute inset-0 z-20', className)}
      style={{ width: '100%', height: '100%' }}
    >
      <defs>
        <marker id="link-arrow" viewBox="0 0 10 10" refX="9" refY="5" markerWidth="6" markerHeight="6" orient="auto-start-reverse">
          <path d="M 0 0 L 10 5 L 0 10 z" fill="currentColor" />
        </marker>
      </defs>
      {visibleArcs.map((arc) => {
        const color = COLOR_MAP[arc.colorType] ?? '#6b7280';
        const opacity = getOpacity(arc.createdAt);
        if (opacity <= 0) return null;
        // 贝塞尔控制点：中点上偏 40px
        const midX = (arc.sourceX + arc.targetX) / 2;
        const midY = Math.min(arc.sourceY, arc.targetY) - 40;
        const d = `M ${arc.sourceX} ${arc.sourceY} Q ${midX} ${midY} ${arc.targetX} ${arc.targetY}`;
        return (
          <g key={arc.id} opacity={opacity}>
            <path
              d={d}
              fill="none"
              stroke={color}
              strokeWidth={2}
              markerEnd="url(#link-arrow)"
              style={{ color, transition: 'opacity 0.3s ease' }}
            />
            {arc.label && (
              <text x={midX} y={midY - 6} textAnchor="middle" className="fill-current text-[10px]" style={{ color }}>
                {arc.label}
              </text>
            )}
          </g>
        );
      })}
    </svg>
  );
}
