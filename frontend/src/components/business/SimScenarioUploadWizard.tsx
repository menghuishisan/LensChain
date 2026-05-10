'use client';

// SimScenarioUploadWizard.tsx
// 教师自定义仿真场景上传（06.2 §十一）。
// 三步流程：① 元信息 → ② 容器/渲染器 → ③ 提交审核。
// 路由 /teacher/sim-scenarios/upload。

import { useState } from 'react';
import { Check, ChevronLeft, ChevronRight, Circle, CircleDot, FileCode, Info, Upload } from 'lucide-react';
import { Badge } from '@/components/ui/Badge';
import { Button } from '@/components/ui/Button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/Card';
import { Input } from '@/components/ui/Input';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/Select';
import { Textarea } from '@/components/ui/Textarea';
import { Switch } from '@/components/ui/Switch';
import { cn } from '@/lib/utils';
import { useSimScenarioMutations, useExperimentFileUploadMutation } from '@/hooks/useExperimentTemplates';
import { useAuthStore } from '@/stores/authStore';

// ─── Types ──────────────────────────────────────────────────
interface MetaInfo {
  code: string;
  nameCN: string;
  nameEN: string;
  description: string;
  category: string;
  difficultyLevel: 'L1' | 'L2' | 'L3';
  timeControlMode: 'process' | 'reactive' | 'continuous';
  joinLinkGroups: string[];
}

type RendererLevel = 'L1' | 'L2' | 'L3';

interface RendererConfig {
  level: RendererLevel;
  algorithmCode: string;
  defaultParamsSchema: string;
  actionDefJSON: string;
  npmPackageName: string;
  npmVersion: string;
  grpcImageURL: string;
  grpcPort: string;
  cpuLimit: string;
  memLimit: string;
}

const STEPS = [
  { id: 'meta', label: '元信息', icon: Info },
  { id: 'renderer', label: '容器/渲染器', icon: FileCode },
  { id: 'submit', label: '提交审核', icon: Upload },
] as const;

type StepId = (typeof STEPS)[number]['id'];

const CATEGORY_OPTIONS = [
  { value: 'node_network', label: '节点与网络' },
  { value: 'consensus', label: '共识过程' },
  { value: 'cryptography', label: '密码学' },
  { value: 'data_structure', label: '数据结构' },
  { value: 'transaction', label: '交易生命周期' },
  { value: 'smart_contract', label: '智能合约' },
  { value: 'attack_security', label: '攻击与安全' },
  { value: 'economic', label: '经济模型' },
  { value: 'generic', label: '教师扩展' },
];

const RESERVED_PREFIXES = ['pow-', 'pbft-', 'sha-', 'ecdsa-', 'raft-', 'pos-', 'dpos-', 'mpt-'];

/**
 * SimScenarioUploadWizard 教师自定义场景三步上传向导（06.2 §十一）。
 */
export function SimScenarioUploadWizard({ className }: { className?: string }) {
  const user = useAuthStore((s) => s.user);
  const schoolId = user?.school_id ?? 'unknown';
  const scenarioMutations = useSimScenarioMutations();
  const uploadMutation = useExperimentFileUploadMutation();

  const [step, setStep] = useState<StepId>('meta');
  const [agreed, setAgreed] = useState(false);

  // §11.3 元信息
  const [meta, setMeta] = useState<MetaInfo>({
    code: '',
    nameCN: '',
    nameEN: '',
    description: '',
    category: 'generic',
    difficultyLevel: 'L1',
    timeControlMode: 'process',
    joinLinkGroups: [],
  });

  // §11.4 渲染器
  const [renderer, setRenderer] = useState<RendererConfig>({
    level: 'L1',
    algorithmCode: '',
    defaultParamsSchema: '{}',
    actionDefJSON: '[]',
    npmPackageName: '',
    npmVersion: '1.0.0',
    grpcImageURL: '',
    grpcPort: '50051',
    cpuLimit: '0.5',
    memLimit: '512Mi',
  });

  const fullSceneCode = `teacher_${schoolId}__${meta.code}`;
  const stepIndex = STEPS.findIndex((s) => s.id === step);

  // ─── Validation ──────────────────────────────────
  const codeReserved = RESERVED_PREFIXES.some((p) => meta.code.startsWith(p));
  const metaValid = meta.code.length > 0 && meta.nameCN.length > 0 && !codeReserved;
  const rendererValid = renderer.level === 'L3' ? renderer.npmPackageName.length > 0 : true;

  const stepValid: Record<StepId, boolean> = {
    meta: metaValid,
    renderer: rendererValid,
    submit: agreed,
  };

  const goNext = () => stepIndex < STEPS.length - 1 && setStep(STEPS[stepIndex + 1].id);
  const goPrev = () => stepIndex > 0 && setStep(STEPS[stepIndex - 1].id);

  // ─── Submit ──────────────────────────────────────
  const handleSubmit = () => {
    scenarioMutations.create.mutate({
      name: meta.nameCN,
      code: fullSceneCode,
      category: meta.category,
      algorithm_type: renderer.level === 'L3' ? 'npm-custom' : 'platform-sandbox',
      time_control_mode: meta.timeControlMode,
      container_image_url: renderer.level === 'L3' ? renderer.grpcImageURL : '',
      data_source_mode: 1,
      default_params: safeParseJSON(renderer.defaultParamsSchema),
      interaction_schema: safeParseJSON(renderer.actionDefJSON),
      default_size: { w: 640, h: 360 },
    });
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

      {/* Step ① 元信息 */}
      {step === 'meta' && (
        <Card>
          <CardHeader><CardTitle>基本信息</CardTitle></CardHeader>
          <CardContent className="space-y-4">
            <div className="grid gap-4 md:grid-cols-2">
              <div>
                <label className="text-xs text-muted-foreground">场景标识码（自动加前缀）</label>
                <div className="flex items-center gap-1 mt-1">
                  <span className="text-xs text-muted-foreground font-mono">teacher_{schoolId}__</span>
                  <Input
                    className="h-8 text-sm font-mono flex-1"
                    value={meta.code}
                    onChange={(e) => setMeta({ ...meta, code: e.target.value.replace(/[^a-z0-9_-]/g, '') })}
                    placeholder="my-scenario"
                  />
                </div>
                {codeReserved && <p className="text-xs text-destructive mt-1">不允许使用平台保留前缀。</p>}
              </div>
              <div>
                <label className="text-xs text-muted-foreground">中文名</label>
                <Input className="h-8 text-sm mt-1" value={meta.nameCN} onChange={(e) => setMeta({ ...meta, nameCN: e.target.value })} placeholder="我的自定义场景" />
              </div>
              <div>
                <label className="text-xs text-muted-foreground">英文名</label>
                <Input className="h-8 text-sm mt-1" value={meta.nameEN} onChange={(e) => setMeta({ ...meta, nameEN: e.target.value })} placeholder="My Custom Scenario" />
              </div>
              <div>
                <label className="text-xs text-muted-foreground">领域分类</label>
                <Select value={meta.category} onValueChange={(v) => setMeta({ ...meta, category: v })}>
                  <SelectTrigger className="h-8 text-xs mt-1"><SelectValue /></SelectTrigger>
                  <SelectContent>
                    {CATEGORY_OPTIONS.map((c) => (
                      <SelectItem key={c.value} value={c.value}>{c.label}</SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
              <div>
                <label className="text-xs text-muted-foreground">难度等级</label>
                <Select value={meta.difficultyLevel} onValueChange={(v) => setMeta({ ...meta, difficultyLevel: v as MetaInfo['difficultyLevel'] })}>
                  <SelectTrigger className="h-8 text-xs mt-1"><SelectValue /></SelectTrigger>
                  <SelectContent>
                    <SelectItem value="L1">L1 基础</SelectItem>
                    <SelectItem value="L2">L2 进阶</SelectItem>
                    <SelectItem value="L3">L3 自定义</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <div>
                <label className="text-xs text-muted-foreground">时间模式</label>
                <Select value={meta.timeControlMode} onValueChange={(v) => setMeta({ ...meta, timeControlMode: v as MetaInfo['timeControlMode'] })}>
                  <SelectTrigger className="h-8 text-xs mt-1"><SelectValue /></SelectTrigger>
                  <SelectContent>
                    <SelectItem value="process">process</SelectItem>
                    <SelectItem value="reactive">reactive</SelectItem>
                    <SelectItem value="continuous">continuous</SelectItem>
                  </SelectContent>
                </Select>
              </div>
            </div>
            <div>
              <label className="text-xs text-muted-foreground">描述</label>
              <Textarea className="mt-1 text-sm" rows={3} value={meta.description} onChange={(e) => setMeta({ ...meta, description: e.target.value })} placeholder="场景用途、教学目标..." />
            </div>
          </CardContent>
        </Card>
      )}

      {/* Step ② 容器与渲染器 */}
      {step === 'renderer' && (
        <Card>
          <CardHeader><CardTitle>容器与渲染器</CardTitle></CardHeader>
          <CardContent className="space-y-4">
            <div className="flex gap-3">
              {(['L1', 'L2', 'L3'] as RendererLevel[]).map((lv) => (
                <Button
                  key={lv}
                  variant={renderer.level === lv ? 'secondary' : 'outline'}
                  size="sm"
                  className="text-xs"
                  onClick={() => setRenderer({ ...renderer, level: lv })}
                >
                  {renderer.level === lv
                    ? <CircleDot className="h-3 w-3 shrink-0" />
                    : <Circle className="h-3 w-3 shrink-0" />}
                  {lv === 'L1' && 'L1 平台原语组合'}
                  {lv === 'L2' && 'L2 平台+1原语函数'}
                  {lv === 'L3' && 'L3 npm 自定义渲染器'}
                </Button>
              ))}
            </div>

            {/* L1/L2: 沙箱代码 */}
            {(renderer.level === 'L1' || renderer.level === 'L2') && (
              <div className="space-y-3">
                <div>
                  <label className="text-xs text-muted-foreground">算法逻辑（JavaScript，沙箱 Worker 内执行）</label>
                  <Textarea
                    className="mt-1 font-mono text-xs"
                    rows={10}
                    value={renderer.algorithmCode}
                    onChange={(e) => setRenderer({ ...renderer, algorithmCode: e.target.value })}
                    placeholder="// 在沙箱内执行的算法代码..."
                  />
                </div>
                <div>
                  <label className="text-xs text-muted-foreground">默认参数 JSON Schema</label>
                  <Textarea
                    className="mt-1 font-mono text-xs"
                    rows={5}
                    value={renderer.defaultParamsSchema}
                    onChange={(e) => setRenderer({ ...renderer, defaultParamsSchema: e.target.value })}
                  />
                </div>
                <div>
                  <label className="text-xs text-muted-foreground">ActionDef JSON（定义交互按钮与字段）</label>
                  <Textarea
                    className="mt-1 font-mono text-xs"
                    rows={5}
                    value={renderer.actionDefJSON}
                    onChange={(e) => setRenderer({ ...renderer, actionDefJSON: e.target.value })}
                  />
                </div>
              </div>
            )}

            {/* L3: npm + gRPC */}
            {renderer.level === 'L3' && (
              <div className="space-y-3">
                <div className="grid gap-3 md:grid-cols-2">
                  <div>
                    <label className="text-xs text-muted-foreground">npm 包名（自动加命名空间前缀）</label>
                    <div className="flex items-center gap-1 mt-1">
                      <span className="text-xs text-muted-foreground font-mono">@chainmirror/renderer-{schoolId}__</span>
                      <Input className="h-8 text-xs font-mono flex-1" value={renderer.npmPackageName} onChange={(e) => setRenderer({ ...renderer, npmPackageName: e.target.value })} />
                    </div>
                  </div>
                  <div>
                    <label className="text-xs text-muted-foreground">版本号</label>
                    <Input className="h-8 text-xs mt-1" value={renderer.npmVersion} onChange={(e) => setRenderer({ ...renderer, npmVersion: e.target.value })} />
                  </div>
                </div>
                <div>
                  <label className="text-xs text-muted-foreground">.tgz 文件上传（≤5MB）</label>
                  <label className="mt-1 flex cursor-pointer items-center gap-2 rounded-lg border border-dashed border-border p-4 hover:bg-muted/50">
                    <Upload className="h-4 w-4" />
                    <span className="text-xs">拖拽或点击上传 .tgz</span>
                    <input className="sr-only" type="file" accept=".tgz" onChange={(e) => {
                      const file = e.target.files?.[0];
                      if (file) uploadMutation.mutate({ file, purpose: 'scenario_package' });
                    }} />
                  </label>
                  {uploadMutation.data && <p className="text-xs text-muted-foreground mt-1">已上传：{uploadMutation.data.file_url}</p>}
                </div>
                <div className="grid gap-3 md:grid-cols-3">
                  <div>
                    <label className="text-xs text-muted-foreground">gRPC 镜像</label>
                    <Input className="h-8 text-xs mt-1" value={renderer.grpcImageURL} onChange={(e) => setRenderer({ ...renderer, grpcImageURL: e.target.value })} placeholder="registry/sim-xxx:v1.0" />
                  </div>
                  <div>
                    <label className="text-xs text-muted-foreground">gRPC 端口</label>
                    <Input className="h-8 text-xs mt-1" value={renderer.grpcPort} onChange={(e) => setRenderer({ ...renderer, grpcPort: e.target.value })} />
                  </div>
                  <div className="grid grid-cols-2 gap-2">
                    <div>
                      <label className="text-xs text-muted-foreground">CPU</label>
                      <Input className="h-8 text-xs mt-1" value={renderer.cpuLimit} onChange={(e) => setRenderer({ ...renderer, cpuLimit: e.target.value })} />
                    </div>
                    <div>
                      <label className="text-xs text-muted-foreground">Mem</label>
                      <Input className="h-8 text-xs mt-1" value={renderer.memLimit} onChange={(e) => setRenderer({ ...renderer, memLimit: e.target.value })} />
                    </div>
                  </div>
                </div>
              </div>
            )}
          </CardContent>
        </Card>
      )}

      {/* Step ③ 提交审核 */}
      {step === 'submit' && (
        <Card>
          <CardHeader><CardTitle>提交审核</CardTitle></CardHeader>
          <CardContent className="space-y-4">
            <div className="rounded-xl border bg-muted/30 p-4 space-y-2">
              <p className="text-xs text-muted-foreground">审核流程预估：3-5 个工作日</p>
              <div className="grid gap-2 sm:grid-cols-2 text-sm">
                <div><span className="text-muted-foreground">场景标识码：</span><span className="font-mono">{fullSceneCode}</span></div>
                <div><span className="text-muted-foreground">中文名：</span>{meta.nameCN}</div>
                <div><span className="text-muted-foreground">领域：</span>{meta.category}</div>
                <div><span className="text-muted-foreground">复杂度：</span><Badge variant="outline">{renderer.level}</Badge></div>
                <div><span className="text-muted-foreground">时间模式：</span>{meta.timeControlMode}</div>
                <div><span className="text-muted-foreground">难度：</span>{meta.difficultyLevel}</div>
              </div>
            </div>
            <label className="flex items-center gap-2 text-sm">
              <Switch checked={agreed} onCheckedChange={setAgreed} />
              我已阅读并同意《教师扩展场景规范》
            </label>
            {scenarioMutations.create.isSuccess && (
              <p className="text-sm text-green-600">场景已提交审核，管理员将在场景库管理页处理。</p>
            )}
          </CardContent>
        </Card>
      )}

      {/* 底部导航 */}
      <div className="flex items-center justify-between">
        <div>
          {stepIndex > 0 && (
            <Button variant="outline" onClick={goPrev}>
              <ChevronLeft className="h-4 w-4 mr-1" />上一步
            </Button>
          )}
        </div>
        <div>
          {stepIndex < STEPS.length - 1 ? (
            <Button onClick={goNext} disabled={!stepValid[step]}>
              下一步<ChevronRight className="h-4 w-4 ml-1" />
            </Button>
          ) : (
            <Button onClick={handleSubmit} disabled={!agreed || scenarioMutations.create.isPending} isLoading={scenarioMutations.create.isPending}>
              提交审核
            </Button>
          )}
        </div>
      </div>
    </div>
  );
}

function safeParseJSON(str: string): Record<string, unknown> {
  try {
    return JSON.parse(str) as Record<string, unknown>;
  } catch {
    return {};
  }
}
