"use client";

// ExperimentInstanceListPanels.tsx
// 模块04实验实例、教师监控和管理端资源页面级业务面板。

import { Activity, BarChart3, CheckCircle, Circle, Eye, Loader2, Play, RotateCcw, Square } from "lucide-react";
import { useRouter, useSearchParams } from "next/navigation";
import { useEffect, useMemo, useState } from "react";

import { Badge } from "@/components/ui/Badge";
import { Button } from "@/components/ui/Button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/Card";
import { EmptyState } from "@/components/ui/EmptyState";
import { ErrorState } from "@/components/ui/ErrorState";
import { FormField } from "@/components/ui/FormField";
import { Input } from "@/components/ui/Input";
import { LoadingState } from "@/components/ui/LoadingState";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/Select";
import { Table, TableBody, TableCell, TableContainer, TableHead, TableHeader, TableRow } from "@/components/ui/Table";
import {
  useCourseExperimentMonitor,
  useCourseExperimentStatistics,
  useExperimentOperationLogs,
  useExperimentInstanceLifecycleMutations,
  useExperimentInstances,
  useExperimentMonitorMutations,
  useSnapshots,
} from "@/hooks/useExperimentInstances";
import { useExperimentGroupMutations, useExperimentGroups } from "@/hooks/useExperimentGroups";
import { useCourseExperimentMonitorRealtime, useExperimentInstanceRealtime } from "@/hooks/useExperimentRealtime";
import { useExperimentTemplate, useExperimentTemplates } from "@/hooks/useExperimentTemplates";
import { formatDateTime, formatScore } from "@/lib/format";
import type { ID } from "@/types/api";
import type { ExperimentSnapshot } from "@/types/experiment";

/**
 * StudentExperimentListPanel 学生实验实例列表和启动入口。
 */
export function StudentExperimentListPanel() {
  const router = useRouter();
  const instancesQuery = useExperimentInstances({ page: 1, page_size: 20 });
  const templatesQuery = useExperimentTemplates({ page: 1, page_size: 50, status: 2 });
  const lifecycle = useExperimentInstanceLifecycleMutations();
  const [templateID, setTemplateID] = useState("");

  const launch = () => {
    lifecycle.create.mutate(
      { template_id: templateID },
      {
        onSuccess: (created) => {
          if (created.instance_id) {
            router.push(`/student/experiment-instances/${created.instance_id}`);
          }
        },
      },
    );
  };

  if (instancesQuery.isLoading) {
    return <LoadingState title="正在加载我的实验" description="正在整理实验记录、状态和报告入口。" />;
  }

  if (instancesQuery.isError) {
    return <ErrorState title="实验列表加载失败" description="请稍后重试。" />;
  }

  const instances = instancesQuery.data?.list ?? [];

  return (
    <div className="space-y-5">
      <div>
        <h1 className="font-display text-3xl font-semibold">我的实验环境</h1>
        <p className="mt-2 text-sm text-muted-foreground">启动实验、进入终端、提交报告并查看操作历史。</p>
      </div>
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Play className="h-5 w-5 text-primary" />
            启动实验
          </CardTitle>
        </CardHeader>
        <CardContent className="grid gap-3 md:grid-cols-[1fr_auto]">
          <FormField label="已发布实验模板">
            <Select value={templateID} onValueChange={setTemplateID}>
              <SelectTrigger><SelectValue placeholder="选择实验模板" /></SelectTrigger>
              <SelectContent>
                {(templatesQuery.data?.list ?? []).map((template) => (
                  <SelectItem key={template.id} value={template.id}>{template.title}</SelectItem>
                ))}
              </SelectContent>
            </Select>
          </FormField>
          <Button className="self-end" disabled={templateID.length === 0} onClick={launch} isLoading={lifecycle.create.isPending}>
            启动
          </Button>
        </CardContent>
      </Card>
      {instances.length === 0 ? (
        <EmptyState title="还没有实验记录" description="选择实验内容后即可开始操作。" />
      ) : (
        <TableContainer>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>实验</TableHead>
                <TableHead>状态</TableHead>
                <TableHead>得分</TableHead>
                <TableHead>开始时间</TableHead>
                <TableHead>操作</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {instances.map((instance) => (
                <TableRow key={instance.id}>
                  <TableCell className="font-semibold">{instance.template_title}</TableCell>
                  <TableCell><Badge variant={instance.status === 7 ? "success" : "outline"}>{instance.status_text}</Badge></TableCell>
                  <TableCell>{formatScore(instance.total_score ?? 0)}</TableCell>
                  <TableCell>{formatDateTime(instance.started_at)}</TableCell>
                  <TableCell>
                    <Button size="sm" variant="outline" onClick={() => router.push(`/student/experiment-instances/${instance.id}`)}>
                      <Eye className="h-4 w-4" />
                      进入
                    </Button>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </TableContainer>
      )}
    </div>
  );
}

/**
 * ExperimentLaunchPanel 学生指定模板实验启动/排队页面。
 * 支持 WS 分步启动进度跟踪和快照恢复选择。
 */
export function ExperimentLaunchPanel({ templateID }: { templateID: ID }) {
  const router = useRouter();
  const searchParams = useSearchParams();
  const templateQuery = useExperimentTemplate(templateID);
  const lifecycle = useExperimentInstanceLifecycleMutations();
  const [courseID, setCourseID] = useState(searchParams.get("course_id") ?? "");
  const [groupID, setGroupID] = useState(searchParams.get("group_id") ?? "");
  const [snapshotID, setSnapshotID] = useState("");
  const [launchedInstanceID, setLaunchedInstanceID] = useState("");

  // 查询该模板历史快照（需要先有实例 ID，这里通过已有实例列表获取最近的快照）
  const instancesQuery = useExperimentInstances({ page: 1, page_size: 5, template_id: templateID });
  const latestInstanceID = instancesQuery.data?.list?.[0]?.id ?? "";
  const snapshotsQuery = useSnapshots(latestInstanceID);
  const snapshots = snapshotsQuery.data ?? [];

  // WS 实时跟踪启动进度
  const realtime = useExperimentInstanceRealtime(launchedInstanceID, launchedInstanceID.length > 0);

  // 从 WS 消息中提取启动进度步骤
  const launchSteps = useMemo(() => {
    const steps: Array<{ label: string; status: "done" | "active" | "pending"; detail?: string }> = [
      { label: "创建实例", status: "pending" },
      { label: "分配资源", status: "pending" },
      { label: "拉取镜像", status: "pending" },
      { label: "启动容器", status: "pending" },
      { label: "就绪检查", status: "pending" },
    ];
    if (!launchedInstanceID) return steps;

    // 根据 WS 推送的 status_change 和 container_status 消息推进步骤
    const statusMessages = realtime.messages.filter((m) => m.type === "status_change" || m.type === "container_status");
    let maxStep = 0;
    for (const msg of statusMessages) {
      const payload = msg.data ?? {};
      const status = (payload as Record<string, unknown>).status as number | undefined;
      if (status === 1) maxStep = Math.max(maxStep, 1);
      if (status === 2) maxStep = Math.max(maxStep, 2);
      if (msg.type === "container_status") maxStep = Math.max(maxStep, 3);
      if (status === 3) maxStep = Math.max(maxStep, 5);
    }

    for (let i = 0; i < steps.length; i++) {
      if (i < maxStep) steps[i].status = "done";
      else if (i === maxStep && maxStep > 0) steps[i].status = "active";
    }
    // 创建成功即标记第一步完成
    if (launchedInstanceID.length > 0 && maxStep === 0) {
      steps[0].status = "done";
      steps[1].status = "active";
    }
    return steps;
  }, [launchedInstanceID, realtime.messages.length]);

  // 当实例就绪（status=3 running）自动跳转
  useEffect(() => {
    if (!launchedInstanceID) return;
    const readyMsg = realtime.messages.find((m) => {
      const msgData = m.data ?? {};
      return m.type === "status_change" && (msgData as Record<string, unknown>).status === 3;
    });
    if (readyMsg) {
      router.push(`/student/experiment-instances/${launchedInstanceID}`);
    }
  }, [realtime.messages.length, launchedInstanceID, router]);

  const launch = () => {
    lifecycle.create.mutate(
      {
        template_id: templateID,
        course_id: courseID || null,
        group_id: groupID || null,
        snapshot_id: snapshotID || null,
      },
      {
        onSuccess: (created) => {
          if (created.instance_id) {
            setLaunchedInstanceID(created.instance_id);
            // 如果直接就绪（无需等待）则跳转
            if (created.status === 3) {
              router.push(`/student/experiment-instances/${created.instance_id}`);
            }
          }
        },
      },
    );
  };

  if (templateQuery.isLoading) {
    return <LoadingState title="正在加载实验内容" description="正在读取实验要求、时长和评分方式。" />;
  }

  if (!templateQuery.data) {
    return <ErrorState title="实验内容不存在" description="请确认内容已开放，且当前账号可以查看。" />;
  }

  const template = templateQuery.data;
  const createResult = lifecycle.create.data;
  const isLaunching = launchedInstanceID.length > 0 && !realtime.messages.some((m) => {
    const msgData = m.data ?? {};
    return m.type === "status_change" && (msgData as Record<string, unknown>).status === 3;
  });

  return (
    <div className="space-y-5">
      <Card className="border-cyan-500/20 bg-gradient-to-br from-slate-950 via-slate-900 to-cyan-950 text-white">
        <CardHeader>
          <CardTitle className="text-white">{template.title}</CardTitle>
        </CardHeader>
        <CardContent className="grid gap-4 md:grid-cols-4">
          <MetricCard title="实验类型" value={template.experiment_type_text} />
          <MetricCard title="拓扑模式" value={template.topology_mode_text} />
          <MetricCard title="最大时长" value={`${template.max_duration} 分钟`} />
          <MetricCard title="总分" value={template.total_score} />
        </CardContent>
        {/* 容器配置摘要 */}
        {template.containers.length > 0 ? (
          <CardContent className="border-t border-white/10 pt-4 text-sm text-white/70">
            <p>容器配置：{template.containers.map((c) => `${c.image_version?.image_display_name ?? c.container_name}:${c.image_version?.version ?? "latest"}`).join(" + ")}</p>
            <p className="mt-1">资源需求：CPU {template.cpu_limit ?? "-"} · 内存 {template.memory_limit ?? "-"} · 磁盘 {template.disk_limit ?? "-"}</p>
          </CardContent>
        ) : null}
      </Card>

      {/* 快照恢复选择器 */}
      {snapshots.length > 0 && !isLaunching ? (
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <RotateCcw className="h-5 w-5 text-primary" />
              启动方式
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            <label className="flex items-center gap-3 rounded-xl border border-border p-4 cursor-pointer hover:bg-muted/30">
              <input type="radio" name="launch-mode" value="" checked={snapshotID === ""} onChange={() => setSnapshotID("")} className="accent-primary" />
              <div>
                <p className="font-semibold">全新开始</p>
                <p className="text-sm text-muted-foreground">创建一个全新的实验实例</p>
              </div>
            </label>
            {snapshots.map((snapshot) => (
              <label key={snapshot.id} className="flex items-center gap-3 rounded-xl border border-border p-4 cursor-pointer hover:bg-muted/30">
                <input type="radio" name="launch-mode" value={snapshot.id} checked={snapshotID === snapshot.id} onChange={() => setSnapshotID(snapshot.id)} className="accent-primary" />
                <div>
                  <p className="font-semibold">从快照恢复 · {snapshot.snapshot_type_text}</p>
                  <p className="text-sm text-muted-foreground">{snapshot.description ?? "无描述"} · {formatDateTime(snapshot.created_at)}</p>
                </div>
              </label>
            ))}
          </CardContent>
        </Card>
      ) : null}

      {/* 启动参数 */}
      {!isLaunching ? (
        <Card>
          <CardHeader>
            <CardTitle>启动参数</CardTitle>
          </CardHeader>
          <CardContent className="grid gap-3 md:grid-cols-[1fr_1fr_auto]">
            <FormField label="课程 ID" description="从课程或作业进入时可自动带入；未填写时按当前可用范围启动。">
              <Input value={courseID} onChange={(event) => setCourseID(event.target.value)} />
            </FormField>
            <FormField label="分组 ID" description="多人协作实验可填写对应分组标识。">
              <Input value={groupID} onChange={(event) => setGroupID(event.target.value)} />
            </FormField>
            <Button className="self-end" onClick={launch} isLoading={lifecycle.create.isPending}>
              启动实验
            </Button>
          </CardContent>
        </Card>
      ) : null}

      {/* WS 分步启动进度 */}
      {isLaunching ? (
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Loader2 className="h-5 w-5 animate-spin text-primary" />
              正在启动实验环境
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="space-y-3">
              {launchSteps.map((step, index) => (
                <div key={index} className="flex items-center gap-3">
                  {step.status === "done" ? (
                    <CheckCircle className="h-5 w-5 text-emerald-500 shrink-0" />
                  ) : step.status === "active" ? (
                    <Loader2 className="h-5 w-5 animate-spin text-primary shrink-0" />
                  ) : (
                    <Circle className="h-5 w-5 text-muted-foreground/40 shrink-0" />
                  )}
                  <span className={step.status === "pending" ? "text-muted-foreground/50" : step.status === "active" ? "font-semibold text-primary" : "text-foreground"}>{step.label}</span>
                </div>
              ))}
            </div>
            <p className="text-sm text-muted-foreground">实验环境准备完成后将自动进入操作页面。</p>
          </CardContent>
        </Card>
      ) : null}

      {/* 排队等待 */}
      {createResult?.status === 10 ? (
        <Card>
          <CardHeader>
            <CardTitle>正在等待实验资源</CardTitle>
          </CardHeader>
          <CardContent className="text-sm text-muted-foreground">
            当前排队位置 {createResult.queue_position ?? "-"}，预计等待 {createResult.estimated_wait_seconds ?? "-"} 秒。实验准备完成后即可进入操作页面。
          </CardContent>
        </Card>
      ) : null}
    </div>
  );
}

/**
 * TeacherExperimentMonitorPanel 教师课程实验监控面板。
 */
export function TeacherExperimentMonitorPanel({ courseID }: { courseID: ID }) {
  const monitorQuery = useCourseExperimentMonitor(courseID);
  const statisticsQuery = useCourseExperimentStatistics(courseID);
  const realtime = useCourseExperimentMonitorRealtime(courseID);
  const mutations = useExperimentMonitorMutations();
  const router = useRouter();

  if (monitorQuery.isLoading) {
    return <LoadingState title="正在加载课堂实验观察" description="正在整理学生进度、评分情况和资源使用。" />;
  }

  const monitor = monitorQuery.data;

  return (
    <div className="space-y-5">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="font-display text-3xl font-semibold">课堂实验观察</h1>
          <p className="mt-2 text-sm text-muted-foreground">实时查看学生进度、实验状态和资源使用情况。</p>
        </div>
        <Badge variant={realtime.status === "open" ? "success" : "outline"}>{realtime.status === "open" ? "实时连接" : "未连接"}</Badge>
      </div>
      <div className="grid gap-4 md:grid-cols-4">
        <MetricCard title="运行中" value={monitor?.summary.running ?? 0} />
        <MetricCard title="已完成" value={monitor?.summary.completed ?? 0} />
        <MetricCard title="平均进度" value={`${monitor?.summary.avg_progress ?? 0}%`} />
        <MetricCard title="实时事件" value={realtime.messages.length} />
      </div>
      <TableContainer>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>学生</TableHead>
              <TableHead>状态</TableHead>
              <TableHead>进度</TableHead>
              <TableHead>资源</TableHead>
              <TableHead>操作</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {(monitor?.students ?? []).map((student) => (
              <TableRow key={student.student_id}>
                <TableCell>
                  <p className="font-semibold">{student.student_name}</p>
                  <p className="text-xs text-muted-foreground">{student.student_no}</p>
                </TableCell>
                <TableCell><Badge variant={student.status === 7 ? "success" : "outline"}>{student.status_text ?? "未开始"}</Badge></TableCell>
                <TableCell>{student.checkpoints_passed}/{student.checkpoints_total} · {student.progress_percent}%</TableCell>
                <TableCell>{student.cpu_usage ?? "-"} / {student.memory_usage ?? "-"}</TableCell>
                <TableCell className="space-x-2">
                  {student.instance_id ? (
                    <>
                      <Button size="sm" variant="outline" onClick={() => router.push(`/teacher/experiment-instances/${student.instance_id}/assist`)}>协助</Button>
                      <Button size="sm" variant="outline" onClick={() => router.push(`/teacher/experiment-instances/${student.instance_id}/grade`)}>评分</Button>
                      <Button size="sm" variant="destructive" onClick={() => mutations.forceDestroy.mutate(student.instance_id ?? "")}>
                        <Square className="h-4 w-4" />
                        销毁
                      </Button>
                    </>
                  ) : null}
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </TableContainer>
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <BarChart3 className="h-5 w-5 text-primary" />
            实验数据概览
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          {(statisticsQuery.data?.templates ?? []).map((template) => (
            <div key={template.template_id} className="rounded-xl border border-border p-4">
              <p className="font-semibold">{template.template_title}</p>
              <p className="mt-1 text-sm text-muted-foreground">实例 {template.total_instances}，已提交 {template.submitted_instances}，均分 {formatScore(template.avg_score ?? 0)}</p>
            </div>
          ))}
        </CardContent>
      </Card>
    </div>
  );
}

/**
 * TeacherExperimentStatisticsPanel 教师课程实验统计专用面板。
 */
export function TeacherExperimentStatisticsPanel({ courseID }: { courseID: ID }) {
  const statisticsQuery = useCourseExperimentStatistics(courseID);

  if (statisticsQuery.isLoading) {
    return <LoadingState title="正在加载实验数据" description="正在整理实验完成情况、得分和通过率。" />;
  }

  return (
    <div className="space-y-5">
      <h1 className="font-display text-3xl font-semibold">实验数据概览</h1>
      {(statisticsQuery.data?.templates ?? []).length === 0 ? (
        <EmptyState title="暂无实验数据" description="学生提交实验后，这里会显示完成情况和得分概览。" />
      ) : (
        <div className="space-y-4">
          {(statisticsQuery.data?.templates ?? []).map((template) => (
            <Card key={template.template_id}>
              <CardHeader>
                <CardTitle>{template.template_title}</CardTitle>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="grid gap-3 md:grid-cols-4">
                  <MetricCard title="实例数" value={template.total_instances} />
                  <MetricCard title="已提交" value={template.submitted_instances} />
                  <MetricCard title="平均分" value={formatScore(template.avg_score ?? 0)} />
                  <MetricCard title="平均耗时" value={template.avg_duration_minutes ? `${template.avg_duration_minutes} 分钟` : "-"} />
                </div>
                <TableContainer>
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>检查点</TableHead>
                        <TableHead>通过率</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {template.checkpoint_pass_rates.map((checkpoint) => (
                        <TableRow key={checkpoint.checkpoint_id}>
                          <TableCell>{checkpoint.title}</TableCell>
                          <TableCell>{checkpoint.pass_rate}%</TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                </TableContainer>
              </CardContent>
            </Card>
          ))}
        </div>
      )}
    </div>
  );
}

/**
 * ExperimentOperationHistoryPanel 实验操作日志面板。
 */
export function ExperimentOperationHistoryPanel({ instanceID }: { instanceID: ID }) {
  const logsQuery = useExperimentOperationLogs(instanceID, { page: 1, page_size: 50 });

  if (logsQuery.isLoading) {
    return <LoadingState title="正在加载操作记录" description="正在整理实验过程中的关键操作记录。" />;
  }

  return (
    <div className="space-y-5">
      <h1 className="font-display text-3xl font-semibold">实验操作记录</h1>
      <TableContainer>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>时间</TableHead>
              <TableHead>操作人</TableHead>
              <TableHead>类型</TableHead>
              <TableHead>详情</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {(logsQuery.data?.list ?? []).map((log) => (
              <TableRow key={log.id}>
                <TableCell>{formatDateTime(log.created_at)}</TableCell>
                <TableCell>{log.operator_name}</TableCell>
                <TableCell><Badge>{log.operation_type}</Badge></TableCell>
                <TableCell>{log.detail ?? "-"}</TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </TableContainer>
    </div>
  );
}

/**
 * ExperimentGroupManagementPanel 教师实验分组列表面板。
 */
export function ExperimentGroupManagementPanel() {
  const groupsQuery = useExperimentGroups({ page: 1, page_size: 30 });
  const mutations = useExperimentGroupMutations();
  const [templateID, setTemplateID] = useState("");
  const [courseID, setCourseID] = useState("");
  const [groupName, setGroupName] = useState("实验小组A");
  const [maxMembers, setMaxMembers] = useState("4");
  const [groupSize, setGroupSize] = useState("4");

  return (
    <div className="space-y-5">
      <h1 className="font-display text-3xl font-semibold">实验分组</h1>
      <Card>
        <CardHeader>
          <CardTitle>创建分组与自动分配</CardTitle>
        </CardHeader>
        <CardContent className="grid gap-3 md:grid-cols-3">
          <FormField label="模板 ID">
            <Input value={templateID} onChange={(event) => setTemplateID(event.target.value)} />
          </FormField>
          <FormField label="课程 ID">
            <Input value={courseID} onChange={(event) => setCourseID(event.target.value)} />
          </FormField>
          <FormField label="分组名称">
            <Input value={groupName} onChange={(event) => setGroupName(event.target.value)} />
          </FormField>
          <FormField label="最大成员数">
            <Input type="number" value={maxMembers} onChange={(event) => setMaxMembers(event.target.value)} />
          </FormField>
          <FormField label="随机分组人数">
            <Input type="number" value={groupSize} onChange={(event) => setGroupSize(event.target.value)} />
          </FormField>
          <div className="flex items-end gap-2">
            <Button
              disabled={!templateID || !courseID}
              onClick={() =>
                mutations.create.mutate({
                  template_id: templateID,
                  course_id: courseID,
                  group_method: 1,
                  groups: [{ group_name: groupName, max_members: Number(maxMembers), members: [] }],
                })
              }
              isLoading={mutations.create.isPending}
            >
              创建分组
            </Button>
            <Button
              variant="outline"
              disabled={!templateID || !courseID}
              onClick={() => mutations.autoAssign.mutate({ template_id: templateID, course_id: courseID, group_size: Number(groupSize), group_name_prefix: groupName.slice(0, 20) })}
              isLoading={mutations.autoAssign.isPending}
            >
              随机分组
            </Button>
          </div>
        </CardContent>
      </Card>
      <TableContainer>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>分组</TableHead>
              <TableHead>成员</TableHead>
              <TableHead>状态</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {(groupsQuery.data?.list ?? []).map((group) => (
              <TableRow key={group.id}>
                <TableCell>{group.group_name}</TableCell>
                <TableCell>{group.member_count}/{group.max_members}</TableCell>
                <TableCell><Badge>{group.status_text}</Badge></TableCell>
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
