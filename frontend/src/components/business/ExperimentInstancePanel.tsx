"use client";

// ExperimentInstancePanel.tsx
// 模块04实验实例详情面板，组合生命周期、终端、检查点、快照、报告和 SimEngine。

import { ExternalLink, FlaskConical, Pause, Play, RotateCcw, Send, Square, Upload, Code, Monitor, Gamepad2, BookOpen, CheckCircle } from "lucide-react";
import { useEffect, useMemo, useRef, useState } from "react";
import { Group as PanelGroup, Panel, Separator as PanelResizeHandle } from "react-resizable-panels";

import { CheckpointPanel } from "@/components/business/CheckpointPanel";
import { ExperimentTerminal } from "@/components/business/ExperimentTerminal";
import { SimEnginePanel } from "@/components/business/SimEnginePanel";
import { SnapshotPanel } from "@/components/business/SnapshotPanel";
import { WebIDEPanel } from "@/components/business/WebIDEPanel";
import { VNCDesktopPanel } from "@/components/business/VNCDesktopPanel";
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

  // 心跳定时器：每60秒上报一次（仅当实例存在且处于运行中）
  const instanceID_ = instanceQuery.data?.id;
  const instanceStatus = instanceQuery.data?.status;
  useEffect(() => {
    if (instanceStatus !== 3) return;
    const interval = setInterval(() => {
      lifecycle.heartbeat.mutate();
    }, 60_000);
    return () => clearInterval(interval);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [instanceID_, instanceStatus]);

  // 处理 WS 推送的空闲超时和时长超限警告
  useEffect(() => {
    const msgs = realtime.messages;
    if (msgs.length === 0) return;
    const last = msgs[msgs.length - 1];
    if (last?.type === 'idle_warning') {
      if (window.confirm('您已较长时间未操作，实验即将因超时被自动暂停。点击"确定"继续实验。')) {
        lifecycle.heartbeat.mutate();
      }
    }
    if (last?.type === 'duration_warning') {
      window.alert('实验即将超时，请尽快完成并提交。');
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [realtime.messages.length]);

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
  const simSessionID = instance.sim_session_id ?? "";
  const simScenes = templateQuery.data?.sim_scenes ?? [];
  const timeControlMode = simScenes[0]?.scenario?.time_control_mode ?? null;
  const canUseSimEngine = simSessionID.length > 0 || simScenes.length > 0;
  const experimentType = templateQuery.data?.experiment_type ?? 2;

  // 从 tools[] 提取工具 URL
  const terminalTool = instance.tools.find((t) => t.kind === "terminal");
  const ideTool = instance.tools.find((t) => t.kind === "ide");
  const desktopTool = instance.tools.find((t) => t.kind === "desktop");
  const explorerTool = instance.tools.find((t) => t.kind === "explorer");
  const monitorTool = instance.tools.find((t) => t.kind === "monitor");

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

  // 教师消息浮动通知
  const latestGuidance = guidanceMessages.length > 0 ? guidanceMessages[guidanceMessages.length - 1] : null;

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
          {latestGuidance ? (
            <div className="fixed bottom-4 right-4 z-40 max-w-md animate-in slide-in-from-bottom-4 rounded-xl border border-cyan-300/20 bg-cyan-950/95 p-4 text-sm text-cyan-50 shadow-lg backdrop-blur">
              <p className="font-medium">教师指导</p>
              <p className="mt-1">{latestGuidance.content ?? "已收到教师消息"}</p>
            </div>
          ) : null}
          <div className="flex flex-wrap gap-2">
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

      <PanelGroup orientation="horizontal" className="min-h-[600px] rounded-xl border border-border">
        {/* 左侧面板：说明 + 检查点 */}
        <Panel defaultSize={40} minSize={25}>
          <Tabs defaultValue={isGradeMode ? "checkpoints" : "instructions"} className="h-full flex flex-col">
            <TabsList className="flex w-full flex-wrap justify-start border-b rounded-none">
              <TabsTrigger value="instructions"><BookOpen className="h-3.5 w-3.5 mr-1" />说明</TabsTrigger>
              <TabsTrigger value="checkpoints"><CheckCircle className="h-3.5 w-3.5 mr-1" />{isGradeMode ? '评分项' : '检查点'}</TabsTrigger>
              {!isGradeMode && <TabsTrigger value="snapshots">快照</TabsTrigger>}
              <TabsTrigger value="report">{isGradeMode ? '报告与总评' : '报告'}</TabsTrigger>
            </TabsList>

            <TabsContent value="instructions" className="flex-1 overflow-auto p-4">
              <p className="whitespace-pre-wrap text-sm leading-7 text-muted-foreground">
                {instance.template.instructions ?? "当前模板未填写实验说明。"}
              </p>
              {isAssistMode && (
                <div className="mt-4 space-y-3 border-t pt-4">
                  <p className="text-sm font-medium">指导消息</p>
                  <Textarea value={guidanceMessage} onChange={(e) => setGuidanceMessage(e.target.value)} rows={4} placeholder="向学生发送指导建议" />
                  <Button size="sm" onClick={submitGuidance} isLoading={monitorMutations.guidance.isPending} disabled={guidanceMessage.trim().length === 0}>
                    <Send className="h-4 w-4" />发送
                  </Button>
                </div>
              )}
            </TabsContent>

            <TabsContent value="checkpoints" className="flex-1 overflow-auto p-4">
              <CheckpointPanel instanceID={instanceID} canGrade={isGradeMode} />
              {isGradeMode && (
                <div className="mt-4 space-y-3 border-t pt-4">
                  <FormField label="手动评分总分">
                    <Input type="number" value={manualScore} onChange={(e) => setManualScore(e.target.value)} placeholder="输入手动评分总分" />
                  </FormField>
                  <FormField label="教师总评">
                    <Textarea value={manualComment} onChange={(e) => setManualComment(e.target.value)} rows={4} placeholder="填写综合评语" />
                  </FormField>
                  <Button size="sm" onClick={submitManualGrade} isLoading={checkpointMutations.manualGrade.isPending}>
                    提交人工评分
                  </Button>
                </div>
              )}
            </TabsContent>

            {!isGradeMode && (
              <TabsContent value="snapshots" className="flex-1 overflow-auto p-4">
                <SnapshotPanel instanceID={instanceID} />
                <div className="mt-4 space-y-3 border-t pt-4">
                  <p className="text-sm font-medium">操作历史</p>
                  {(logsQuery.data?.list ?? []).length === 0 ? <p className="text-sm text-muted-foreground">暂无操作历史。</p> : null}
                  {(logsQuery.data?.list ?? []).map((log) => (
                    <div key={log.id} className="rounded-lg border border-border p-3 text-sm">
                      <p className="font-medium">{log.operation_type}</p>
                      <p className="mt-1 text-muted-foreground">{log.detail ?? "无详细信息"}</p>
                      <p className="mt-1 text-xs text-muted-foreground">{formatDateTime(log.created_at)}</p>
                    </div>
                  ))}
                </div>
              </TabsContent>
            )}

            <TabsContent value="report" className="flex-1 overflow-auto p-4">
              <FormField label="报告正文">
                <Textarea value={reportContent} onChange={(e) => setReportContent(e.target.value)} rows={8} />
              </FormField>
              {reportQuery.data?.file_name && (
                <div className="mt-2 rounded-lg border bg-muted/35 p-3 text-sm">
                  当前附件：{reportQuery.data.file_name}，{formatFileSize(reportQuery.data.file_size ?? 0)}
                </div>
              )}
              {!isGradeMode && (
                <div className="mt-3 flex flex-wrap gap-2">
                  <Button size="sm" onClick={saveReport} isLoading={reportMutations.create.isPending || reportMutations.update.isPending}>保存报告</Button>
                  <label className="inline-flex cursor-pointer items-center gap-2 rounded-lg border px-3 py-1.5 text-sm font-medium hover:bg-muted">
                    <Upload className="h-4 w-4" />上传附件
                    <input className="sr-only" type="file" accept=".pdf,.doc,.docx,.md,.txt,.zip" onChange={(e) => { const f = e.target.files?.[0]; if (f) uploadReportFile(f); }} />
                  </label>
                  {uploadProgress !== null && <span className="self-center text-sm text-muted-foreground">上传 {uploadProgress}%</span>}
                </div>
              )}
            </TabsContent>
          </Tabs>
        </Panel>

        <PanelResizeHandle className="w-1.5 bg-border hover:bg-primary/20 transition-colors cursor-col-resize" />

        {/* 右侧面板：终端/IDE/桌面/仿真（按实验类型动态显隐） */}
        <Panel defaultSize={60} minSize={35}>
          <Tabs defaultValue={experimentType === 1 ? "sim" : (terminalTool ? "terminal" : (ideTool ? "ide" : "sim"))} className="h-full flex flex-col">
            <TabsList className="flex w-full flex-wrap justify-start border-b rounded-none">
              {terminalTool && (
                <TabsTrigger value="terminal">{isAssistMode ? '只读终端' : '终端'}</TabsTrigger>
              )}
              {ideTool && (
                <TabsTrigger value="ide"><Code className="h-3.5 w-3.5 mr-1" />IDE</TabsTrigger>
              )}
              {desktopTool && (
                <TabsTrigger value="desktop"><Monitor className="h-3.5 w-3.5 mr-1" />桌面</TabsTrigger>
              )}
              {explorerTool && (
                <TabsTrigger value="explorer">浏览器</TabsTrigger>
              )}
              {monitorTool && (
                <TabsTrigger value="monitor">监控</TabsTrigger>
              )}
              {canUseSimEngine && (
                <TabsTrigger value="sim"><Gamepad2 className="h-3.5 w-3.5 mr-1" />仿真</TabsTrigger>
              )}
            </TabsList>

            {terminalTool && (
              <TabsContent value="terminal" className="flex-1 overflow-hidden p-2">
                <ExperimentTerminal
                  instanceID={instanceID}
                  containers={instance.containers ?? []}
                  readOnly={isAssistMode || isGradeMode}
                  className="h-full"
                />
              </TabsContent>
            )}

            {ideTool && (
              <TabsContent value="ide" className="flex-1 overflow-hidden p-2">
                <WebIDEPanel accessUrl={ideTool.proxy_url} className="h-full" />
              </TabsContent>
            )}

            {desktopTool && (
              <TabsContent value="desktop" className="flex-1 overflow-hidden p-2">
                <VNCDesktopPanel accessUrl={desktopTool.proxy_url} className="h-full" />
              </TabsContent>
            )}

            {explorerTool && (
              <TabsContent value="explorer" className="flex-1 overflow-hidden p-2">
                <iframe src={explorerTool.proxy_url} className="h-full w-full rounded border" title="区块链浏览器" />
              </TabsContent>
            )}

            {monitorTool && (
              <TabsContent value="monitor" className="flex-1 overflow-hidden p-2">
                <iframe src={monitorTool.proxy_url} className="h-full w-full rounded border" title="监控仪表盘" />
              </TabsContent>
            )}

            {canUseSimEngine && (
              <TabsContent value="sim" className="flex-1 overflow-auto p-2">
                <SimEnginePanel sessionID={simSessionID} scenes={simScenes} />
              </TabsContent>
            )}
          </Tabs>
        </Panel>
      </PanelGroup>
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
