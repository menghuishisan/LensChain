'use client';

/**
 * SimSceneSlot — 单场景插槽（重构版，对齐 sim-engine-redesign-proposal.html ②/③ 节）。
 *
 * 与旧版差异：
 *   1. 不再内嵌 InteractionForm（统一一份在主区下方，由 SimEnginePanel 渲染）。
 *   2. 新增 header ▼ dropdown：点击展开"内容类原语"卡片（code_block / math_formula /
 *      register_row）+ 衍生指标（progress_bar / risk_gauge），即 HTML 行 957-958
 *      "每场景的 code/formula/progress 在该场景头部 ▼ dropdown 展开（避免侧栏堆叠 4 套）"。
 *   3. 不显式接受 schema / userRole / onSubmitAction —— 这些是 InteractionForm 的关心。
 *
 * 单场景模式（mode=single）下父组件可隐藏 header（hideHeader=true），把舞台让给主画布。
 */

import { useMemo, useState } from 'react';
import { Camera, ChevronDown, ChevronRight, Copy, Gauge, Maximize2 } from 'lucide-react';
import { Button } from '@/components/ui/Button';
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/Tooltip';
import { Badge } from '@/components/ui/Badge';
import { cn } from '@/lib/utils';
import { SimSceneCanvas } from '@/components/business/SimSceneCanvas';
import type { Primitive, RenderState } from '@lenschain/sim-engine-renderers';

export interface SimSceneSlotProps {
  sceneCode: string;
  title: string;
  /** 完整 RenderState；未连接 / 首帧未到时为 null。 */
  state: RenderState | null;
  /** 是否隐藏 slot 自身 header（single 模式由父级 TopBar 表达标题）。 */
  hideHeader?: boolean;
  /** 双击触发：grid → focus。 */
  onFocus?: (sceneCode: string) => void;
  onCapture?: (sceneCode: string) => void;
  attachScene: (sceneCode: string, canvas: HTMLCanvasElement) => void;
  detachScene: (sceneCode: string) => void;
  className?: string;
}

const CONTENT_PRIMITIVE_TYPES = new Set([
  'code_block', 'math_formula', 'register_row', 'progress_bar', 'risk_gauge',
]);

export function SimSceneSlot(props: SimSceneSlotProps) {
  const {
    sceneCode, title, state, hideHeader, onFocus, onCapture,
    attachScene, detachScene, className,
  } = props;

  const [contentOpen, setContentOpen] = useState(false);

  const tick = state?.tick ?? 0;
  const contentPrimitives = useMemo(
    () => (state?.envelope.primitives ?? []).filter(p => CONTENT_PRIMITIVE_TYPES.has(p.type)),
    [state?.envelope.primitives],
  );
  const hasContent = contentPrimitives.length > 0;

  return (
    <TooltipProvider>
      <div className={cn(
        'flex h-full flex-col overflow-hidden rounded-md border border-border bg-card',
        className,
      )}>
        {!hideHeader && (
          <header className="flex h-7 items-center justify-between border-b border-border bg-muted/30 px-2">
            <button
              type="button"
              disabled={!hasContent}
              onClick={() => setContentOpen(v => !v)}
              className={cn(
                'flex items-center gap-1 text-[11px] font-medium',
                hasContent ? 'cursor-pointer hover:text-primary' : 'cursor-default text-muted-foreground',
              )}
            >
              {hasContent && (contentOpen ? (
                <ChevronDown className="h-3 w-3" />
              ) : (
                <ChevronRight className="h-3 w-3" />
              ))}
              <span>{title}</span>
            </button>
            <div className="flex items-center gap-1">
              <Badge variant="outline" className="text-[10px]">tick {tick}</Badge>
              {onCapture && (
                <Tooltip>
                  <TooltipTrigger asChild>
                    <Button size="sm" variant="ghost" onClick={() => onCapture(sceneCode)} className="h-5 w-5 p-0">
                      <Camera className="h-3 w-3" />
                    </Button>
                  </TooltipTrigger>
                  <TooltipContent>截图</TooltipContent>
                </Tooltip>
              )}
              {onFocus && (
                <Tooltip>
                  <TooltipTrigger asChild>
                    <Button size="sm" variant="ghost" onClick={() => onFocus(sceneCode)} className="h-5 w-5 p-0">
                      <Maximize2 className="h-3 w-3" />
                    </Button>
                  </TooltipTrigger>
                  <TooltipContent>聚焦</TooltipContent>
                </Tooltip>
              )}
            </div>
          </header>
        )}

        {/* 内容类原语 dropdown — slot 内统一展开点（HTML 设计稿 ▼） */}
        {!hideHeader && contentOpen && hasContent && (
          <div className="border-b border-border bg-muted/10 p-2">
            <ContentPrimitivesPanel primitives={contentPrimitives} />
          </div>
        )}

        {/* 主画布 */}
        <div className="relative flex-1 p-1">
          <SimSceneCanvas
            sceneCode={sceneCode}
            attachScene={attachScene}
            detachScene={detachScene}
            onDoubleClick={onFocus ? () => onFocus(sceneCode) : undefined}
          />
        </div>
      </div>
    </TooltipProvider>
  );
}

/**
 * ContentPrimitivesPanel — 把 code_block / math_formula / register_row 三类
 * "强文本/交互/a11y 必走 DOM" 的原语统一渲染为 DOM 卡片。
 * 单 slot 内多个同类原语按出现顺序竖排。
 */
function ContentPrimitivesPanel(props: { primitives: readonly Primitive[] }) {
  return (
    <div className="flex flex-col gap-2">
      {props.primitives.map(p => {
        if (p.type === 'code_block') return <CodeBlockCard key={p.id} primitive={p} />;
        if (p.type === 'math_formula') return <MathFormulaCard key={p.id} primitive={p} />;
        if (p.type === 'register_row') return <RegisterRowCard key={p.id} primitive={p} />;
        if (p.type === 'progress_bar') return <ProgressBarCard key={p.id} primitive={p} />;
        if (p.type === 'risk_gauge') return <RiskGaugeCard key={p.id} primitive={p} />;
        return null;
      })}
    </div>
  );
}

function CodeBlockCard(props: { primitive: Primitive }) {
  const params = props.primitive.params as {
    language?: string;
    code?: string;
    highlight_lines?: number[];
    title?: string;
  };
  const lines = (params.code ?? '').split('\n');
  const highlighted = new Set(params.highlight_lines ?? []);
  return (
    <div className="rounded border border-border/60 bg-background/60 p-2">
      <div className="mb-1 flex items-center justify-between text-[10px] text-muted-foreground">
        <span>code_block · {params.language ?? 'text'}{params.title ? ` · ${params.title}` : ''}</span>
        <button
          type="button"
          onClick={() => { void navigator.clipboard?.writeText(params.code ?? ''); }}
          className="inline-flex items-center gap-1 hover:text-primary"
        ><Copy className="h-3 w-3" /> 复制</button>
      </div>
      <pre className="m-0 overflow-x-auto rounded bg-muted/40 p-1 font-mono text-[10px] leading-relaxed">
        {lines.map((ln, i) => (
          <div key={i} className={cn('whitespace-pre', highlighted.has(i + 1) && 'bg-emerald-500/10 text-emerald-600')}>
            <span className="mr-2 inline-block w-5 select-none text-right text-muted-foreground">{i + 1}</span>
            {ln || ' '}
          </div>
        ))}
      </pre>
    </div>
  );
}

function MathFormulaCard(props: { primitive: Primitive }) {
  const params = props.primitive.params as { latex?: string; description?: string; label?: string };
  return (
    <div className="rounded border border-border/60 bg-background/60 p-2">
      <div className="mb-1 text-[10px] text-muted-foreground">math_formula · KaTeX{params.label ? ` · ${params.label}` : ''}</div>
      {/* 设计稿要求 KaTeX 渲染；项目暂无 KaTeX 依赖时退回到 mono 文本展示，保留协议不损失。 */}
      <div className="rounded bg-muted/40 p-1 font-mono text-[11px] italic">{params.latex ?? '（empty）'}</div>
      {params.description && (
        <div className="mt-1 text-[10px] text-muted-foreground">{params.description}</div>
      )}
    </div>
  );
}

function RegisterRowCard(props: { primitive: Primitive }) {
  const params = props.primitive.params as {
    label?: string;
    columns?: { name: string; value: string; highlight?: boolean }[];
  };
  const cols = params.columns ?? [];
  return (
    <div className="rounded border border-border/60 bg-background/60 p-2">
      <div className="mb-1 text-[10px] text-muted-foreground">register_row{params.label ? ` · ${params.label}` : ''}</div>
      <table className="w-full text-[10px]">
        <thead>
          <tr className="text-left text-muted-foreground">
            {cols.map(c => <th key={c.name} className="px-1 font-normal">{c.name}</th>)}
          </tr>
        </thead>
        <tbody>
          <tr>
            {cols.map(c => (
              <td key={c.name} className={cn('px-1 font-mono', c.highlight && 'bg-yellow-500/15 text-yellow-700')}>
                {c.value}
              </td>
            ))}
          </tr>
        </tbody>
      </table>
    </div>
  );
}

function ProgressBarCard(props: { primitive: Primitive }) {
  const params = props.primitive.params as {
    label?: string;
    current?: number;
    value?: number;
    max?: number;
    total?: number;
  };
  const cur = Number(params.current ?? params.value ?? 0);
  const max = Math.max(1, Number(params.max ?? params.total ?? 100));
  const pct = Math.min(100, Math.max(0, (cur / max) * 100));
  return (
    <div className="rounded border border-border/60 bg-background/60 p-2">
      <div className="mb-1 flex items-center justify-between text-[10px] text-muted-foreground">
        <span>progress_bar{params.label ? ` · ${params.label}` : ''}</span>
        <span className="font-mono">{cur}/{max}</span>
      </div>
      <div className="h-1.5 overflow-hidden rounded bg-muted/40">
        <div className="h-full rounded bg-primary transition-[width]" style={{ width: `${pct}%` }} />
      </div>
    </div>
  );
}

function RiskGaugeCard(props: { primitive: Primitive }) {
  const params = props.primitive.params as {
    label?: string;
    value?: number;
    level?: 'low' | 'mid' | 'high';
  };
  const level = params.level ?? (typeof params.value === 'number'
    ? (params.value >= 67 ? 'high' : params.value >= 34 ? 'mid' : 'low')
    : 'mid');
  const toneClass = level === 'high'
    ? 'text-destructive border-destructive/40'
    : level === 'low'
      ? 'text-emerald-600 border-emerald-500/40'
      : 'text-yellow-700 border-yellow-500/40';
  return (
    <div className={cn('flex items-center gap-2 rounded border bg-background/60 p-2', toneClass)}>
      <Gauge className="h-4 w-4" />
      <div className="flex-1 text-[10px]">
        <div className="text-muted-foreground">risk_gauge{params.label ? ` · ${params.label}` : ''}</div>
        <div className="font-mono font-medium">
          {level.toUpperCase()}{typeof params.value === 'number' ? ` · ${params.value}` : ''}
        </div>
      </div>
    </div>
  );
}
