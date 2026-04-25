"use client";

// ExperimentInstancePanel.tsx
// 模块04实验实例详情面板，组合生命周期、终端、检查点、快照、报告和 SimEngine。

import { ExternalLink, FlaskConical, Pause, Play, RotateCcw, Send, Square, Upload } from "lucide-react";
import { useEffect, useMemo, useState } from "react";

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
import { Input } from "@/components/ui/Input";
import { LoadingState } from "@/components/ui/LoadingState";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/Tabs";
import { Textarea } from "@/components/ui/Textarea";
import { useCheckpointMutations, useExperimentInstance, useExperimentInstanceLifecycleMutations, useExperimentOperationLogs, useExperimentReport, useExperimentReportMutations, useExperimentMonitorMutations } from "@/hooks/useExperimentInstances";
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
  mode?: "student" | "assist" | "grade";
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
export function ExperimentInstancePanel({ instanceID, mode = "student" }: ExperimentInstancePanelProps) {
  const instanceQuery = useExperimentInstance(instanceID);
  const lifecycle = useExperimentInstanceLifecycleMutations(instanceID);
  const realtime = useExperimentInstanceRealtime(instanceID);
  const checkpointMutations = useCheckpointMutations(instanceID);
  const monitorMutations = useExperimentMonitorMutations();
  const [reportContent, setReportContent] = useState("");
  const [uploadProgress, setUploadProgress] = useState<number | null>(null);
  const [guidanceMessage, setGuidanceMessage] = useState("");
  const [manualScore, setManualScore] = useState("");
  const [manualComment, setManualComment] = useState("");
  const reportQuery = useExperimentReport(instanceID);
  const logsQuery = useExperimentOperationLogs(instanceID, { page: 1, page_size: 20 });
  const reportMutations = useExperimentReportMutations(instanceID);
  const templateID = instanceQuery.data?.template.id ?? "";
  const templateQuery = useExperimentTemplate(templateID);
  const guidanceMessages = useMemo(
    () => realtime.messages.filter((message) => message.type === "guidance_message"),
    [realtime.messages],
  );

  useEffect(() => {
    if (reportQuery.data?.content !== undefined && reportQuery.data.content !== null) {
      setReportContent(reportQuery.data.content);
    }
  }, [reportQuery.data?.content]);

  if (instanceQuery.isLoading) {
    return <LoadingState title="正在加载实验详情" description="正在整理实验环境、评分进度、快照和报告内容。" />;
  }

  if (instanceQuery.isError || !instanceQuery.data) {
    return <ErrorState title="实验详情加载失败" description="请确认实验仍可访问，或稍后再试。" />;
  }

  const instance = instanceQuery.data;
  const isAssistMode = mode === "assist";
  const isGradeMode = mode === "grade";
  const isStudentMode = mode === "student";
  const primaryContainer = instance.containers[0]?.container_name;
  const simSessionID = instance.sim_session_id ?? "";
  const simScenes = templateQuery.data?.sim_scenes ?? [];
  const timeControlMode = simScenes[0]?.scenario?.time_control_mode ?? null;
  const canUseSimEngine = simSessionID.length > 0 || simScenes.length > 0;

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

  const submitGuidance = () => {
    const content = guidanceMessage.trim();
    if (content.length === 0) {
      return;
    }
    monitorMutations.guidance.mutate({ instanceID, content });
    setGuidanceMessage("");
  };

  const submitManualGrade = () => {
    checkpointMutations.manualGrade.mutate({
      manual_score: Number(manualScore || "0"),
      comment: manualComment || null,
    });
  };

  const availableTabs = [
    { id: "terminal", label: isAssistMode ? "只读终端" : "终端", visible: true },
    { id: "checkpoints", label: isGradeMode ? "评分项" : "检查点", visible: true },
    { id: "snapshots", label: "快照", visible: !isGradeMode },
    { id: "report", label: isGradeMode ? "实验报告与总评" : "报告", visible: true },
    { id: "sim", label: "仿真", visible: canUseSimEngine },
  ].filter((tab) => tab.visible);
  const defaultTab = isGradeMode ? "checkpoints" : isStudentMode ? "terminal" : "terminal";

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
              <p className="mt-2 text-sm text-white/60">
                学生 {instance.student.name} · 尝试第 {instance.attempt_no} 次 · 创建于 {formatDateTime(instance.created_at)}
                {isAssistMode ? " · 教师远程协助视图" : ""}
                {isGradeMode ? " · 教师手动评分视图" : ""}
              </p>
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
          {guidanceMessages.length > 0 ? (
            <div className="rounded-xl border border-cyan-300/20 bg-cyan-300/10 p-4 text-sm text-cyan-50">
              最近指导：{guidanceMessages[guidanceMessages.length - 1]?.content ?? "已收到教师消息"}
            </div>
          ) : null}
          <div className="flex flex-wrap gap-2">
            {instance.access_url && isStudentMode ? (
              <Button variant="secondary" size="sm" onClick={() => window.open(instance.access_url ?? "", "_blank", "noopener,noreferrer")}>
                <ExternalLink className="h-4 w-4" />
                打开实验环境
              </Button>
            ) : null}
            <Button variant="outline" size="sm" className="border-white/18 bg-white/8 text-white hover:bg-white/14" onClick={() => lifecycle.pause.mutate()} isLoading={lifecycle.pause.isPending} disabled={isAssistMode || isGradeMode}>
              <Pause className="h-4 w-4" />
              暂停
            </Button>
            <Button variant="outline" size="sm" className="border-white/18 bg-white/8 text-white hover:bg-white/14" onClick={() => lifecycle.resume.mutate({})} isLoading={lifecycle.resume.isPending} disabled={isAssistMode || isGradeMode}>
              <Play className="h-4 w-4" />
              恢复
            </Button>
            <Button variant="outline" size="sm" className="border-white/18 bg-white/8 text-white hover:bg-white/14" onClick={() => lifecycle.restart.mutate()} isLoading={lifecycle.restart.isPending} disabled={isAssistMode}>
              <RotateCcw className="h-4 w-4" />
              重启
            </Button>
            {isStudentMode ? <ConfirmDialog
              title="确认提交实验"
              description="提交后将触发检查点评分和报告验收，运行态实例会进入已提交状态。"
              confirmText="提交"
              confirmVariant="primary"
              trigger={<Button size="sm">提交实验</Button>}
              onConfirm={() => lifecycle.submit.mutate()}
            /> : null}
            <ConfirmDialog
              title="确认销毁实验环境"
              description="结束后会释放实验环境，未保存的运行数据可能丢失。"
              trigger={
                <Button variant="destructive" size="sm">
                  <Square className="h-4 w-4" />
                  销毁
                </Button>
              }
              onConfirm={() => lifecycle.destroy.mutate()}
            />
            {isAssistMode ? (
              <Button variant="outline" size="sm" className="border-white/18 bg-white/8 text-white hover:bg-white/14" onClick={() => monitorMutations.forceDestroy.mutate(instanceID)}>
                强制回收
              </Button>
            ) : null}
          </div>
        </CardContent>
      </Card>

      <Tabs defaultValue={defaultTab}>
        <TabsList className="flex w-full flex-wrap justify-start">
          {availableTabs.map((tab) => (
            <TabsTrigger key={tab.id} value={tab.id}>{tab.label}</TabsTrigger>
          ))}
        </TabsList>
        <TabsContent value="terminal">
          <div className="grid gap-4 xl:grid-cols-[1.1fr_0.9fr]">
            <ExperimentTerminal instanceID={instanceID} container={primaryContainer} readOnly={isAssistMode || isGradeMode} />
            <Card>
              <CardHeader>
                <CardTitle>{isAssistMode ? "指导消息" : "实验说明"}</CardTitle>
              </CardHeader>
              <CardContent className="space-y-4">
                <p className="whitespace-pre-wrap text-sm leading-7 text-muted-foreground">{instance.template.instructions ?? "当前模板未填写实验说明。"}</p>
                {isAssistMode ? (
                  <div className="space-y-3">
                    <Textarea value={guidanceMessage} onChange={(event) => setGuidanceMessage(event.target.value)} rows={5} placeholder="向学生发送指导建议" />
                    <Button onClick={submitGuidance} isLoading={monitorMutations.guidance.isPending} disabled={guidanceMessage.trim().length === 0}>
                      <Send className="h-4 w-4" />
                      发送指导消息
                    </Button>
                  </div>
                ) : null}
              </CardContent>
            </Card>
          </div>
        </TabsContent>
        <TabsContent value="checkpoints">
          <div className="grid gap-4 xl:grid-cols-[1fr_0.9fr]">
            <CheckpointPanel instanceID={instanceID} canGrade={isGradeMode} />
            {isGradeMode ? (
              <Card>
                <CardHeader>
                  <CardTitle>手动评分与总评</CardTitle>
                </CardHeader>
                <CardContent className="space-y-4">
                  <FormField label="手动评分总分">
                    <Input type="number" value={manualScore} onChange={(event) => setManualScore(event.target.value)} placeholder="输入手动评分总分" />
                  </FormField>
                  <FormField label="教师总评">
                    <Textarea value={manualComment} onChange={(event) => setManualComment(event.target.value)} rows={6} placeholder="填写综合评语、问题与改进建议" />
                  </FormField>
                  <Button onClick={submitManualGrade} isLoading={checkpointMutations.manualGrade.isPending}>
                    提交人工评分
                  </Button>
                </CardContent>
              </Card>
            ) : null}
          </div>
        </TabsContent>
        <TabsContent value="snapshots">
          <div className="grid gap-4 xl:grid-cols-[1fr_0.9fr]">
            <SnapshotPanel instanceID={instanceID} />
            <Card>
              <CardHeader>
                <CardTitle>操作历史</CardTitle>
              </CardHeader>
              <CardContent className="space-y-3">
                {(logsQuery.data?.list ?? []).length === 0 ? <p className="text-sm text-muted-foreground">暂无操作历史。</p> : null}
                {(logsQuery.data?.list ?? []).map((log) => (
                  <div key={log.id} className="rounded-xl border border-border p-3 text-sm">
                    <p className="font-medium">{log.operation_type}</p>
                    <p className="mt-1 text-muted-foreground">{log.detail ?? "无详细信息"}</p>
                    <p className="mt-1 text-xs text-muted-foreground">{formatDateTime(log.created_at)}</p>
                  </div>
                ))}
              </CardContent>
            </Card>
          </div>
        </TabsContent>
        <TabsContent value="report">
          <div className="grid gap-4 xl:grid-cols-[1fr_0.9fr]">
            <Card>
              <CardHeader>
                <CardTitle>{isGradeMode ? "实验报告与评分参考" : "实验报告"}</CardTitle>
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
                {!isGradeMode ? (
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
                ) : null}
              </CardContent>
            </Card>
            <Card>
              <CardHeader>
                <CardTitle>评分参考</CardTitle>
              </CardHeader>
              <CardContent className="space-y-3">
                <div className="rounded-xl border border-border p-4 text-sm">
                  <p className="font-medium">自动评分</p>
                  <p className="mt-1 text-muted-foreground">{formatScore(instance.scores.auto_score ?? 0)}</p>
                </div>
                <div className="rounded-xl border border-border p-4 text-sm">
                  <p className="font-medium">手动评分</p>
                  <p className="mt-1 text-muted-foreground">{formatScore(instance.scores.manual_score ?? 0)}</p>
                </div>
                <div className="rounded-xl border border-border p-4 text-sm">
                  <p className="font-medium">总分</p>
                  <p className="mt-1 text-muted-foreground">{formatScore(instance.scores.total_score ?? 0)}</p>
                </div>
              </CardContent>
            </Card>
          </div>
        </TabsContent>
        <TabsContent value="sim">
          {canUseSimEngine ? (
            <div className="space-y-4">
              <Card>
                <CardHeader>
                  <CardTitle>仿真控制信息</CardTitle>
                </CardHeader>
                <CardContent className="grid gap-3 md:grid-cols-3">
                  <div className="rounded-xl border border-border p-4 text-sm">
                    <p className="text-muted-foreground">时间控制模式</p>
                    <p className="mt-1 font-semibold">{timeControlMode === "process" ? "过程化" : timeControlMode === "reactive" ? "交互响应式" : "持续运行式"}</p>
                  </div>
                  <div className="rounded-xl border border-border p-4 text-sm">
                    <p className="text-muted-foreground">仿真场景数量</p>
                    <p className="mt-1 font-semibold">{simScenes.length}</p>
                  </div>
                  <div className="rounded-xl border border-border p-4 text-sm">
                    <p className="text-muted-foreground">会话标识</p>
                    <p className="mt-1 break-all font-semibold">{simSessionID || "纯仿真场景等待会话同步"}</p>
                  </div>
                </CardContent>
              </Card>
              <SimEnginePanel sessionID={simSessionID} scenes={simScenes} />
            </div>
          ) : (
            <EmptyState title="暂时无法显示仿真画面" description="当前实验尚未准备好可视化内容，请稍后再试。" />
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
    return <ErrorState title="实验结果加载失败" description="请确认当前账号仍可查看这次实验结果。" />;
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
