"use client";

// ExperimentTemplatePanels.tsx
// 模块04模板、镜像和仿真场景页面级业务面板。

import { Plus, ShieldCheck } from "lucide-react";
import { useRouter } from "next/navigation";
import { useEffect, useMemo, useState } from "react";

import { ExperimentTemplateCard } from "@/components/business/ExperimentTemplateCard";
import { Badge } from "@/components/ui/Badge";
import { Button } from "@/components/ui/Button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/Card";
import { EmptyState } from "@/components/ui/EmptyState";
import { ErrorState } from "@/components/ui/ErrorState";
import { FormField } from "@/components/ui/FormField";
import { Input } from "@/components/ui/Input";
import { LoadingState } from "@/components/ui/LoadingState";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/Select";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/Tabs";
import { Textarea } from "@/components/ui/Textarea";
import {
  useExperimentTemplate,
  useExperimentTemplateMutations,
  useExperimentTemplates,
  useImage,
  useImageConfigTemplate,
  useImages,
  useSimScenarios,
  useSimLinkGroups,
  useTemplateConfigMutations,
} from "@/hooks/useExperimentTemplates";
import { buildServiceDiscoveryEnvVars, detectPortConflicts, resolveConditionalEnvVars } from "@/lib/experiment";
import type { ID } from "@/types/api";
import type { ExperimentTemplateRequest, ExperimentType, TemplateCheckpointRequest, TemplateContainerRequest } from "@/types/experiment";

const DEFAULT_TEMPLATE_REQUEST: ExperimentTemplateRequest = {
  title: "",
  description: "",
  objectives: "",
  instructions: "",
  reference_materials: "",
  experiment_type: 2,
  topology_mode: 1,
  judge_mode: 3,
  auto_weight: 60,
  manual_weight: 40,
  total_score: 100,
  max_duration: 120,
  idle_timeout: 30,
  cpu_limit: "2",
  memory_limit: "4Gi",
  disk_limit: "20Gi",
  score_strategy: 1,
};

const TEMPLATE_STEPS = [
  { id: "basic", label: "1. 基本信息", appliesTo: [1, 2, 3] as ExperimentType[] },
  { id: "images", label: "2. 镜像编排", appliesTo: [2, 3] as ExperimentType[] },
  { id: "sim", label: "3. 工具与仿真场景", appliesTo: [1, 2, 3] as ExperimentType[] },
  { id: "instructions", label: "4. 实验说明", appliesTo: [1, 2, 3] as ExperimentType[] },
  { id: "validate", label: "5. 检查点与评分", appliesTo: [1, 2, 3] as ExperimentType[] },
  { id: "resource", label: "6. 资源与发布", appliesTo: [1, 2, 3] as ExperimentType[] },
] as const;

/**
 * ExperimentTemplateListPanel 教师实验模板列表面板。
 */
export function ExperimentTemplateListPanel() {
  const router = useRouter();
  const [keyword, setKeyword] = useState("");
  const templatesQuery = useExperimentTemplates({ page: 1, page_size: 20, keyword });

  if (templatesQuery.isLoading) {
    return <LoadingState title="正在加载实验内容" description="正在整理你可管理的实验内容列表。" />;
  }

  if (templatesQuery.isError) {
    return <ErrorState title="实验内容加载失败" description="请稍后重试，或确认当前账号是否有访问权限。" />;
  }

  const templates = templatesQuery.data?.list ?? [];

  return (
    <div className="space-y-5">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="font-display text-3xl font-semibold">实验内容中心</h1>
          <p className="mt-2 text-sm text-muted-foreground">创建、检查并发布实验内容，完善环境、说明和评分要求。</p>
        </div>
        <Button onClick={() => router.push("/teacher/experiment-templates/create")}>
          <Plus className="h-4 w-4" />
          新建实验内容
        </Button>
      </div>
      <Input value={keyword} onChange={(event) => setKeyword(event.target.value)} placeholder="搜索实验标题" />
      {templates.length === 0 ? (
        <EmptyState title="暂无实验内容" description="先创建一份实验草稿，再补充说明、环境和评分要求。" />
      ) : (
        <div className="grid gap-4 xl:grid-cols-2">
          {templates.map((template) => (
            <TemplateCardWithActions key={template.id} template={template} onOpen={(id) => router.push(`/teacher/experiment-templates/${id}`)} />
          ))}
        </div>
      )}
    </div>
  );
}

/**
 * ExperimentTemplateEditorPanel 实验模板编辑器面板。
 */
export function ExperimentTemplateEditorPanel({ templateID }: { templateID?: ID }) {
  const router = useRouter();
  const isEdit = templateID !== undefined && templateID.length > 0;
  const templateQuery = useExperimentTemplate(templateID ?? "");
  const templateMutations = useExperimentTemplateMutations(templateID);
  const configMutations = useTemplateConfigMutations(templateID ?? "");
  const imagesQuery = useImages({ page: 1, page_size: 50, status: 1 });
  const scenariosQuery = useSimScenarios({ page: 1, page_size: 50, status: 1 });
  const linkGroupsQuery = useSimLinkGroups();
  const [form, setForm] = useState<ExperimentTemplateRequest>(DEFAULT_TEMPLATE_REQUEST);
  const [selectedImageID, setSelectedImageID] = useState("");
  const [selectedScenarioID, setSelectedScenarioID] = useState("");
  const [selectedLinkGroupID, setSelectedLinkGroupID] = useState("");
  const [activeStep, setActiveStep] = useState<(typeof TEMPLATE_STEPS)[number]["id"]>("basic");
  const [k8sConfigText, setK8sConfigText] = useState("{\n  \"namespace_resource\": {},\n  \"network_policy\": {}\n}");
  const imageQuery = useImage(selectedImageID);
  const imageConfigQuery = useImageConfigTemplate(selectedImageID);
  const [containerName, setContainerName] = useState("primary-node");
  const [checkpointTitle, setCheckpointTitle] = useState("关键任务完成");
  const [checkpointScore, setCheckpointScore] = useState("20");

  const activeTemplate = templateQuery.data;
  const canEditSubresources = isEdit && templateID !== undefined;
  const selectedVersion = imageQuery.data?.versions.find((version) => version.is_default) ?? imageQuery.data?.versions[0];
  const portConflicts = detectPortConflicts((activeTemplate?.containers ?? []).map((container) => ({ container_name: container.container_name, ports: container.ports })));
  const serviceDiscoveryVars = buildServiceDiscoveryEnvVars((activeTemplate?.containers ?? []).map((container) => ({ container_name: container.container_name, ports: container.ports })));
  const visibleSteps = useMemo(() => TEMPLATE_STEPS.filter((step) => step.appliesTo.includes(form.experiment_type)), [form.experiment_type]);
  const currentStepIndex = visibleSteps.findIndex((step) => step.id === activeStep);
  const totalCheckpointScore = (activeTemplate?.checkpoints ?? []).reduce((sum, checkpoint) => sum + checkpoint.score, 0);
  const validationSummary = templateMutations.validate.data?.summary ?? null;
  const publishBlocked = templateMutations.validate.data ? !templateMutations.validate.data.is_publishable : portConflicts.length > 0;

  useEffect(() => {
    if (!activeTemplate) {
      return;
    }

    setForm({
      title: activeTemplate.title,
      description: activeTemplate.description ?? "",
      objectives: activeTemplate.objectives ?? "",
      instructions: activeTemplate.instructions ?? "",
      reference_materials: activeTemplate.reference_materials ?? "",
      experiment_type: activeTemplate.experiment_type,
      topology_mode: activeTemplate.topology_mode,
      judge_mode: activeTemplate.judge_mode,
      auto_weight: activeTemplate.auto_weight ?? 60,
      manual_weight: activeTemplate.manual_weight ?? 40,
      total_score: activeTemplate.total_score,
      max_duration: activeTemplate.max_duration,
      idle_timeout: activeTemplate.idle_timeout ?? 30,
      cpu_limit: activeTemplate.cpu_limit ?? "2",
      memory_limit: activeTemplate.memory_limit ?? "4Gi",
      disk_limit: activeTemplate.disk_limit ?? "20Gi",
      score_strategy: activeTemplate.score_strategy,
    });
    setK8sConfigText(JSON.stringify(activeTemplate.k8s_config ?? { namespace_resource: {}, network_policy: {} }, null, 2));
  }, [activeTemplate]);

  useEffect(() => {
    if (visibleSteps.some((step) => step.id === activeStep)) {
      return;
    }
    setActiveStep(visibleSteps[0]?.id ?? "basic");
  }, [activeStep, visibleSteps]);

  const submitTemplate = () => {
    if (isEdit && templateID) {
      templateMutations.update.mutate(form);
      return;
    }
    templateMutations.create.mutate(form, {
      onSuccess: (created) => router.push(`/teacher/experiment-templates/${created.id}`),
    });
  };

  const addContainer = () => {
    if (!templateID || !selectedVersion) {
      return;
    }
    const config = imageConfigQuery.data;
    const payload: TemplateContainerRequest = {
      image_version_id: selectedVersion.id,
      container_name: containerName,
      env_vars: config?.default_env_vars.map((item) => ({ key: item.key, value: item.value })) ?? [],
      ports: config?.default_ports.map((item) => ({ container: item.port, protocol: item.protocol })) ?? [],
      volumes: [],
      cpu_limit: config?.resource_recommendation.cpu ?? null,
      memory_limit: config?.resource_recommendation.memory ?? null,
      depends_on: config?.typical_companions.required.map((item) => item.image) ?? [],
      startup_order: (activeTemplate?.containers.length ?? 0) + 1,
      is_primary: (activeTemplate?.containers.length ?? 0) === 0,
    };
    configMutations.createContainer.mutate(payload);
  };

  const addCheckpoint = () => {
    if (!templateID) {
      return;
    }
    const payload: TemplateCheckpointRequest = {
      title: checkpointTitle,
      description: "由模板编辑器创建的检查点",
      check_type: form.experiment_type === 1 ? 3 : 1,
      script_content: form.experiment_type === 1 ? null : "echo ok",
      script_language: form.experiment_type === 1 ? null : "bash",
      target_container: form.experiment_type === 1 ? null : containerName,
      assertion_config: form.experiment_type === 1 ? { path: "state.ready", operator: "eq", value: true } : null,
      score: Number(checkpointScore),
      scope: 1,
      sort_order: (activeTemplate?.checkpoints.length ?? 0) + 1,
    };
    configMutations.createCheckpoint.mutate(payload);
  };

  const addSimScene = () => {
    if (!templateID || !selectedScenarioID) {
      return;
    }
    configMutations.createScene.mutate({
      scenario_id: selectedScenarioID,
      link_group_id: selectedLinkGroupID || null,
      scene_params: {},
      initial_state: {},
      data_source_mode: form.experiment_type === 1 ? 1 : 3,
      data_source_config: {},
      layout_position: { order: activeTemplate?.sim_scenes.length ?? 0, column_span: 6 },
    });
  };

  const saveK8sConfig = () => {
    try {
      templateMutations.setK8s.mutate(JSON.parse(k8sConfigText) as Record<string, unknown>);
    } catch {
      setK8sConfigText("{\n  \"error\": \"K8s JSON 解析失败，请修正后重新保存\"\n}");
    }
  };

  return (
    <div className="space-y-5">
      <div>
        <h1 className="font-display text-3xl font-semibold">{isEdit ? "编辑实验内容" : "创建实验内容"}</h1>
        <p className="mt-2 text-sm text-muted-foreground">按步骤完善实验类型、环境内容、说明、评分要求和发布设置。</p>
      </div>
      <Tabs value={activeStep} onValueChange={(value) => setActiveStep(value as (typeof TEMPLATE_STEPS)[number]["id"])}>
        <TabsList className="flex w-full flex-wrap justify-start gap-2 bg-transparent p-0">
          {TEMPLATE_STEPS.map((step) => {
            const isVisible = visibleSteps.some((visibleStep) => visibleStep.id === step.id);
            return (
              <TabsTrigger
                key={step.id}
                disabled={!isVisible}
                value={step.id}
                className={isVisible ? "" : "cursor-not-allowed opacity-45"}
              >
                {step.label}
              </TabsTrigger>
            );
          })}
        </TabsList>
        <TabsContent value="basic">
          <Card>
            <CardHeader>
              <CardTitle>基本信息与拓扑选择</CardTitle>
            </CardHeader>
            <CardContent className="grid gap-4 md:grid-cols-2">
              <FormField label="标题" required>
                <Input value={form.title} onChange={(event) => setForm((current) => ({ ...current, title: event.target.value }))} placeholder="例如：以太坊智能合约开发入门" />
              </FormField>
              <FormField label="实验类型" required>
                <Select value={String(form.experiment_type)} onValueChange={(value) => setForm((current) => ({ ...current, experiment_type: Number(value) as 1 | 2 | 3 }))}>
                  <SelectTrigger><SelectValue /></SelectTrigger>
                  <SelectContent>
                    <SelectItem value="1">纯仿真</SelectItem>
                    <SelectItem value="2">真实环境</SelectItem>
                    <SelectItem value="3">混合实验</SelectItem>
                  </SelectContent>
                </Select>
              </FormField>
              <FormField label="拓扑模式" required>
                <Select value={String(form.topology_mode)} onValueChange={(value) => setForm((current) => ({ ...current, topology_mode: Number(value) as 1 | 2 | 3 | 4 }))}>
                  <SelectTrigger><SelectValue /></SelectTrigger>
                  <SelectContent>
                    <SelectItem value="1">单人单节点</SelectItem>
                    <SelectItem value="2">单人多节点</SelectItem>
                    <SelectItem value="3">多人协作组网</SelectItem>
                    <SelectItem value="4">共享基础设施</SelectItem>
                  </SelectContent>
                </Select>
              </FormField>
              <FormField label="评分模式" required>
                <Select value={String(form.judge_mode)} onValueChange={(value) => setForm((current) => ({ ...current, judge_mode: Number(value) as 1 | 2 | 3 }))}>
                  <SelectTrigger><SelectValue /></SelectTrigger>
                  <SelectContent>
                    <SelectItem value="1">纯自动</SelectItem>
                    <SelectItem value="2">纯手动</SelectItem>
                    <SelectItem value="3">混合评分</SelectItem>
                  </SelectContent>
                </Select>
              </FormField>
              <FormField label="总分">
                <Input type="number" value={form.total_score} onChange={(event) => setForm((current) => ({ ...current, total_score: Number(event.target.value) }))} />
              </FormField>
              <FormField label="最大时长（分钟）">
                <Input type="number" value={form.max_duration} onChange={(event) => setForm((current) => ({ ...current, max_duration: Number(event.target.value) }))} />
              </FormField>
              <FormField label="实验目标" className="md:col-span-2">
                <Textarea value={form.objectives ?? ""} onChange={(event) => setForm((current) => ({ ...current, objectives: event.target.value }))} rows={4} />
              </FormField>
              <FormField label="实验描述" className="md:col-span-2">
                <Textarea value={form.description ?? ""} onChange={(event) => setForm((current) => ({ ...current, description: event.target.value }))} rows={4} />
              </FormField>
              <FormField label="标签参考资料" className="md:col-span-2">
                <Textarea value={form.reference_materials ?? ""} onChange={(event) => setForm((current) => ({ ...current, reference_materials: event.target.value }))} rows={3} />
              </FormField>
              <div className="md:col-span-2 flex justify-between">
                <Button variant="outline" disabled>
                  上一步
                </Button>
                <Button variant="outline" onClick={() => setActiveStep(visibleSteps[Math.min(currentStepIndex + 1, visibleSteps.length - 1)]?.id ?? activeStep)}>
                  下一步
                </Button>
              </div>
            </CardContent>
          </Card>
        </TabsContent>
        <TabsContent value="images">
          <Card>
            <CardHeader>
              <CardTitle>镜像编排</CardTitle>
            </CardHeader>
            <CardContent className="space-y-5">
              {form.experiment_type === 1 ? (
                <div className="rounded-xl border border-border bg-muted/25 p-4 text-sm text-muted-foreground">纯仿真实验无需配置容器环境，这一步可以直接跳过。</div>
              ) : null}
              <div className="grid gap-4 md:grid-cols-3">
                <FormField label="选择镜像">
                  <Select value={selectedImageID} onValueChange={setSelectedImageID}>
                    <SelectTrigger><SelectValue placeholder="选择镜像" /></SelectTrigger>
                    <SelectContent>
                      {(imagesQuery.data?.list ?? []).map((image) => (
                        <SelectItem key={image.id} value={image.id}>{image.display_name}</SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </FormField>
                <FormField label="容器名称">
                  <Input value={containerName} onChange={(event) => setContainerName(event.target.value)} />
                </FormField>
                <Button className="self-end" disabled={!canEditSubresources || !selectedVersion || form.experiment_type === 1} onClick={addContainer} isLoading={configMutations.createContainer.isPending}>
                  添加容器
                </Button>
              </div>
              {imageConfigQuery.data ? (
                <div className="grid gap-4 lg:grid-cols-3">
                  <CompatibilityBlock title="必须搭配" items={imageConfigQuery.data.typical_companions.required} />
                  <CompatibilityBlock title="推荐搭配" items={imageConfigQuery.data.typical_companions.recommended} />
                  <CompatibilityBlock title="可选搭配" items={imageConfigQuery.data.typical_companions.optional} />
                </div>
              ) : null}
              <div className="grid gap-4 md:grid-cols-2">
                {(activeTemplate?.containers ?? []).map((container) => (
                  <div key={container.id} className="rounded-xl border border-border p-4">
                    <p className="font-semibold">{container.container_name}</p>
                    <p className="mt-1 text-sm text-muted-foreground">{container.image_version?.image_display_name ?? "未关联镜像"} · {container.memory_limit ?? "未设内存"} · 启动顺序 {container.startup_order}</p>
                  </div>
                ))}
              </div>
              {portConflicts.length > 0 ? (
                <div className="rounded-xl border border-destructive/30 bg-destructive/8 p-4 text-sm text-destructive">
                  {portConflicts.map((conflict) => (
                    <p key={`${conflict.protocol}-${conflict.port}`}>{conflict.protocol}/{conflict.port} 被 {conflict.containers.join("、")} 重复占用。</p>
                  ))}
                </div>
              ) : null}
              <div className="rounded-xl border border-border bg-muted/25 p-4">
                <p className="font-semibold">服务发现环境变量预览</p>
                <div className="mt-3 grid gap-2 md:grid-cols-2">
                  {serviceDiscoveryVars.map((item) => (
                    <code key={item.key} className="rounded bg-background px-2 py-1 text-xs">{item.key}={item.value}</code>
                  ))}
                </div>
              </div>
              <div className="flex justify-between">
                <Button variant="outline" onClick={() => setActiveStep(visibleSteps[Math.max(currentStepIndex - 1, 0)]?.id ?? activeStep)}>
                  上一步
                </Button>
                <Button variant="outline" onClick={() => setActiveStep(visibleSteps[Math.min(currentStepIndex + 1, visibleSteps.length - 1)]?.id ?? activeStep)}>
                  下一步
                </Button>
              </div>
            </CardContent>
          </Card>
        </TabsContent>
        <TabsContent value="sim">
          <Card>
            <CardHeader>
              <CardTitle>工具与仿真场景</CardTitle>
            </CardHeader>
            <CardContent className="space-y-5">
              <div className="grid gap-4 md:grid-cols-[1fr_1fr_auto]">
                <FormField label="仿真场景">
                  <Select value={selectedScenarioID} onValueChange={setSelectedScenarioID}>
                    <SelectTrigger><SelectValue placeholder="选择仿真内容" /></SelectTrigger>
                    <SelectContent>
                      {(scenariosQuery.data?.list ?? []).map((scenario) => (
                        <SelectItem key={scenario.id} value={scenario.id}>{scenario.name}</SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </FormField>
                <FormField label="联动组">
                  <Select value={selectedLinkGroupID} onValueChange={setSelectedLinkGroupID}>
                    <SelectTrigger><SelectValue placeholder="可选联动关系" /></SelectTrigger>
                    <SelectContent>
                      {(linkGroupsQuery.data ?? []).map((group) => (
                        <SelectItem key={group.id} value={group.id}>{group.name}</SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </FormField>
                <Button className="self-end" disabled={!canEditSubresources || !selectedScenarioID || form.experiment_type === 2} onClick={addSimScene} isLoading={configMutations.createScene.isPending}>添加内容</Button>
              </div>
              <div className="grid gap-4 lg:grid-cols-2">
                {(activeTemplate?.sim_scenes ?? []).map((scene) => (
                  <div key={scene.id} className="rounded-xl border border-border p-4">
                    <p className="font-semibold">{scene.scenario?.name ?? "未命名场景"}</p>
                    <p className="mt-1 text-sm text-muted-foreground">数据源：{scene.data_source_mode_text} · 联动组：{scene.link_group_name ?? "无"} · 时间控制：{scene.scenario?.time_control_mode ?? "未知"}</p>
                    <pre className="mt-3 overflow-auto rounded-lg bg-muted p-3 text-xs">{JSON.stringify(scene.layout_position ?? {}, null, 2)}</pre>
                  </div>
                ))}
              </div>
              <div className="flex justify-between">
                <Button variant="outline" onClick={() => setActiveStep(visibleSteps[Math.max(currentStepIndex - 1, 0)]?.id ?? activeStep)}>
                  上一步
                </Button>
                <Button variant="outline" onClick={() => setActiveStep(visibleSteps[Math.min(currentStepIndex + 1, visibleSteps.length - 1)]?.id ?? activeStep)}>
                  下一步
                </Button>
              </div>
            </CardContent>
          </Card>
        </TabsContent>
        <TabsContent value="instructions">
          <Card>
            <CardHeader>
              <CardTitle>实验说明</CardTitle>
            </CardHeader>
            <CardContent className="grid gap-4">
              <FormField label="实验说明" className="md:col-span-2">
                <Textarea value={form.instructions ?? ""} onChange={(event) => setForm((current) => ({ ...current, instructions: event.target.value }))} rows={6} />
              </FormField>
              <div className="flex justify-between">
                <Button variant="outline" onClick={() => setActiveStep(visibleSteps[Math.max(currentStepIndex - 1, 0)]?.id ?? activeStep)}>
                  上一步
                </Button>
                <Button variant="outline" onClick={() => setActiveStep(visibleSteps[Math.min(currentStepIndex + 1, visibleSteps.length - 1)]?.id ?? activeStep)}>
                  下一步
                </Button>
              </div>
            </CardContent>
          </Card>
        </TabsContent>
        <TabsContent value="validate">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <ShieldCheck className="h-5 w-5 text-primary" />
                检查点与评分
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="grid gap-4 md:grid-cols-4">
                <SummaryCard label="检查点数量" value={activeTemplate?.checkpoints.length ?? 0} />
                <SummaryCard label="检查点总分" value={totalCheckpointScore} />
                <SummaryCard label="自动权重" value={`${form.auto_weight ?? 0}%`} />
                <SummaryCard label="手动权重" value={`${form.manual_weight ?? 0}%`} />
              </div>
              <div className="grid gap-3 md:grid-cols-[1fr_160px_auto]">
                <Input value={checkpointTitle} onChange={(event) => setCheckpointTitle(event.target.value)} placeholder="检查点标题" />
                <Input type="number" value={checkpointScore} onChange={(event) => setCheckpointScore(event.target.value)} placeholder="分值" />
                <Button disabled={!canEditSubresources} onClick={addCheckpoint} isLoading={configMutations.createCheckpoint.isPending}>添加检查点</Button>
              </div>
              <div className="grid gap-4 md:grid-cols-3">
                {(activeTemplate?.checkpoints ?? []).map((checkpoint) => (
                  <div key={checkpoint.id} className="rounded-xl border border-border p-4">
                    <p className="font-semibold">{checkpoint.title}</p>
                    <p className="mt-1 text-sm text-muted-foreground">{checkpoint.check_type_text} · {checkpoint.scope_text} · {checkpoint.score} 分</p>
                  </div>
                ))}
              </div>
              <div className="grid gap-4 md:grid-cols-3">
                <FormField label="自动评分权重">
                  <Input type="number" value={form.auto_weight ?? 0} onChange={(event) => setForm((current) => ({ ...current, auto_weight: Number(event.target.value) }))} />
                </FormField>
                <FormField label="手动评分权重">
                  <Input type="number" value={form.manual_weight ?? 0} onChange={(event) => setForm((current) => ({ ...current, manual_weight: Number(event.target.value) }))} />
                </FormField>
                <FormField label="成绩策略">
                  <Select value={String(form.score_strategy)} onValueChange={(value) => setForm((current) => ({ ...current, score_strategy: Number(value) as 1 | 2 }))}>
                    <SelectTrigger><SelectValue /></SelectTrigger>
                    <SelectContent>
                      <SelectItem value="1">取最后一次提交</SelectItem>
                      <SelectItem value="2">取最高分</SelectItem>
                    </SelectContent>
                  </Select>
                </FormField>
              </div>
              <Button disabled={!canEditSubresources} onClick={() => templateMutations.validate.mutate()} isLoading={templateMutations.validate.isPending}>
                检查发布条件
              </Button>
              {templateMutations.validate.data ? (
                <div className="space-y-3">
                  <div className="rounded-xl border border-border bg-muted/35 p-4 text-sm">
                    问题 {templateMutations.validate.data.summary.errors}，提醒 {templateMutations.validate.data.summary.warnings}，建议 {templateMutations.validate.data.summary.hints}
                  </div>
                  {templateMutations.validate.data.results.map((level) => (
                    <div key={level.level} className="rounded-xl border border-border p-4">
                      <div className="flex items-center justify-between">
                        <p className="font-semibold">{level.level}. {level.level_name}</p>
                        <Badge variant={level.passed ? "success" : "destructive"}>{level.passed ? "通过" : level.severity}</Badge>
                      </div>
                      <pre className="mt-3 overflow-auto rounded-lg bg-muted p-3 text-xs">{JSON.stringify(level.issues, null, 2)}</pre>
                    </div>
                  ))}
                </div>
              ) : null}
              <div className="flex justify-between">
                <Button variant="outline" onClick={() => setActiveStep(visibleSteps[Math.max(currentStepIndex - 1, 0)]?.id ?? activeStep)}>
                  上一步
                </Button>
                <Button variant="outline" onClick={() => setActiveStep(visibleSteps[Math.min(currentStepIndex + 1, visibleSteps.length - 1)]?.id ?? activeStep)}>
                  下一步
                </Button>
              </div>
            </CardContent>
          </Card>
        </TabsContent>
        <TabsContent value="resource">
          <Card>
            <CardHeader>
              <CardTitle>资源与发布</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="grid gap-4 md:grid-cols-3">
                <FormField label="CPU 限制">
                  <Input value={form.cpu_limit ?? ""} onChange={(event) => setForm((current) => ({ ...current, cpu_limit: event.target.value }))} />
                </FormField>
                <FormField label="内存限制">
                  <Input value={form.memory_limit ?? ""} onChange={(event) => setForm((current) => ({ ...current, memory_limit: event.target.value }))} />
                </FormField>
                <FormField label="磁盘限制">
                  <Input value={form.disk_limit ?? ""} onChange={(event) => setForm((current) => ({ ...current, disk_limit: event.target.value }))} />
                </FormField>
                <FormField label="空闲超时（分钟）">
                  <Input type="number" value={form.idle_timeout ?? 30} onChange={(event) => setForm((current) => ({ ...current, idle_timeout: Number(event.target.value) }))} />
                </FormField>
                <FormField label="K8s 编排">
                  <Textarea value={k8sConfigText} onChange={(event) => setK8sConfigText(event.target.value)} rows={8} className="font-mono" />
                </FormField>
                <div className="rounded-xl border border-border bg-muted/25 p-4 text-sm text-muted-foreground">
                  {validationSummary ? (
                    <div className="space-y-2">
                      <p>错误 {validationSummary.errors}，警告 {validationSummary.warnings}，提示 {validationSummary.hints}</p>
                      <p>{publishBlocked ? "当前还不能发布，请先处理检查中发现的问题。" : "当前已满足发布条件，可以继续发布。"}</p>
                    </div>
                  ) : (
                    <p>建议先检查发布条件，再决定是否对学生开放。</p>
                  )}
                </div>
              </div>
              <div className="flex flex-wrap justify-between gap-3">
                <Button variant="outline" onClick={() => setActiveStep(visibleSteps[Math.max(currentStepIndex - 1, 0)]?.id ?? activeStep)}>
                  上一步
                </Button>
                <div className="flex gap-3">
                  <Button variant="outline" disabled={!canEditSubresources} onClick={saveK8sConfig} isLoading={templateMutations.setK8s.isPending}>
                    保存环境设置
                  </Button>
                  <Button variant="outline" onClick={submitTemplate} isLoading={templateMutations.create.isPending || templateMutations.update.isPending}>
                    保存草稿
                  </Button>
                  <Button disabled={!canEditSubresources || publishBlocked} onClick={() => templateMutations.publish.mutate()} isLoading={templateMutations.publish.isPending}>
                    发布实验内容
                  </Button>
                </div>
              </div>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  );
}



function TemplateCardWithActions({ template, onOpen }: { template: Parameters<typeof ExperimentTemplateCard>[0]["template"]; onOpen: (id: ID) => void }) {
  const mutations = useExperimentTemplateMutations(template.id);
  return (
    <ExperimentTemplateCard
      template={template}
      onOpen={onOpen}
      onPublish={() => mutations.publish.mutate()}
      onClone={() => mutations.clone.mutate()}
      onShare={(_, isShared) => mutations.share.mutate(isShared)}
      isOperating={mutations.publish.isPending || mutations.clone.isPending || mutations.share.isPending}
    />
  );
}

function CompatibilityBlock({ title, items }: { title: string; items: Array<{ image: string; reason: string }> }) {
  return (
    <div className="rounded-xl border border-border bg-muted/25 p-4">
      <p className="font-semibold">{title}</p>
      <div className="mt-3 space-y-2">
        {items.length === 0 ? <p className="text-sm text-muted-foreground">无</p> : null}
        {items.map((item) => (
          <div key={item.image} className="rounded-lg bg-background p-3 text-sm">
            <p className="font-semibold">{item.image}</p>
            <p className="mt-1 text-muted-foreground">{item.reason}</p>
          </div>
        ))}
      </div>
    </div>
  );
}

function SummaryCard({ label, value }: { label: string; value: string | number }) {
  return (
    <div className="rounded-xl border border-border bg-muted/25 p-4">
      <p className="text-sm text-muted-foreground">{label}</p>
      <p className="mt-2 text-lg font-semibold">{value}</p>
    </div>
  );
}
