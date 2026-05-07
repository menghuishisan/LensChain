"use client";

// BackupPanel.tsx
// 模块08数据备份组件，覆盖备份配置、手动触发、历史列表与下载操作。

import { Clock3, DatabaseBackup, Download, RefreshCcw, ShieldCheck, TriangleAlert } from "lucide-react";
import { useEffect, useMemo, useState } from "react";

import { Button } from "@/components/ui/Button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/Card";
import { ConfirmDialog } from "@/components/ui/ConfirmDialog";
import { EmptyState } from "@/components/ui/EmptyState";
import { ErrorState } from "@/components/ui/ErrorState";
import { FormField } from "@/components/ui/FormField";
import { Input } from "@/components/ui/Input";
import { LoadingState } from "@/components/ui/LoadingState";
import { Pagination } from "@/components/ui/Pagination";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/Select";
import { Table, TableBody, TableCell, TableContainer, TableHead, TableHeader, TableRow } from "@/components/ui/Table";
import { useToast } from "@/components/ui/Toast";
import { useBackupConfig, useBackupMutations, useBackups } from "@/hooks/useBackups";
import { formatDateTime, formatFileSize, formatNumber } from "@/lib/format";
import {
  formatSystemDuration,
  normalizeSystemCronExpression,
  saveDownloadedFile,
  SYSTEM_BACKUP_STATUS_OPTIONS,
} from "@/lib/system";
import type { SystemBackupConfig, SystemBackupStatus } from "@/types/system";

/**
 * BackupPanel 数据备份管理组件。
 */
export function BackupPanel() {
  const { showToast } = useToast();
  const [params, setParams] = useState<{ page: number; page_size: number; status?: SystemBackupStatus }>({
    page: 1,
    page_size: 20,
  });
  const backupsQuery = useBackups(params);
  const configQuery = useBackupConfig();
  const mutations = useBackupMutations();
  const [draft, setDraft] = useState<SystemBackupConfig | null>(null);
  const [cronError, setCronError] = useState<string | null>(null);

  useEffect(() => {
    if (configQuery.data !== undefined) {
      setDraft(configQuery.data);
    }
  }, [configQuery.data]);

  const isInitialLoading =
    backupsQuery.isLoading &&
    configQuery.isLoading &&
    backupsQuery.data === undefined &&
    configQuery.data === undefined;

  const blockingError =
    (backupsQuery.isError && backupsQuery.data === undefined && backupsQuery.error) ||
    (configQuery.isError && configQuery.data === undefined && configQuery.error) ||
    null;

  const hasRunningBackup = useMemo(
    () => (backupsQuery.data?.list ?? []).some((item) => item.status === 1),
    [backupsQuery.data?.list],
  );

  if (isInitialLoading) {
    return <LoadingState variant="hero" title="正在加载数据备份" description="链镜正在读取备份配置、备份历史和当前执行状态。" />;
  }

  if (blockingError !== null) {
    return (
      <ErrorState
        title="数据备份加载失败"
        description={blockingError.message}
        action={
          <Button
            type="button"
            variant="outline"
            onClick={() => {
              void Promise.all([backupsQuery.refetch(), configQuery.refetch()]);
            }}
          >
            <RefreshCcw className="h-4 w-4" />
            重新加载
          </Button>
        }
      />
    );
  }

  return (
    <div className="space-y-6">
      <Card className="overflow-hidden border-none bg-[linear-gradient(135deg,hsl(220_40%_12%),hsl(var(--primary)/0.45)_52%,hsl(220_40%_14%))] text-white shadow-[0_32px_90px_-40px_rgba(15,23,42,0.72)]">
        <CardContent className="p-6 md:p-8">
          <div className="flex flex-col gap-5 lg:flex-row lg:items-end lg:justify-between">
            <div className="space-y-3">
              <div className="inline-flex items-center gap-2 rounded-full border border-white/12 bg-white/10 px-3 py-1 text-xs font-semibold uppercase tracking-[0.2em] text-emerald-100/90">
                <DatabaseBackup className="h-3.5 w-3.5" />
                数据备份管理
              </div>
              <div>
                <h2 className="font-display text-3xl font-semibold tracking-tight">自动保留，手动可控</h2>
                <p className="mt-2 max-w-2xl text-sm leading-7 text-slate-300">
                  备份文件存储到 MinIO，支持手动触发 PostgreSQL 全量备份、下载成功备份文件，并按保留策略自动清理旧备份。
                </p>
              </div>
            </div>
            <div className="flex flex-wrap items-center gap-3">
              <div className="rounded-2xl border border-white/12 bg-white/8 px-4 py-3">
                <p className="text-xs uppercase tracking-[0.18em] text-slate-400">当前状态</p>
                <p className="mt-1 text-sm font-semibold text-white">{hasRunningBackup ? "有备份正在执行" : "当前无执行中的备份"}</p>
              </div>
              <ConfirmDialog
                title="确认手动触发备份"
                description="系统将异步执行 PostgreSQL 全量备份。备份进行中期间不允许重复触发。"
                confirmText="立即备份"
                confirmVariant="primary"
                trigger={
                  <Button type="button" className="bg-white text-slate-950 hover:bg-slate-100" disabled={hasRunningBackup}>
                    <DatabaseBackup className="h-4 w-4" />
                    手动备份
                  </Button>
                }
                onConfirm={() => {
                  mutations.trigger.mutate(undefined, {
                    onSuccess: () => {
                      showToast({ title: "备份任务已触发", description: "请在备份历史中查看执行状态。", variant: "success" });
                    },
                    onError: (error) => {
                      showToast({ title: "触发失败", description: error.message, variant: "destructive" });
                    },
                  });
                }}
              />
            </div>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader className="flex flex-row items-start justify-between gap-4">
          <div className="space-y-1">
            <CardTitle>备份配置</CardTitle>
            <CardDescription>
              `auto_backup_cron` 仅接受 6 段秒级 cron，例如 <code>0 0 2 * * *</code>；提交时会原样回传三个
              <code>updated_at</code> 字段。
            </CardDescription>
          </div>
          <Button
            type="button"
            variant="outline"
            isLoading={configQuery.isRefetching}
            onClick={() => void configQuery.refetch()}
          >
            <RefreshCcw className="h-4 w-4" />
            刷新配置
          </Button>
        </CardHeader>
        <CardContent>
          {draft === null ? (
            <LoadingState title="正在读取备份配置" description="链镜正在同步自动备份策略。" className="min-h-44" />
          ) : (
            <form
              className="grid gap-6 xl:grid-cols-[1.2fr_0.8fr]"
              onSubmit={(event) => {
                event.preventDefault();
                const normalizedCron = normalizeSystemCronExpression(draft.cron);
                if (normalizedCron === null) {
                  setCronError("cron 必须是 6 段秒级表达式，例如 0 0 2 * * *");
                  return;
                }

                setCronError(null);
                mutations.updateConfig.mutate(
                  {
                    ...draft,
                    cron: normalizedCron,
                  },
                  {
                    onSuccess: (response) => {
                      setDraft(response);
                      showToast({ title: "备份配置已保存", variant: "success" });
                    },
                    onError: (error) => {
                      showToast({ title: "备份配置保存失败", description: error.message, variant: "destructive" });
                    },
                  },
                );
              }}
            >
              <div className="space-y-5 rounded-[1.75rem] border border-border/70 bg-muted/20 p-5">
                <div className="grid gap-4 md:grid-cols-2">
                  <label className="flex items-center gap-3 rounded-2xl border border-border bg-card px-4 py-4">
                    <input
                      type="checkbox"
                      checked={draft.auto_enabled}
                      onChange={(event) =>
                        setDraft((current) =>
                          current === null
                            ? current
                            : {
                                ...current,
                                auto_enabled: event.target.checked,
                              },
                        )
                      }
                    />
                    <div>
                      <p className="font-semibold text-foreground">启用自动备份</p>
                      <p className="mt-1 text-sm text-muted-foreground">关闭后将移除自动备份调度任务。</p>
                    </div>
                  </label>
                  <div className="rounded-2xl border border-border bg-card px-4 py-4">
                    <p className="text-sm font-semibold text-foreground">当前保留策略</p>
                    <p className="mt-2 text-sm leading-6 text-muted-foreground">
                      默认保留最近 {formatNumber(draft.retention_count)} 份备份，超出后自动删除最旧备份。
                    </p>
                  </div>
                </div>

                <FormField
                  label="自动备份 cron"
                  required
                  description="统一使用 6 段秒级 cron 表达式，例如每日凌晨 2:00 为 0 0 2 * * *。"
                  error={cronError ?? undefined}
                >
                  <Input
                    value={draft.cron}
                    hasError={cronError !== null}
                    onChange={(event) => {
                      setCronError(null);
                      setDraft((current) => (current === null ? current : { ...current, cron: event.target.value }));
                    }}
                    placeholder="0 0 2 * * *"
                  />
                </FormField>

                <FormField label="保留份数" required description="最少保留 1 份备份文件。">
                  <Input
                    type="number"
                    min={1}
                    value={draft.retention_count}
                    onChange={(event) =>
                      setDraft((current) =>
                        current === null
                          ? current
                          : {
                              ...current,
                              retention_count: Math.max(1, Number(event.target.value)),
                            },
                      )
                    }
                  />
                </FormField>

                <div className="flex flex-wrap items-center gap-3">
                  <Button type="submit" isLoading={mutations.updateConfig.isPending}>
                    <ShieldCheck className="h-4 w-4" />
                    保存备份配置
                  </Button>
                  <span className="text-sm text-muted-foreground">
                    最近更新于 {formatDateTime(draft.cron_updated_at)}
                  </span>
                </div>
              </div>

              <div className="rounded-[1.75rem] border border-border/70 bg-slate-950 p-5 text-white">
                <div className="flex items-start gap-3">
                  <div className="rounded-2xl bg-white/10 p-3 text-teal-300">
                    <Clock3 className="h-5 w-5" />
                  </div>
                  <div>
                    <p className="font-semibold">备份策略摘要</p>
                    <p className="mt-2 text-sm leading-6 text-slate-300">
                      自动备份 {draft.auto_enabled ? "已启用" : "已关闭"}，当前 cron 为 <code>{draft.cron}</code>，保留最近{" "}
                      {formatNumber(draft.retention_count)} 份。
                    </p>
                  </div>
                </div>
                <div className="mt-5 space-y-3 text-sm">
                  <SummaryItem label="auto_enabled.updated_at" value={formatDateTime(draft.auto_enabled_updated_at)} />
                  <SummaryItem label="cron.updated_at" value={formatDateTime(draft.cron_updated_at)} />
                  <SummaryItem label="retention.updated_at" value={formatDateTime(draft.retention_count_updated_at)} />
                </div>
              </div>
            </form>
          )}
        </CardContent>
      </Card>

      <Card>
        <CardHeader className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
          <div className="space-y-1">
            <CardTitle>备份历史</CardTitle>
            <CardDescription>展示备份时间、类型、状态、文件大小、耗时以及失败原因。</CardDescription>
          </div>
          <div className="w-full max-w-56">
            <Select
              value={params.status === undefined ? "all" : String(params.status)}
              onValueChange={(value) =>
                setParams((current) => ({
                  ...current,
                  page: 1,
                  status: value === "all" ? undefined : (Number(value) as SystemBackupStatus),
                }))
              }
            >
              <SelectTrigger>
                <SelectValue placeholder="全部状态" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">全部状态</SelectItem>
                {SYSTEM_BACKUP_STATUS_OPTIONS.map((item) => (
                  <SelectItem key={item.value} value={String(item.value)}>
                    {item.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        </CardHeader>
        <CardContent className="space-y-4">
          {backupsQuery.isError && backupsQuery.data === undefined ? (
            <ErrorState description={backupsQuery.error.message} />
          ) : null}

          {(backupsQuery.data?.list ?? []).length === 0 ? (
            <EmptyState title="暂无备份记录" description="当前筛选条件下没有匹配的备份历史。" className="min-h-56" />
          ) : (
            <TableContainer>
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>开始时间</TableHead>
                    <TableHead>类型</TableHead>
                    <TableHead>数据库</TableHead>
                    <TableHead>大小</TableHead>
                    <TableHead>耗时</TableHead>
                    <TableHead>状态</TableHead>
                    <TableHead>操作</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {(backupsQuery.data?.list ?? []).map((item) => (
                    <TableRow key={item.id}>
                      <TableCell>
                        <div>
                          <p className="font-medium text-foreground">{formatDateTime(item.started_at)}</p>
                          <p className="mt-1 text-xs text-muted-foreground">
                            完成于 {item.completed_at !== null ? formatDateTime(item.completed_at) : "—"}
                          </p>
                        </div>
                      </TableCell>
                      <TableCell>{item.backup_type_text}</TableCell>
                      <TableCell>{item.database_name}</TableCell>
                      <TableCell>{item.file_size_text ?? formatFileSize(item.file_size ?? -1)}</TableCell>
                      <TableCell>{formatSystemDuration(item.duration_seconds)}</TableCell>
                      <TableCell>
                        <div className="space-y-2">
                          <span
                            className={
                              item.status === 1
                                ? "inline-flex rounded-full bg-amber-500/12 px-2.5 py-1 text-xs font-semibold text-amber-700 dark:text-amber-300"
                                : item.status === 2
                                  ? "inline-flex rounded-full bg-emerald-500/12 px-2.5 py-1 text-xs font-semibold text-emerald-700 dark:text-emerald-300"
                                  : "inline-flex rounded-full bg-destructive/12 px-2.5 py-1 text-xs font-semibold text-destructive"
                            }
                          >
                            {item.status_text}
                          </span>
                          {item.status === 1 ? (
                            <div className="h-1.5 w-28 overflow-hidden rounded-full bg-muted">
                              <div className="h-full w-2/3 animate-pulse rounded-full bg-gradient-to-r from-amber-400 to-orange-500" />
                            </div>
                          ) : null}
                          {item.status === 3 && item.error_message ? (
                            <p className="max-w-xs text-xs leading-5 text-destructive">{item.error_message}</p>
                          ) : null}
                        </div>
                      </TableCell>
                      <TableCell>
                        {item.status === 2 ? (
                          <Button
                            type="button"
                            size="sm"
                            variant="outline"
                            isLoading={mutations.download.isPending}
                            onClick={() => {
                              mutations.download.mutate(item.id, {
                                onSuccess: (result) => {
                                  saveDownloadedFile(result.blob, result.filename ?? `backup-${item.id}.sql.gz`);
                                  showToast({ title: "备份文件开始下载", variant: "success" });
                                },
                                onError: (error) => {
                                  showToast({ title: "下载失败", description: error.message, variant: "destructive" });
                                },
                              });
                            }}
                          >
                            <Download className="h-4 w-4" />
                            下载
                          </Button>
                        ) : item.status === 3 ? (
                          <span className="inline-flex items-center gap-2 text-xs text-destructive">
                            <TriangleAlert className="h-4 w-4" />
                            失败详情已显示
                          </span>
                        ) : (
                          <span className="text-xs text-muted-foreground">执行中</span>
                        )}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </TableContainer>
          )}

          {backupsQuery.data?.pagination !== undefined ? (
            <Pagination
              page={backupsQuery.data.pagination.page}
              totalPages={backupsQuery.data.pagination.total_pages}
              total={backupsQuery.data.pagination.total}
              onPageChange={(page) => setParams((current) => ({ ...current, page }))}
            />
          ) : null}
        </CardContent>
      </Card>
    </div>
  );
}

function SummaryItem({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex items-center justify-between gap-3 rounded-2xl border border-white/12 bg-white/8 px-4 py-3">
      <span className="text-xs uppercase tracking-[0.16em] text-slate-400">{label}</span>
      <span className="text-sm font-semibold text-white">{value}</span>
    </div>
  );
}
