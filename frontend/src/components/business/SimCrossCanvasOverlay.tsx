'use client';

/**
 * SimCrossCanvasOverlay — 跨画布因果弧线（06.2 §6.2，仅 C 联动 + 多场景同屏启用）。
 *
 * 监听 linkTriggers，对每条 trigger：
 *   1. 通过 getAnchorPosition 查 source / target anchor 在视口的像素位置；
 *   2. 用 SVG 绘制贝塞尔弧线 + 头部箭头 + 文字标签；
 *   3. 1.5-2s 后 fade out。
 *
 * 配色按 LinkGroup 类目（详 06.2 §6.2 末）。
 */

import { useEffect, useMemo, useState } from 'react';
import type { LinkTrigger } from '@lenschain/sim-engine-renderers';
import { cn } from '@/lib/utils';

export interface AnchorPosition {
  x: number; // 视口绝对像素
  y: number;
}

export interface SimCrossCanvasOverlayProps {
  /** 当前活跃的联动事件（最新触发的若干条）。 */
  triggers: readonly LinkTrigger[];
  /** 解析 anchor 在视口中的像素位置；找不到返回 null。 */
  getAnchorPosition: (sceneCode: string, anchorId: string) => AnchorPosition | null;
  className?: string;
}

interface ActiveArc {
  id: string;
  source: AnchorPosition;
  target: AnchorPosition;
  label: string;
  group: string;
  expiresAt: number;
}

/** LinkGroup 名称 → 弧线颜色（06.2 §6.2 末）。 */
function colorForGroup(group: string): string {
  if (group.includes('attack')) return '#ef4444';        // red
  if (group.includes('network')) return '#3b82f6';       // blue
  if (group.includes('crypto')) return '#06b6d4';        // cyan
  if (group.includes('economic')) return '#f59e0b';      // gold
  if (group.includes('consensus')) return '#a855f7';     // purple
  if (group.includes('integrity')) return '#6b7280';     // gray
  return '#64748b';                                      // slate fallback
}

const ARC_TTL_MS = 1800;

export function SimCrossCanvasOverlay(props: SimCrossCanvasOverlayProps) {
  const { triggers, getAnchorPosition, className } = props;
  const [now, setNow] = useState(() => Date.now());

  const activeArcs = useMemo<ActiveArc[]>(() => {
    const arcs: ActiveArc[] = [];
    for (const t of triggers) {
      if (!t.source_anchor_id || !t.target_anchor_id) continue;
      const src = getAnchorPosition(t.source_scene, t.source_anchor_id);
      // target_scene 在 LinkTrigger 上没有显式字段；这里假设跨画布弧线的 target 也在 source_scene 同一 anchor 命名空间下，
      // 真实业务通常 target_anchor 属于另一场景，需要由父组件提前把 target_scene 注入 anchor 解析；
      // 为保留 anchor_id 接口兼容，先用 source_scene 解析两端。父组件可以叠加 anchor → 全局键映射。
      const tgt = getAnchorPosition(t.source_scene, t.target_anchor_id);
      if (!src || !tgt) continue;
      arcs.push({
        id: t.id,
        source: src,
        target: tgt,
        label: t.changed_fields[0] ?? t.source_action,
        group: t.link_group,
        expiresAt: t.ts + ARC_TTL_MS,
      });
    }
    return arcs.filter(a => a.expiresAt > now);
  }, [triggers, getAnchorPosition, now]);

  // 60ms 一次刷新本地 now，使过期的弧线自动消失
  useEffect(() => {
    if (activeArcs.length === 0) return;
    const id = setInterval(() => setNow(Date.now()), 60);
    return () => clearInterval(id);
  }, [activeArcs.length]);

  if (activeArcs.length === 0) return null;

  return (
    <svg
      className={cn('pointer-events-none absolute inset-0 z-30', className)}
      width="100%"
      height="100%"
    >
      {activeArcs.map((arc) => {
        const color = colorForGroup(arc.group);
        const remain = Math.max(0, arc.expiresAt - now);
        const opacity = Math.min(1, remain / ARC_TTL_MS);
        const path = bezierPath(arc.source, arc.target);
        return (
          <g key={arc.id} opacity={opacity}>
            <path d={path} fill="none" stroke={color} strokeWidth={2} strokeDasharray="4 3" />
            <ArrowHead source={arc.source} target={arc.target} color={color} />
            <text
              x={(arc.source.x + arc.target.x) / 2}
              y={(arc.source.y + arc.target.y) / 2 - 6}
              textAnchor="middle"
              fontSize="10"
              fill={color}
              className="font-mono"
            >
              ↗ {arc.label}
            </text>
          </g>
        );
      })}
    </svg>
  );
}

function bezierPath(s: AnchorPosition, t: AnchorPosition): string {
  const cx1 = s.x;
  const cy1 = (s.y + t.y) / 2 - 60;
  const cx2 = t.x;
  const cy2 = (s.y + t.y) / 2 - 60;
  return `M ${s.x} ${s.y} C ${cx1} ${cy1}, ${cx2} ${cy2}, ${t.x} ${t.y}`;
}

function ArrowHead(props: { source: AnchorPosition; target: AnchorPosition; color: string }) {
  const { source, target, color } = props;
  const dx = target.x - source.x;
  const dy = target.y - source.y;
  const angle = Math.atan2(dy, dx);
  const size = 8;
  const x1 = target.x - Math.cos(angle - Math.PI / 6) * size;
  const y1 = target.y - Math.sin(angle - Math.PI / 6) * size;
  const x2 = target.x - Math.cos(angle + Math.PI / 6) * size;
  const y2 = target.y - Math.sin(angle + Math.PI / 6) * size;
  return (
    <polygon
      points={`${target.x},${target.y} ${x1},${y1} ${x2},${y2}`}
      fill={color}
    />
  );
}
