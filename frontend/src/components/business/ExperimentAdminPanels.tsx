"use client";

// ExperimentAdminPanels.tsx
// 模块04实验资源治理、学校侧监控和平台管理相关业务面板。

import { Activity, DatabaseZap, Server } from "lucide-react";
import { useState } from "react";

import { Badge } from "@/components/ui/Badge";
import { Button } from "@/components/ui/Button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/Card";
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
 */
export function AdminExperimentDashboardPanel() {
  const dashboard = useExperimentAdminDashboard({ page: 1, page_size: 20 });
  const adminInstances = useAdminExperimentInstances({ page: 1, page_size: 20 });

  return (
    <div className="space-y-5">
      <h1 className="font-display text-3xl font-semibold">实验支持总览</h1>
      <div className="grid gap-4 md:grid-cols-4">
        <MetricCard title="总实例" value={dashboard.overview.data?.total_instances ?? 0} />
        <MetricCard title="运行实例" value={dashboard.overview.data?.running_instances ?? 0} />
        <MetricCard title="待审核镜像" value={dashboard.overview.data?.pending_images ?? 0} />
        <MetricCard title="待审核场景" value={dashboard.overview.data?.pending_scenarios ?? 0} />
      </div>
      <div className="grid gap-4 lg:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Server className="h-5 w-5 text-primary" />
              K8s 集群
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            {(dashboard.k8s.data?.nodes ?? []).map((node) => (
              <div key={node.name} className="rounded-xl border border-border p-4">
                <p className="font-semibold">{node.name}</p>
                <p className="mt-1 text-sm text-muted-foreground">
                  {node.status} · CPU {node.cpu_used}/{node.cpu_allocatable} · 内存 {node.memory_used}/{node.memory_allocatable}
                </p>
              </div>
            ))}
          </CardContent>
        </Card>
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <DatabaseZap className="h-5 w-5 text-primary" />
              容器资源
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            {(dashboard.containers.data?.list ?? []).map((item) => (
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
      <TableContainer>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>实验</TableHead>
              <TableHead>学生</TableHead>
              <TableHead>状态</TableHead>
              <TableHead>学校</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {(adminInstances.data?.list ?? []).map((instance) => (
              <TableRow key={instance.id}>
                <TableCell>{instance.template_title}</TableCell>
                <TableCell>{instance.student_name ?? "-"}</TableCell>
                <TableCell><Badge>{instance.status_text}</Badge></TableCell>
                <TableCell>{instance.school_name ?? "-"}</TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </TableContainer>
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

/**
 * ResourceQuotaPanel 学校管理员资源配额面板。
 */
export function ResourceQuotaPanel({ schoolID = "" }: { schoolID?: ID }) {
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

  return (
    <div className="space-y-5">
      <h1 className="font-display text-3xl font-semibold">资源与实验</h1>
      <div className="grid gap-4 md:grid-cols-3">
        <MetricCard title="运行实例" value={schoolMonitorQuery.data?.summary.running ?? 0} />
        <MetricCard title="本校镜像" value={schoolImagesQuery.data?.pagination.total ?? 0} />
        <MetricCard title="配额规则" value={quotasQuery.data?.pagination.total ?? 0} />
      </div>
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
      <div className="grid gap-4 lg:grid-cols-2">
        {(quotasQuery.data?.list ?? []).map((quota) => (
          <Card key={quota.id}>
            <CardHeader>
              <CardTitle>{quota.course_title ?? quota.school_name}</CardTitle>
            </CardHeader>
            <CardContent className="text-sm text-muted-foreground">
              {quota.quota_level_text} · 并发 {quota.used_concurrency}/{quota.max_concurrency} · 单人 {quota.max_per_student} · CPU {quota.used_cpu}/{quota.max_cpu} · 内存 {quota.used_memory}/{quota.max_memory} · 存储 {quota.used_storage}/{quota.max_storage}
            </CardContent>
          </Card>
        ))}
      </div>
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
 */
export function ImagePullStatusPanel() {
  const pullStatusQuery = useImagePullStatus({ page: 1, page_size: 50 });
  const triggerMutation = useTriggerImagePullMutation();
  const [imageVersionID, setImageVersionID] = useState("");

  return (
    <div className="space-y-5">
      <h1 className="font-display text-3xl font-semibold">镜像预拉取状态</h1>
      <Card>
        <CardHeader>
          <CardTitle>手动触发预拉取</CardTitle>
        </CardHeader>
        <CardContent className="grid gap-3 md:grid-cols-[1fr_auto]">
          <FormField label="镜像版本 ID">
            <Input value={imageVersionID} onChange={(event) => setImageVersionID(event.target.value)} placeholder="输入 image_version_id；留空表示全量策略由后端决定" />
          </FormField>
          <Button className="self-end" onClick={() => triggerMutation.mutate({ image_version_id: imageVersionID || undefined, scope: "all" })} isLoading={triggerMutation.isPending}>
            触发预拉取
          </Button>
        </CardContent>
      </Card>
      <TableContainer>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>镜像</TableHead>
              <TableHead>版本</TableHead>
              <TableHead>节点</TableHead>
              <TableHead>状态</TableHead>
              <TableHead>更新时间</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {(pullStatusQuery.data?.list ?? []).map((item) => (
              <TableRow key={item.id}>
                <TableCell>{item.image_name}</TableCell>
                <TableCell>{item.version}</TableCell>
                <TableCell>{item.node_name}</TableCell>
                <TableCell><Badge variant={item.status === 2 ? "success" : "outline"}>{item.status_text}</Badge></TableCell>
                <TableCell>{formatDateTime(item.updated_at)}</TableCell>
              </TableRow>
            ))}
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
