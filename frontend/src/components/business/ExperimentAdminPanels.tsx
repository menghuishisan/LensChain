"use client";

// ExperimentAdminPanels.tsx
// 模块04实验资源治理、学校侧监控和平台管理相关业务面板。

import { Activity, AlertTriangle, Check, DatabaseZap, Server, X } from "lucide-react";
import { useState } from "react";

import { Badge } from "@/components/ui/Badge";
import { Button } from "@/components/ui/Button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/Card";
import { ConfirmDialog } from "@/components/ui/ConfirmDialog";
import { EmptyState } from "@/components/ui/EmptyState";
import { FormField } from "@/components/ui/FormField";
import { Input } from "@/components/ui/Input";
import { LoadingState } from "@/components/ui/LoadingState";
import { Table, TableBody, TableCell, TableContainer, TableHead, TableHeader, TableRow } from "@/components/ui/Table";
import {
  useAdminExperimentInstances,
  useExperimentAdminDashboard,
  useExperimentInstanceLifecycleMutations,
  useResourceQuotaMutations,
  useResourceQuotas,
  useSchoolExperimentMonitor,
  useSchoolImages,
} from "@/hooks/useExperimentInstances";
import { useAuth } from "@/hooks/useAuth";
import { useImagePullStatus, useTriggerImagePullMutation } from "@/hooks/useExperimentTemplates";
import { formatDateTime } from "@/lib/format";
import type { ID } from "@/types/api";

/**
 * AdminExperimentDashboardPanel 管理端实验资源总览面板。
 * 含 K8s 集群状态、各学校资源使用汇总表和异常实例告警。
 */
export function AdminExperimentDashboardPanel() {
  const dashboard = useExperimentAdminDashboard({ page: 1, page_size: 20 });
  const adminInstances = useAdminExperimentInstances({ page: 1, page_size: 20 });
  const quotasQuery = useResourceQuotas({ page: 1, page_size: 50 });

  // 异常实例 = 状态为异常(8)或创建失败(9)
  const abnormalInstances = (adminInstances.data?.list ?? []).filter((inst) => inst.status === 8 || inst.status === 9);

  return (
    <div className="space-y-5">
      <h1 className="font-display text-3xl font-semibold">全平台资源监控</h1>
      <div className="grid gap-4 md:grid-cols-4">
        <MetricCard title="运行实例" value={dashboard.overview.data?.running_instances ?? 0} />
        <MetricCard title="总实例" value={dashboard.overview.data?.total_instances ?? 0} />
        <MetricCard title="镜像总数" value={dashboard.overview.data?.total_images ?? 0} />
        <MetricCard title="待审核" value={(dashboard.overview.data?.pending_images ?? 0) + (dashboard.overview.data?.pending_scenarios ?? 0)} />
      </div>

      {/* K8s 集群状态 + 进度条 */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Server className="h-5 w-5 text-primary" />
            K8s 集群状态
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          {(dashboard.k8s.data?.nodes ?? []).map((node) => {
            const cpuUsed = parseFloat(node.cpu_used) || 0;
            const cpuTotal = parseFloat(node.cpu_allocatable) || 1;
            const memUsed = parseFloat(node.memory_used) || 0;
            const memTotal = parseFloat(node.memory_allocatable) || 1;
            return (
              <div key={node.name} className="rounded-xl border border-border p-4 space-y-2">
                <div className="flex items-center justify-between">
                  <p className="font-semibold">{node.name}</p>
                  <Badge variant={node.status === "Ready" ? "success" : "destructive"}>{node.status}</Badge>
                </div>
                <div className="space-y-1">
                  <div className="flex items-center justify-between text-xs text-muted-foreground">
                    <span>CPU</span><span>{node.cpu_used}/{node.cpu_allocatable}</span>
                  </div>
                  <div className="h-2 rounded-full bg-muted">
                    <div className="h-2 rounded-full bg-primary" style={{ width: `${Math.min(100, Math.round((cpuUsed / cpuTotal) * 100))}%` }} />
                  </div>
                </div>
                <div className="space-y-1">
                  <div className="flex items-center justify-between text-xs text-muted-foreground">
                    <span>内存</span><span>{node.memory_used}/{node.memory_allocatable}</span>
                  </div>
                  <div className="h-2 rounded-full bg-muted">
                    <div className="h-2 rounded-full bg-blue-500" style={{ width: `${Math.min(100, Math.round((memUsed / memTotal) * 100))}%` }} />
                  </div>
                </div>
              </div>
            );
          })}
        </CardContent>
      </Card>

      {/* 各学校资源使用汇总表 */}
      <Card>
        <CardHeader>
          <CardTitle>各学校资源使用</CardTitle>
        </CardHeader>
        <CardContent>
          <TableContainer>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>学校</TableHead>
                  <TableHead>CPU 使用</TableHead>
                  <TableHead>内存使用</TableHead>
                  <TableHead>存储使用</TableHead>
                  <TableHead>并发实例</TableHead>
                  <TableHead>配额使用率</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {(quotasQuery.data?.list ?? []).filter((q) => q.quota_level === 1).map((quota) => {
                  const cpuRatio = calcRatio(quota.used_cpu, quota.max_cpu);
                  return (
                    <TableRow key={quota.id}>
                      <TableCell className="font-semibold">{quota.school_name}</TableCell>
                      <TableCell>{quota.used_cpu}/{quota.max_cpu}</TableCell>
                      <TableCell>{quota.used_memory}/{quota.max_memory}</TableCell>
                      <TableCell>{quota.used_storage}/{quota.max_storage}</TableCell>
                      <TableCell>{quota.used_concurrency}/{quota.max_concurrency}</TableCell>
                      <TableCell>
                        <div className="flex items-center gap-2">
                          <div className="h-2 w-20 rounded-full bg-muted">
                            <div className={`h-2 rounded-full ${cpuRatio > 80 ? "bg-destructive" : "bg-primary"}`} style={{ width: `${Math.min(100, cpuRatio)}%` }} />
                          </div>
                          <span className="text-xs text-muted-foreground">{cpuRatio}%</span>
                        </div>
                      </TableCell>
                    </TableRow>
                  );
                })}
              </TableBody>
            </Table>
          </TableContainer>
        </CardContent>
      </Card>

      {/* 异常实例告警 */}
      {abnormalInstances.length > 0 ? (
        <Card className="border-destructive/30">
          <CardHeader>
            <CardTitle className="flex items-center gap-2 text-destructive">
              <AlertTriangle className="h-5 w-5" />
              异常实例告警 ({abnormalInstances.length})
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            {abnormalInstances.map((instance) => (
              <AbnormalInstanceRow key={instance.id} instance={instance} />
            ))}
          </CardContent>
        </Card>
      ) : null}

      {/* 容器资源详情 */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <DatabaseZap className="h-5 w-5 text-primary" />
            容器资源
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          {(dashboard.containers.data?.list ?? []).slice(0, 10).map((item) => (
            <div key={`${item.instance_id}-${item.container_name}`} className="rounded-xl border border-border p-4">
              <p className="font-semibold">{item.container_name}</p>
              <p className="mt-1 text-sm text-muted-foreground">
                {item.school_name} · {item.student_name} · CPU {item.cpu_usage} · 内存 {item.memory_usage}
              </p>
            </div>
          ))}
        </CardContent>
      </Card>
    </div>
  );
}

/**
 * AdminExperimentInstancesPanel 全平台实验实例管理专用面板。
 */
export function AdminExperimentInstancesPanel() {
  const instancesQuery = useAdminExperimentInstances({ page: 1, page_size: 50 });

  return (
    <div className="space-y-5">
      <h1 className="font-display text-3xl font-semibold">全平台实验记录</h1>
      <TableContainer>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>实验</TableHead>
              <TableHead>学生/学校</TableHead>
              <TableHead>状态</TableHead>
              <TableHead>更新时间</TableHead>
              <TableHead>操作</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {(instancesQuery.data?.list ?? []).map((instance) => (
              <AdminInstanceRow key={instance.id} instance={instance} />
            ))}
          </TableBody>
        </Table>
      </TableContainer>
    </div>
  );
}

/**
 * K8sClusterStatusPanel K8s 集群状态专用面板。
 */
export function K8sClusterStatusPanel() {
  const dashboard = useExperimentAdminDashboard();

  return (
    <div className="space-y-5">
      <h1 className="font-display text-3xl font-semibold">K8s 集群状态</h1>
      <div className="grid gap-4 lg:grid-cols-2">
        {(dashboard.k8s.data?.nodes ?? []).map((node) => (
          <Card key={node.name}>
            <CardHeader>
              <CardTitle>{node.name}</CardTitle>
            </CardHeader>
            <CardContent className="space-y-2 text-sm text-muted-foreground">
              <p>状态：{node.status}</p>
              <p>CPU：{node.cpu_used}/{node.cpu_allocatable}</p>
              <p>内存：{node.memory_used}/{node.memory_allocatable}</p>
            </CardContent>
          </Card>
        ))}
      </div>
      <TableContainer>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Namespace</TableHead>
              <TableHead>Pod 总数</TableHead>
              <TableHead>运行中</TableHead>
              <TableHead>失败</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {(dashboard.k8s.data?.namespaces ?? []).map((namespace) => (
              <TableRow key={namespace.name}>
                <TableCell>{namespace.name}</TableCell>
                <TableCell>{namespace.pod_count}</TableCell>
                <TableCell>{namespace.running_pods}</TableCell>
                <TableCell>{namespace.failed_pods}</TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </TableContainer>
    </div>
  );
}

function AdminInstanceRow({ instance }: { instance: NonNullable<ReturnType<typeof useAdminExperimentInstances>["data"]>["list"][number] }) {
  const lifecycle = useExperimentInstanceLifecycleMutations(instance.id);
  return (
    <TableRow>
      <TableCell>{instance.template_title}</TableCell>
      <TableCell>
        <p>{instance.student_name ?? "-"}</p>
        <p className="text-xs text-muted-foreground">{instance.school_name ?? "-"}</p>
      </TableCell>
      <TableCell><Badge>{instance.status_text}</Badge></TableCell>
      <TableCell>{formatDateTime(instance.updated_at ?? instance.created_at)}</TableCell>
      <TableCell>
        <Button variant="destructive" size="sm" onClick={() => lifecycle.adminForceDestroy.mutate()} isLoading={lifecycle.adminForceDestroy.isPending}>
          强制销毁
        </Button>
      </TableCell>
    </TableRow>
  );
}

function AbnormalInstanceRow({ instance }: { instance: NonNullable<ReturnType<typeof useAdminExperimentInstances>["data"]>["list"][number] }) {
  const lifecycle = useExperimentInstanceLifecycleMutations(instance.id);
  return (
    <div className="flex items-center justify-between rounded-xl border border-destructive/20 bg-destructive/5 p-4">
      <div>
        <p className="font-semibold">{instance.student_name ?? "未知学生"} ({instance.school_name ?? "未知学校"})</p>
        <p className="mt-1 text-sm text-muted-foreground">{instance.template_title} · {instance.error_message ?? instance.status_text}</p>
      </div>
      <Button variant="destructive" size="sm" onClick={() => lifecycle.adminForceDestroy.mutate()} isLoading={lifecycle.adminForceDestroy.isPending}>
        强制回收
      </Button>
    </div>
  );
}

/**
 * ResourceQuotaPanel 学校管理员资源配额面板。
 * 支持 CPU/内存/存储/并发进度条可视化和只读模式。
 */
export function ResourceQuotaPanel({ schoolID = "", readOnly = false }: { schoolID?: ID; readOnly?: boolean }) {
  const { user } = useAuth();
  const quotasQuery = useResourceQuotas({ page: 1, page_size: 20 });
  const schoolImagesQuery = useSchoolImages({ page: 1, page_size: 12 });
  const schoolMonitorQuery = useSchoolExperimentMonitor();
  const quotaMutations = useResourceQuotaMutations();
  const [schoolIDInput, setSchoolIDInput] = useState(schoolID || user?.school_id || "");
  const [courseID, setCourseID] = useState("");
  const [maxCPU, setMaxCPU] = useState("40");
  const [maxMemory, setMaxMemory] = useState("80Gi");
  const [maxStorage, setMaxStorage] = useState("500Gi");
  const [maxConcurrency, setMaxConcurrency] = useState("20");
  const [maxPerStudent, setMaxPerStudent] = useState("2");
  const activeSchoolID = schoolID || user?.school_id || schoolIDInput;

  // 找学校级配额用于进度条显示
  const schoolQuota = (quotasQuery.data?.list ?? []).find((q) => q.quota_level === 1);
  const courseQuotas = (quotasQuery.data?.list ?? []).filter((q) => q.quota_level === 2);

  return (
    <div className="space-y-5">
      <h1 className="font-display text-3xl font-semibold">本校资源配额</h1>
      <div className="grid gap-4 md:grid-cols-3">
        <MetricCard title="运行实例" value={schoolMonitorQuery.data?.summary.running ?? 0} />
        <MetricCard title="本校镜像" value={schoolImagesQuery.data?.pagination.total ?? 0} />
        <MetricCard title="配额规则" value={quotasQuery.data?.pagination.total ?? 0} />
      </div>

      {/* 学校级配额进度条 */}
      {schoolQuota ? (
        <Card>
          <CardHeader>
            <CardTitle>学校级配额</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <QuotaProgressBar label="CPU" used={schoolQuota.used_cpu} total={schoolQuota.max_cpu} />
            <QuotaProgressBar label="内存" used={schoolQuota.used_memory} total={schoolQuota.max_memory} />
            <QuotaProgressBar label="存储" used={schoolQuota.used_storage} total={schoolQuota.max_storage} />
            <QuotaProgressBar label="并发实例" used={String(schoolQuota.used_concurrency)} total={String(schoolQuota.max_concurrency)} />
          </CardContent>
        </Card>
      ) : null}

      {/* 课程配额列表 */}
      {courseQuotas.length > 0 ? (
        <Card>
          <CardHeader>
            <CardTitle>课程配额分配</CardTitle>
          </CardHeader>
          <CardContent>
            <TableContainer>
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>课程</TableHead>
                    <TableHead>并发上限</TableHead>
                    <TableHead>当前并发</TableHead>
                    <TableHead>每人上限</TableHead>
                    {!readOnly ? <TableHead>操作</TableHead> : null}
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {courseQuotas.map((quota) => (
                    <TableRow key={quota.id}>
                      <TableCell className="font-semibold">{quota.course_title ?? quota.id}</TableCell>
                      <TableCell>{quota.max_concurrency}</TableCell>
                      <TableCell>{quota.used_concurrency}</TableCell>
                      <TableCell>{quota.max_per_student}</TableCell>
                      {!readOnly ? <TableCell><Button size="sm" variant="outline">编辑</Button></TableCell> : null}
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </TableContainer>
          </CardContent>
        </Card>
      ) : null}

      {/* 创建配额（仅非只读模式） */}
      {!readOnly ? (
        <Card>
          <CardHeader>
            <CardTitle>资源配额配置</CardTitle>
          </CardHeader>
          <CardContent className="grid gap-3 md:grid-cols-3">
            <FormField label="学校 ID">
              <Input value={schoolIDInput} onChange={(event) => setSchoolIDInput(event.target.value)} placeholder="学校雪花ID" disabled={Boolean(schoolID || user?.school_id)} />
            </FormField>
            <FormField label="课程 ID">
              <Input value={courseID} onChange={(event) => setCourseID(event.target.value)} placeholder="留空表示学校级配额" />
            </FormField>
            <FormField label="最大并发">
              <Input type="number" value={maxConcurrency} onChange={(event) => setMaxConcurrency(event.target.value)} />
            </FormField>
            <FormField label="CPU 配额">
              <Input value={maxCPU} onChange={(event) => setMaxCPU(event.target.value)} />
            </FormField>
            <FormField label="内存配额">
              <Input value={maxMemory} onChange={(event) => setMaxMemory(event.target.value)} />
            </FormField>
            <FormField label="存储配额">
              <Input value={maxStorage} onChange={(event) => setMaxStorage(event.target.value)} />
            </FormField>
            <FormField label="单人并发">
              <Input type="number" value={maxPerStudent} onChange={(event) => setMaxPerStudent(event.target.value)} />
            </FormField>
            <Button
              className="self-end"
              disabled={activeSchoolID.length === 0}
              onClick={() =>
                quotaMutations.create.mutate({
                  quota_level: courseID.length > 0 ? 2 : 1,
                  school_id: activeSchoolID,
                  course_id: courseID || null,
                  max_cpu: maxCPU,
                  max_memory: maxMemory,
                  max_storage: maxStorage,
                  max_concurrency: Number(maxConcurrency),
                  max_per_student: Number(maxPerStudent),
                })
              }
              isLoading={quotaMutations.create.isPending}
            >
              创建配额
            </Button>
            <Button
              variant="outline"
              className="self-end"
              disabled={courseID.length === 0}
              onClick={() =>
                quotaMutations.assignCourse.mutate({
                  courseID,
                  payload: { max_concurrency: Number(maxConcurrency), max_per_student: Number(maxPerStudent) },
                })
              }
              isLoading={quotaMutations.assignCourse.isPending}
            >
              分配课程并发
            </Button>
          </CardContent>
        </Card>
      ) : null}
    </div>
  );
}

/**
 * SchoolImageLibraryPanel 学校管理员本校镜像管理面板。
 */
export function SchoolImageLibraryPanel() {
  const imagesQuery = useSchoolImages({ page: 1, page_size: 30 });

  if (imagesQuery.isLoading) {
    return <LoadingState title="正在加载本校镜像" description="读取本校自定义镜像和审核状态。" />;
  }

  return (
    <div className="space-y-5">
      <h1 className="font-display text-3xl font-semibold">本校镜像管理</h1>
      {(imagesQuery.data?.list ?? []).length === 0 ? (
        <EmptyState title="暂无本校镜像" description="教师上传的自定义镜像审核通过后会出现在这里。" />
      ) : (
        <div className="grid gap-4 lg:grid-cols-2">
          {(imagesQuery.data?.list ?? []).map((image) => (
            <Card key={image.id}>
              <CardHeader>
                <CardTitle>{image.display_name}</CardTitle>
              </CardHeader>
              <CardContent className="space-y-2 text-sm text-muted-foreground">
                <div className="flex flex-wrap gap-2">
                  <Badge>{image.category_name}</Badge>
                  <Badge variant={image.status === 1 ? "success" : "outline"}>{image.status_text}</Badge>
                  <Badge variant="secondary">{image.source_type_text}</Badge>
                </div>
                <p>版本数 {image.version_count} · 使用次数 {image.usage_count}</p>
              </CardContent>
            </Card>
          ))}
        </div>
      )}
    </div>
  );
}

/**
 * ImagePullStatusPanel 镜像预拉取状态和手动触发面板。
 * 含镜像×节点矩阵视图、展开详情、全量拉取按钮和进度条。
 */
export function ImagePullStatusPanel() {
  const pullStatusQuery = useImagePullStatus({ page: 1, page_size: 200 });
  const triggerMutation = useTriggerImagePullMutation();
  const [imageVersionID, setImageVersionID] = useState("");
  const [expandedImage, setExpandedImage] = useState("");

  const pullItems = pullStatusQuery.data?.list ?? [];

  // 按镜像+版本分组，统计各节点状态
  const imageGroups = new Map<string, { imageName: string; version: string; imageVersionID: string; nodes: Array<{ node: string; status: number; statusText: string }> }>();
  for (const item of pullItems) {
    const key = `${item.image_name}:${item.version}`;
    if (!imageGroups.has(key)) {
      imageGroups.set(key, { imageName: item.image_name, version: item.version, imageVersionID: item.image_version_id, nodes: [] });
    }
    imageGroups.get(key)?.nodes.push({ node: item.node_name, status: item.status, statusText: item.status_text });
  }
  const groupList = Array.from(imageGroups.values());

  // 全局统计
  const allNodes = new Set(pullItems.map((i) => i.node_name));
  const totalImages = groupList.length;
  const fullyReady = groupList.filter((g) => g.nodes.length === allNodes.size && g.nodes.every((n) => n.status === 2)).length;
  const partialMissing = totalImages - fullyReady;

  return (
    <div className="space-y-5">
      <h1 className="font-display text-3xl font-semibold">镜像预拉取状态</h1>

      <div className="grid gap-4 md:grid-cols-4">
        <MetricCard title="镜像总数" value={totalImages} />
        <MetricCard title="节点总数" value={allNodes.size} />
        <MetricCard title="全量就绪" value={fullyReady} />
        <MetricCard title="部分缺失" value={partialMissing} />
      </div>

      {/* 手动触发 + 全量拉取 */}
      <Card>
        <CardHeader>
          <CardTitle>触发预拉取</CardTitle>
        </CardHeader>
        <CardContent className="grid gap-3 md:grid-cols-[1fr_auto_auto]">
          <FormField label="镜像版本 ID（留空=全量）">
            <Input value={imageVersionID} onChange={(e) => setImageVersionID(e.target.value)} placeholder="image_version_id" />
          </FormField>
          <Button className="self-end" onClick={() => triggerMutation.mutate({ image_version_id: imageVersionID || undefined, scope: "all" })} isLoading={triggerMutation.isPending}>
            指定拉取
          </Button>
          <Button className="self-end" variant="primary" onClick={() => triggerMutation.mutate({ scope: "all" })} isLoading={triggerMutation.isPending}>
            全量拉取所有缺失镜像
          </Button>
        </CardContent>
      </Card>

      {/* 拉取进度（如果有正在进行的任务） */}
      {triggerMutation.isPending ? (
        <Card>
          <CardContent className="p-4">
            <p className="text-sm font-semibold">拉取任务进行中...</p>
            <div className="mt-2 h-2 rounded-full bg-muted">
              <div className="h-2 rounded-full bg-primary animate-pulse" style={{ width: "60%" }} />
            </div>
          </CardContent>
        </Card>
      ) : null}

      {/* 镜像×节点矩阵表格 */}
      <TableContainer>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>镜像名称</TableHead>
              <TableHead>版本</TableHead>
              <TableHead>节点覆盖</TableHead>
              <TableHead>状态</TableHead>
              <TableHead>操作</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {groupList.map((group) => {
              const key = `${group.imageName}:${group.version}`;
              const readyCount = group.nodes.filter((n) => n.status === 2).length;
              const totalNodeCount = allNodes.size;
              const isFullReady = readyCount === totalNodeCount;
              const isExpanded = expandedImage === key;
              return (
                <>
                  <TableRow key={key} className="cursor-pointer" onClick={() => setExpandedImage(isExpanded ? "" : key)}>
                    <TableCell className="font-semibold">{group.imageName}</TableCell>
                    <TableCell>{group.version}</TableCell>
                    <TableCell>
                      <div className="flex items-center gap-2">
                        <div className="h-2 w-16 rounded-full bg-muted">
                          <div className={`h-2 rounded-full ${isFullReady ? "bg-emerald-500" : readyCount > 0 ? "bg-yellow-500" : "bg-destructive"}`} style={{ width: `${totalNodeCount === 0 ? 0 : Math.round((readyCount / totalNodeCount) * 100)}%` }} />
                        </div>
                        <span className="text-xs text-muted-foreground">{readyCount}/{totalNodeCount}</span>
                      </div>
                    </TableCell>
                    <TableCell>
                      <Badge variant={isFullReady ? "success" : readyCount > 0 ? "outline" : "destructive"}>
                        {isFullReady ? "全量就绪" : readyCount > 0 ? "部分缺失" : "未拉取"}
                      </Badge>
                    </TableCell>
                    <TableCell>
                      {!isFullReady ? (
                        <Button size="sm" variant="outline" onClick={(e) => { e.stopPropagation(); triggerMutation.mutate({ image_version_id: group.imageVersionID, scope: "all" }); }}>
                          拉取
                        </Button>
                      ) : null}
                    </TableCell>
                  </TableRow>
                  {isExpanded ? (
                    <TableRow key={`${key}-detail`}>
                      <TableCell colSpan={5}>
                        <div className="flex flex-wrap gap-2 p-2">
                          {Array.from(allNodes).map((nodeName) => {
                            const nodeStatus = group.nodes.find((n) => n.node === nodeName);
                            const ready = nodeStatus?.status === 2;
                            return (
                              <span key={nodeName} className={`inline-flex items-center gap-1 rounded-lg px-2 py-1 text-xs ${ready ? "bg-emerald-500/15 text-emerald-700" : "bg-destructive/15 text-destructive"}`}>
                                {ready ? <Check className="h-3 w-3" /> : <X className="h-3 w-3" />} {nodeName}
                              </span>
                            );
                          })}
                        </div>
                      </TableCell>
                    </TableRow>
                  ) : null}
                </>
              );
            })}
          </TableBody>
        </Table>
      </TableContainer>
    </div>
  );
}

/**
 * SchoolExperimentMonitorPanel 学校管理员本校实验监控面板。
 */
export function SchoolExperimentMonitorPanel() {
  const monitorQuery = useSchoolExperimentMonitor();

  if (monitorQuery.isLoading) {
    return <LoadingState title="正在加载本校实验概览" description="正在整理本校实验进度和资源使用情况。" />;
  }

  const monitor = monitorQuery.data;

  return (
    <div className="space-y-5">
      <h1 className="font-display text-3xl font-semibold">本校实验概览</h1>
      <div className="grid gap-4 md:grid-cols-4">
        <MetricCard title="总学生" value={monitor?.summary.total_students ?? 0} />
        <MetricCard title="运行中" value={monitor?.summary.running ?? 0} />
        <MetricCard title="已完成" value={monitor?.summary.completed ?? 0} />
        <MetricCard title="平均进度" value={`${monitor?.summary.avg_progress ?? 0}%`} />
      </div>
      <TableContainer>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>学生</TableHead>
              <TableHead>状态</TableHead>
              <TableHead>进度</TableHead>
              <TableHead>资源</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {(monitor?.students ?? []).map((student) => (
              <TableRow key={student.student_id}>
                <TableCell>{student.student_name}</TableCell>
                <TableCell><Badge>{student.status_text ?? "未开始"}</Badge></TableCell>
                <TableCell>{student.progress_percent}%</TableCell>
                <TableCell>{student.cpu_usage ?? "-"} / {student.memory_usage ?? "-"}</TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </TableContainer>
    </div>
  );
}

function QuotaProgressBar({ label, used, total }: { label: string; used: string; total: string }) {
  const ratio = calcRatio(used, total);
  return (
    <div className="space-y-1">
      <div className="flex items-center justify-between text-sm">
        <span>{label}</span>
        <span className="text-muted-foreground">{used} / {total} ({ratio}%)</span>
      </div>
      <div className="h-3 rounded-full bg-muted">
        <div className={`h-3 rounded-full transition-all ${ratio > 80 ? "bg-destructive" : ratio > 60 ? "bg-yellow-500" : "bg-primary"}`} style={{ width: `${ratio}%` }} />
      </div>
    </div>
  );
}

function calcRatio(used: string, max: string) {
  const u = parseFloat((used ?? "0").replace(/[^\d.]/g, ""));
  const m = parseFloat((max ?? "0").replace(/[^\d.]/g, ""));
  if (!Number.isFinite(u) || !Number.isFinite(m) || m <= 0) return 0;
  return Math.max(0, Math.min(100, Math.round((u / m) * 100)));
}

function MetricCard({ title, value }: { title: string; value: string | number }) {
  return (
    <Card>
      <CardContent className="p-5">
        <div className="flex items-center gap-3">
          <div className="rounded-xl bg-primary/10 p-2 text-primary">
            <Activity className="h-5 w-5" />
          </div>
          <div>
            <p className="text-sm text-muted-foreground">{title}</p>
            <p className="font-display text-2xl font-semibold">{value}</p>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}
