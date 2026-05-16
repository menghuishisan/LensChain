'use client';

/**
 * SimInteractionForm — InteractionDefinition 渲染器（06.2 §五）。
 *
 * 把单场景的 InteractionSchema.actions 渲染为一组按钮 + 展开字段表单。
 * 处理三种 trigger：submit（展开 → 填字段 → 提交）/ immediate（点击即触发）/ hold（按住持续）。
 * 字段 type → shadcn/ui 组件映射见 06.2 §5.3。
 *
 * 不写兑底：未知 trigger / category / type 抛错，用 ConfirmDialog 处理 attack_inject 二次确认。
 */

import { useCallback, useMemo, useState } from 'react';
import { ChevronDown, ChevronRight, Container, Eye, Link2, ShieldAlert, Sliders, ZapIcon } from 'lucide-react';
import {
  validateInputs,
  type InteractionAction,
  type InteractionField,
  type InteractionInputMap,
  type InteractionSchema,
  type JsonObject,
} from '@lenschain/sim-engine-renderers';
import { Button } from '@/components/ui/Button';
import { Input } from '@/components/ui/Input';
import { Switch } from '@/components/ui/Switch';
import { Slider } from '@/components/ui/Slider';
import { Textarea } from '@/components/ui/Textarea';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/Select';
import { Badge } from '@/components/ui/Badge';
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/Tooltip';
import { ConfirmDialog } from '@/components/ui/ConfirmDialog';
import { cn } from '@/lib/utils';

export interface SimInteractionFormProps {
  sceneCode: string;
  schema: InteractionSchema | null;
  /** 当前用户 role；用于过滤 ActionDef.roles。 */
  userRole: 'student' | 'teacher';
  /** 提交回调；buildSimAction 后由上层走 panel.dispatchAction。 */
  onSubmit: (sceneCode: string, actionCode: string, params: JsonObject) => void;
  className?: string;
}

const CATEGORY_STYLE = {
  param_tune: { Icon: Sliders, label: '调参', accent: 'bg-blue-500/10 text-blue-600 border-blue-500/30' },
  attack_inject: { Icon: ShieldAlert, label: '攻击', accent: 'bg-red-500/10 text-red-600 border-red-500/30' },
  primary: { Icon: ZapIcon, label: '操作', accent: 'bg-emerald-500/10 text-emerald-600 border-emerald-500/30' },
  observe: { Icon: Eye, label: '观察', accent: 'bg-muted text-muted-foreground border-border' },
} as const;

export function SimInteractionForm(props: SimInteractionFormProps) {
  const { sceneCode, schema, userRole, onSubmit, className } = props;

  const visibleActions = useMemo(() => {
    if (!schema) return [];
    return schema.actions.filter(a => a.roles.includes(userRole));
  }, [schema, userRole]);

  if (!schema || visibleActions.length === 0) return null;

  return (
    <div className={cn('rounded-md border border-border bg-card p-2', className)}>
      <div className="mb-1 flex items-center gap-1 text-xs text-muted-foreground">
        <span>🎮 操作面板</span>
        <span className="opacity-50">·</span>
        <span className="opacity-70">{schema.actions.length} 项</span>
      </div>
      <div className="flex flex-wrap gap-2">
        {visibleActions.map(action => (
          <ActionItem
            key={action.actionCode}
            sceneCode={sceneCode}
            action={action}
            onSubmit={onSubmit}
          />
        ))}
      </div>
    </div>
  );
}

function ActionItem(props: {
  sceneCode: string;
  action: InteractionAction;
  onSubmit: (sceneCode: string, actionCode: string, params: JsonObject) => void;
}) {
  const { sceneCode, action, onSubmit } = props;
  const [expanded, setExpanded] = useState(false);
  const [inputs, setInputs] = useState<InteractionInputMap>(() => initialInputs(action));
  const [pendingConfirm, setPendingConfirm] = useState(false);
  const [cooldownEndsAt, setCooldownEndsAt] = useState<number | null>(null);

  const isCoolingDown = cooldownEndsAt !== null && Date.now() < cooldownEndsAt;
  const issues = useMemo(() => validateInputs(action, inputs), [action, inputs]);
  const canSubmit = issues.length === 0 && !isCoolingDown;

  const triggerSubmit = useCallback(() => {
    const params: JsonObject = {};
    for (const f of action.fields) {
      const v = inputs[f.key];
      if (v === undefined) continue;
      params[f.key] = v as JsonObject[keyof JsonObject];
    }
    onSubmit(sceneCode, action.actionCode, params);
    setExpanded(false);
    if (action.cooldownMs && action.cooldownMs > 0) {
      setCooldownEndsAt(Date.now() + action.cooldownMs);
      setTimeout(() => setCooldownEndsAt(null), action.cooldownMs);
    }
  }, [sceneCode, action, inputs, onSubmit]);

  const handlePrimaryClick = useCallback(() => {
    if (isCoolingDown) return;
    if (action.category === 'attack_inject') {
      setPendingConfirm(true);
      return;
    }
    if (action.trigger === 'immediate') {
      triggerSubmit();
    } else if (action.trigger === 'submit') {
      setExpanded(v => !v);
    } else if (action.trigger === 'hold') {
      // hold 模式：按住按钮持续触发；mousedown/mouseup 在 ButtonHold 处理
      // 此分支由独立按钮处理，不进入此 onClick。
      throw new Error(`SimInteractionForm: hold trigger 应由 ButtonHold 处理 (${action.actionCode})`);
    } else {
      throw new Error(`SimInteractionForm: 未知 trigger "${(action.trigger as string)}"`);
    }
  }, [action, isCoolingDown, triggerSubmit]);

  const style = CATEGORY_STYLE[action.category as keyof typeof CATEGORY_STYLE];
  if (!style) {
    throw new Error(`SimInteractionForm: 未知 category "${action.category}"`);
  }
  const { Icon, label: catLabel, accent } = style;

  return (
    <div className="flex flex-col">
      <TooltipProvider>
        <div className="flex items-center gap-1">
          {action.trigger === 'hold' ? (
            <ButtonHold action={action} disabled={isCoolingDown} onTrigger={triggerSubmit} accent={accent} Icon={Icon} catLabel={catLabel} />
          ) : (
            <Button
              size="sm"
              variant="outline"
              disabled={isCoolingDown}
              onClick={handlePrimaryClick}
              className={cn('h-7 gap-1 text-xs', accent)}
            >
              <Icon className="h-3.5 w-3.5" />
              <span>[{catLabel}] {action.label}</span>
              {action.trigger === 'submit' && (
                expanded ? <ChevronDown className="h-3 w-3" /> : <ChevronRight className="h-3 w-3" />
              )}
            </Button>
          )}
          {action.writesOwnedFields && action.writesOwnedFields.length > 0 && (
            <Tooltip>
              <TooltipTrigger asChild>
                <Badge variant="secondary" className="h-5 gap-0.5 px-1 text-[10px]">
                  <Link2 className="h-2.5 w-2.5" />
                </Badge>
              </TooltipTrigger>
              <TooltipContent>
                联动写入：{action.writesOwnedFields.join(', ')}
              </TooltipContent>
            </Tooltip>
          )}
          {action.hybridChannel === 'container' && (
            <Tooltip>
              <TooltipTrigger asChild>
                <Badge variant="warning" className="h-5 px-1 text-[10px]"><Container className="h-2.5 w-2.5" /></Badge>
              </TooltipTrigger>
              <TooltipContent>真链通道：执行后跳 Web Terminal</TooltipContent>
            </Tooltip>
          )}
          {isCoolingDown && cooldownEndsAt && (
            <span className="text-[10px] text-muted-foreground">
              {Math.ceil((cooldownEndsAt - Date.now()) / 1000)}s
            </span>
          )}
        </div>
      </TooltipProvider>

      {expanded && action.trigger === 'submit' && (
        <div className="mt-2 flex flex-col gap-2 rounded border border-border/60 bg-muted/30 p-2">
          {action.description && (
            <p className="text-[11px] text-muted-foreground">{action.description}</p>
          )}
          {action.fields.map(field => (
            <FieldRenderer
              key={field.key}
              field={field}
              value={inputs[field.key]}
              onChange={(v) => setInputs(prev => ({ ...prev, [field.key]: v as InteractionInputMap[string] }))}
            />
          ))}
          {issues.length > 0 && (
            <ul className="text-[10px] text-destructive">
              {issues.map(i => <li key={i.fieldKey}>· {i.message}</li>)}
            </ul>
          )}
          <div className="flex justify-end gap-1">
            <Button size="sm" variant="ghost" onClick={() => setExpanded(false)} className="h-6 text-xs">取消</Button>
            <Button size="sm" variant="primary" disabled={!canSubmit} onClick={triggerSubmit} className="h-6 text-xs">提交</Button>
          </div>
        </div>
      )}

      <ConfirmDialog
        open={pendingConfirm}
        onOpenChange={setPendingConfirm}
        title="二次确认：注入攻击"
        description={`即将执行 ${action.label}。攻击行为可能造成共识异常，确定继续？`}
        confirmText="继续"
        confirmVariant="destructive"
        onConfirm={() => {
          setPendingConfirm(false);
          if (action.trigger === 'immediate') triggerSubmit();
          else setExpanded(true);
        }}
      />
    </div>
  );
}

function ButtonHold(props: {
  action: InteractionAction;
  disabled: boolean;
  onTrigger: () => void;
  accent: string;
  Icon: typeof ChevronDown;
  catLabel: string;
}) {
  const { action, disabled, onTrigger, accent, Icon, catLabel } = props;
  const [holding, setHolding] = useState(false);
  // 简单实现：mousedown/touchstart 启动一个定时器 100ms 触发一次；mouseup/leave 停止。
  // 真实持续触发的频率交由后端节流；前端 100ms 一次足够。
  const intervalRef = useState<{ id: ReturnType<typeof setInterval> | null }>({ id: null })[0];

  const start = () => {
    if (disabled || holding) return;
    setHolding(true);
    onTrigger();
    intervalRef.id = setInterval(onTrigger, 100);
  };
  const stop = () => {
    if (!holding) return;
    setHolding(false);
    if (intervalRef.id) { clearInterval(intervalRef.id); intervalRef.id = null; }
  };

  return (
    <Button
      size="sm"
      variant={holding ? 'primary' : 'outline'}
      disabled={disabled}
      onMouseDown={start}
      onMouseUp={stop}
      onMouseLeave={stop}
      onTouchStart={start}
      onTouchEnd={stop}
      className={cn('h-7 gap-1 text-xs', accent)}
    >
      <Icon className="h-3.5 w-3.5" />
      <span>[{catLabel}] {action.label}（按住）</span>
    </Button>
  );
}

function FieldRenderer(props: {
  field: InteractionField;
  value: unknown;
  onChange: (v: unknown) => void;
}) {
  const { field, value, onChange } = props;
  const labelEl = (
    <label className="text-[11px] font-medium text-foreground">
      {field.label}{field.required && <span className="text-destructive">*</span>}
    </label>
  );

  switch (field.type) {
    case 'string':
      return (
        <div className="flex flex-col gap-1">
          {labelEl}
          <Input value={(value as string | undefined) ?? ''} onChange={e => onChange(e.target.value)} className="h-7 text-xs" />
        </div>
      );
    case 'number':
      return (
        <div className="flex flex-col gap-1">
          {labelEl}
          <Input
            type="number"
            value={(value as number | undefined) ?? ''}
            onChange={e => onChange(e.target.value === '' ? undefined : Number(e.target.value))}
            className="h-7 text-xs"
          />
        </div>
      );
    case 'boolean':
      return (
        <div className="flex items-center gap-2">
          {labelEl}
          <Switch checked={Boolean(value)} onCheckedChange={onChange} />
        </div>
      );
    case 'select':
    case 'enum': {
      const options = field.options ?? [];
      return (
        <div className="flex flex-col gap-1">
          {labelEl}
          <Select value={String(value ?? '')} onValueChange={(v: string) => onChange(v)}>
            <SelectTrigger className="h-7 text-xs"><SelectValue /></SelectTrigger>
            <SelectContent>
              {options.map(opt => (
                <SelectItem key={String(opt.value)} value={String(opt.value)}>{opt.label}</SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      );
    }
    case 'range':
      return (
        <div className="flex flex-col gap-1">
          {labelEl}
          <div className="flex items-center gap-2">
            <Slider
              value={[Number(value ?? 0)]}
              onValueChange={(v: number[]) => onChange(v[0])}
              min={0}
              max={100}
              step={1}
              className="flex-1"
            />
            <span className="w-8 text-right text-[11px] text-muted-foreground">{String(value ?? 0)}</span>
          </div>
        </div>
      );
    case 'multi_select': {
      const options = field.options ?? [];
      const arr = (Array.isArray(value) ? value : []) as (string | number)[];
      return (
        <div className="flex flex-col gap-1">
          {labelEl}
          <div className="flex flex-wrap gap-1">
            {options.map(opt => {
              const active = arr.some(v => String(v) === String(opt.value));
              return (
                <Button
                  key={String(opt.value)}
                  size="sm"
                  variant={active ? 'primary' : 'outline'}
                  className="h-6 px-2 text-xs"
                  onClick={() => {
                    if (active) onChange(arr.filter(v => String(v) !== String(opt.value)));
                    else onChange([...arr, opt.value]);
                  }}
                >
                  {opt.label}
                </Button>
              );
            })}
          </div>
        </div>
      );
    }
    case 'json':
      return (
        <div className="flex flex-col gap-1">
          {labelEl}
          <Textarea
            value={typeof value === 'string' ? value : JSON.stringify((value ?? {}) as unknown, null, 2)}
            onChange={e => {
              const text = e.target.value;
              try { onChange(JSON.parse(text)); }
              catch { onChange(text); /* 暂存原始字符串，提交时 validate 报错 */ }
            }}
            className="h-20 text-xs font-mono"
          />
        </div>
      );
    default:
      throw new Error(`SimInteractionForm.FieldRenderer: 未知 FieldType "${(field.type as string)}"`);
  }
}

function initialInputs(action: InteractionAction): InteractionInputMap {
  const inputs: InteractionInputMap = {};
  for (const f of action.fields) {
    if (f.defaultValue !== undefined) inputs[f.key] = f.defaultValue as InteractionInputMap[string];
  }
  return inputs;
}
