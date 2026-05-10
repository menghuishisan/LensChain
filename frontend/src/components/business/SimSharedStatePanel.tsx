'use client';

// SimSharedStatePanel.tsx
// 联动模式 SharedState 面板（06.2 §6.3）。
// 手风琴样式按联动组分组，每组列出全部共享字段及 owner 场景。
// 仅 mode=linkage 时挂载。

import { Accordion, AccordionContent, AccordionItem, AccordionTrigger } from '@/components/ui/Accordion';
import { Badge } from '@/components/ui/Badge';
import { cn } from '@/lib/utils';
import type { SimSharedStateGroup } from '@/types/experiment';

export interface SimSharedStatePanelProps {
  groups: SimSharedStateGroup[];
  className?: string;
}

/**
 * SimSharedStatePanel 展示联动组的共享状态（06.2 §6.3）。
 * 默认展开第一组，按字段名列表渲染值和 owner 场景标签。
 */
export function SimSharedStatePanel({ groups, className }: SimSharedStatePanelProps) {
  if (groups.length === 0) {
    return (
      <div className={cn('px-3 py-4 text-xs text-muted-foreground text-center', className)}>
        暂无联动共享数据
      </div>
    );
  }

  return (
    <div className={cn('border-t', className)}>
      <div className="px-3 py-2 border-b">
        <p className="text-xs font-medium text-muted-foreground">SharedState 联动数据</p>
      </div>
      <Accordion type="single" collapsible defaultValue={groups[0]?.link_group_id}>
        {groups.map((group) => (
          <AccordionItem key={group.link_group_id} value={group.link_group_id}>
            <AccordionTrigger className="px-3 py-2 text-xs">
              <span className="flex items-center gap-2">
                {group.link_group_name}
                <Badge variant="outline" className="text-[10px] h-4">
                  {group.fields.length} 字段
                </Badge>
              </span>
            </AccordionTrigger>
            <AccordionContent className="px-3">
              <div className="space-y-2">
                {group.fields.map((field) => (
                  <div key={field.field_name} className="rounded border px-2 py-1.5">
                    <div className="flex items-center justify-between mb-1">
                      <span className="text-xs font-mono font-medium">{field.field_name}</span>
                      <Badge variant="outline" className="text-[10px] h-4">
                        {field.owner_scene_label}
                      </Badge>
                    </div>
                    <pre className="text-[11px] text-muted-foreground font-mono whitespace-pre-wrap break-all max-h-20 overflow-y-auto">
                      {typeof field.value === 'string' ? field.value : JSON.stringify(field.value, null, 2)}
                    </pre>
                  </div>
                ))}
              </div>
            </AccordionContent>
          </AccordionItem>
        ))}
      </Accordion>
    </div>
  );
}
