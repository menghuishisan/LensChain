"use client";

// ExperimentTemplatePanels.tsx
// 模块04模板、镜像和仿真场景页面级业务面板。

import { FileArchive, Image as ImageIcon, Plus, ShieldCheck, UploadCloud } from "lucide-react";
import { useRouter } from "next/navigation";
import { useState } from "react";

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
  useExperimentFileUploadMutation,
  useExperimentTemplate,
  useExperimentTemplateMutations,
  useExperimentTemplates,
  useImage,
  useImageCategories,
  useImageConfigTemplate,
  useImageMutations,
  useImages,
  useSimScenarioMutations,
  useSimScenarios,
  useSimLinkGroups,
  useTemplateConfigMutations,
} from "@/hooks/useExperimentTemplates";
import { formatFileSize } from "@/lib/format";
import { buildServiceDiscoveryEnvVars, detectPortConflicts, resolveConditionalEnvVars } from "@/lib/experiment";
import type { ID } from "@/types/api";
import type { CreateImageRequest, ExperimentTemplateRequest, TemplateCheckpointRequest, TemplateContainerRequest } from "@/types/experiment";

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

/**
 * ExperimentTemplateListPanel 教师实验模板列表面板。
 */
export function ExperimentTemplateListPanel() {
  const router = useRouter();
  const [keyword, setKeyword] = useState("");
  const templatesQuery = useExperimentTemplates({ page: 1, page_size: 20, keyword });

  if (templatesQuery.isLoading) {
    return <LoadingState title="正在加载实验模板" description="读取教师可管理的模板列表。" />;
  }

  if (templatesQuery.isError) {
    return <ErrorState title="实验模板加载失败" description="请稍后重试或检查权限。" />;
  }

  const templates = templatesQuery.data?.list ?? [];

  return (
    <div className="space-y-5">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="font-display text-3xl font-semibold">实验教学工作台</h1>
          <p className="mt-2 text-sm text-muted-foreground">创建、验证、发布实验模板，并绑定镜像、检查点和 SimEngine 场景。</p>
        </div>
        <Button onClick={() => router.push("/teacher/experiment-templates/create")}>
          <Plus className="h-4 w-4" />
          新建模板
        </Button>
      </div>
      <Input value={keyword} onChange={(event) => setKeyword(event.target.value)} placeholder="搜索模板标题" />
      {templates.length === 0 ? (
        <EmptyState title="暂无实验模板" description="请先创建模板草稿，再按五层验证结果发布。" />
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
        <h1 className="font-display text-3xl font-semibold">{isEdit ? "编辑实验模板" : "创建实验模板"}</h1>
        <p className="mt-2 text-sm text-muted-foreground">按文档配置基础信息、镜像兼容性、检查点和五层验证，不在前端绕过后端规则。</p>
      </div>
      <Tabs defaultValue="basic">
        <TabsList className="flex w-full flex-wrap justify-start">
          <TabsTrigger value="basic">基础信息</TabsTrigger>
          <TabsTrigger value="images">镜像配置模板</TabsTrigger>
          <TabsTrigger value="sim">仿真场景</TabsTrigger>
          <TabsTrigger value="k8s">K8s 编排</TabsTrigger>
          <TabsTrigger value="validate">五层验证</TabsTrigger>
        </TabsList>
        <TabsContent value="basic">
          <Card>
            <CardHeader>
              <CardTitle>模板基础信息</CardTitle>
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
              <FormField label="总分">
                <Input type="number" value={form.total_score} onChange={(event) => setForm((current) => ({ ...current, total_score: Number(event.target.value) }))} />
              </FormField>
              <FormField label="最大时长（分钟）">
                <Input type="number" value={form.max_duration} onChange={(event) => setForm((current) => ({ ...current, max_duration: Number(event.target.value) }))} />
              </FormField>
              <FormField label="说明" className="md:col-span-2">
                <Textarea value={form.instructions ?? ""} onChange={(event) => setForm((current) => ({ ...current, instructions: event.target.value }))} rows={6} />
              </FormField>
              <Button className="md:col-span-2" onClick={submitTemplate} isLoading={templateMutations.create.isPending || templateMutations.update.isPending}>
                保存模板
              </Button>
            </CardContent>
          </Card>
        </TabsContent>
        <TabsContent value="images">
          <Card>
            <CardHeader>
              <CardTitle>镜像配置模板与工具兼容性</CardTitle>
            </CardHeader>
            <CardContent className="space-y-5">
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
                <Button className="self-end" disabled={!canEditSubresources || !selectedVersion} onClick={addContainer} isLoading={configMutations.createContainer.isPending}>
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
                    <p className="mt-1 text-sm text-muted-foreground">{container.image_version?.image_display_name ?? "未关联镜像"} · {container.memory_limit ?? "未设内存"}</p>
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
            </CardContent>
          </Card>
        </TabsContent>
        <TabsContent value="sim">
          <Card>
            <CardHeader>
              <CardTitle>SimEngine 场景与联动组</CardTitle>
            </CardHeader>
            <CardContent className="space-y-5">
              <div className="grid gap-4 md:grid-cols-[1fr_1fr_auto]">
                <FormField label="仿真场景">
                  <Select value={selectedScenarioID} onValueChange={setSelectedScenarioID}>
                    <SelectTrigger><SelectValue placeholder="选择场景" /></SelectTrigger>
                    <SelectContent>
                      {(scenariosQuery.data?.list ?? []).map((scenario) => (
                        <SelectItem key={scenario.id} value={scenario.id}>{scenario.name}</SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </FormField>
                <FormField label="联动组">
                  <Select value={selectedLinkGroupID} onValueChange={setSelectedLinkGroupID}>
                    <SelectTrigger><SelectValue placeholder="可选联动组" /></SelectTrigger>
                    <SelectContent>
                      {(linkGroupsQuery.data ?? []).map((group) => (
                        <SelectItem key={group.id} value={group.id}>{group.name}</SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </FormField>
                <Button className="self-end" disabled={!canEditSubresources || !selectedScenarioID} onClick={addSimScene} isLoading={configMutations.createScene.isPending}>添加场景</Button>
              </div>
              <div className="grid gap-4 lg:grid-cols-2">
                {(activeTemplate?.sim_scenes ?? []).map((scene) => (
                  <div key={scene.id} className="rounded-xl border border-border p-4">
                    <p className="font-semibold">{scene.scenario?.name ?? "未命名场景"}</p>
                    <p className="mt-1 text-sm text-muted-foreground">数据源：{scene.data_source_mode_text} · 联动组：{scene.link_group_name ?? "无"}</p>
                    <pre className="mt-3 overflow-auto rounded-lg bg-muted p-3 text-xs">{JSON.stringify(scene.layout_position ?? {}, null, 2)}</pre>
                  </div>
                ))}
              </div>
            </CardContent>
          </Card>
        </TabsContent>
        <TabsContent value="k8s">
          <Card>
            <CardHeader>
              <CardTitle>K8s 编排微调</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <p className="text-sm text-muted-foreground">K8s 配置由后端最终校验，前端只编辑文档允许的 namespace_resource、network_policy 等编排片段。</p>
              <Textarea value={k8sConfigText} onChange={(event) => setK8sConfigText(event.target.value)} rows={12} className="font-mono" />
              <Button disabled={!canEditSubresources} onClick={saveK8sConfig} isLoading={templateMutations.setK8s.isPending}>保存 K8s 配置</Button>
            </CardContent>
          </Card>
        </TabsContent>
        <TabsContent value="validate">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <ShieldCheck className="h-5 w-5 text-primary" />
                五层验证
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="grid gap-3 md:grid-cols-[1fr_160px_auto]">
                <Input value={checkpointTitle} onChange={(event) => setCheckpointTitle(event.target.value)} placeholder="检查点标题" />
                <Input type="number" value={checkpointScore} onChange={(event) => setCheckpointScore(event.target.value)} placeholder="分值" />
                <Button disabled={!canEditSubresources} onClick={addCheckpoint} isLoading={configMutations.createCheckpoint.isPending}>添加检查点</Button>
              </div>
              <Button disabled={!canEditSubresources} onClick={() => templateMutations.validate.mutate()} isLoading={templateMutations.validate.isPending}>
                执行后端五层验证
              </Button>
              {templateMutations.validate.data ? (
                <div className="space-y-3">
                  <div className="rounded-xl border border-border bg-muted/35 p-4 text-sm">
                    错误 {templateMutations.validate.data.summary.errors}，警告 {templateMutations.validate.data.summary.warnings}，提示 {templateMutations.validate.data.summary.hints}
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
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  );
}

/**
 * ExperimentImageUploadPanel 教师自定义镜像上传和配置模板登记面板。
 */
export function ExperimentImageUploadPanel() {
  const categoriesQuery = useImageCategories();
  const imageMutations = useImageMutations();
  const uploadMutation = useExperimentFileUploadMutation();
  const [categoryID, setCategoryID] = useState("");
  const [name, setName] = useState("");
  const [displayName, setDisplayName] = useState("");
  const [description, setDescription] = useState("");
  const [registryURL, setRegistryURL] = useState("");
  const [version, setVersion] = useState("1.0.0");
  const [ecosystem, setEcosystem] = useState("ethereum");
  const [documentationURL, setDocumentationURL] = useState("");

  const submit = () => {
    const payload: CreateImageRequest = {
      category_id: categoryID,
      name,
      display_name: displayName,
      description,
      ecosystem,
      default_ports: [{ port: 8545, protocol: "tcp", name: "JSON-RPC" }],
      default_env_vars: [{ key: "CHAIN_ECOSYSTEM", value: ecosystem, desc: "链生态", conditions: [] }],
      default_volumes: [{ path: "/data", desc: "节点数据目录" }],
      typical_companions: { required: [], recommended: [], optional: [] },
      required_dependencies: [],
      resource_recommendation: { cpu: "2", memory: "4Gi", disk: "20Gi" },
      documentation_url: documentationURL || null,
      versions: [{ version, registry_url: registryURL, min_cpu: "1", min_memory: "2Gi", min_disk: "10Gi", is_default: true }],
    };
    imageMutations.create.mutate(payload);
  };

  return (
    <div className="space-y-5">
      <h1 className="font-display text-3xl font-semibold">自定义镜像上传</h1>
      <Card>
        <CardHeader>
          <CardTitle>镜像元数据与默认版本</CardTitle>
        </CardHeader>
        <CardContent className="grid gap-4 md:grid-cols-2">
          <FormField label="镜像分类" required>
            <Select value={categoryID} onValueChange={setCategoryID}>
              <SelectTrigger><SelectValue placeholder="选择分类" /></SelectTrigger>
              <SelectContent>
                {(categoriesQuery.data ?? []).map((category) => (
                  <SelectItem key={category.id} value={category.id}>{category.display_name}</SelectItem>
                ))}
              </SelectContent>
            </Select>
          </FormField>
          <FormField label="生态">
            <Input value={ecosystem} onChange={(event) => setEcosystem(event.target.value)} />
          </FormField>
          <FormField label="镜像名称" required>
            <Input value={name} onChange={(event) => setName(event.target.value)} placeholder="chainspace/geth-lab" />
          </FormField>
          <FormField label="显示名称" required>
            <Input value={displayName} onChange={(event) => setDisplayName(event.target.value)} placeholder="Geth 实验节点" />
          </FormField>
          <FormField label="版本号" required>
            <Input value={version} onChange={(event) => setVersion(event.target.value)} />
          </FormField>
          <FormField label="Registry URL" required>
            <Input value={registryURL} onChange={(event) => setRegistryURL(event.target.value)} placeholder="registry.example/geth-lab:1.0.0" />
          </FormField>
          <FormField label="镜像说明" className="md:col-span-2">
            <Textarea value={description} onChange={(event) => setDescription(event.target.value)} />
          </FormField>
          <FormField label="文档对象 Key" className="md:col-span-2">
            <Input value={documentationURL} onChange={(event) => setDocumentationURL(event.target.value)} placeholder="可通过下方上传文档自动获取" />
          </FormField>
          <label className="inline-flex cursor-pointer items-center gap-2 rounded-lg border border-border px-4 py-2 text-sm font-semibold hover:bg-muted">
            <UploadCloud className="h-4 w-4" />
            上传镜像文档
            <input className="sr-only" type="file" accept=".md,.pdf,.txt" onChange={(event) => {
              const file = event.target.files?.[0];
              if (file) {
                uploadMutation.mutate(
                  { file, purpose: "image_document" },
                  { onSuccess: (uploaded) => setDocumentationURL(uploaded.file_url) },
                );
              }
            }} />
          </label>
          <Button disabled={!categoryID || !name || !displayName || !registryURL} onClick={submit} isLoading={imageMutations.create.isPending}>
            提交审核
          </Button>
        </CardContent>
      </Card>
    </div>
  );
}

/**
 * ExperimentImageLibraryPanel 镜像库与预拉取状态面板。
 */
export function ExperimentImageLibraryPanel({ reviewMode = false }: { reviewMode?: boolean }) {
  const imagesQuery = useImages({ page: 1, page_size: 30 });

  if (imagesQuery.isLoading) {
    return <LoadingState title="正在加载镜像库" description="读取镜像、版本和审核状态。" />;
  }

  return (
    <div className="space-y-5">
      <h1 className="font-display text-3xl font-semibold">镜像库</h1>
      <div className="grid gap-4 xl:grid-cols-2">
        {(imagesQuery.data?.list ?? []).map((image) => (
          <ImageLibraryCard key={image.id} image={image} reviewMode={reviewMode} />
        ))}
      </div>
    </div>
  );
}

/**
 * ExperimentImageDetailPanel 镜像详情、版本、配置模板和结构化文档编辑面板。
 */
export function ExperimentImageDetailPanel({ imageID }: { imageID: ID }) {
  const imageQuery = useImage(imageID);
  const configQuery = useImageConfigTemplate(imageID);
  const imageMutations = useImageMutations(imageID);
  const uploadMutation = useExperimentFileUploadMutation();
  const [displayName, setDisplayName] = useState("");
  const [description, setDescription] = useState("");
  const [documentationURL, setDocumentationURL] = useState("");

  const image = imageQuery.data;
  const config = configQuery.data;
  const serviceVars = buildServiceDiscoveryEnvVars(
    image
      ? [
          {
            container_name: image.name,
            ports: image.default_ports.map((port) => ({ container: port.port, protocol: port.protocol })),
          },
        ]
      : [],
  );
  const conditionalVars = config ? resolveConditionalEnvVars(config.default_env_vars, Object.fromEntries(config.default_env_vars.map((item) => [item.key, item.value]))) : [];

  const saveImage = () => {
    imageMutations.update.mutate({
      display_name: displayName || image?.display_name,
      description: description || image?.description,
      documentation_url: documentationURL || image?.documentation_url,
    });
  };

  if (imageQuery.isLoading) {
    return <LoadingState title="正在加载镜像详情" description="读取镜像版本、配置模板和结构化文档。" />;
  }

  if (!image) {
    return <ErrorState title="镜像不存在" description="请确认镜像 ID 正确且当前账号有权限访问。" />;
  }

  return (
    <div className="space-y-5">
      <div>
        <h1 className="font-display text-3xl font-semibold">{image.display_name}</h1>
        <p className="mt-2 text-sm text-muted-foreground">{image.name} · {image.category_name} · {image.status_text}</p>
      </div>
      <Tabs defaultValue="basic">
        <TabsList className="flex w-full flex-wrap justify-start">
          <TabsTrigger value="basic">基本信息</TabsTrigger>
          <TabsTrigger value="versions">版本</TabsTrigger>
          <TabsTrigger value="config">配置模板</TabsTrigger>
          <TabsTrigger value="docs">文档与敏感字段</TabsTrigger>
        </TabsList>
        <TabsContent value="basic">
          <Card>
            <CardHeader>
              <CardTitle>镜像信息编辑</CardTitle>
            </CardHeader>
            <CardContent className="grid gap-4 md:grid-cols-2">
              <FormField label="显示名称">
                <Input value={displayName || image.display_name} onChange={(event) => setDisplayName(event.target.value)} />
              </FormField>
              <FormField label="文档对象 URL">
                <Input value={documentationURL || image.documentation_url || ""} onChange={(event) => setDocumentationURL(event.target.value)} placeholder="对象存储 key 或后端允许展示的文档地址" />
              </FormField>
              <FormField label="描述" className="md:col-span-2">
                <Textarea value={description || image.description || ""} onChange={(event) => setDescription(event.target.value)} />
              </FormField>
              <Button onClick={saveImage} isLoading={imageMutations.update.isPending}>保存镜像信息</Button>
            </CardContent>
          </Card>
        </TabsContent>
        <TabsContent value="versions">
          <div className="grid gap-4 lg:grid-cols-2">
            {image.versions.map((version) => (
              <Card key={version.id}>
                <CardHeader>
                  <CardTitle className="flex items-center justify-between">
                    {version.version}
                    {version.is_default ? <Badge variant="success">默认</Badge> : null}
                  </CardTitle>
                </CardHeader>
                <CardContent className="space-y-2 text-sm text-muted-foreground">
                  <p>Registry: {version.registry_url}</p>
                  <p>Digest: {version.digest ?? "未记录"}</p>
                  <p>大小: {version.image_size ? formatFileSize(version.image_size) : "未记录"}</p>
                </CardContent>
              </Card>
            ))}
          </div>
        </TabsContent>
        <TabsContent value="config">
          <div className="grid gap-4 lg:grid-cols-2">
            <Card>
              <CardHeader>
                <CardTitle>端口与服务发现</CardTitle>
              </CardHeader>
              <CardContent className="space-y-3">
                {image.default_ports.map((port) => (
                  <div key={`${port.protocol}-${port.port}`} className="rounded-xl border border-border p-3 text-sm">{port.name} · {port.protocol}/{port.port}</div>
                ))}
                {serviceVars.map((item) => (
                  <div key={item.key} className="rounded-lg bg-muted p-2 font-mono text-xs">{item.key}={item.value}</div>
                ))}
              </CardContent>
            </Card>
            <Card>
              <CardHeader>
                <CardTitle>条件环境变量</CardTitle>
              </CardHeader>
              <CardContent className="space-y-2">
                {conditionalVars.map((item) => (
                  <div key={item.key} className="rounded-lg bg-muted p-2 font-mono text-xs">{item.key}={item.value}</div>
                ))}
              </CardContent>
            </Card>
            <div className="lg:col-span-2 grid gap-4 lg:grid-cols-3">
              <CompatibilityBlock title="必须搭配" items={image.typical_companions.required} />
              <CompatibilityBlock title="推荐搭配" items={image.typical_companions.recommended} />
              <CompatibilityBlock title="可选搭配" items={image.typical_companions.optional} />
            </div>
          </div>
        </TabsContent>
        <TabsContent value="docs">
          <Card>
            <CardHeader>
              <CardTitle>镜像文档上传与展示</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <p className="text-sm text-muted-foreground">敏感字段只展示后端允许返回的结构化内容；未允许的密钥、Token、私有 Registry 凭证不会在前端明文回显。</p>
              <label className="inline-flex cursor-pointer items-center gap-2 rounded-lg border border-border px-4 py-2 text-sm font-semibold hover:bg-muted">
                <UploadCloud className="h-4 w-4" />
                上传镜像文档
                <input className="sr-only" type="file" accept=".md,.pdf,.txt" onChange={(event) => {
                  const file = event.target.files?.[0];
                  if (file) {
                    uploadMutation.mutate({ file, purpose: "image_document" });
                  }
                }} />
              </label>
              {uploadMutation.data ? <p className="text-sm text-muted-foreground">文档已上传：{uploadMutation.data.file_url}</p> : null}
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  );
}

/**
 * ExperimentImageReviewPanel 单个镜像审核面板。
 * 按 /admin/images/:id/review 路由参数加载目标镜像，避免审核详情页退化为全量列表。
 */
export function ExperimentImageReviewPanel({ imageID }: { imageID: ID }) {
  const imageQuery = useImage(imageID);
  const imageMutations = useImageMutations(imageID);
  const [comment, setComment] = useState("");

  if (imageQuery.isLoading) {
    return <LoadingState title="正在加载镜像审核详情" description="读取目标镜像、版本和配置模板。" />;
  }

  if (imageQuery.isError || !imageQuery.data) {
    return <ErrorState title="镜像审核详情加载失败" description="请确认镜像 ID 正确，且当前账号拥有超级管理员权限。" />;
  }

  const image = imageQuery.data;

  return (
    <div className="space-y-5">
      <div>
        <h1 className="font-display text-3xl font-semibold">镜像审核：{image.display_name}</h1>
        <p className="mt-2 text-sm text-muted-foreground">{image.name} · {image.category_name} · {image.status_text}</p>
      </div>
      <div className="grid gap-5 xl:grid-cols-[1.2fr_0.8fr]">
        <Card>
          <CardHeader>
            <CardTitle>镜像详情</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="grid gap-3 md:grid-cols-2">
              <ReviewField label="镜像名称" value={image.name} />
              <ReviewField label="显示名称" value={image.display_name} />
              <ReviewField label="分类" value={image.category_name} />
              <ReviewField label="生态" value={image.ecosystem ?? "未填写"} />
              <ReviewField label="引用模板数" value={String(image.usage_count)} />
              <ReviewField label="版本数" value={String(image.versions.length)} />
            </div>
            <div className="rounded-xl border border-border bg-muted/25 p-4">
              <p className="text-sm font-semibold text-foreground">描述</p>
              <p className="mt-2 whitespace-pre-wrap text-sm text-muted-foreground">{image.description || "未填写"}</p>
            </div>
            <div className="space-y-3">
              {image.versions.map((version) => (
                <div key={version.id} className="rounded-xl border border-border p-4">
                  <div className="flex flex-wrap items-center gap-2">
                    <p className="font-semibold">{version.version}</p>
                    {version.is_default ? <Badge variant="success">默认版本</Badge> : null}
                  </div>
                  <p className="mt-2 break-all text-sm text-muted-foreground">{version.registry_url}</p>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader>
            <CardTitle>审核操作</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <FormField label="审核意见">
              <Textarea value={comment} onChange={(event) => setComment(event.target.value)} rows={7} placeholder="请填写审核意见；拒绝时必须说明原因。" />
            </FormField>
            <div className="grid gap-3 sm:grid-cols-2">
              <Button
                onClick={() => imageMutations.review.mutate({ action: "approve", comment: comment || "审核通过" })}
                isLoading={imageMutations.review.isPending}
              >
                审核通过
              </Button>
              <Button
                variant="destructive"
                disabled={!comment.trim()}
                onClick={() => imageMutations.review.mutate({ action: "reject", comment })}
                isLoading={imageMutations.review.isPending}
              >
                审核拒绝
              </Button>
            </div>
            <p className="text-xs leading-5 text-muted-foreground">
              审核通过只表示镜像状态变为正常；镜像预拉取由后端异步触发，进度请在镜像预拉取状态页查看。
            </p>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}

/**
 * SimScenarioLibraryPanel 仿真场景库和场景包上传面板。
 */
export function SimScenarioLibraryPanel({ reviewMode = false }: { reviewMode?: boolean }) {
  const scenariosQuery = useSimScenarios({ page: 1, page_size: 20 });
  const scenarioMutations = useSimScenarioMutations();
  const uploadMutation = useExperimentFileUploadMutation();
  const [containerImageURL, setContainerImageURL] = useState("");
  const [name, setName] = useState("");

  const createScenario = () => {
    scenarioMutations.create.mutate({
      name,
      code: name.toLowerCase().replace(/\s+/g, "-"),
      category: "custom",
      algorithm_type: "custom-container",
      time_control_mode: "process",
      container_image_url: containerImageURL,
      data_source_mode: 1,
      default_params: {},
      interaction_schema: {},
      default_size: { w: 640, h: 360 },
    });
  };

  return (
    <div className="space-y-5">
      <h1 className="font-display text-3xl font-semibold">仿真场景库</h1>
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <UploadCloud className="h-5 w-5 text-primary" />
            自定义场景登记
          </CardTitle>
        </CardHeader>
        <CardContent className="grid gap-4 md:grid-cols-3">
          <FormField label="场景名称">
            <Input value={name} onChange={(event) => setName(event.target.value)} placeholder="PBFT 可视化扩展" />
          </FormField>
          <FormField label="场景容器镜像 URL">
            <Input value={containerImageURL} onChange={(event) => setContainerImageURL(event.target.value)} placeholder="registry.example/scenario:1.0" />
          </FormField>
          <Button className="self-end" onClick={createScenario} isLoading={scenarioMutations.create.isPending}>创建场景</Button>
          <label className="inline-flex cursor-pointer items-center gap-2 rounded-lg border border-border px-4 py-2 text-sm font-semibold hover:bg-muted md:col-span-3">
            <FileArchive className="h-4 w-4" />
            上传场景包
            <input className="sr-only" type="file" accept=".zip,.tar,.tar.gz" onChange={(event) => {
              const file = event.target.files?.[0];
              if (file) {
                uploadMutation.mutate({ file, purpose: "scenario_package" });
              }
            }} />
          </label>
          {uploadMutation.data ? <p className="text-sm text-muted-foreground md:col-span-3">场景包已上传：{uploadMutation.data.file_url}，{formatFileSize(uploadMutation.data.file_size)}</p> : null}
        </CardContent>
      </Card>
      <div className="grid gap-4 xl:grid-cols-2">
        {(scenariosQuery.data?.list ?? []).map((scenario) => (
          <SimScenarioCard key={scenario.id} scenario={scenario} reviewMode={reviewMode} />
        ))}
      </div>
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

function ImageLibraryCard({ image, reviewMode }: { image: NonNullable<ReturnType<typeof useImages>["data"]>["list"][number]; reviewMode: boolean }) {
  const imageMutations = useImageMutations(image.id);
  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <ImageIcon className="h-5 w-5 text-primary" />
          {image.display_name}
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-3">
        <div className="flex flex-wrap gap-2">
          <Badge>{image.category_name}</Badge>
          <Badge variant="outline">{image.status_text}</Badge>
          <Badge variant="secondary">{image.version_count} 版本</Badge>
        </div>
        {reviewMode ? (
          <div className="flex gap-2">
            <Button size="sm" onClick={() => imageMutations.review.mutate({ action: "approve", comment: "审核通过" })}>通过</Button>
            <Button size="sm" variant="destructive" onClick={() => imageMutations.review.mutate({ action: "reject", comment: "请补充镜像文档" })}>拒绝</Button>
          </div>
        ) : null}
      </CardContent>
    </Card>
  );
}

function SimScenarioCard({ scenario, reviewMode }: { scenario: NonNullable<ReturnType<typeof useSimScenarios>["data"]>["list"][number]; reviewMode: boolean }) {
  const scenarioMutations = useSimScenarioMutations(scenario.id);
  return (
    <Card>
      <CardHeader>
        <CardTitle>{scenario.name}</CardTitle>
      </CardHeader>
      <CardContent className="space-y-3">
        <div className="flex flex-wrap gap-2">
          <Badge>{scenario.category_text}</Badge>
          <Badge variant="outline">{scenario.time_control_mode}</Badge>
          <Badge variant={scenario.status === 1 ? "success" : "outline"}>{scenario.status_text}</Badge>
        </div>
        {reviewMode ? (
          <div className="flex gap-2">
            <Button size="sm" onClick={() => scenarioMutations.review.mutate({ action: "approve" })}>通过</Button>
            <Button size="sm" variant="destructive" onClick={() => scenarioMutations.review.mutate({ action: "reject", comment: "场景包验证未通过" })}>拒绝</Button>
          </div>
        ) : null}
      </CardContent>
    </Card>
  );
}

function ReviewField({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-xl border border-border bg-background/70 p-3">
      <p className="text-xs text-muted-foreground">{label}</p>
      <p className="mt-1 break-all text-sm font-semibold text-foreground">{value || "未填写"}</p>
    </div>
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
