'use client';

/**
 * SimSharedStatePanel — 联动 SharedState 内联面板（06.2 §6.3 / 设计稿 ④）。
 *
 * 联动模式下取代右侧 SimSidebar 位置（margin-right:+360px 让位），不再使用 Sheet 抽屉。
 * 每 LinkGroup 一个手风琴 section；每字段一行 [字段名 / 值 / owner]。
 * 字段值变化 1 秒黄色高亮闪烁。
 */

import { useState, useEffect, useRef } from 'react';
import { Database } from 'lucide-react';
import { Accordion, AccordionContent, AccordionItem, AccordionTrigger } from '@/components/ui/Accordion';
import { Badge } from '@/components/ui/Badge';
import { cn } from '@/lib/utils';
import type { SimSharedStateGroup } from '@/types/experiment';

export interface SimSharedStatePanelProps {
  groups: readonly SimSharedStateGroup[];
  className?: string;
}

export function SimSharedStatePanel(props: SimSharedStatePanelProps) {
  const { groups, className } = props;
  return (
    <div className={cn('flex h-full w-[360px] flex-col overflow-y-auto border-l border-primary/40 bg-card', className)}>
      <header className="flex items-center gap-1.5 border-b border-border bg-muted/30 px-3 py-2 text-xs font-medium">
        <Database className="h-3.5 w-3.5" /> 共享状态（联动）
      </header>
      <div className="flex-1 overflow-y-auto p-2">
        {groups.length === 0 ? (
          <p className="px-2 py-3 text-xs text-muted-foreground">无活跃联动组</p>
        ) : (
          <Accordion type="multiple" defaultValue={groups.map(g => String(g.link_group_id))}>
            {groups.map(group => (
              <AccordionItem key={group.link_group_id} value={String(group.link_group_id)}>
                <AccordionTrigger className="px-2 py-2 text-xs">{group.link_group_name}</AccordionTrigger>
                <AccordionContent className="px-2 pb-2">
                  <ul className="flex flex-col gap-1">
                    {group.fields.map(f => (
                      <SharedStateRow key={f.field_name} name={f.field_name} value={f.value} owner={f.owner_scene} />
                    ))}
                  </ul>
                </AccordionContent>
              </AccordionItem>
            ))}
          </Accordion>
        )}
      </div>
    </div>
  );
}

function SharedStateRow(props: { name: string; value: unknown; owner: string }) {
  const { name, value, owner } = props;
  const [flash, setFlash] = useState(false);
  const lastValueRef = useRef(value);

  useEffect(() => {
    if (lastValueRef.current !== value) {
      setFlash(true);
      lastValueRef.current = value;
      const t = setTimeout(() => setFlash(false), 1000);
      return () => clearTimeout(t);
    }
    return undefined;
  }, [value]);

  return (
    <li className={cn(
      'flex items-center justify-between gap-2 rounded border border-border/60 px-2 py-1 transition-colors',
      flash && 'bg-yellow-500/20',
    )}>
      <div className="flex flex-col">
        <span className="text-[11px] font-medium">{name}</span>
        <span className="text-[10px] text-muted-foreground">owner: {owner}</span>
      </div>
      <Badge variant="secondary" className="text-[10px] font-mono">{formatValue(value)}</Badge>
    </li>
  );
}

function formatValue(v: unknown): string {
  if (v === null || v === undefined) return 'null';
  if (typeof v === 'string') return v;
  if (typeof v === 'number' || typeof v === 'boolean') return String(v);
  try { return JSON.stringify(v); } catch { return '[unserializable]'; }
}
