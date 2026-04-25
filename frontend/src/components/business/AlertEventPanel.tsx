"use client";

// AlertEventPanel.tsx
// 模块08告警事件组件，覆盖告警列表、详情查看、处理与忽略联动刷新。

import { AlertTriangle, RefreshCcw, ShieldCheck } from "lucide-react";
import { useSearchParams } from "next/navigation";
import { useEffect, useState } from "react";

import { Button } from "@/components/ui/Button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/Card";
import { EmptyState } from "@/components/ui/EmptyState";
import { ErrorState } from "@/components/ui/ErrorState";
import { FormField } from "@/components/ui/FormField";
import { Input } from "@/components/ui/Input";
import { LoadingState } from "@/components/ui/LoadingState";
import { Pagination } from "@/components/ui/Pagination";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/Select";
import { Table, TableBody, TableCell, TableContainer, TableHead, TableHeader, TableRow } from "@/components/ui/Table";
import { Textarea } from "@/components/ui/Textarea";
import { useToast } from "@/components/ui/Toast";
import { useAlertEvent, useAlertEventMutations, useAlertEvents } from "@/hooks/useAlertEvents";
import { formatDateTime, formatScore } from "@/lib/format";
import {
  createSystemRecentRange,
  fromSystemDateTimeLocalValue,
  getSystemAlertEventStatusClassName,
  getSystemAlertLevelClassName,
  SYSTEM_ALERT_EVENT_STATUS_OPTIONS,
  SYSTEM_ALERT_LEVEL_OPTIONS,
} from "@/lib/system";
import { cn } from "@/lib/utils";
import type { SystemAlertEventListParams, SystemAlertEventStatus, SystemAlertLevel } from "@/types/system";

const DEFAULT_ALERT_EVENT_RANGE = createSystemRecentRange(7);

interface AlertEventFilterDraft {
  level: string;
  status: string;
  ruleID: string;
  dateFrom: string;
  dateTo: string;
}

function createInitialDraft(): AlertEventFilterDraft {
  return {
    level: "all",
    status: "all",
    ruleID: "",
    dateFrom: DEFAULT_ALERT_EVENT_RANGE.dateFrom,
    dateTo: DEFAULT_ALERT_EVENT_RANGE.dateTo,
  };
}

function buildEventParams(draft: AlertEventFilterDraft, page = 1): SystemAlertEventListParams {
  return {
    page,
    page_size: 20,
    level: draft.level === "all" ? undefined : (Number(draft.level) as SystemAlertLevel),
    status: draft.status === "all" ? undefined : (Number(draft.status) as SystemAlertEventStatus),
    rule_id: draft.ruleID || undefined,
    date_from: fromSystemDateTimeLocalValue(draft.dateFrom),
    date_to: fromSystemDateTimeLocalValue(draft.dateTo),
  };
}

/**
 * AlertEventPanel 告警事件列表与处理组件。
 */
export function AlertEventPanel() {
  const searchParams = useSearchParams();
  const { showToast } = useToast();
  const [draft, setDraft] = useState(createInitialDraft);
  const [params, setParams] = useState<SystemAlertEventListParams>(() => buildEventParams(createInitialDraft()));
  const [selectedEventID, setSelectedEventID] = useState(searchParams.get("selected") ?? "");
  const [handleNote, setHandleNote] = useState("");
  const listQuery = useAlertEvents(params);
  const detailQuery = useAlertEvent(selectedEventID);
  const mutations = useAlertEventMutations(selectedEventID);

  useEffect(() => {
    const eventID = searchParams.get("selected");
    if (eventID !== null && eventID !== "") {
      setSelectedEventID(eventID);
    }
  }, [searchParams]);

  useEffect(() => {
    if (selectedEventID === "" && (listQuery.data?.list.length ?? 0) > 0) {
      setSelectedEventID(listQuery.data?.list[0]?.id ?? "");
    }
  }, [listQuery.data, selectedEventID]);

  useEffect(() => {
    if (detailQuery.data !== undefined) {
      setHandleNote(detailQuery.data.handle_note ?? "");
    }
  }, [detailQuery.data]);

  if (listQuery.isLoading && listQuery.data === undefined) {
    return <LoadingState title="正在加载运行提醒" description="正在整理提醒列表、概览数据和当前详情。" />;
  }

  if (listQuery.isError && listQuery.data === undefined) {
    return <ErrorState title="告警事件加载失败" description={listQuery.error.message} />;
  }

  return (
    <div className="space-y-6">
      <Card className="overflow-hidden border-none bg-[linear-gradient(135deg,#1e1b4b_0%,#5b21b6_45%,#7f1d1d_100%)] text-white shadow-[0_34px_95px_-42px_rgba(30,27,75,0.72)]">
        <CardContent className="p-6 md:p-8">
          <div className="flex flex-col gap-5 lg:flex-row lg:items-end lg:justify-between">
            <div className="space-y-3">
              <div className="inline-flex items-center gap-2 rounded-full border border-white/12 bg-white/10 px-3 py-1 text-xs font-semibold uppercase tracking-[0.2em] text-rose-100/90">
                <AlertTriangle className="h-3.5 w-3.5" />
                告警事件
              </div>
              <div>
                <h2 className="font-display text-3xl font-semibold tracking-tight">待处理、已处理、已忽略</h2>
                <p className="mt-2 max-w-2xl text-sm leading-7 text-slate-300">
                  提醒列表会定时刷新。处理或忽略后，列表和详情会同步更新。
                </p>
              </div>
            </div>
            <Button
              type="button"
              variant="outline"
              className="border-white/20 bg-white/10 text-white hover:bg-white/15"
              isLoading={listQuery.isRefetching || detailQuery.isRefetching}
              onClick={() => {
                void Promise.all([listQuery.refetch(), detailQuery.refetch()]);
              }}
            >
              <RefreshCcw className="h-4 w-4" />
              刷新
            </Button>
          </div>
          <div className="mt-6 flex flex-wrap gap-3">
            <CountPill label="待处理" value={listQuery.data?.status_counts.pending ?? 0} tone="danger" />
            <CountPill label="已处理" value={listQuery.data?.status_counts.handled ?? 0} tone="success" />
            <CountPill label="已忽略" value={listQuery.data?.status_counts.ignored ?? 0} tone="muted" />
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>筛选条件</CardTitle>
          <CardDescription>支持按级别、状态、规则编号和时间范围筛选提醒记录。</CardDescription>
        </CardHeader>
        <CardContent className="space-y-5">
          <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-5">
            <FormField label="级别">
              <Select value={draft.level} onValueChange={(value) => setDraft((current) => ({ ...current, level: value }))}>
                <SelectTrigger>
                  <SelectValue placeholder="全部级别" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">全部级别</SelectItem>
                  {SYSTEM_ALERT_LEVEL_OPTIONS.map((item) => (
                    <SelectItem key={item.value} value={String(item.value)}>
                      {item.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </FormField>
            <FormField label="状态">
              <Select value={draft.status} onValueChange={(value) => setDraft((current) => ({ ...current, status: value }))}>
                <SelectTrigger>
                  <SelectValue placeholder="全部状态" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">全部状态</SelectItem>
                  {SYSTEM_ALERT_EVENT_STATUS_OPTIONS.map((item) => (
                    <SelectItem key={item.value} value={String(item.value)}>
                      {item.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </FormField>
            <FormField label="规则 ID">
              <Input value={draft.ruleID} onChange={(event) => setDraft((current) => ({ ...current, ruleID: event.target.value }))} placeholder="输入规则编号进行筛选" />
            </FormField>
            <FormField label="开始时间">
              <Input type="datetime-local" value={draft.dateFrom} onChange={(event) => setDraft((current) => ({ ...current, dateFrom: event.target.value }))} />
            </FormField>
            <FormField label="结束时间">
              <Input type="datetime-local" value={draft.dateTo} onChange={(event) => setDraft((current) => ({ ...current, dateTo: event.target.value }))} />
            </FormField>
          </div>
          <div className="flex flex-wrap items-center gap-3">
            <Button
              type="button"
              onClick={() => {
                setParams(buildEventParams(draft));
              }}
            >
              搜索
            </Button>
            <Button
              type="button"
              variant="outline"
              onClick={() => {
                const nextDraft = createInitialDraft();
                setDraft(nextDraft);
                setParams(buildEventParams(nextDraft));
              }}
            >
              重置为最近 7 天
            </Button>
          </div>
        </CardContent>
      </Card>

      <div className="grid gap-6 xl:grid-cols-[1.15fr_0.85fr]">
        <Card>
          <CardHeader>
            <CardTitle>告警列表</CardTitle>
            <CardDescription>选择任意告警以查看详情，并在右侧完成处理或忽略操作。</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            {(listQuery.data?.list ?? []).length === 0 ? (
              <EmptyState title="暂无告警事件" description="当前筛选条件下没有匹配的告警事件。" className="min-h-72" />
            ) : (
              <>
                <TableContainer>
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>时间</TableHead>
                        <TableHead>级别</TableHead>
                        <TableHead>规则名称</TableHead>
                        <TableHead>告警标题</TableHead>
                        <TableHead>状态</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {(listQuery.data?.list ?? []).map((item) => (
                        <TableRow
                          key={item.id}
                          className={cn("cursor-pointer", selectedEventID === item.id ? "bg-primary/5" : "")}
                          onClick={() => setSelectedEventID(item.id)}
                        >
                          <TableCell>{formatDateTime(item.triggered_at)}</TableCell>
                          <TableCell>
                            <span className={cn("inline-flex rounded-full px-2.5 py-1 text-xs font-semibold", getSystemAlertLevelClassName(item.level))}>
                              {item.level_text}
                            </span>
                          </TableCell>
                          <TableCell>{item.rule_name}</TableCell>
                          <TableCell className="max-w-sm truncate">{item.title}</TableCell>
                          <TableCell>
                            <span className={cn("inline-flex rounded-full px-2.5 py-1 text-xs font-semibold", getSystemAlertEventStatusClassName(item.status))}>
                              {item.status_text}
                            </span>
                          </TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                </TableContainer>
                {listQuery.data?.pagination !== undefined ? (
                  <Pagination
                    page={listQuery.data.pagination.page}
                    totalPages={listQuery.data.pagination.total_pages}
                    total={listQuery.data.pagination.total}
                    onPageChange={(page) => setParams((current) => ({ ...current, page }))}
                  />
                ) : null}
              </>
            )}
          </CardContent>
        </Card>

        <Card className="xl:sticky xl:top-20">
          <CardHeader>
            <CardTitle>告警详情</CardTitle>
            <CardDescription>查看触发上下文，并在待处理状态下提交处理备注。</CardDescription>
          </CardHeader>
          <CardContent>
            {selectedEventID === "" ? (
              <EmptyState title="请选择一条告警" description="左侧选择告警后，这里会展示完整详情与处理操作。" className="min-h-72" />
            ) : detailQuery.isLoading && detailQuery.data === undefined ? (
              <LoadingState title="正在加载告警详情" description="链镜正在读取选中告警的详细触发上下文。" className="min-h-72" />
            ) : detailQuery.isError && detailQuery.data === undefined ? (
              <ErrorState description={detailQuery.error.message} />
            ) : detailQuery.data !== undefined ? (
              <div className="space-y-5">
                <div className="rounded-[1.5rem] border border-border/70 bg-muted/20 p-5">
                  <div className="flex flex-wrap items-center gap-2">
                    <span className={cn("inline-flex rounded-full px-2.5 py-1 text-xs font-semibold", getSystemAlertLevelClassName(detailQuery.data.level))}>
                      {detailQuery.data.level_text}
                    </span>
                    <span className={cn("inline-flex rounded-full px-2.5 py-1 text-xs font-semibold", getSystemAlertEventStatusClassName(detailQuery.data.status))}>
                      {detailQuery.data.status_text}
                    </span>
                  </div>
                  <h3 className="mt-4 font-display text-2xl font-semibold tracking-tight">{detailQuery.data.title}</h3>
                  <div className="mt-4 grid gap-2 text-sm text-muted-foreground">
                    <DetailItem label="规则名称" value={detailQuery.data.rule_name} />
                    <DetailItem label="触发时间" value={formatDateTime(detailQuery.data.triggered_at)} />
                    <DetailItem label="处理人" value={detailQuery.data.handled_by_name ?? "—"} />
                    <DetailItem label="处理时间" value={detailQuery.data.handled_at ? formatDateTime(detailQuery.data.handled_at) : "—"} />
                  </div>
                </div>

                <div className="rounded-[1.5rem] border border-border/70 bg-card p-5">
                  <p className="text-sm font-semibold text-foreground">触发详情</p>
                  <div className="mt-4 space-y-2 text-sm text-muted-foreground">
                    <DetailItem label="指标" value={detailQuery.data.detail.metric ?? detailQuery.data.detail.event_source ?? "—"} />
                    <DetailItem label="当前值" value={detailQuery.data.detail.current_value !== undefined ? `${formatScore(detailQuery.data.detail.current_value, 1)}%` : "—"} />
                    <DetailItem label="阈值" value={detailQuery.data.detail.threshold !== undefined ? `${formatScore(detailQuery.data.detail.threshold, 1)}%` : "—"} />
                    <DetailItem label="持续时间" value={detailQuery.data.detail.duration_seconds !== undefined ? `${detailQuery.data.detail.duration_seconds} 秒` : "—"} />
                    <DetailItem label="节点 / 分组" value={detailQuery.data.detail.node ?? detailQuery.data.detail.group_value ?? "—"} />
                    <DetailItem label="事件数" value={detailQuery.data.detail.event_count !== undefined ? String(detailQuery.data.detail.event_count) : "—"} />
                    <DetailItem label="时间窗口" value={detailQuery.data.detail.time_window !== undefined ? `${detailQuery.data.detail.time_window} 秒` : "—"} />
                  </div>

                  {(detailQuery.data.detail.sample_events ?? []).length > 0 ? (
                    <div className="mt-5 rounded-2xl border border-border/70 bg-muted/20 p-4">
                      <p className="text-sm font-semibold text-foreground">样本事件</p>
                      <div className="mt-3 space-y-3">
                        {(detailQuery.data.detail.sample_events ?? []).map((sample, index) => (
                          <div key={`${sample.created_at}-${index}`} className="rounded-xl border border-border/60 bg-card p-3 text-sm text-muted-foreground">
                            <p>用户 {sample.user_id ?? "—"}</p>
                            <p className="mt-1">原因 {sample.fail_reason ?? "—"}</p>
                            <p className="mt-1">时间 {formatDateTime(sample.created_at)}</p>
                          </div>
                        ))}
                      </div>
                    </div>
                  ) : null}
                </div>

                {detailQuery.data.status === 1 ? (
                  <div className="rounded-[1.5rem] border border-border/70 bg-slate-950 p-5 text-white">
                    <p className="text-sm font-semibold">处理动作</p>
                    <p className="mt-2 text-sm leading-6 text-slate-300">
                      处理或忽略前请先填写备注。提交成功后，列表和详情会一并刷新。
                    </p>
                    <div className="mt-4">
                      <FormField label="处理备注" required>
                        <Textarea
                          rows={5}
                          value={handleNote}
                          onChange={(event) => setHandleNote(event.target.value)}
                          placeholder="例如：已扩容节点，CPU 使用率恢复正常。"
                        />
                      </FormField>
                    </div>
                    <div className="mt-5 flex flex-wrap gap-3">
                      <Button
                        type="button"
                        variant="outline"
                        className="border-white/15 bg-white/10 text-white hover:bg-white/15"
                        isLoading={mutations.ignore.isPending}
                        onClick={() => {
                          if (handleNote.trim() === "") {
                            showToast({ title: "请先填写备注", description: "忽略告警前需要提供说明。", variant: "destructive" });
                            return;
                          }
                          mutations.ignore.mutate(
                            { handle_note: handleNote.trim() },
                            {
                              onSuccess: () => {
                                showToast({ title: "告警已忽略", variant: "success" });
                              },
                              onError: (error) => {
                                showToast({ title: "忽略失败", description: error.message, variant: "destructive" });
                              },
                            },
                          );
                        }}
                      >
                        忽略告警
                      </Button>
                      <Button
                        type="button"
                        isLoading={mutations.handle.isPending}
                        onClick={() => {
                          if (handleNote.trim() === "") {
                            showToast({ title: "请先填写备注", description: "处理告警前需要记录处理说明。", variant: "destructive" });
                            return;
                          }
                          mutations.handle.mutate(
                            { handle_note: handleNote.trim() },
                            {
                              onSuccess: () => {
                                showToast({ title: "告警已标记为已处理", variant: "success" });
                              },
                              onError: (error) => {
                                showToast({ title: "处理失败", description: error.message, variant: "destructive" });
                              },
                            },
                          );
                        }}
                      >
                        <ShieldCheck className="h-4 w-4" />
                        标记已处理
                      </Button>
                    </div>
                  </div>
                ) : (
                  <div className="rounded-[1.5rem] border border-border/70 bg-muted/20 p-5">
                    <p className="text-sm font-semibold text-foreground">处理备注</p>
                    <p className="mt-3 text-sm leading-7 text-muted-foreground">
                      {detailQuery.data.handle_note ?? "当前未记录处理备注。"}
                    </p>
                  </div>
                )}
              </div>
            ) : null}
          </CardContent>
        </Card>
      </div>
    </div>
  );
}

function CountPill({
  label,
  value,
  tone,
}: {
  label: string;
  value: number;
  tone: "danger" | "success" | "muted";
}) {
  return (
    <div
      className={cn(
        "rounded-full border px-4 py-2",
        tone === "danger"
          ? "border-rose-400/25 bg-rose-500/12"
          : tone === "success"
            ? "border-emerald-400/20 bg-emerald-500/12"
            : "border-white/12 bg-white/8",
      )}
    >
      <span className="text-sm text-slate-300">{label}</span>
      <span className="ml-3 font-semibold text-white">{value}</span>
    </div>
  );
}

function DetailItem({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex items-center justify-between gap-4 border-b border-border/70 pb-2 last:border-b-0 last:pb-0">
      <span>{label}</span>
      <span className="max-w-[60%] text-right text-foreground">{value}</span>
    </div>
  );
}
