"use client";

// ExperimentInstancePanel.tsx
// 模块04实验实例详情面板，组合生命周期、终端、检查点、快照、报告和 SimEngine。

import { ExternalLink, FlaskConical, Pause, Play, RotateCcw, Send, Square, Upload, Code, Monitor, Gamepad2, BookOpen, CheckCircle, Camera, FileText, PanelRightOpen } from "lucide-react";
import dynamic from "next/dynamic";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useCallback, useEffect, useMemo, useState } from "react";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";

import { CheckpointPanel } from "@/components/business/CheckpointPanel";
import { SnapshotPanel } from "@/components/business/SnapshotPanel";
import { LoadingState } from "@/components/ui/LoadingState";
import { Sheet, SheetContent, SheetHeader, SheetTitle } from "@/components/ui/Sheet";

// 重型子组件全部按需懒加载，避免在初始路由编译阶段把 xterm.js / sim-engine-renderers / iframe 桥接逻辑
// 一次性打入同一个 dev chunk，否则 Next.js dev 编译会因 sourcemap 序列化体积过大触发 V8 OOM。
const ExperimentTerminal = dynamic(
  () => import("@/components/business/ExperimentTerminal").then((mod) => ({ default: mod.ExperimentTerminal })),
  { ssr: false, loading: () => <LoadingState variant="spinner" /> },
);
const SimEnginePanel = dynamic(
  () => import("@/components/business/SimEnginePanel").then((mod) => ({ default: mod.SimEnginePanel })),
  { ssr: false, loading: () => <LoadingState variant="spinner" /> },
);
const WebIDEPanel = dynamic(
  () => import("@/components/business/WebIDEPanel").then((mod) => ({ default: mod.WebIDEPanel })),
  { ssr: false, loading: () => <LoadingState variant="spinner" /> },
);
const VNCDesktopPanel = dynamic(
  () => import("@/components/business/VNCDesktopPanel").then((mod) => ({ default: mod.VNCDesktopPanel })),
  { ssr: false, loading: () => <LoadingState variant="spinner" /> },
);
const ToolIframe = dynamic(
  () => import("@/components/business/ToolIframe").then((mod) => ({ default: mod.ToolIframe })),
  { ssr: false, loading: () => <LoadingState variant="spinner" /> },
);
import { Badge } from "@/components/ui/Badge";
import { Button } from "@/components/ui/Button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/Card";
import { ConfirmDialog } from "@/components/ui/ConfirmDialog";
import { ErrorState } from "@/components/ui/ErrorState";
import { FormField } from "@/components/ui/FormField";
import { Input } from "@/components/ui/Input";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/Tabs";
import { Textarea } from "@/components/ui/Textarea";
import { useQueryClient } from "@tanstack/react-query";
import { experimentInstanceQueryKey, useCheckpointMutations, useExperimentInstance, useExperimentInstanceLifecycleMutations, useExperimentReport, useExperimentReportMutations, useExperimentMonitorMutations } from "@/hooks/useExperimentInstances";
import { useExperimentInstanceRealtime } from "@/hooks/useExperimentRealtime";
import { useStudentExperimentTemplate } from "@/hooks/useExperimentTemplates";
import { formatFileSize, formatScore } from "@/lib/format";
import { buildExperimentResultSummary, getInstanceStatusVariant, instanceStateMachine } from "@/lib/experiment";
import type { ID } from "@/types/api";

/**
 * ExperimentInstancePanel 组件属性。
 */
export interface ExperimentInstancePanelProps {
  instanceID: ID;
  mode?: "student" | "assist" | "grade";
}


/**
 * ExperimentInstancePanel 实验实例工作台组件。
 */
export function ExperimentInstancePanel({ instanceID, mode = "student" }: ExperimentInstancePanelProps) {
  const router = useRouter();
  const instanceQuery = useExperimentInstance(instanceID);
  const lifecycle = useExperimentInstanceLifecycleMutations(instanceID);
  const realtime = useExperimentInstanceRealtime(instanceID);
  const queryClient = useQueryClient();
  const checkpointMutations = useCheckpointMutations(instanceID);
  const monitorMutations = useExperimentMonitorMutations();
  const [reportContent, setReportContent] = useState("");
  const [guidanceMessage, setGuidanceMessage] = useState("");
  const [manualScore, setManualScore] = useState("");
  const [manualComment, setManualComment] = useState("");
  const reportQuery = useExperimentReport(instanceID);
  const templateID = instanceQuery.data?.template.id ?? "";
  const templateQuery = useStudentExperimentTemplate(templateID);
  const guidanceMessages = useMemo(
    () => realtime.messages.filter((message) => message.type === "guidance_message"),
    [realtime.messages],
  );

  // 浮动参考抽屉（说明+检查点）
  const [refDrawerOpen, setRefDrawerOpen] = useState(false);
  const [refDrawerTab, setRefDrawerTab] = useState<"instructions" | "checkpoints">("instructions");
  const openRef = useCallback((tab: "instructions" | "checkpoints") => {
    setRefDrawerTab(tab);
    setRefDrawerOpen(true);
  }, []);

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

  // 处理 WS 推送：状态变更 / 检查点结果 → 失效 React Query 缓存让 UI 跟随；空闲超时 / 时长超限 → 用户提醒。
  // 注意：这里使用 messages.length 作为 effect 触发条件，确保只对最新一条消息执行一次副作用。
  useEffect(() => {
    const msgs = realtime.messages;
    if (msgs.length === 0) return;
    const last = msgs[msgs.length - 1];
    switch (last?.type) {
      case 'status_change':
        // 后端在状态变更（创建中→运行中、运行中→已暂停 等）时推送，触发详情查询重新拉取。
        void queryClient.invalidateQueries({ queryKey: experimentInstanceQueryKey(instanceID) });
        void queryClient.invalidateQueries({ queryKey: ["experiment", "instances"] });
        break;
      case 'checkpoint_result':
        // 检查点完成后失效检查点列表与实例详情（实例分数会同步更新）。
        void queryClient.invalidateQueries({ queryKey: ["experiment", "checkpoints", instanceID] });
        void queryClient.invalidateQueries({ queryKey: experimentInstanceQueryKey(instanceID) });
        break;
      case 'container_status':
        // 容器状态变更只影响实例详情中的 containers/tools 列表。
        void queryClient.invalidateQueries({ queryKey: experimentInstanceQueryKey(instanceID) });
        break;
      case 'idle_warning':
        if (window.confirm('您已较长时间未操作，实验即将因超时被自动暂停。点击"确定"继续实验。')) {
          lifecycle.heartbeat.mutate();
        }
        break;
      case 'duration_warning':
        window.alert('实验即将超时，请尽快完成并提交。');
        break;
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [realtime.messages.length]);

  // Ctrl+I 快捷键切换参考抽屉
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if ((e.ctrlKey || e.metaKey) && e.key === "i") {
        e.preventDefault();
        setRefDrawerOpen((prev) => !prev);
      }
    };
    window.addEventListener("keydown", handler);
    return () => window.removeEventListener("keydown", handler);
  }, []);

  if (instanceQuery.isLoading) {
    return <LoadingState variant="hero" title="正在加载实验详情" description="正在整理实验环境、评分进度、快照和报告内容。" />;
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

  // 仅将 tool_kind="terminal" 的容器传给终端组件，避免默认选中非终端容器导致 WS 400
  const terminalContainers = (instance.containers ?? [])
    .filter((c) => c.tool_kind === "terminal")
    .map((c) => ({ container_name: c.container_name, display_name: c.container_name }));

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

  // 计算默认激活 Tab
  const defaultTab = experimentType === 1
    ? "sim"
    : terminalTool ? "terminal" : ideTool ? "ide" : canUseSimEngine ? "sim" : "instructions";

  return (
    <div className="flex flex-col gap-3">
      {/* 精简状态栏：标题 + 状态 + 分数 + 操作按钮 */}
      <div className="rounded-xl border border-cyan-500/20 bg-gradient-to-r from-slate-950 via-slate-900 to-cyan-950 px-5 py-3 text-white">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <div className="flex items-center gap-3">
            <FlaskConical className="h-5 w-5 text-cyan-200 shrink-0" />
            <div className="min-w-0">
              <h2 className="truncate font-display text-base font-semibold">{instance.template.title}</h2>
              <p className="text-xs text-white/50">
                {instance.student.name} · 第{instance.attempt_no}次
                {isAssistMode ? " · 协助模式" : ""}
                {isGradeMode ? " · 评分模式" : ""}
              </p>
            </div>
            <Badge variant={getInstanceStatusVariant(instance.status)} className="shrink-0">{instance.status_text}</Badge>
          </div>
          <div className="flex items-center gap-2 text-xs">
            <span className="rounded-md border border-white/10 bg-white/7 px-2.5 py-1">自动 <strong>{formatScore(instance.scores.auto_score ?? 0)}</strong></span>
            <span className="rounded-md border border-white/10 bg-white/7 px-2.5 py-1">人工 <strong>{formatScore(instance.scores.manual_score ?? 0)}</strong></span>
            <span className="rounded-md border border-white/10 bg-white/7 px-2.5 py-1">总分 <strong>{formatScore(instance.scores.total_score ?? 0)}</strong></span>
            <span className={`rounded-md border px-2.5 py-1 ${realtime.status === "open" ? "border-emerald-400/30 bg-emerald-500/15 text-emerald-200" : "border-white/10 bg-white/7"}`}>
              {realtime.status === "open" ? "已连接" : "未连接"}
            </span>
          </div>
        </div>
        {instance.error_message ? (
          <div className="mt-2 rounded-lg border border-red-300/30 bg-red-500/10 px-3 py-2 text-xs text-red-100">
            <strong>环境异常：</strong>{instance.error_message}
          </div>
        ) : null}
        <div className="mt-2 flex flex-wrap gap-1.5">
          <Button variant="outline" size="sm" className="h-7 border-white/18 bg-white/8 text-xs text-white hover:bg-white/14" onClick={() => lifecycle.pause.mutate()} isLoading={lifecycle.pause.isPending} disabled={isAssistMode || isGradeMode || !instanceStateMachine.canPause(instance.status)}>
            <Pause className="h-3.5 w-3.5" />暂停
          </Button>
          <Button variant="outline" size="sm" className="h-7 border-white/18 bg-white/8 text-xs text-white hover:bg-white/14" onClick={() => lifecycle.resume.mutate({})} isLoading={lifecycle.resume.isPending} disabled={isAssistMode || isGradeMode || !instanceStateMachine.canResume(instance.status)}>
            <Play className="h-3.5 w-3.5" />恢复
          </Button>
          <Button variant="outline" size="sm" className="h-7 border-white/18 bg-white/8 text-xs text-white hover:bg-white/14" onClick={() => lifecycle.restart.mutate(undefined, { onSuccess: (resp) => { if (resp?.instance_id && resp.instance_id !== instanceID) router.push(`/student/experiment-instances/${resp.instance_id}`); } })} isLoading={lifecycle.restart.isPending} disabled={isAssistMode || !instanceStateMachine.canRestart(instance.status)}>
            <RotateCcw className="h-3.5 w-3.5" />重启
          </Button>
          {isStudentMode ? (
            <>
              <ConfirmDialog title="确认提交实验" description="提交后将触发检查点评分和报告验收，运行态实例会进入已提交状态。" confirmText="提交" confirmVariant="primary" trigger={<Button size="sm" className="h-7 text-xs" disabled={!instanceStateMachine.canSubmit(instance.status)}>提交实验</Button>} onConfirm={() => lifecycle.submit.mutate()} />
              <Button variant="outline" size="sm" className="h-7 border-white/18 bg-white/8 text-xs text-white hover:bg-white/14" onClick={() => router.push(`/student/experiment-instances/${instanceID}/report`)}><ExternalLink className="h-3.5 w-3.5" />报告</Button>
              <Button variant="outline" size="sm" className="h-7 border-white/18 bg-white/8 text-xs text-white hover:bg-white/14" onClick={() => router.push(`/student/experiment-instances/${instanceID}/result`)}>结果</Button>
            </>
          ) : null}
          <ConfirmDialog title="确认销毁实验环境" description="结束后会释放实验环境，未保存的运行数据可能丢失。" trigger={<Button variant="destructive" size="sm" className="h-7 text-xs" disabled={!instanceStateMachine.canDestroy(instance.status)}><Square className="h-3.5 w-3.5" />销毁</Button>} onConfirm={() => lifecycle.destroy.mutate()} />
          {isAssistMode ? (
            <Button variant="outline" size="sm" className="h-7 border-white/18 bg-white/8 text-xs text-white hover:bg-white/14" onClick={() => monitorMutations.forceDestroy.mutate(instanceID)}>强制回收</Button>
          ) : null}
        </div>
      </div>

      {/* 教师指导浮动通知 */}
      {latestGuidance ? (
        <div className="fixed bottom-4 right-4 z-40 max-w-md animate-in slide-in-from-bottom-4 rounded-xl border border-cyan-300/20 bg-cyan-950/95 p-4 text-sm text-cyan-50 shadow-lg backdrop-blur">
          <p className="font-medium">教师指导</p>
          <p className="mt-1">{latestGuidance.content ?? "已收到教师消息"}</p>
        </div>
      ) : null}

      {/* 全宽扁平 Tab 工作区 */}
      <Tabs defaultValue={defaultTab} className="flex min-h-[600px] flex-col rounded-xl border border-border">
        <div className="flex items-center border-b">
          <TabsList className="flex flex-1 flex-wrap justify-start rounded-none border-b-0">
            {/* 工具 Tab — 按实验类型动态显隐 */}
            {terminalTool && (
              <TabsTrigger value="terminal">{isAssistMode ? "只读终端" : "终端"}</TabsTrigger>
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
            {/* 内容 Tab */}
            <TabsTrigger value="instructions"><BookOpen className="h-3.5 w-3.5 mr-1" />说明</TabsTrigger>
            <TabsTrigger value="checkpoints"><CheckCircle className="h-3.5 w-3.5 mr-1" />{isGradeMode ? "评分项" : "检查点"}</TabsTrigger>
            {!isGradeMode && <TabsTrigger value="snapshots"><Camera className="h-3.5 w-3.5 mr-1" />快照</TabsTrigger>}
            <TabsTrigger value="report"><FileText className="h-3.5 w-3.5 mr-1" />{isGradeMode ? "报告与总评" : "报告"}</TabsTrigger>
          </TabsList>
          {/* 浮动参考按钮：在工具 Tab 激活时可快速打开说明/检查点抽屉 */}
          <div className="flex shrink-0 items-center gap-1 px-2">
            <button type="button" title="打开说明 (Ctrl+I)" className="rounded-md p-1.5 text-muted-foreground transition hover:bg-muted hover:text-foreground" onClick={() => openRef("instructions")}>
              <PanelRightOpen className="h-4 w-4" />
            </button>
          </div>
        </div>

        {/* 工具内容区 — 每个 Tab 独占全宽 */}
        {terminalTool && (
          <TabsContent value="terminal" className="flex-1 overflow-hidden p-2">
            <ExperimentTerminal instanceID={instanceID} containers={terminalContainers} readOnly={isAssistMode || isGradeMode} className="h-full" />
          </TabsContent>
        )}
        {ideTool && (
          <TabsContent value="ide" className="flex-1 overflow-hidden p-2">
            <WebIDEPanel accessUrl={ideTool.proxy_url} instanceID={instanceID} toolKind={ideTool.kind} className="h-full" />
          </TabsContent>
        )}
        {desktopTool && (
          <TabsContent value="desktop" className="flex-1 overflow-hidden p-2">
            <VNCDesktopPanel accessUrl={desktopTool.proxy_url} instanceID={instanceID} toolKind={desktopTool.kind} className="h-full" />
          </TabsContent>
        )}
        {explorerTool && (
          <TabsContent value="explorer" className="flex-1 overflow-hidden p-2">
            <ToolIframe src={explorerTool.proxy_url} instanceID={instanceID} toolKind={explorerTool.kind} title="区块链浏览器" className="h-full" />
          </TabsContent>
        )}
        {monitorTool && (
          <TabsContent value="monitor" className="flex-1 overflow-hidden p-2">
            <ToolIframe src={monitorTool.proxy_url} instanceID={instanceID} toolKind={monitorTool.kind} title="监控仪表盘" className="h-full" />
          </TabsContent>
        )}
        {canUseSimEngine && (
          <TabsContent value="sim" className="flex-1 overflow-auto p-2">
            <SimEnginePanel sessionID={simSessionID} instanceID={instanceID} scenes={simScenes} experimentType={experimentType} userRole={mode === "assist" ? "teacher" : "student"} />
          </TabsContent>
        )}

        {/* 内容 Tab 区 — 全宽文档阅读 */}
        <TabsContent value="instructions" className="flex-1 overflow-auto p-6">
          <div className="prose prose-sm max-w-none dark:prose-invert text-muted-foreground">
            <ReactMarkdown remarkPlugins={[remarkGfm]}>{instance.template.instructions ?? "当前模板未填写实验说明。"}</ReactMarkdown>
          </div>
          {isAssistMode && (
            <div className="mt-6 space-y-3 border-t pt-4">
              <p className="text-sm font-medium">指导消息</p>
              <Textarea value={guidanceMessage} onChange={(e) => setGuidanceMessage(e.target.value)} rows={4} placeholder="向学生发送指导建议" />
              <Button size="sm" onClick={submitGuidance} isLoading={monitorMutations.guidance.isPending} disabled={guidanceMessage.trim().length === 0}>
                <Send className="h-4 w-4" />发送
              </Button>
            </div>
          )}
        </TabsContent>

        <TabsContent value="checkpoints" className="flex-1 overflow-auto p-6">
          <CheckpointPanel instanceID={instanceID} canGrade={isGradeMode} />
          {isGradeMode && (
            <div className="mt-6 space-y-3 border-t pt-4">
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
          <TabsContent value="snapshots" className="flex-1 overflow-auto p-6">
            <SnapshotPanel instanceID={instanceID} />
            {isStudentMode && (
              <div className="mt-4 border-t pt-4">
                <Link className="text-sm font-medium text-primary hover:underline" href={`/student/experiment-instances/${instanceID}/history`}>查看完整操作历史 →</Link>
              </div>
            )}
          </TabsContent>
        )}

        <TabsContent value="report" className="flex-1 overflow-auto p-6">
          {isGradeMode ? (
            <>
              <FormField label="报告正文">
                <Textarea value={reportContent} onChange={(e) => setReportContent(e.target.value)} rows={8} disabled />
              </FormField>
              {reportQuery.data?.file_name && (
                <div className="mt-2 rounded-lg border bg-muted/35 p-3 text-sm">
                  当前附件：{reportQuery.data.file_name}，{formatFileSize(reportQuery.data.file_size ?? 0)}
                </div>
              )}
            </>
          ) : (
            <div className="space-y-3">
              <p className="text-sm text-muted-foreground">{reportQuery.data?.content ? `报告已提交（${reportQuery.data.content.length}字）` : "报告尚未提交"}</p>
              {reportQuery.data?.file_name && (
                <p className="text-sm text-muted-foreground">附件：{reportQuery.data.file_name}，{formatFileSize(reportQuery.data.file_size ?? 0)}</p>
              )}
              <Link className="text-sm font-medium text-primary hover:underline" href={`/student/experiment-instances/${instanceID}/report`}>前往独立报告页编辑 →</Link>
            </div>
          )}
        </TabsContent>
      </Tabs>

      {/* 浮动参考抽屉：在工具 Tab 操作时快速查看说明/检查点 */}
      <Sheet open={refDrawerOpen} onOpenChange={setRefDrawerOpen}>
        <SheetContent width="w-[420px]">
          <SheetHeader>
            <SheetTitle>快速参考</SheetTitle>
          </SheetHeader>
          <Tabs value={refDrawerTab} onValueChange={(v) => setRefDrawerTab(v as "instructions" | "checkpoints")} className="flex flex-1 flex-col overflow-hidden">
            <TabsList className="mx-4 mt-2 flex w-auto justify-start rounded-none border-b border-b-0">
              <TabsTrigger value="instructions"><BookOpen className="h-3.5 w-3.5 mr-1" />说明</TabsTrigger>
              <TabsTrigger value="checkpoints"><CheckCircle className="h-3.5 w-3.5 mr-1" />检查点</TabsTrigger>
            </TabsList>
            <TabsContent value="instructions" className="flex-1 overflow-auto px-5 py-4">
              <div className="prose prose-sm max-w-none dark:prose-invert text-muted-foreground">
                <ReactMarkdown remarkPlugins={[remarkGfm]}>{instance.template.instructions ?? "当前模板未填写实验说明。"}</ReactMarkdown>
              </div>
            </TabsContent>
            <TabsContent value="checkpoints" className="flex-1 overflow-auto px-5 py-4">
              <CheckpointPanel instanceID={instanceID} canGrade={false} />
            </TabsContent>
          </Tabs>
        </SheetContent>
      </Sheet>
    </div>
  );
}

/**
 * ExperimentReportPanel 独立实验报告编辑组件。
 */
export function ExperimentReportPanel({ instanceID }: { instanceID: ID }) {
  const instanceQuery = useExperimentInstance(instanceID);
  const reportQuery = useExperimentReport(instanceID);
  const reportMutations = useExperimentReportMutations(instanceID);
  const [reportContent, setReportContent] = useState("");
  const [uploadProgress, setUploadProgress] = useState<number | null>(null);

  useEffect(() => {
    if (reportQuery.data?.content !== undefined && reportQuery.data.content !== null) {
      setReportContent(reportQuery.data.content);
    }
  }, [reportQuery.data?.content]);

  if (instanceQuery.isLoading) {
    return <LoadingState variant="hero" title="正在加载实验报告" description="读取报告信息。" />;
  }

  if (!instanceQuery.data) {
    return <ErrorState title="加载失败" description="请确认当前账号仍可查看此实验。" />;
  }

  const instance = instanceQuery.data;

  const uploadReportFile = (file: File) => {
    reportMutations.upload.mutate(
      { file, purpose: "experiment_report", onUploadProgress: setUploadProgress },
      {
        onSuccess: (uploaded) => {
          const payload = { content: reportContent, file_url: uploaded.file_url, file_name: uploaded.file_name, file_size: uploaded.file_size };
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
      <Card className="overflow-hidden border-emerald-500/20 bg-gradient-to-br from-slate-950 via-emerald-950 to-cyan-950 text-white">
        <CardHeader>
          <CardTitle className="text-white">{instance.template.title} · 实验报告</CardTitle>
        </CardHeader>
      </Card>
      <Card>
        <CardHeader>
          <CardTitle>报告正文</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <FormField label="报告内容">
            <Textarea value={reportContent} onChange={(e) => setReportContent(e.target.value)} rows={12} />
          </FormField>
          {reportQuery.data?.file_name && (
            <div className="rounded-lg border bg-muted/35 p-3 text-sm">
              当前附件：{reportQuery.data.file_name}，{formatFileSize(reportQuery.data.file_size ?? 0)}
            </div>
          )}
          <div className="flex flex-wrap gap-2">
            <Button size="sm" disabled={!reportContent.trim() && !reportQuery.data?.file_url} onClick={saveReport} isLoading={reportMutations.create.isPending || reportMutations.update.isPending}>保存报告</Button>
            <label className="inline-flex cursor-pointer items-center gap-2 rounded-lg border px-3 py-1.5 text-sm font-medium hover:bg-muted">
              <Upload className="h-4 w-4" />上传附件
              <input className="sr-only" type="file" accept=".pdf,.doc,.docx,.md,.txt,.zip" onChange={(e) => { const f = e.target.files?.[0]; if (f) uploadReportFile(f); }} />
            </label>
            {uploadProgress !== null && <span className="self-center text-sm text-muted-foreground">上传 {uploadProgress}%</span>}
          </div>
        </CardContent>
      </Card>
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
    return <LoadingState variant="hero" title="正在加载实验结果" description="读取检查点、成绩和报告信息。" />;
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
          <div className="prose prose-sm max-w-none dark:prose-invert text-muted-foreground">
            <ReactMarkdown remarkPlugins={[remarkGfm]}>{reportQuery.data?.content ?? "未提交报告正文。"}</ReactMarkdown>
          </div>
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
