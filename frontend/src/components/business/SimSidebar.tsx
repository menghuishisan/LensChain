'use client';

/**
 * SimSidebar — 右侧侧栏（06.2 §1.1, 1.2）。
 *
 * 4 个 section 按 mode 启用：
 *   • 指标（Metrics）         — A/B/C/D 都展示
 *   • 联动状态（Linkage）     — C/D 启用
 *   • 容器健康（Containers）   — D 启用
 *   • 时间线（Timeline / MicroSteps） — A/B/C/D 都展示
 *
 * width 240-280；A/B/C 默认展开，D 默认收起（由父组件控制 collapsed）。
 */

import { useMemo } from 'react';
import { Activity, Boxes, Clock3, FileCode, Link2 } from 'lucide-react';
import { Accordion, AccordionContent, AccordionItem, AccordionTrigger } from '@/components/ui/Accordion';
import { Badge } from '@/components/ui/Badge';
import { cn } from '@/lib/utils';
import type { Primitive, TimelineEvent, LinkTrigger, RenderMetric } from '@lenschain/sim-engine-renderers';
import type { SimMode } from '@/types/experiment';

export interface ContainerHealthItem {
  containerName: string;
  status: 'healthy' | 'warning' | 'error' | 'unknown';
  message?: string;
}

export interface SimSidebarProps {
  mode: SimMode;
  metrics: readonly RenderMetric[];
  timeline: readonly TimelineEvent[];
  linkTriggers: readonly LinkTrigger[];
  containerHealth?: readonly ContainerHealthItem[];
  /** 内容类原语（code_block / math_formula / register_row）——仅 single 模式下传入，
   * 多场景模式中该内容在各自 slot dropdown 展开。 */
  contentPrimitives?: readonly Primitive[];
  className?: string;
}

export function SimSidebar(props: SimSidebarProps) {
  const { mode, metrics, timeline, linkTriggers, containerHealth, contentPrimitives, className } = props;

  const showLinkage = mode === 'linkage' || mode === 'hybrid';
  const showContainers = mode === 'hybrid';
  const showContent = (contentPrimitives?.length ?? 0) > 0;

  const defaultOpen = useMemo(() => {
    const open: string[] = ['metrics', 'timeline'];
    if (showLinkage) open.push('linkage');
    if (showContainers) open.push('containers');
    if (showContent) open.push('content');
    // D 模式默认全部收起（spec 要求），其他模式默认展开
    return mode === 'hybrid' ? [] : open;
  }, [mode, showLinkage, showContainers, showContent]);

  return (
    <aside className={cn('flex w-[260px] flex-col border-l border-border bg-card', className)}>
      <Accordion type="multiple" defaultValue={defaultOpen} className="flex-1 overflow-y-auto">
        <AccordionItem value="metrics">
          <AccordionTrigger className="px-3 py-2 text-xs">
            <Activity className="mr-2 h-3.5 w-3.5" /> 指标 ({metrics.length})
          </AccordionTrigger>
          <AccordionContent className="px-3 pb-2">
            {metrics.length === 0 ? (
              <p className="text-[11px] text-muted-foreground">暂无指标数据</p>
            ) : (
              <ul className="flex flex-col gap-1.5">
                {metrics.map((m, i) => (
                  <li key={i} className="flex items-center justify-between rounded border border-border/60 px-2 py-1">
                    <span className="text-[11px] text-muted-foreground">{m.label}</span>
                    <Badge variant={m.tone === 'danger' ? 'destructive' : m.tone === 'warning' ? 'warning' : 'secondary'} className="text-[11px]">
                      {m.value}
                    </Badge>
                  </li>
                ))}
              </ul>
            )}
          </AccordionContent>
        </AccordionItem>

        {showLinkage && (
          <AccordionItem value="linkage">
            <AccordionTrigger className="px-3 py-2 text-xs">
              <Link2 className="mr-2 h-3.5 w-3.5" /> 联动状态 ({linkTriggers.length})
            </AccordionTrigger>
            <AccordionContent className="px-3 pb-2">
              {linkTriggers.length === 0 ? (
                <p className="text-[11px] text-muted-foreground">暂无联动事件</p>
              ) : (
                <ul className="flex flex-col gap-1">
                  {linkTriggers.slice(-10).reverse().map((t) => (
                    <li key={t.id} className="flex flex-col rounded border border-border/60 px-2 py-1">
                      <span className="text-[11px] font-medium">{t.source_action}</span>
                      <span className="text-[10px] text-muted-foreground">
                        {t.source_scene} · {t.link_group}
                      </span>
                      {t.changed_fields.length > 0 && (
                        <span className="text-[10px] text-muted-foreground/80">
                          字段: {t.changed_fields.join(', ')}
                        </span>
                      )}
                    </li>
                  ))}
                </ul>
              )}
            </AccordionContent>
          </AccordionItem>
        )}

        {showContainers && (
          <AccordionItem value="containers">
            <AccordionTrigger className="px-3 py-2 text-xs">
              <Boxes className="mr-2 h-3.5 w-3.5" /> 容器健康
            </AccordionTrigger>
            <AccordionContent className="px-3 pb-2">
              {!containerHealth || containerHealth.length === 0 ? (
                <p className="text-[11px] text-muted-foreground">无容器</p>
              ) : (
                <ul className="flex flex-col gap-1">
                  {containerHealth.map((c) => (
                    <li key={c.containerName} className="flex items-center justify-between rounded border border-border/60 px-2 py-1">
                      <span className="text-[11px]">{c.containerName}</span>
                      <Badge
                        variant={c.status === 'healthy' ? 'secondary' : c.status === 'warning' ? 'warning' : c.status === 'error' ? 'destructive' : 'outline'}
                        className="text-[10px]"
                      >
                        {c.status}
                      </Badge>
                    </li>
                  ))}
                </ul>
              )}
            </AccordionContent>
          </AccordionItem>
        )}

        {showContent && (
          <AccordionItem value="content">
            <AccordionTrigger className="px-3 py-2 text-xs">
              <FileCode className="mr-2 h-3.5 w-3.5" /> 内容类原语 ({contentPrimitives!.length})
            </AccordionTrigger>
            <AccordionContent className="px-3 pb-2">
              <ul className="flex flex-col gap-1.5">
                {contentPrimitives!.map(p => (
                  <li key={p.id} className="rounded border border-border/60 bg-background/40 p-1.5">
                    <div className="text-[10px] text-muted-foreground">{p.type} · {p.id}</div>
                    <pre className="m-0 mt-1 max-h-24 overflow-x-auto rounded bg-muted/40 p-1 font-mono text-[10px]">
{formatPrimitiveSummary(p)}
                    </pre>
                  </li>
                ))}
              </ul>
            </AccordionContent>
          </AccordionItem>
        )}

        <AccordionItem value="timeline">
          <AccordionTrigger className="px-3 py-2 text-xs">
            <Clock3 className="mr-2 h-3.5 w-3.5" /> 时间线 ({timeline.length})
          </AccordionTrigger>
          <AccordionContent className="px-3 pb-2">
            {timeline.length === 0 ? (
              <p className="text-[11px] text-muted-foreground">暂无事件</p>
            ) : (
              <ul className="flex flex-col gap-1">
                {timeline.slice(-15).reverse().map((e) => (
                  <li key={e.id} className="flex flex-col rounded border border-border/60 px-2 py-1">
                    <div className="flex items-center justify-between">
                      <span className="text-[11px] font-medium">{e.title}</span>
                      <span className="text-[10px] text-muted-foreground">tick {e.tick}</span>
                    </div>
                    {e.description && (
                      <span className="mt-0.5 text-[10px] text-muted-foreground">{e.description}</span>
                    )}
                  </li>
                ))}
              </ul>
            )}
          </AccordionContent>
        </AccordionItem>
      </Accordion>
    </aside>
  );
}

/** 内容类原语在 sidebar 限高环境下只呈现要点。完整交互（高亮/复制/KaTeX）在 slot dropdown 中由
 * SimSceneSlot.ContentPrimitivesPanel 提供，双路不重复。
 *
 * 支持的内容类原语（与设计稿 §1.2 内容类原语 section 对齐）：
 *   code_block / math_formula / register_row / progress_bar / risk_gauge
 */
function formatPrimitiveSummary(p: Primitive): string {
  const params = p.params as Record<string, unknown>;
  if (p.type === 'code_block') return String(params.code ?? '').split('\n').slice(0, 4).join('\n');
  if (p.type === 'math_formula') return String(params.latex ?? '');
  if (p.type === 'register_row') {
    const cols = (params.columns as { name: string; value: string }[] | undefined) ?? [];
    return cols.map(c => `${c.name}=${c.value}`).join('  ');
  }
  if (p.type === 'progress_bar') {
    const cur = String(params.current ?? params.value ?? 0);
    const max = String(params.max ?? params.total ?? 100);
    const label = params.label ? `${String(params.label)} ` : '';
    return `${label}${cur}/${max}`;
  }
  if (p.type === 'risk_gauge') {
    const v = params.value ?? params.level ?? 0;
    const label = String(params.label ?? '风险');
    return `${label}: ${String(v)}${typeof v === 'number' ? '%' : ''}`;
  }
  return JSON.stringify(params);
}
