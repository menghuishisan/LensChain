"use client";

// ExperimentAssetPanels.tsx
// 模块04镜像与仿真场景业务面板，负责镜像上传、镜像详情/审核和仿真场景库。

import { FileArchive, Grid3X3, Image as ImageIcon, List, ShieldCheck, ShieldAlert, UploadCloud } from "lucide-react";
import { useState } from "react";

import { Badge } from "@/components/ui/Badge";
import { Button } from "@/components/ui/Button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/Card";
import { ConfirmDialog } from "@/components/ui/ConfirmDialog";
import { EmptyState } from "@/components/ui/EmptyState";
import { ErrorState } from "@/components/ui/ErrorState";
import { FormField } from "@/components/ui/FormField";
import { Input } from "@/components/ui/Input";
import { LoadingState } from "@/components/ui/LoadingState";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/Select";
import { Table, TableBody, TableCell, TableContainer, TableHead, TableHeader, TableRow } from "@/components/ui/Table";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/Tabs";
import { Textarea } from "@/components/ui/Textarea";
import {
  useExperimentFileUploadMutation,
  useImage,
  useImageCategories,
  useImageConfigTemplate,
  useImageMutations,
  useImages,
  useSimScenarioMutations,
  useSimScenarios,
} from "@/hooks/useExperimentTemplates";
import { formatFileSize } from "@/lib/format";
import { buildServiceDiscoveryEnvVars, resolveConditionalEnvVars } from "@/lib/experiment";
import type { ID } from "@/types/api";
import type { AssetStatus, CreateImageRequest, ImageListItem, ImageSourceType } from "@/types/experiment";

// ExperimentImageUploadPanel 教师上传自定义镜像并登记默认版本与文档。
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
      <h1 className="font-display text-3xl font-semibold">上传自定义镜像</h1>
      <Card>
        <CardHeader>
          <CardTitle>镜像信息与默认版本</CardTitle>
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
          <FormField label="Registry 地址" required>
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

// ExperimentImageLibraryPanel 镜像库列表，支持四类Tab、组合筛选、搜索、卡片/表格切换、审核高亮和分页。
export function ExperimentImageLibraryPanel({ reviewMode = false }: { reviewMode?: boolean }) {
  const categoriesQuery = useImageCategories();
  const [categoryTab, setCategoryTab] = useState("all");
  const [keyword, setKeyword] = useState("");
  const [sourceFilter, setSourceFilter] = useState<string>("all");
  const [ecosystemFilter, setEcosystemFilter] = useState("");
  const [statusFilter, setStatusFilter] = useState<string>("all");
  const [viewMode, setViewMode] = useState<"card" | "table">("card");
  const [page, setPage] = useState(1);
  const pageSize = 12;

  // 根据分类Tab映射到 category_id
  const categoryMap = new Map((categoriesQuery.data ?? []).map((c) => [c.name, c.id]));
  const selectedCategoryID = categoryTab !== "all" ? categoryMap.get(categoryTab) ?? undefined : undefined;

  const imagesQuery = useImages({
    page,
    page_size: pageSize,
    keyword: keyword || undefined,
    category_id: selectedCategoryID,
    source_type: sourceFilter !== "all" ? (Number(sourceFilter) as ImageSourceType) : undefined,
    status: statusFilter !== "all" ? (Number(statusFilter) as AssetStatus) : undefined,
  });

  const images = imagesQuery.data?.list ?? [];
  const totalPages = Math.ceil((imagesQuery.data?.pagination?.total ?? 0) / pageSize);

  return (
    <div className="space-y-5">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <h1 className="font-display text-3xl font-semibold">镜像仓库管理</h1>
        <div className="flex gap-2">
          <Button variant={viewMode === "card" ? "primary" : "outline"} size="sm" onClick={() => setViewMode("card")}>
            <Grid3X3 className="h-4 w-4" />
          </Button>
          <Button variant={viewMode === "table" ? "primary" : "outline"} size="sm" onClick={() => setViewMode("table")}>
            <List className="h-4 w-4" />
          </Button>
        </div>
      </div>

      {/* 四类Tab分类筛选 */}
      <Tabs value={categoryTab} onValueChange={(v) => { setCategoryTab(v); setPage(1); }}>
        <TabsList>
          <TabsTrigger value="all">全部</TabsTrigger>
          <TabsTrigger value="chain_node">链节点</TabsTrigger>
          <TabsTrigger value="middleware">中间件</TabsTrigger>
          <TabsTrigger value="tool">工具</TabsTrigger>
          <TabsTrigger value="base">环境基础</TabsTrigger>
        </TabsList>
      </Tabs>

      {/* 搜索 + 组合筛选 */}
      <div className="flex flex-wrap items-center gap-3">
        <Input className="w-64" placeholder="搜索镜像名称" value={keyword} onChange={(e) => { setKeyword(e.target.value); setPage(1); }} />
        <Select value={sourceFilter} onValueChange={(v) => { setSourceFilter(v); setPage(1); }}>
          <SelectTrigger className="w-32"><SelectValue placeholder="来源" /></SelectTrigger>
          <SelectContent>
            <SelectItem value="all">全部来源</SelectItem>
            <SelectItem value="1">官方</SelectItem>
            <SelectItem value="2">自定义</SelectItem>
          </SelectContent>
        </Select>
        <Input className="w-36" placeholder="生态筛选" value={ecosystemFilter} onChange={(e) => setEcosystemFilter(e.target.value)} />
        <Select value={statusFilter} onValueChange={(v) => { setStatusFilter(v); setPage(1); }}>
          <SelectTrigger className="w-32"><SelectValue placeholder="状态" /></SelectTrigger>
          <SelectContent>
            <SelectItem value="all">全部状态</SelectItem>
            <SelectItem value="1">正常</SelectItem>
            <SelectItem value="2">待审核</SelectItem>
            <SelectItem value="3">审核拒绝</SelectItem>
            <SelectItem value="4">已下架</SelectItem>
          </SelectContent>
        </Select>
      </div>

      {imagesQuery.isLoading ? (
        <LoadingState title="正在加载镜像库" description="读取镜像、版本和审核状态。" />
      ) : images.length === 0 ? (
        <EmptyState title="暂无匹配镜像" description="调整筛选条件或上传新镜像。" />
      ) : viewMode === "card" ? (
        <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-4">
          {images.filter((img) => !ecosystemFilter || (img.ecosystem ?? "").toLowerCase().includes(ecosystemFilter.toLowerCase())).map((image) => (
            <ImageLibraryCard key={image.id} image={image} reviewMode={reviewMode} />
          ))}
        </div>
      ) : (
        <TableContainer>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>镜像</TableHead>
                <TableHead>分类</TableHead>
                <TableHead>来源</TableHead>
                <TableHead>生态</TableHead>
                <TableHead>版本数</TableHead>
                <TableHead>引用</TableHead>
                <TableHead>状态</TableHead>
                <TableHead>操作</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {images.filter((img) => !ecosystemFilter || (img.ecosystem ?? "").toLowerCase().includes(ecosystemFilter.toLowerCase())).map((image) => (
                <TableRow key={image.id} className={image.status === 2 ? "ring-2 ring-yellow-500 ring-inset" : ""}>
                  <TableCell className="font-semibold">{image.display_name}</TableCell>
                  <TableCell>{image.category_name}</TableCell>
                  <TableCell>{image.source_type_text}</TableCell>
                  <TableCell>{image.ecosystem ?? "-"}</TableCell>
                  <TableCell>{image.version_count}</TableCell>
                  <TableCell>{image.usage_count}</TableCell>
                  <TableCell><Badge variant={image.status === 2 ? "outline" : image.status === 1 ? "success" : "secondary"}>{image.status_text}</Badge></TableCell>
                  <TableCell>
                    <Button size="sm" variant="outline" onClick={() => window.location.assign(`/admin/images/${image.id}`)}>
                      {image.status === 2 ? "审核" : "详情"}
                    </Button>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </TableContainer>
      )}

      {/* 分页 */}
      {totalPages > 1 ? (
        <div className="flex items-center justify-center gap-2">
          <Button variant="outline" size="sm" disabled={page <= 1} onClick={() => setPage(page - 1)}>上一页</Button>
          <span className="text-sm text-muted-foreground">{page} / {totalPages}</span>
          <Button variant="outline" size="sm" disabled={page >= totalPages} onClick={() => setPage(page + 1)}>下一页</Button>
        </div>
      ) : null}
    </div>
  );
}

// ExperimentImageDetailPanel 管理端镜像详情与配置模板查看。
// 含安全扫描区域、下架按钮、版本增删UI。
export function ExperimentImageDetailPanel({ imageID }: { imageID: ID }) {
  const imageQuery = useImage(imageID);
  const configQuery = useImageConfigTemplate(imageID);
  const imageMutations = useImageMutations(imageID);
  const uploadMutation = useExperimentFileUploadMutation();
  const [displayName, setDisplayName] = useState("");
  const [description, setDescription] = useState("");
  const [documentationURL, setDocumentationURL] = useState("");
  const [newVersion, setNewVersion] = useState("");
  const [newRegistryURL, setNewRegistryURL] = useState("");

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
                <Input value={documentationURL || image.documentation_url || ""} onChange={(event) => setDocumentationURL(event.target.value)} placeholder="填写可公开查看的文档地址" />
              </FormField>
              <FormField label="描述" className="md:col-span-2">
                <Textarea value={description || image.description || ""} onChange={(event) => setDescription(event.target.value)} />
              </FormField>
              <Button onClick={saveImage} isLoading={imageMutations.update.isPending}>保存镜像信息</Button>
              {image.status === 1 ? (
                <ConfirmDialog
                  title="下架镜像"
                  description={`确认下架 "${image.display_name}"？下架后教师将无法在新实验模板中选用此镜像。已使用此镜像的模板不受影响。`}
                  confirmText="确认下架"
                  confirmVariant="destructive"
                  trigger={<Button variant="destructive">下架镜像</Button>}
                  onConfirm={() => imageMutations.remove.mutate()}
                />
              ) : null}
            </CardContent>
          </Card>
        </TabsContent>
        <TabsContent value="versions">
          <div className="space-y-4">
            {/* 添加新版本 */}
            <Card>
              <CardHeader>
                <CardTitle>添加版本</CardTitle>
              </CardHeader>
              <CardContent className="grid gap-3 md:grid-cols-[1fr_1.5fr_auto]">
                <FormField label="版本号">
                  <Input value={newVersion} onChange={(e) => setNewVersion(e.target.value)} placeholder="1.15" />
                </FormField>
                <FormField label="Registry 地址">
                  <Input value={newRegistryURL} onChange={(e) => setNewRegistryURL(e.target.value)} placeholder="registry.lianjing.com/geth:1.15" />
                </FormField>
                <Button className="self-end" disabled={!newVersion || !newRegistryURL} onClick={() => {
                  imageMutations.createVersion.mutate({ version: newVersion, registry_url: newRegistryURL, is_default: false });
                  setNewVersion("");
                  setNewRegistryURL("");
                }} isLoading={imageMutations.createVersion.isPending}>
                  添加版本
                </Button>
              </CardContent>
            </Card>
            {/* 版本列表 */}
            <div className="grid gap-4 lg:grid-cols-2">
              {image.versions.map((version) => (
                <Card key={version.id}>
                  <CardHeader>
                    <CardTitle className="flex items-center justify-between">
                      {version.version}
                      {version.is_default ? <Badge variant="success">默认</Badge> : null}
                    </CardTitle>
                  </CardHeader>
                  <CardContent className="space-y-3 text-sm text-muted-foreground">
                    <p>镜像地址：{version.registry_url}</p>
                    <p>Digest：{version.digest ?? "未记录"}</p>
                    <p>大小：{version.image_size ? formatFileSize(version.image_size) : "未记录"}</p>
                    <p>最低要求：CPU {version.min_cpu ?? "-"} · 内存 {version.min_memory ?? "-"} · 磁盘 {version.min_disk ?? "-"}</p>
                    {/* 安全扫描区域 */}
                    <div className="rounded-xl border border-border bg-muted/25 p-3">
                      <p className="font-semibold text-foreground">安全扫描</p>
                      {version.scanned_at ? (
                        <p className="mt-1 flex items-center gap-1">
                          {version.scan_result ? <ShieldCheck className="h-4 w-4 text-emerald-500" /> : <ShieldAlert className="h-4 w-4 text-yellow-500" />}
                          {version.scan_result ? "通过" : "有风险"} · 扫描时间：{version.scanned_at}
                        </p>
                      ) : (
                        <p className="mt-1">未扫描</p>
                      )}
                    </div>
                    <div className="flex gap-2">
                      {!version.is_default ? (
                        <Button size="sm" variant="outline" onClick={() => imageMutations.setDefaultVersion.mutate(version.id)} isLoading={imageMutations.setDefaultVersion.isPending}>
                          设为默认
                        </Button>
                      ) : null}
                      <ConfirmDialog
                        title="删除版本"
                        description={`确认删除版本 ${version.version}？此操作不可恢复。`}
                        confirmVariant="destructive"
                        trigger={<Button size="sm" variant="destructive">删除</Button>}
                        onConfirm={() => imageMutations.deleteVersion.mutate(version.id)}
                      />
                    </div>
                  </CardContent>
                </Card>
              ))}
            </div>
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
            <div className="grid gap-4 lg:col-span-2 lg:grid-cols-3">
              <CompatibilityBlock title="必须搭配" level="required" items={image.typical_companions.required} />
              <CompatibilityBlock title="推荐搭配" level="recommended" items={image.typical_companions.recommended} />
              <CompatibilityBlock title="可选搭配" level="optional" items={image.typical_companions.optional} />
            </div>
          </div>
        </TabsContent>
        <TabsContent value="docs">
          <Card>
            <CardHeader>
              <CardTitle>镜像文档上传与展示</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <p className="text-sm text-muted-foreground">敏感内容不会直接显示在页面中，页面只展示可安全查看的信息。</p>
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

// ExperimentImageReviewPanel 超级管理员审核单个镜像。
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
        <h1 className="font-display text-3xl font-semibold">审核镜像：{image.display_name}</h1>
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
              <Button onClick={() => imageMutations.review.mutate({ action: "approve", comment: comment || "审核通过" })} isLoading={imageMutations.review.isPending}>
                审核通过
              </Button>
              <Button variant="destructive" disabled={!comment.trim()} onClick={() => imageMutations.review.mutate({ action: "reject", comment })} isLoading={imageMutations.review.isPending}>
                审核拒绝
              </Button>
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}

// SimScenarioLibraryPanel 教师或管理员查看仿真场景库与上传场景包。
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
          <CardTitle>登记或上传自定义场景</CardTitle>
        </CardHeader>
        <CardContent className="grid gap-4 md:grid-cols-3">
          <FormField label="场景名称">
            <Input value={name} onChange={(event) => setName(event.target.value)} placeholder="PBFT 可视化扩展" />
          </FormField>
          <FormField label="场景容器镜像">
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

function ImageLibraryCard({ image, reviewMode }: { image: ImageListItem; reviewMode: boolean }) {
  const imageMutations = useImageMutations(image.id);
  return (
    <Card className={image.status === 2 ? "ring-2 ring-yellow-500" : ""}>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <ImageIcon className="h-5 w-5 text-primary" />
          {image.display_name}
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-3">
        <div className="flex flex-wrap gap-2">
          <Badge>{image.category_name}</Badge>
          <Badge variant="outline">{image.source_type_text}</Badge>
          <Badge variant="secondary">{image.version_count} 版本</Badge>
          {image.ecosystem ? <Badge variant="outline">{image.ecosystem}</Badge> : null}
        </div>
        <p className="text-sm text-muted-foreground">引用 {image.usage_count} 次 · {image.status_text}</p>
        <div className="flex gap-2">
          <Button size="sm" variant="outline" onClick={() => window.location.assign(`/admin/images/${image.id}`)}>
            {image.status === 2 ? "审核" : "查看"}
          </Button>
          {reviewMode && image.status === 2 ? (
            <>
              <Button size="sm" onClick={() => imageMutations.review.mutate({ action: "approve", comment: "审核通过" })}>通过</Button>
              <Button size="sm" variant="destructive" onClick={() => imageMutations.review.mutate({ action: "reject", comment: "请补充镜像文档" })}>拒绝</Button>
            </>
          ) : null}
        </div>
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

function CompatibilityBlock({ title, level, items }: { title: string; level?: "required" | "recommended" | "optional"; items: Array<{ image: string; reason: string }> }) {
  const levelStyles = {
    required: "border-destructive bg-destructive/10",
    recommended: "border-blue-500 bg-blue-500/10",
    optional: "border-border bg-muted/25",
  };
  return (
    <div className={`rounded-xl border p-4 ${levelStyles[level ?? "optional"]}`}>
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
