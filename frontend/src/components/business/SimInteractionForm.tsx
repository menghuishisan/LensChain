'use client';

// SimInteractionForm.tsx
// 场景交互面板（06.2 §五）。
// 根据 ActionDef 列表按 category 分组渲染折叠区域，每个 ActionDef 按 FieldDef→shadcn/ui 映射生成表单。
// 支持 submit / immediate / hold 三种触发模式。

import { useCallback, useEffect, useRef, useState } from 'react';
import { AlertTriangle, Eye, Play, Settings } from 'lucide-react';
import { Button } from '@/components/ui/Button';
import { Input } from '@/components/ui/Input';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/Select';
import { Slider } from '@/components/ui/Slider';
import { Switch } from '@/components/ui/Switch';
import { cn } from '@/lib/utils';
import type { JsonObject, SimActionCategory, SimActionDef, SimFieldDef } from '@/types/experiment';

/** category → 配色与图标（06.2 §5.2 / API §4.2）。 */
const CATEGORY_META: Record<SimActionCategory, { label: string; icon: React.ElementType; className: string }> = {
  param_tune: { label: '调参', icon: Settings, className: 'text-blue-600 border-blue-200 bg-blue-50 dark:text-blue-300 dark:border-blue-800 dark:bg-blue-950/30' },
  attack_inject: { label: '攻击', icon: AlertTriangle, className: 'text-red-600 border-red-200 bg-red-50 dark:text-red-300 dark:border-red-800 dark:bg-red-950/30' },
  primary: { label: '操作', icon: Play, className: 'text-green-600 border-green-200 bg-green-50 dark:text-green-300 dark:border-green-800 dark:bg-green-950/30' },
  observe: { label: '观察', icon: Eye, className: 'text-gray-600 border-gray-200 bg-gray-50 dark:text-gray-300 dark:border-gray-700 dark:bg-gray-900/30' },
};

export interface SimInteractionFormProps {
  actions: SimActionDef[];
  connected: boolean;
  onSubmit: (actionCode: string, params: JsonObject) => void;
  className?: string;
}

/**
 * SimInteractionForm 渲染全部可执行 ActionDef（06.2 §五）。
 * 按 category 分组，每组用折叠面板包裹，内部每个 ActionDef 渲染为子表单。
 */
export function SimInteractionForm({ actions, connected, onSubmit, className }: SimInteractionFormProps) {
  if (actions.length === 0) return null;

  const grouped = groupByCategory(actions);

  return (
    <div className={cn('space-y-3 px-4 py-3 border-t', className)}>
      <p className="text-xs font-medium text-muted-foreground">交互操作</p>
      {Object.entries(grouped).map(([cat, acts]) => {
        const meta = CATEGORY_META[cat as SimActionCategory] ?? CATEGORY_META.primary;
        const Icon = meta.icon;
        return (
          <div key={cat} className={cn('rounded-lg border p-3', meta.className)}>
            <div className="flex items-center gap-1.5 mb-2">
              <Icon className="h-3.5 w-3.5" />
              <span className="text-xs font-medium">{meta.label}</span>
            </div>
            <div className="space-y-3">
              {acts.map((action) => (
                <ActionBlock key={action.action_code} action={action} connected={connected} onSubmit={onSubmit} />
              ))}
            </div>
          </div>
        );
      })}
    </div>
  );
}

/** ActionBlock 单个 ActionDef 渲染（含字段集 + 触发按钮）。 */
function ActionBlock({
  action,
  connected,
  onSubmit,
}: {
  action: SimActionDef;
  connected: boolean;
  onSubmit: (actionCode: string, params: JsonObject) => void;
}) {
  const [values, setValues] = useState<Record<string, unknown>>(() => buildDefaults(action.fields));
  const [confirming, setConfirming] = useState(false);
  const [cooldown, setCooldown] = useState(false);
  const cooldownTimer = useRef<number | null>(null);

  const setValue = useCallback((name: string, value: unknown) => {
    setValues((prev) => ({ ...prev, [name]: value }));
  }, []);

  const handleSubmit = useCallback(() => {
    if (action.category === 'attack_inject' && !confirming) {
      setConfirming(true);
      return;
    }
    setConfirming(false);
    onSubmit(action.action_code, values as JsonObject);

    if (action.cooldown_ms && action.cooldown_ms > 0) {
      setCooldown(true);
      cooldownTimer.current = window.setTimeout(() => setCooldown(false), action.cooldown_ms);
    }
  }, [action, confirming, onSubmit, values]);

  const handleImmediateChange = useCallback(
    (name: string, value: unknown) => {
      setValue(name, value);
      onSubmit(action.action_code, { ...values, [name]: value } as JsonObject);
    },
    [action, onSubmit, setValue, values],
  );

  useEffect(() => {
    return () => {
      if (cooldownTimer.current !== null) window.clearTimeout(cooldownTimer.current);
    };
  }, []);

  const isDisabled = !connected || cooldown;

  return (
    <div className="space-y-2">
      <div>
        <p className="text-sm font-medium">{action.label}</p>
        {action.description && <p className="text-xs text-muted-foreground">{action.description}</p>}
      </div>

      {action.fields.map((field) => (
        <FieldInput
          key={field.name}
          field={field}
          value={values[field.name]}
          onChange={(val) => (action.trigger === 'immediate' ? handleImmediateChange(field.name, val) : setValue(field.name, val))}
          disabled={isDisabled}
        />
      ))}

      {action.trigger === 'submit' && (
        <div className="flex items-center gap-2">
          {confirming ? (
            <>
              <Button variant="destructive" size="sm" className="h-7 text-xs" disabled={isDisabled} onClick={handleSubmit}>
                确认执行
              </Button>
              <Button variant="outline" size="sm" className="h-7 text-xs" onClick={() => setConfirming(false)}>
                取消
              </Button>
            </>
          ) : (
            <Button
              variant={action.category === 'attack_inject' ? 'destructive' : 'primary'}
              size="sm"
              className="h-7 text-xs"
              disabled={isDisabled}
              onClick={handleSubmit}
            >
              {cooldown ? '冷却中...' : action.label}
            </Button>
          )}
        </div>
      )}
    </div>
  );
}

/** FieldInput 单个 FieldDef → shadcn/ui 映射（06.2 §5.3 表格）。 */
function FieldInput({
  field,
  value,
  onChange,
  disabled,
}: {
  field: SimFieldDef;
  value: unknown;
  onChange: (value: unknown) => void;
  disabled: boolean;
}) {
  switch (field.type) {
    case 'string':
      return (
        <div className="space-y-1">
          <label className="text-xs text-muted-foreground">{field.label}</label>
          <Input
            className="h-7 text-xs"
            value={String(value ?? '')}
            onChange={(e) => onChange(e.target.value)}
            disabled={disabled}
          />
        </div>
      );

    case 'number':
      return (
        <div className="space-y-1">
          <label className="text-xs text-muted-foreground">{field.label}</label>
          <Input
            type="number"
            className="h-7 text-xs"
            value={String(value ?? '')}
            min={field.min}
            max={field.max}
            step={field.step}
            onChange={(e) => onChange(Number(e.target.value))}
            disabled={disabled}
          />
        </div>
      );

    case 'boolean':
      return (
        <div className="flex items-center gap-2">
          <Switch checked={Boolean(value)} onCheckedChange={onChange} disabled={disabled} />
          <label className="text-xs">{field.label}</label>
        </div>
      );

    case 'select':
    case 'enum':
      return (
        <div className="space-y-1">
          <label className="text-xs text-muted-foreground">{field.label}</label>
          <Select value={String(value ?? '')} onValueChange={onChange} disabled={disabled}>
            <SelectTrigger className="h-7 text-xs">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {(field.options ?? []).map((opt) => (
                <SelectItem key={opt.value} value={opt.value}>
                  {opt.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      );

    case 'range':
      return (
        <div className="space-y-1">
          <div className="flex items-center justify-between">
            <label className="text-xs text-muted-foreground">{field.label}</label>
            <span className="text-xs font-medium">{String(value ?? field.default ?? field.min ?? 0)}</span>
          </div>
          <Slider
            min={field.min}
            max={field.max}
            step={field.step}
            value={[Number(value ?? field.default ?? field.min ?? 0)]}
            onValueChange={([v]) => onChange(v)}
            disabled={disabled}
          />
        </div>
      );

    case 'multi_select':
      return (
        <div className="space-y-1">
          <label className="text-xs text-muted-foreground">{field.label}</label>
          <div className="flex flex-wrap gap-1">
            {(field.options ?? []).map((opt) => {
              const selected = Array.isArray(value) && (value as string[]).includes(opt.value);
              return (
                <Button
                  key={opt.value}
                  variant={selected ? 'secondary' : 'outline'}
                  size="sm"
                  className="h-6 px-2 text-xs"
                  disabled={disabled}
                  onClick={() => {
                    const current = Array.isArray(value) ? (value as string[]) : [];
                    onChange(selected ? current.filter((v) => v !== opt.value) : [...current, opt.value]);
                  }}
                >
                  {opt.label}
                </Button>
              );
            })}
          </div>
        </div>
      );

    case 'json':
      return (
        <div className="space-y-1">
          <label className="text-xs text-muted-foreground">{field.label}</label>
          <textarea
            className="w-full rounded-md border bg-background px-2 py-1 text-xs font-mono resize-y min-h-[60px]"
            value={typeof value === 'string' ? value : JSON.stringify(value ?? {}, null, 2)}
            onChange={(e) => {
              try { onChange(JSON.parse(e.target.value)); } catch { onChange(e.target.value); }
            }}
            disabled={disabled}
          />
        </div>
      );

    default:
      return null;
  }
}

/** groupByCategory 按 category 分组 ActionDef。 */
function groupByCategory(actions: SimActionDef[]) {
  const groups: Partial<Record<SimActionCategory, SimActionDef[]>> = {};
  for (const action of actions) {
    const cat = action.category;
    if (!groups[cat]) groups[cat] = [];
    groups[cat]!.push(action);
  }
  return groups;
}

/** buildDefaults 构建字段默认值映射。 */
function buildDefaults(fields: SimFieldDef[]): Record<string, unknown> {
  const defaults: Record<string, unknown> = {};
  for (const f of fields) {
    if (f.default !== undefined) {
      defaults[f.name] = f.default;
    } else if (f.type === 'boolean') {
      defaults[f.name] = false;
    } else if (f.type === 'multi_select') {
      defaults[f.name] = [];
    } else if (f.type === 'number' || f.type === 'range') {
      defaults[f.name] = f.min ?? 0;
    } else {
      defaults[f.name] = '';
    }
  }
  return defaults;
}
