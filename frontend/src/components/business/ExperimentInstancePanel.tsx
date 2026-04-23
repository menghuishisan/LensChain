"use client";

// ExperimentInstancePanel.tsx
// 模块04实验实例详情面板，组合生命周期、终端、检查点、快照、报告和 SimEngine。

import { ExternalLink, FlaskConical, Pause, Play, RotateCcw, Square, Upload } from "lucide-react";
import { useEffect, useState } from "react";

import { CheckpointPanel } from "@/components/business/CheckpointPanel";
import { ExperimentTerminal } from "@/components/business/ExperimentTerminal";
import { SimEnginePanel } from "@/components/business/SimEnginePanel";
import { SnapshotPanel } from "@/components/business/SnapshotPanel";
import { Badge } from "@/components/ui/Badge";
import { Button } from "@/components/ui/Button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/Card";
import { ConfirmDialog } from "@/components/ui/ConfirmDialog";
import { EmptyState } from "@/components/ui/EmptyState";
import { ErrorState } from "@/components/ui/ErrorState";
import { FormField } from "@/components/ui/FormField";
import { LoadingState } from "@/components/ui/LoadingState";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/Tabs";
import { Textarea } from "@/components/ui/Textarea";
import { useExperimentInstance, useExperimentInstanceLifecycleMutations, useExperimentReport, useExperimentReportMutations } from "@/hooks/useExperimentInstances";
import { useExperimentInstanceRealtime } from "@/hooks/useExperimentRealtime";
import { useExperimentTemplate } from "@/hooks/useExperimentTemplates";
import { formatDateTime, formatFileSize, formatScore } from "@/lib/format";
import { buildExperimentResultSummary } from "@/lib/experiment";
import type { ID } from "@/types/api";

/**
 * ExperimentInstancePanel 组件属性。
 */
export interface ExperimentInstancePanelProps {
  instanceID: ID;
  canGrade?: boolean;
  terminalReadOnly?: boolean;
}

function getInstanceStatusVariant(status: number) {
  if ([3, 7].includes(status)) {
    return "success" as const;
  }
  if ([8, 9].includes(status)) {
    return "destructive" as const;
  }
  return "outline" as const;
}

/**
 * ExperimentInstancePanel 实验实例工作台组件。
 */
export function ExperimentInstancePanel({ instanceID, canGrade = false, terminalReadOnly = false }: ExperimentInstancePanelProps) {
  const instanceQuery = useExperimentInstance(instanceID);
  const lifecycle = useExperimentInstanceLifecycleMutations(instanceID);
  const realtime = useExperimentInstanceRealtime(instanceID);
  const [reportContent, setReportContent] = useState("");
  const [uploadProgress, setUploadProgress] = useState<number | null>(null);
  const reportQuery = useExperimentReport(instanceID);
  const reportMutations = useExperimentReportMutations(instanceID);
  const templateID = instanceQuery.data?.template.id ?? "";
  const templateQuery = useExperimentTemplate(templateID);

  useEffect(() => {
    if (reportQuery.data?.content !== undefined && reportQuery.data.content !== null) {
      setReportContent(reportQuery.data.content);
    }
  }, [reportQuery.data?.content]);

  if (instanceQuery.isLoading) {
    return <LoadingState title="正在加载实验实例" description="读取容器、检查点、快照和报告状态。" />;
  }

  if (instanceQuery.isError || !instanceQuery.data) {
    return <ErrorState title="实验实例加载失败" description="请确认实例存在且当前账号有权限访问。" />;
  }

  const instance = instanceQuery.data;
  const primaryContainer = instance.containers[0]?.container_name;
  const simSessionID = instance.sim_session_id ?? "";
  const simScenes = templateQuery.data?.sim_scenes ?? [];

  const uploadReportFile = (file: File) => {
    reportMutations.upload.mutate(
      { file, purpose: "experiment_report", onUploadProgress: setUploadProgress },
      {
        onSuccess: (uploaded) => {
          const payload = {
            content: reportContent,
            file_url: uploaded.file_url,
            file_name: uploaded.file_name,
            file_size: uploaded.file_size,
          };
          if (reportQuery.data) {
            reportMutations.update.mutate(payload);
          } else {
            reportMutations.create.mutate(payload);
          }
        },
        onSettled: () => setUploadProgress(null),
      },
    );
  };

  const saveReport = () => {
    const payload = {
      content: reportContent,
      file_url: reportQuery.data?.file_url ?? null,
      file_name: reportQuery.data?.file_name ?? null,
      file_size: reportQuery.data?.file_size ?? null,
    };
    if (reportQuery.data) {
      reportMutations.update.mutate(payload);
    } else {
      reportMutations.create.mutate(payload);
    }
  };

  return (
    <div className="space-y-6">
      <Card className="overflow-hidden border-cyan-500/20 bg-gradient-to-br from-slate-950 via-slate-900 to-cyan-950 text-white">
        <CardHeader>
          <div className="flex flex-wrap items-start justify-between gap-4">
            <div>
              <CardTitle className="flex items-center gap-2 text-white">
                <FlaskConical className="h-5 w-5 text-cyan-200" />
                {instance.template.title}
              </CardTitle>
              <p className="mt-2 text-sm text-white/60">学生 {instance.student.name} · 尝试第 {instance.attempt_no} 次 · 创建于 {formatDateTime(instance.created_at)}</p>
            </div>
            <Badge variant={getInstanceStatusVariant(instance.status)}>{instance.status_text}</Badge>
          </div>
        </CardHeader>
        <CardContent className="space-y-5">
          <div className="grid gap-3 md:grid-cols-4">
            <div className="rounded-xl border border-white/10 bg-white/7 p-4">
              <p className="text-xs text-white/50">自动分</p>
              <p className="mt-1 text-xl font-semibold">{formatScore(instance.scores.auto_score ?? 0)}</p>
            </div>
            <div className="rounded-xl border border-white/10 bg-white/7 p-4">
              <p className="text-xs text-white/50">人工分</p>
              <p className="mt-1 text-xl font-semibold">{formatScore(instance.scores.manual_score ?? 0)}</p>
            </div>
            <div className="rounded-xl border border-white/10 bg-white/7 p-4">
              <p className="text-xs text-white/50">总分</p>
              <p className="mt-1 text-xl font-semibold">{formatScore(instance.scores.total_score ?? 0)}</p>
            </div>
            <div className="rounded-xl border border-white/10 bg-white/7 p-4">
              <p className="text-xs text-white/50">实时通道</p>
              <p className="mt-1 text-sm font-semibold">{realtime.status === "open" ? "已连接" : "未连接"}</p>
            </div>
          </div>
          <div className="flex flex-wrap gap-2">
            {instance.access_url ? (
              <Button variant="secondary" size="sm" onClick={() => window.open(instance.access_url ?? "", "_blank", "noopener,noreferrer")}>
                <ExternalLink className="h-4 w-4" />
                打开实验环境
              </Button>
            ) : null}
            <Button variant="outline" size="sm" className="border-white/18 bg-white/8 text-white hover:bg-white/14" onClick={() => lifecycle.pause.mutate()} isLoading={lifecycle.pause.isPending}>
              <Pause className="h-4 w-4" />
              暂停
            </Button>
            <Button variant="outline" size="sm" className="border-white/18 bg-white/8 text-white hover:bg-white/14" onClick={() => lifecycle.resume.mutate({})} isLoading={lifecycle.resume.isPending}>
              <Play className="h-4 w-4" />
              恢复
            </Button>
            <Button variant="outline" size="sm" className="border-white/18 bg-white/8 text-white hover:bg-white/14" onClick={() => lifecycle.restart.mutate()} isLoading={lifecycle.restart.isPending}>
              <RotateCcw className="h-4 w-4" />
              重启
            </Button>
            <ConfirmDialog
              title="确认提交实验"
              description="提交后将触发检查点评分和报告验收，运行态实例会进入已提交状态。"
              confirmText="提交"
              confirmVariant="primary"
              trigger={<Button size="sm">提交实验</Button>}
              onConfirm={() => lifecycle.submit.mutate()}
            />
            <ConfirmDialog
              title="确认销毁实验环境"
              description="销毁会释放容器和 SimEngine 会话，未保存的运行态数据可能丢失。"
              trigger={
                <Button variant="destructive" size="sm">
                  <Square className="h-4 w-4" />
                  销毁
                </Button>
              }
              onConfirm={() => lifecycle.destroy.mutate()}
            />
          </div>
        </CardContent>
      </Card>

      <Tabs defaultValue="terminal">
        <TabsList className="flex w-full flex-wrap justify-start">
          <TabsTrigger value="terminal">终端</TabsTrigger>
          <TabsTrigger value="checkpoints">检查点</TabsTrigger>
          <TabsTrigger value="snapshots">快照</TabsTrigger>
          <TabsTrigger value="report">报告</TabsTrigger>
          <TabsTrigger value="sim">SimEngine</TabsTrigger>
        </TabsList>
        <TabsContent value="terminal">
          <ExperimentTerminal instanceID={instanceID} container={primaryContainer} readOnly={terminalReadOnly} />
        </TabsContent>
        <TabsContent value="checkpoints">
          <CheckpointPanel instanceID={instanceID} canGrade={canGrade} />
        </TabsContent>
        <TabsContent value="snapshots">
          <SnapshotPanel instanceID={instanceID} />
        </TabsContent>
        <TabsContent value="report">
          <Card>
            <CardHeader>
              <CardTitle>实验报告</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <FormField label="报告正文" description="报告正文和附件都会保存到后端报告接口，附件必须先走统一上传服务。">
                <Textarea value={reportContent} onChange={(event) => setReportContent(event.target.value)} rows={8} />
              </FormField>
              {reportQuery.data?.file_name ? (
                <div className="rounded-xl border border-border bg-muted/35 p-4 text-sm">
                  当前附件：{reportQuery.data.file_name}，{formatFileSize(reportQuery.data.file_size ?? 0)}
                </div>
              ) : null}
              <div className="flex flex-wrap gap-2">
                <Button onClick={saveReport} isLoading={reportMutations.create.isPending || reportMutations.update.isPending}>保存报告</Button>
                <label className="inline-flex cursor-pointer items-center gap-2 rounded-lg border border-border px-4 py-2 text-sm font-semibold hover:bg-muted">
                  <Upload className="h-4 w-4" />
                  上传附件
                  <input className="sr-only" type="file" accept=".pdf,.doc,.docx,.md,.txt,.zip" onChange={(event) => {
                    const file = event.target.files?.[0];
                    if (file) {
                      uploadReportFile(file);
                    }
                  }} />
                </label>
                {uploadProgress !== null ? <span className="self-center text-sm text-muted-foreground">上传进度 {uploadProgress}%</span> : null}
              </div>
            </CardContent>
          </Card>
        </TabsContent>
        <TabsContent value="sim">
          {simSessionID.length > 0 || simScenes.length > 0 ? (
            <SimEnginePanel sessionID={simSessionID} scenes={simScenes} />
          ) : (
            <EmptyState title="未绑定 SimEngine 会话" description="真实环境实验不会创建 SimEngine 会话；纯仿真或混合实验需要等待后端返回 sim_session_id。" />
          )}
        </TabsContent>
      </Tabs>
    </div>
  );
}

/**
 * ExperimentResultPanel 学生实验结果查看组件。
 */
export function ExperimentResultPanel({ instanceID }: { instanceID: ID }) {
  const instanceQuery = useExperimentInstance(instanceID);
  const reportQuery = useExperimentReport(instanceID);

  if (instanceQuery.isLoading) {
    return <LoadingState title="正在加载实验结果" description="读取检查点、成绩和报告信息。" />;
  }

  if (!instanceQuery.data) {
    return <ErrorState title="实验结果加载失败" description="请确认实例存在且当前账号有权限查看。" />;
  }

  const instance = instanceQuery.data;
  const summary = buildExperimentResultSummary(instance);

  return (
    <div className="space-y-6">
      <Card className="overflow-hidden border-emerald-500/20 bg-gradient-to-br from-slate-950 via-emerald-950 to-cyan-950 text-white">
        <CardHeader>
          <CardTitle className="text-white">{instance.template.title} · 实验结果</CardTitle>
        </CardHeader>
        <CardContent className="grid gap-4 md:grid-cols-4">
          <div className="rounded-xl border border-white/10 bg-white/7 p-4">
            <p className="text-xs text-white/50">总分</p>
            <p className="mt-1 text-3xl font-semibold">{formatScore(summary.totalScore)}</p>
          </div>
          <div className="rounded-xl border border-white/10 bg-white/7 p-4">
            <p className="text-xs text-white/50">检查点得分</p>
            <p className="mt-1 text-3xl font-semibold">{formatScore(summary.checkpointScore)}</p>
          </div>
          <div className="rounded-xl border border-white/10 bg-white/7 p-4">
            <p className="text-xs text-white/50">通过率</p>
            <p className="mt-1 text-3xl font-semibold">{summary.passRate}%</p>
          </div>
          <div className="rounded-xl border border-white/10 bg-white/7 p-4">
            <p className="text-xs text-white/50">通过检查点</p>
            <p className="mt-1 text-3xl font-semibold">{summary.passed}/{summary.total}</p>
          </div>
        </CardContent>
      </Card>
      <Card>
        <CardHeader>
          <CardTitle>检查点结果</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          {instance.checkpoints.map((checkpoint) => (
            <div key={checkpoint.checkpoint_id} className="rounded-xl border border-border p-4">
              <div className="flex items-center justify-between gap-3">
                <div>
                  <p className="font-semibold">{checkpoint.title}</p>
                  <p className="mt-1 text-sm text-muted-foreground">满分 {checkpoint.score} · 得分 {checkpoint.result?.score ?? 0}</p>
                </div>
                <Badge variant={checkpoint.result?.is_passed ? "success" : "outline"}>{checkpoint.result?.is_passed ? "通过" : "未通过"}</Badge>
              </div>
            </div>
          ))}
        </CardContent>
      </Card>
      <Card>
        <CardHeader>
          <CardTitle>实验报告</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          <p className="whitespace-pre-wrap text-sm leading-7 text-muted-foreground">{reportQuery.data?.content ?? "未提交报告正文。"}</p>
          {reportQuery.data?.file_name ? (
            <div className="rounded-xl border border-border bg-muted/35 p-4 text-sm">
              附件：{reportQuery.data.file_name} · {formatFileSize(reportQuery.data.file_size ?? 0)}
            </div>
          ) : null}
        </CardContent>
      </Card>
    </div>
  );
}
