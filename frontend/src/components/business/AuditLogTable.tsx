"use client";

// AuditLogTable.tsx
// 模块08统一审计组件，覆盖聚合查询、来源筛选、详情展开与导出下载。

import { Download, RefreshCcw, Search, ShieldCheck } from "lucide-react";
import { Fragment } from "react";
import { useState } from "react";

import { Button } from "@/components/ui/Button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/Card";
import { EmptyState } from "@/components/ui/EmptyState";
import { ErrorState } from "@/components/ui/ErrorState";
import { FormField } from "@/components/ui/FormField";
import { Input } from "@/components/ui/Input";
import { LoadingState } from "@/components/ui/LoadingState";
import { Pagination } from "@/components/ui/Pagination";
import { Table, TableBody, TableCell, TableContainer, TableHead, TableHeader, TableRow } from "@/components/ui/Table";
import { useToast } from "@/components/ui/Toast";
import { useSystemAudit, useSystemAuditExport } from "@/hooks/useSystemAudit";
import { formatDateTime } from "@/lib/format";
import {
  createSystemRecentRange,
  fromSystemDateTimeLocalValue,
  saveDownloadedFile,
  SYSTEM_AUDIT_SOURCE_OPTIONS,
  validateSystemAuditScope,
} from "@/lib/system";
import { cn } from "@/lib/utils";
import type { SystemAuditListParams, SystemAuditLogItem, SystemAuditSource, SystemJSONValue } from "@/types/system";

interface AuditFilterDraft {
  source: SystemAuditSource | "all";
  dateFrom: string;
  dateTo: string;
  operatorID: string;
  action: string;
  ip: string;
  keyword: string;
}

const DEFAULT_AUDIT_RANGE = createSystemRecentRange(7);

function createInitialDraft(): AuditFilterDraft {
  return {
    source: "all",
    dateFrom: DEFAULT_AUDIT_RANGE.dateFrom,
    dateTo: DEFAULT_AUDIT_RANGE.dateTo,
    operatorID: "",
    action: "",
    ip: "",
    keyword: "",
  };
}

function buildAuditParams(draft: AuditFilterDraft, page = 1): SystemAuditListParams {
  return {
    page,
    page_size: 20,
    source: draft.source === "all" ? undefined : draft.source,
    date_from: fromSystemDateTimeLocalValue(draft.dateFrom),
    date_to: fromSystemDateTimeLocalValue(draft.dateTo),
    operator_id: draft.operatorID || undefined,
    action: draft.action || undefined,
    ip: draft.ip || undefined,
    keyword: draft.keyword || undefined,
  };
}

/**
 * AuditLogTable 统一审计中心组件。
 */
export function AuditLogTable() {
  const { showToast } = useToast();
  const [draft, setDraft] = useState<AuditFilterDraft>(createInitialDraft);
  const [params, setParams] = useState<SystemAuditListParams>(() => buildAuditParams(createInitialDraft()));
  const [expandedID, setExpandedID] = useState<string | null>(null);
  const query = useSystemAudit(params);
  const exportMutation = useSystemAuditExport();

  const validationError = validateSystemAuditScope(
    draft.source,
    fromSystemDateTimeLocalValue(draft.dateFrom),
    fromSystemDateTimeLocalValue(draft.dateTo),
  );

  return (
    <div className="space-y-6">
      <Card className="overflow-hidden border-none bg-[linear-gradient(135deg,#0f172a_0%,#1d4d4f_54%,#7c2d12_100%)] text-white shadow-[0_34px_95px_-42px_rgba(15,23,42,0.75)]">
        <CardContent className="p-6 md:p-8">
          <div className="flex flex-col gap-5 lg:flex-row lg:items-end lg:justify-between">
            <div className="space-y-3">
              <div className="inline-flex items-center gap-2 rounded-full border border-white/12 bg-white/10 px-3 py-1 text-xs font-semibold uppercase tracking-[0.2em] text-teal-100/90">
                <ShieldCheck className="h-3.5 w-3.5" />
                操作记录中心
              </div>
              <div>
                <h2 className="font-display text-3xl font-semibold tracking-tight">平台操作记录查询</h2>
                <p className="mt-2 max-w-2xl text-sm leading-7 text-slate-300">
                  可按来源、时间、操作人和关键词查看平台关键记录，帮助定位问题与追踪变更。
                </p>
              </div>
            </div>
            <div className="flex flex-wrap items-center gap-3">
              <Button
                type="button"
                variant="outline"
                className="border-white/20 bg-white/10 text-white hover:bg-white/15"
                isLoading={query.isRefetching}
                onClick={() => void query.refetch()}
              >
                <RefreshCcw className="h-4 w-4" />
                刷新
              </Button>
              <Button
                type="button"
                variant="outline"
                className="border-white/20 bg-white/10 text-white hover:bg-white/15"
                isLoading={exportMutation.isPending}
                onClick={() => {
                  if (validationError !== null) {
                    showToast({ title: "导出条件不合法", description: validationError, variant: "destructive" });
                    return;
                  }

                  exportMutation.mutate(
                    {
                      ...buildAuditParams(draft),
                      format: "excel",
                    },
                    {
                      onSuccess: (result) => {
                        saveDownloadedFile(result.blob, result.filename ?? "audit_logs.xlsx");
                        showToast({ title: "记录已开始导出", variant: "success" });
                      },
                      onError: (error) => {
                        showToast({ title: "导出失败", description: error.message, variant: "destructive" });
                      },
                    },
                  );
                }}
              >
                <Download className="h-4 w-4" />
                导出 Excel
              </Button>
              <Button
                type="button"
                variant="outline"
                className="border-white/20 bg-white/10 text-white hover:bg-white/15"
                isLoading={exportMutation.isPending}
                onClick={() => {
                  if (validationError !== null) {
                    showToast({ title: "导出条件不合法", description: validationError, variant: "destructive" });
                    return;
                  }

                  exportMutation.mutate(
                    {
                      ...buildAuditParams(draft),
                      format: "csv",
                    },
                    {
                      onSuccess: (result) => {
                        saveDownloadedFile(result.blob, result.filename ?? "audit_logs.csv");
                        showToast({ title: "CSV 导出已开始", variant: "success" });
                      },
                      onError: (error) => {
                        showToast({ title: "导出失败", description: error.message, variant: "destructive" });
                      },
                    },
                  );
                }}
              >
                <Download className="h-4 w-4" />
                导出 CSV
              </Button>
            </div>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>筛选条件</CardTitle>
          <CardDescription>支持来源、时间范围、操作人、动作、IP 和关键词组合筛选。</CardDescription>
        </CardHeader>
        <CardContent className="space-y-5">
          <div className="flex flex-wrap gap-2">
            {SYSTEM_AUDIT_SOURCE_OPTIONS.map((item) => (
              <Button
                key={item.value}
                type="button"
                size="sm"
                variant={draft.source === item.value ? "primary" : "outline"}
                onClick={() => setDraft((current) => ({ ...current, source: item.value }))}
              >
                {item.label}
              </Button>
            ))}
          </div>

          <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
            <FormField
              label="开始时间"
              required={draft.source === "all"}
              error={draft.source === "all" && validationError !== null ? validationError : undefined}
            >
              <Input
                type="datetime-local"
                value={draft.dateFrom}
                hasError={draft.source === "all" && validationError !== null}
                onChange={(event) => setDraft((current) => ({ ...current, dateFrom: event.target.value }))}
              />
            </FormField>
            <FormField label="结束时间" required={draft.source === "all"}>
              <Input
                type="datetime-local"
                value={draft.dateTo}
                hasError={draft.source === "all" && validationError !== null}
                onChange={(event) => setDraft((current) => ({ ...current, dateTo: event.target.value }))}
              />
            </FormField>
            <FormField label="操作人 ID">
              <Input
                value={draft.operatorID}
                onChange={(event) => setDraft((current) => ({ ...current, operatorID: event.target.value }))}
                placeholder="按 operator_id 精确筛选"
              />
            </FormField>
            <FormField label="操作类型">
              <Input
                value={draft.action}
                onChange={(event) => setDraft((current) => ({ ...current, action: event.target.value }))}
                placeholder="例如 login_success / terminal_command"
              />
            </FormField>
            <FormField label="IP 地址">
              <Input
                value={draft.ip}
                onChange={(event) => setDraft((current) => ({ ...current, ip: event.target.value }))}
                placeholder="192.168.1.100"
              />
            </FormField>
            <FormField label="关键词">
              <Input
                value={draft.keyword}
                onChange={(event) => setDraft((current) => ({ ...current, keyword: event.target.value }))}
                placeholder="支持操作人、动作、命令、IP 等关键词"
              />
            </FormField>
          </div>

          <div className="flex flex-wrap items-center gap-3">
            <Button
              type="button"
              isLoading={query.isFetching}
              onClick={() => {
                if (validationError !== null) {
                  showToast({ title: "筛选条件不合法", description: validationError, variant: "destructive" });
                  return;
                }
                setExpandedID(null);
                setParams(buildAuditParams(draft));
              }}
            >
              <Search className="h-4 w-4" />
              搜索
            </Button>
            <Button
              type="button"
              variant="outline"
              onClick={() => {
                const nextDraft = createInitialDraft();
                setDraft(nextDraft);
                setExpandedID(null);
                setParams(buildAuditParams(nextDraft));
              }}
            >
              重置为最近 7 天
            </Button>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>来源概览</CardTitle>
          <CardDescription>展示当前结果中各类记录的数量分布。</CardDescription>
        </CardHeader>
        <CardContent className="flex flex-wrap gap-3">
          <SourceCountPill label="登录日志" value={query.data?.source_counts.login ?? 0} />
          <SourceCountPill label="操作日志" value={query.data?.source_counts.operation ?? 0} />
          <SourceCountPill label="实验操作日志" value={query.data?.source_counts.experiment ?? 0} />
        </CardContent>
      </Card>

      {query.isLoading && query.data === undefined ? <LoadingState title="正在查询操作记录" description="正在整理不同来源的记录并按时间排序。" /> : null}
      {query.isError && query.data === undefined ? <ErrorState description={query.error.message} /> : null}

      {query.data !== undefined && query.data.list.length === 0 ? (
        <EmptyState title="暂无操作记录" description="当前筛选条件下没有匹配的记录。" className="min-h-72" />
      ) : null}

      {query.data !== undefined && query.data.list.length > 0 ? (
        <Card>
          <CardHeader>
            <CardTitle>查询结果</CardTitle>
            <CardDescription>点击任意一行可展开查看更完整的记录详情。</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <TableContainer>
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>时间</TableHead>
                    <TableHead>来源</TableHead>
                    <TableHead>操作人</TableHead>
                    <TableHead>操作类型</TableHead>
                    <TableHead>目标</TableHead>
                    <TableHead>详情摘要</TableHead>
                    <TableHead>IP</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {query.data.list.map((item) => {
                    const isExpanded = expandedID === item.id;

                    return (
                      <Fragment key={item.id}>
                        <TableRow
                          className={cn("cursor-pointer", isExpanded ? "bg-primary/5" : "")}
                          onClick={() => setExpandedID((current) => (current === item.id ? null : item.id))}
                        >
                          <TableCell>{formatDateTime(item.created_at)}</TableCell>
                          <TableCell>
                            <span className={sourceBadgeClassName(item.source)}>{item.source_text}</span>
                          </TableCell>
                          <TableCell>{item.operator_name ?? item.operator_id ?? "系统"}</TableCell>
                          <TableCell>{item.action_text}</TableCell>
                          <TableCell>{renderAuditTarget(item)}</TableCell>
                          <TableCell className="max-w-sm truncate text-muted-foreground">{summarizeAuditDetail(item.detail)}</TableCell>
                          <TableCell>{item.ip ?? "—"}</TableCell>
                        </TableRow>
                        {isExpanded ? (
                          <TableRow key={`${item.id}-detail`}>
                            <TableCell colSpan={7} className="bg-muted/15">
                              <div className="grid gap-4 lg:grid-cols-[0.8fr_1.2fr]">
                                <div className="space-y-2 rounded-2xl border border-border/70 bg-card p-4">
                                  <p className="text-sm font-semibold text-foreground">记录概览</p>
                                  <DetailLine label="记录ID" value={item.id} />
                                  <DetailLine label="来源" value={item.source_text} />
                                  <DetailLine label="操作人" value={item.operator_name ?? item.operator_id ?? "系统"} />
                                  <DetailLine label="操作" value={item.action_text} />
                                  <DetailLine label="IP" value={item.ip ?? "—"} />
                                  <DetailLine label="User-Agent" value={item.user_agent ?? "—"} />
                                </div>
                                <div className="space-y-2 rounded-2xl border border-border/70 bg-slate-950 p-4 text-slate-100">
                                  <p className="text-sm font-semibold text-white">详细内容</p>
                                  <pre className="max-h-80 overflow-auto rounded-xl bg-black/25 p-4 text-xs leading-6">
                                    {JSON.stringify(item.detail, null, 2)}
                                  </pre>
                                </div>
                              </div>
                            </TableCell>
                          </TableRow>
                        ) : null}
                      </Fragment>
                    );
                  })}
                </TableBody>
              </Table>
            </TableContainer>

            <Pagination
              page={query.data.pagination.page}
              totalPages={query.data.pagination.total_pages}
              total={query.data.pagination.total}
              onPageChange={(page) => {
                setExpandedID(null);
                setParams((current) => ({ ...current, page }));
              }}
            />
          </CardContent>
        </Card>
      ) : null}
    </div>
  );
}

function SourceCountPill({ label, value }: { label: string; value: number }) {
  return (
    <div className="rounded-full border border-border/70 bg-muted/25 px-4 py-2">
      <span className="text-sm text-muted-foreground">{label}</span>
      <span className="ml-3 font-semibold text-foreground">{value}</span>
    </div>
  );
}

function DetailLine({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex items-center justify-between gap-4 border-b border-border/70 pb-2 text-sm last:border-b-0 last:pb-0">
      <span className="text-muted-foreground">{label}</span>
      <span className="max-w-[70%] truncate text-right text-foreground">{value}</span>
    </div>
  );
}

function renderAuditTarget(item: SystemAuditLogItem) {
  if (item.target === null) {
    return "—";
  }

  return item.target.id ? `${item.target.type}:${item.target.id}` : item.target.type;
}

function summarizeAuditDetail(detail: SystemJSONValue) {
  if (detail === null) {
    return "—";
  }
  if (typeof detail === "string" || typeof detail === "number" || typeof detail === "boolean") {
    return String(detail);
  }
  if (Array.isArray(detail)) {
    return `${detail.length} 项明细`;
  }

  const entries = Object.entries(detail).slice(0, 3);
  return entries
    .map(([key, value]) => `${key}: ${typeof value === "object" ? JSON.stringify(value) : String(value)}`)
    .join("；");
}

function sourceBadgeClassName(source: SystemAuditSource) {
  if (source === "login") {
    return "inline-flex rounded-full bg-sky-500/12 px-2.5 py-1 text-xs font-semibold text-sky-700 dark:text-sky-300";
  }
  if (source === "operation") {
    return "inline-flex rounded-full bg-amber-500/12 px-2.5 py-1 text-xs font-semibold text-amber-700 dark:text-amber-300";
  }
  return "inline-flex rounded-full bg-emerald-500/12 px-2.5 py-1 text-xs font-semibold text-emerald-700 dark:text-emerald-300";
}
