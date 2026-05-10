'use client';

// SimConfigWizard.tsx
// 教师实验配置 SimEngine 子向导（06.2 §八）。
// 5 步：① 选择场景 → ② 配置参数 → ③ 联动组 → ④ 多场景布局 → ⑤ 预览。
// 嵌入 P-21 ExperimentTemplateEditorPanel step=3 sub=sim-config。

import { useCallback, useMemo, useState } from 'react';
import {
  Check,
  ChevronLeft,
  ChevronRight,
  Eye,
  GripVertical,
  Link2,
  ListChecks,
  LayoutGrid,
  Settings2,
} from 'lucide-react';
import { Badge } from '@/components/ui/Badge';
import { Button } from '@/components/ui/Button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/Card';
import { Input } from '@/components/ui/Input';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/Select';
import { Switch } from '@/components/ui/Switch';
import { Accordion, AccordionContent, AccordionItem, AccordionTrigger } from '@/components/ui/Accordion';
import { cn } from '@/lib/utils';
import type { ID } from '@/types/api';
import type {
  SimScenarioCatalogItem,
  SimLinkGroupDetail,
  JsonObject,
} from '@/types/experiment';

// ─── Types ─────────────────────────────────────────────────
export interface SelectedScene {
  scenarioId: ID;
  code: string;
  name: string;
  category: string;
  timeControlMode: string;
  params: JsonObject;
  defaultParams: JsonObject;
}

export interface LinkGroupConfig {
  groupId: ID;
  groupName: string;
  enabled: boolean;
  scenes: string[];
  fields: Array<{ field_name: string; owner_scene: string }>;
}

export interface SceneLayoutConfig {
  sceneCode: string;
  title: string;
  layoutRole: 'primary' | 'secondary' | 'auxiliary';
  displayMode: string;
  linkToPrimary: boolean;
  defaultVisible: boolean;
}

export interface SimConfigWizardProps {
  scenarios: SimScenarioCatalogItem[];
  linkGroups: SimLinkGroupDetail[];
  initialScenes?: SelectedScene[];
  onPublish: (config: SimConfigResult) => void;
  onCancel?: () => void;
  className?: string;
}

export interface SimConfigResult {
  scenes: SelectedScene[];
  linkGroupConfigs: LinkGroupConfig[];
  layoutConfigs: SceneLayoutConfig[];
  defaultLayout: 'grid' | 'focus' | 'carousel';
}

const STEPS = [
  { id: 'scenes', label: '选择场景', icon: ListChecks },
  { id: 'params', label: '配置参数', icon: Settings2 },
  { id: 'linkage', label: '联动组', icon: Link2 },
  { id: 'layout', label: '多场景布局', icon: LayoutGrid },
  { id: 'preview', label: '预览', icon: Eye },
] as const;

type StepId = (typeof STEPS)[number]['id'];

const CATEGORY_LABELS: Record<string, string> = {
  node_network: '节点网络',
  consensus: '共识机制',
  cryptography: '密码学',
  data_structure: '数据结构',
  transaction: '交易流程',
  smart_contract: '智能合约',
  attack_security: '攻击安全',
  economic: '经济模型',
  generic: '教师扩展',
};

/**
 * SimConfigWizard 5 步 SimEngine 配置向导（06.2 §八）。
 */
export function SimConfigWizard({
  scenarios,
  linkGroups,
  initialScenes = [],
  onPublish,
  onCancel,
  className,
}: SimConfigWizardProps) {
  const [step, setStep] = useState<StepId>('scenes');
  const [selectedScenes, setSelectedScenes] = useState<SelectedScene[]>(initialScenes);
  const [linkGroupConfigs, setLinkGroupConfigs] = useState<LinkGroupConfig[]>([]);
  const [layoutConfigs, setLayoutConfigs] = useState<SceneLayoutConfig[]>([]);
  const [defaultLayout, setDefaultLayout] = useState<'grid' | 'focus' | 'carousel'>('grid');
  const [categoryFilter, setCategoryFilter] = useState('all');

  const stepIndex = STEPS.findIndex((s) => s.id === step);
  const canNext = stepIndex < STEPS.length - 1;
  const canPrev = stepIndex > 0;

  const goNext = () => {
    if (!canNext) return;
    const nextStep = STEPS[stepIndex + 1].id;
    // entering linkage step: auto-detect available link groups
    if (nextStep === 'linkage') {
      detectLinkGroups();
    }
    // entering layout step: auto-init layout configs
    if (nextStep === 'layout') {
      initLayoutConfigs();
    }
    setStep(nextStep);
  };

  const goPrev = () => {
    if (!canPrev) return;
    setStep(STEPS[stepIndex - 1].id);
  };

  // ─── Step ① 场景选择 ────────────────────────────
  const toggleScene = useCallback(
    (scenario: SimScenarioCatalogItem) => {
      setSelectedScenes((prev) => {
        const exists = prev.find((s) => s.scenarioId === scenario.id);
        if (exists) return prev.filter((s) => s.scenarioId !== scenario.id);
        return [
          ...prev,
          {
            scenarioId: scenario.id,
            code: scenario.code,
            name: scenario.name,
            category: scenario.category,
            timeControlMode: scenario.time_control_mode,
            params: { ...(scenario.default_params ?? {}) },
            defaultParams: { ...(scenario.default_params ?? {}) },
          },
        ];
      });
    },
    [],
  );

  const filteredScenarios = useMemo(
    () =>
      categoryFilter === 'all'
        ? scenarios
        : scenarios.filter((s) => s.category === categoryFilter),
    [scenarios, categoryFilter],
  );

  // ─── Step ② 参数 ─────────────────────────────────
  const updateParam = useCallback((scenarioId: ID, key: string, value: unknown) => {
    setSelectedScenes((prev) =>
      prev.map((s) =>
        s.scenarioId === scenarioId
          ? { ...s, params: { ...s.params, [key]: value as string | number | boolean } }
          : s,
      ),
    );
  }, []);

  const resetParams = useCallback((scenarioId: ID) => {
    setSelectedScenes((prev) =>
      prev.map((s) =>
        s.scenarioId === scenarioId ? { ...s, params: { ...s.defaultParams } } : s,
      ),
    );
  }, []);

  // ─── Step ③ 联动组自动检测 ─────────────────────
  const detectLinkGroups = useCallback(() => {
    const selectedCodes = new Set(selectedScenes.map((s) => s.code));
    const configs: LinkGroupConfig[] = linkGroups
      .map((group) => {
        const groupSceneCodes = (group.scenes ?? []).map((s) => s.scene_code);
        const matchingScenes = groupSceneCodes.filter((code) => selectedCodes.has(code));
        return {
          groupId: group.id,
          groupName: group.name,
          enabled: matchingScenes.length >= 2,
          scenes: matchingScenes,
          fields: (group.schema_fields ?? []).map((f) => ({ field_name: f.field_path, owner_scene: f.owner_scene_code })),
        };
      })
      .filter((c) => c.scenes.length >= 1);
    setLinkGroupConfigs(configs);
  }, [selectedScenes, linkGroups]);

  const toggleLinkGroup = useCallback((groupId: ID) => {
    setLinkGroupConfigs((prev) =>
      prev.map((c) => (c.groupId === groupId ? { ...c, enabled: !c.enabled } : c)),
    );
  }, []);

  // ─── Step ④ 布局初始化 ──────────────────────────
  const initLayoutConfigs = useCallback(() => {
    setLayoutConfigs(
      selectedScenes.map((s, i) => ({
        sceneCode: s.code,
        title: s.name,
        layoutRole: i === 0 ? 'primary' : 'secondary',
        displayMode: selectedScenes.length <= 2 ? 'split-2' : 'grid-4',
        linkToPrimary: linkGroupConfigs.some((c) => c.enabled && c.scenes.includes(s.code)),
        defaultVisible: true,
      })),
    );
  }, [selectedScenes, linkGroupConfigs]);

  const updateLayoutConfig = useCallback(
    (sceneCode: string, patch: Partial<SceneLayoutConfig>) => {
      setLayoutConfigs((prev) =>
        prev.map((c) => (c.sceneCode === sceneCode ? { ...c, ...patch } : c)),
      );
    },
    [],
  );

  // ─── Step ⑤ 发布 ────────────────────────────────
  const handlePublish = () => {
    onPublish({
      scenes: selectedScenes,
      linkGroupConfigs: linkGroupConfigs.filter((c) => c.enabled),
      layoutConfigs,
      defaultLayout,
    });
  };

  // ─── Validation ──────────────────────────────────
  const stepValid: Record<StepId, boolean> = {
    scenes: selectedScenes.length >= 1,
    params: true,
    linkage: true,
    layout: layoutConfigs.filter((c) => c.layoutRole === 'primary').length === 1,
    preview: true,
  };

  return (
    <div className={cn('space-y-4', className)}>
      {/* Stepper */}
      <div className="flex items-center gap-1">
        {STEPS.map((s, i) => {
          const Icon = s.icon;
          const isActive = s.id === step;
          const isDone = i < stepIndex;
          return (
            <div key={s.id} className="flex items-center gap-1">
              {i > 0 && <div className={cn('h-px w-6', isDone ? 'bg-primary' : 'bg-border')} />}
              <button
                className={cn(
                  'flex items-center gap-1.5 rounded-full px-3 py-1 text-xs font-medium transition-colors',
                  isActive && 'bg-primary text-primary-foreground',
                  isDone && 'bg-primary/10 text-primary',
                  !isActive && !isDone && 'text-muted-foreground',
                )}
                onClick={() => i <= stepIndex && setStep(s.id)}
              >
                {isDone ? <Check className="h-3.5 w-3.5" /> : <Icon className="h-3.5 w-3.5" />}
                {s.label}
              </button>
            </div>
          );
        })}
      </div>

      {/* Step ① 选择场景 */}
      {step === 'scenes' && (
        <Card>
          <CardHeader>
            <CardTitle>选择仿真场景</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            {/* 领域筛选 */}
            <div className="flex flex-wrap gap-1">
              <Button
                variant={categoryFilter === 'all' ? 'secondary' : 'ghost'}
                size="sm"
                className="h-7 text-xs"
                onClick={() => setCategoryFilter('all')}
              >
                全部
              </Button>
              {Object.entries(CATEGORY_LABELS).map(([key, label]) => (
                <Button
                  key={key}
                  variant={categoryFilter === key ? 'secondary' : 'ghost'}
                  size="sm"
                  className="h-7 text-xs"
                  onClick={() => setCategoryFilter(key)}
                >
                  {label}
                </Button>
              ))}
            </div>

            {/* 场景卡片网格 */}
            <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
              {filteredScenarios.map((scenario) => {
                const isSelected = selectedScenes.some((s) => s.scenarioId === scenario.id);
                return (
                  <button
                    key={scenario.id}
                    className={cn(
                      'rounded-xl border p-3 text-left transition-all hover:shadow-md',
                      isSelected && 'border-primary ring-2 ring-primary/20 bg-primary/5',
                    )}
                    onClick={() => toggleScene(scenario)}
                  >
                    <p className="text-sm font-medium">{scenario.name}</p>
                    <p className="text-xs text-muted-foreground font-mono">{scenario.code}</p>
                    <div className="flex flex-wrap gap-1 mt-2">
                      <Badge variant="outline" className="text-[10px]">
                        {CATEGORY_LABELS[scenario.category] ?? scenario.category}
                      </Badge>
                      <Badge variant="outline" className="text-[10px]">
                        L{scenario.difficulty_level}
                      </Badge>
                      <Badge variant="outline" className="text-[10px]">
                        {scenario.time_control_mode}
                      </Badge>
                    </div>
                  </button>
                );
              })}
            </div>

            {/* 已选序列 */}
            {selectedScenes.length > 0 && (
              <div className="flex flex-wrap gap-2 pt-2 border-t">
                {selectedScenes.map((s) => (
                  <Badge key={s.scenarioId} className="gap-1 cursor-pointer" onClick={() => toggleScene({ id: s.scenarioId, code: s.code, name: s.name } as SimScenarioCatalogItem)}>
                    {s.name}
                    <span className="text-[10px] opacity-60">&times;</span>
                  </Badge>
                ))}
              </div>
            )}
          </CardContent>
        </Card>
      )}

      {/* Step ② 配置参数 */}
      {step === 'params' && (
        <Card>
          <CardHeader>
            <CardTitle>配置参数</CardTitle>
          </CardHeader>
          <CardContent>
            <Accordion type="single" collapsible defaultValue={selectedScenes[0]?.code}>
              {selectedScenes.map((scene) => (
                <AccordionItem key={scene.code} value={scene.code}>
                  <AccordionTrigger className="text-sm">
                    {scene.name}（{scene.code}）
                  </AccordionTrigger>
                  <AccordionContent className="space-y-3 pb-4">
                    {Object.entries(scene.defaultParams).map(([key, defaultVal]) => (
                      <div key={key} className="grid grid-cols-[120px_1fr] items-center gap-2">
                        <label className="text-xs text-muted-foreground truncate">{key}</label>
                        <Input
                          className="h-8 text-sm"
                          type={typeof defaultVal === 'number' ? 'number' : 'text'}
                          value={String(scene.params[key] ?? defaultVal ?? '')}
                          onChange={(e) =>
                            updateParam(
                              scene.scenarioId,
                              key,
                              typeof defaultVal === 'number' ? Number(e.target.value) : e.target.value,
                            )
                          }
                        />
                      </div>
                    ))}
                    <Button variant="outline" size="sm" className="text-xs" onClick={() => resetParams(scene.scenarioId)}>
                      恢复默认
                    </Button>
                  </AccordionContent>
                </AccordionItem>
              ))}
            </Accordion>
          </CardContent>
        </Card>
      )}

      {/* Step ③ 联动组 */}
      {step === 'linkage' && (
        <Card>
          <CardHeader>
            <CardTitle>联动组配置</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            {linkGroupConfigs.length === 0 ? (
              <p className="text-sm text-muted-foreground py-4 text-center">
                当前已选场景未匹配到任何联动组。可继续到下一步。
              </p>
            ) : (
              linkGroupConfigs.map((config) => (
                <div key={config.groupId} className="rounded-xl border p-4 space-y-3">
                  <div className="flex items-center justify-between">
                    <div>
                      <p className="text-sm font-medium">{config.groupName}</p>
                      <p className="text-xs text-muted-foreground">
                        匹配场景：{config.scenes.join(', ')}
                      </p>
                    </div>
                    <div className="flex items-center gap-2">
                      <span className="text-xs text-muted-foreground">
                        {config.enabled ? '已启用' : '未启用'}
                      </span>
                      <Switch
                        checked={config.enabled}
                        onCheckedChange={() => toggleLinkGroup(config.groupId)}
                        disabled={config.scenes.length < 2}
                      />
                    </div>
                  </div>
                  {config.enabled && config.fields.length > 0 && (
                    <div className="rounded border p-2 space-y-1">
                      <p className="text-xs font-medium text-muted-foreground">共享字段 Owner</p>
                      {config.fields.map((f) => (
                        <div key={f.field_name} className="flex items-center gap-2 text-xs">
                          <span className="font-mono">{f.field_name}</span>
                          <span className="text-muted-foreground">←</span>
                          <Badge variant="outline" className="text-[10px]">
                            {f.owner_scene}
                          </Badge>
                        </div>
                      ))}
                      <p className="text-[10px] text-muted-foreground mt-1">
                        联动模式下时钟强制同步，无法独立调步。
                      </p>
                    </div>
                  )}
                </div>
              ))
            )}
          </CardContent>
        </Card>
      )}

      {/* Step ④ 多场景布局 */}
      {step === 'layout' && (
        <Card>
          <CardHeader>
            <CardTitle>多场景布局</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="flex items-center gap-3">
              <span className="text-sm">学生默认视图：</span>
              {(['grid', 'focus', 'carousel'] as const).map((v) => (
                <Button
                  key={v}
                  variant={defaultLayout === v ? 'secondary' : 'outline'}
                  size="sm"
                  className="h-7 text-xs"
                  onClick={() => setDefaultLayout(v)}
                >
                  {v === 'grid' ? '网格' : v === 'focus' ? '焦点' : '轮播'}
                </Button>
              ))}
            </div>

            <div className="grid gap-3 lg:grid-cols-2">
              {layoutConfigs.map((config) => (
                <div key={config.sceneCode} className="rounded-xl border p-4 space-y-3">
                  <div className="flex items-center gap-2">
                    <GripVertical className="h-4 w-4 text-muted-foreground" />
                    <p className="text-sm font-medium">{config.title}</p>
                    <Badge variant="outline" className="text-[10px] ml-auto">
                      {config.sceneCode}
                    </Badge>
                  </div>
                  <div className="grid grid-cols-2 gap-3">
                    <div>
                      <label className="text-xs text-muted-foreground">布局角色</label>
                      <Select
                        value={config.layoutRole}
                        onValueChange={(v) =>
                          updateLayoutConfig(config.sceneCode, {
                            layoutRole: v as SceneLayoutConfig['layoutRole'],
                          })
                        }
                      >
                        <SelectTrigger className="h-8 text-xs mt-1">
                          <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value="primary">primary（主场景）</SelectItem>
                          <SelectItem value="secondary">secondary（副场景）</SelectItem>
                          <SelectItem value="auxiliary">auxiliary（辅助）</SelectItem>
                        </SelectContent>
                      </Select>
                    </div>
                    <div>
                      <label className="text-xs text-muted-foreground">展示模式</label>
                      <Select
                        value={config.displayMode}
                        onValueChange={(v) => updateLayoutConfig(config.sceneCode, { displayMode: v })}
                      >
                        <SelectTrigger className="h-8 text-xs mt-1">
                          <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value="single">single</SelectItem>
                          <SelectItem value="split-2">split-2</SelectItem>
                          <SelectItem value="split-3">split-3</SelectItem>
                          <SelectItem value="grid-4">grid-4</SelectItem>
                        </SelectContent>
                      </Select>
                    </div>
                  </div>
                  <div className="flex items-center gap-4">
                    <label className="flex items-center gap-2 text-xs">
                      <Switch
                        checked={config.linkToPrimary}
                        onCheckedChange={(v) => updateLayoutConfig(config.sceneCode, { linkToPrimary: v })}
                        disabled={config.layoutRole === 'primary'}
                      />
                      同步主场景时钟
                    </label>
                    <label className="flex items-center gap-2 text-xs">
                      <Switch
                        checked={config.defaultVisible}
                        onCheckedChange={(v) => updateLayoutConfig(config.sceneCode, { defaultVisible: v })}
                      />
                      默认展开
                    </label>
                  </div>
                </div>
              ))}
            </div>
            {layoutConfigs.filter((c) => c.layoutRole === 'primary').length !== 1 && (
              <p className="text-sm text-destructive">primary 角色必须且仅能有 1 个场景。</p>
            )}
          </CardContent>
        </Card>
      )}

      {/* Step ⑤ 预览 */}
      {step === 'preview' && (
        <Card>
          <CardHeader>
            <CardTitle>预览与发布</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="rounded-xl border bg-muted/30 p-4">
              <p className="text-xs text-muted-foreground mb-2">预览模式 · 对学生不可见</p>
              <div className="grid gap-2 sm:grid-cols-2">
                {selectedScenes.map((s) => (
                  <div key={s.code} className="rounded border p-3">
                    <p className="text-sm font-medium">{s.name}</p>
                    <div className="flex flex-wrap gap-1 mt-1">
                      <Badge variant="outline" className="text-[10px]">{s.category}</Badge>
                      <Badge variant="outline" className="text-[10px]">{s.timeControlMode}</Badge>
                      {linkGroupConfigs
                        .filter((c) => c.enabled && c.scenes.includes(s.code))
                        .map((c) => (
                          <Badge key={c.groupId} className="text-[10px] gap-0.5">
                            <Link2 className="h-2.5 w-2.5" /> {c.groupName}
                          </Badge>
                        ))}
                    </div>
                    <pre className="text-[10px] text-muted-foreground mt-2 overflow-auto max-h-24">
                      {JSON.stringify(s.params, null, 2)}
                    </pre>
                  </div>
                ))}
              </div>
              <p className="text-xs text-muted-foreground mt-3">
                默认布局：{defaultLayout} · 场景数：{selectedScenes.length} ·
                联动组：{linkGroupConfigs.filter((c) => c.enabled).length}
              </p>
            </div>
          </CardContent>
        </Card>
      )}

      {/* 底部导航 */}
      <div className="flex items-center justify-between">
        <div className="flex gap-2">
          {canPrev && (
            <Button variant="outline" onClick={goPrev}>
              <ChevronLeft className="h-4 w-4 mr-1" />
              上一步
            </Button>
          )}
          {onCancel && (
            <Button variant="ghost" onClick={onCancel}>
              取消
            </Button>
          )}
        </div>
        <div>
          {canNext ? (
            <Button onClick={goNext} disabled={!stepValid[step]}>
              下一步
              <ChevronRight className="h-4 w-4 ml-1" />
            </Button>
          ) : (
            <Button onClick={handlePublish}>
              发布配置
            </Button>
          )}
        </div>
      </div>
    </div>
  );
}
