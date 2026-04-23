"use client";

// SystemConfigPanel.tsx
// 模块08全局配置组件，覆盖配置分组编辑与配置变更记录查询。

import { ArrowLeft, ArrowRight, FileClock, RefreshCcw, Save, Settings2 } from "lucide-react";
import Link from "next/link";
import { useEffect, useState } from "react";

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
import { useSystemConfigChangeLogs, useSystemConfigMutations, useSystemConfigs } from "@/hooks/useSystemConfigs";
import { createSystemRecentRange, fromSystemDateTimeLocalValue } from "@/lib/system";
import { formatDateTime } from "@/lib/format";
import type {
  SystemConfigChangeLogListParams,
  SystemConfigGroup,
  SystemConfigGroupItem,
  SystemConfigItem,
  SystemConfigValueType,
} from "@/types/system";

const DEFAULT_CONFIG_LOG_RANGE = createSystemRecentRange(7);

type ConfigDraftMap = Record<string, string>;

interface ConfigLogFilterDraft {
  configGroup: "all" | SystemConfigGroup;
  configKey: string;
  dateFrom: string;
  dateTo: string;
}

/**
 * SystemConfigPanel 组件属性。
 */
export interface SystemConfigPanelProps {
  mode?: "configs" | "change-logs";
}

function configDraftKey(group: SystemConfigGroup, key: string) {
  return `${group}.${key}`;
}

function createDraftMap(groups: SystemConfigGroupItem[]) {
  return groups.reduce<ConfigDraftMap>((result, group) => {
    group.configs.forEach((item) => {
      result[configDraftKey(group.group, item.key)] = item.value;
    });
    return result;
  }, {});
}

function createLogDraft(): ConfigLogFilterDraft {
  return {
    configGroup: "all",
    configKey: "",
    dateFrom: DEFAULT_CONFIG_LOG_RANGE.dateFrom,
    dateTo: DEFAULT_CONFIG_LOG_RANGE.dateTo,
  };
}

function buildLogParams(draft: ConfigLogFilterDraft, page = 1): SystemConfigChangeLogListParams {
  return {
    page,
    page_size: 20,
    config_group: draft.configGroup === "all" ? undefined : draft.configGroup,
    config_key: draft.configKey || undefined,
    date_from: fromSystemDateTimeLocalValue(draft.dateFrom),
    date_to: fromSystemDateTimeLocalValue(draft.dateTo),
  };
}

/**
 * SystemConfigPanel 全局配置与变更记录组件。
 */
export function SystemConfigPanel({ mode = "configs" }: SystemConfigPanelProps) {
  if (mode === "change-logs") {
    return <SystemConfigChangeLogView />;
  }

  return <SystemConfigEditorView />;
}

function SystemConfigEditorView() {
  const { showToast } = useToast();
  const query = useSystemConfigs();
  const mutations = useSystemConfigMutations();
  const [draftMap, setDraftMap] = useState<ConfigDraftMap>({});

  useEffect(() => {
    if (query.data !== undefined) {
      const editorGroups = query.data.groups.filter((group) => group.group !== "backup");
      setDraftMap(createDraftMap(editorGroups));
    }
  }, [query.data]);

  useEffect(() => {
    const platformName = draftMap["platform.name"];
    if (platformName !== undefined && platformName.trim() !== "") {
      document.title = `${platformName} | 系统管理`;
    }
  }, [draftMap]);

  if (query.isLoading && query.data === undefined) {
    return <LoadingState title="正在加载全局配置" description="链镜正在读取平台、存储、安全与备份配置分组。" />;
  }

  if (query.isError && query.data === undefined) {
    return <ErrorState title="全局配置加载失败" description={query.error.message} />;
  }

  const groups = (query.data?.groups ?? []).filter((group) => group.group !== "backup");

  return (
    <div className="space-y-6">
      <Card className="overflow-hidden border-none bg-[linear-gradient(135deg,#0f172a_0%,#0f766e_45%,#1e293b_100%)] text-white shadow-[0_32px_90px_-40px_rgba(15,23,42,0.72)]">
        <CardContent className="p-6 md:p-8">
          <div className="flex flex-col gap-5 lg:flex-row lg:items-end lg:justify-between">
            <div className="space-y-3">
              <div className="inline-flex items-center gap-2 rounded-full border border-white/12 bg-white/10 px-3 py-1 text-xs font-semibold uppercase tracking-[0.2em] text-emerald-100/90">
                <Settings2 className="h-3.5 w-3.5" />
                全局配置管理
              </div>
              <div>
                <h2 className="font-display text-3xl font-semibold tracking-tight">平台名、默认值与安全基线</h2>
                <p className="mt-2 max-w-2xl text-sm leading-7 text-slate-300">
                  每个配置项独立保存，提交时原样回传后端返回的 <code>updated_at</code>，避免并发修改覆盖。
                </p>
              </div>
            </div>
            <div className="flex flex-wrap items-center gap-3">
              <Link href="/super/system/configs/change-logs">
                <Button type="button" variant="outline" className="border-white/20 bg-white/10 text-white hover:bg-white/15">
                  <FileClock className="h-4 w-4" />
                  变更记录
                </Button>
              </Link>
              <Link href="/super/system/backups">
                <Button type="button" variant="outline" className="border-white/20 bg-white/10 text-white hover:bg-white/15">
                  备份配置
                  <ArrowRight className="h-4 w-4" />
                </Button>
              </Link>
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
            </div>
          </div>
        </CardContent>
      </Card>

      {groups.map((group) => (
        <Card key={group.group}>
          <CardHeader>
            <CardTitle>{group.group_text}</CardTitle>
            <CardDescription>按配置项独立保存，避免误改同组其他配置。</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            {group.configs.map((item) => {
              const key = configDraftKey(group.group, item.key);
              const draftValue = draftMap[key] ?? item.value;
              const isDirty = draftValue !== item.value;
              const isLoading = mutations.updateOne.isPending;

              return (
                <div key={item.key} className="rounded-[1.5rem] border border-border/70 bg-muted/20 p-5">
                  <div className="flex flex-col gap-4 xl:flex-row xl:items-start xl:justify-between">
                    <div className="min-w-0 space-y-2 xl:max-w-sm">
                      <div className="flex flex-wrap items-center gap-2">
                        <h3 className="font-semibold text-foreground">{item.description || item.key}</h3>
                        <span className="rounded-full bg-primary/10 px-2.5 py-1 text-xs font-semibold text-primary">
                          {group.group}.{item.key}
                        </span>
                      </div>
                      <p className="text-sm leading-6 text-muted-foreground">
                        值类型 {getValueTypeText(item.value_type)}，最近更新于 {formatDateTime(item.updated_at)}。
                      </p>
                    </div>

                    <div className="grid min-w-0 flex-1 gap-4 md:grid-cols-[1fr_auto]">
                      <ConfigValueField
                        item={item}
                        value={draftValue}
                        onChange={(value) => setDraftMap((current) => ({ ...current, [key]: value }))}
                      />
                      <div className="flex items-center justify-end">
                        <ConfirmDialog
                          title="确认保存配置"
                          description={`将 ${group.group}.${item.key} 从「${item.value || "空"}」更新为「${draftValue || "空"}」。`}
                          confirmText="确认保存"
                          confirmVariant="primary"
                          trigger={
                            <Button type="button" disabled={!isDirty || isLoading}>
                              <Save className="h-4 w-4" />
                              保存
                            </Button>
                          }
                          onConfirm={() => {
                            mutations.updateOne.mutate(
                              {
                                group: group.group,
                                key: item.key,
                                payload: {
                                  value: draftValue,
                                  updated_at: item.updated_at,
                                },
                              },
                              {
                                onSuccess: () => {
                                  showToast({ title: "配置已保存", description: `${group.group}.${item.key} 已更新。`, variant: "success" });
                                  if (group.group === "platform" && item.key === "name" && draftValue.trim() !== "") {
                                    document.title = `${draftValue} | 系统管理`;
                                  }
                                },
                                onError: (error) => {
                                  showToast({ title: "配置保存失败", description: error.message, variant: "destructive" });
                                },
                              },
                            );
                          }}
                        />
                      </div>
                    </div>
                  </div>
                </div>
              );
            })}
          </CardContent>
        </Card>
      ))}
    </div>
  );
}

function ConfigValueField({
  item,
  value,
  onChange,
}: {
  item: SystemConfigItem;
  value: string;
  onChange: (value: string) => void;
}) {
  if (item.value_type === 3) {
    return (
      <label className="flex items-center gap-3 rounded-2xl border border-border bg-card px-4 py-3">
        <input type="checkbox" checked={value === "true"} onChange={(event) => onChange(String(event.target.checked))} />
        <div>
          <p className="font-semibold text-foreground">{value === "true" ? "已启用" : "已关闭"}</p>
          <p className="mt-1 text-sm text-muted-foreground">布尔配置以字符串形式回传给后端。</p>
        </div>
      </label>
    );
  }

  return (
    <FormField label="配置值" description={item.is_sensitive ? "敏感配置不会在响应中明文展示。" : undefined}>
      <Input
        type={item.value_type === 2 ? "number" : item.is_sensitive ? "password" : "text"}
        value={value}
        onChange={(event) => onChange(event.target.value)}
        placeholder={item.is_sensitive ? "请输入新的敏感值" : undefined}
      />
    </FormField>
  );
}

function getValueTypeText(valueType: SystemConfigValueType) {
  if (valueType === 1) {
    return "字符串";
  }
  if (valueType === 2) {
    return "数字";
  }
  if (valueType === 3) {
    return "布尔";
  }
  return "JSON";
}

function SystemConfigChangeLogView() {
  const [draft, setDraft] = useState(createLogDraft);
  const [params, setParams] = useState<SystemConfigChangeLogListParams>(() => buildLogParams(createLogDraft()));
  const query = useSystemConfigChangeLogs(params);

  if (query.isLoading && query.data === undefined) {
    return <LoadingState title="正在加载配置变更记录" description="链镜正在读取 config_change_logs 历史。" />;
  }

  if (query.isError && query.data === undefined) {
    return <ErrorState title="配置变更记录加载失败" description={query.error.message} />;
  }

  return (
    <div className="space-y-6">
      <Card className="overflow-hidden border-none bg-[linear-gradient(135deg,#0f172a_0%,#1d4d4f_45%,#312e81_100%)] text-white shadow-[0_32px_90px_-40px_rgba(15,23,42,0.72)]">
        <CardContent className="p-6 md:p-8">
          <div className="flex flex-col gap-5 lg:flex-row lg:items-end lg:justify-between">
            <div className="space-y-3">
              <Link href="/super/system/configs" className="inline-flex items-center gap-2 text-sm font-semibold text-slate-200">
                <ArrowLeft className="h-4 w-4" />
                返回全局配置
              </Link>
              <div>
                <h2 className="font-display text-3xl font-semibold tracking-tight">配置变更记录</h2>
                <p className="mt-2 max-w-2xl text-sm leading-7 text-slate-300">
                  记录平台配置变更前后值、操作人和变更时间，用于审计和回溯。
                </p>
              </div>
            </div>
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
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>筛选条件</CardTitle>
          <CardDescription>支持按配置分组、配置键和时间范围筛选变更记录。</CardDescription>
        </CardHeader>
        <CardContent className="space-y-5">
          <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
            <FormField label="配置分组">
              <Select value={draft.configGroup} onValueChange={(value) => setDraft((current) => ({ ...current, configGroup: value as ConfigLogFilterDraft["configGroup"] }))}>
                <SelectTrigger>
                  <SelectValue placeholder="全部分组" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">全部分组</SelectItem>
                  <SelectItem value="platform">platform</SelectItem>
                  <SelectItem value="storage">storage</SelectItem>
                  <SelectItem value="security">security</SelectItem>
                  <SelectItem value="backup">backup</SelectItem>
                </SelectContent>
              </Select>
            </FormField>
            <FormField label="配置键">
              <Input value={draft.configKey} onChange={(event) => setDraft((current) => ({ ...current, configKey: event.target.value }))} placeholder="例如 name / session_timeout_hours" />
            </FormField>
            <FormField label="开始时间">
              <Input type="datetime-local" value={draft.dateFrom} onChange={(event) => setDraft((current) => ({ ...current, dateFrom: event.target.value }))} />
            </FormField>
            <FormField label="结束时间">
              <Input type="datetime-local" value={draft.dateTo} onChange={(event) => setDraft((current) => ({ ...current, dateTo: event.target.value }))} />
            </FormField>
          </div>
          <div className="flex flex-wrap items-center gap-3">
            <Button type="button" onClick={() => setParams(buildLogParams(draft))}>
              查询记录
            </Button>
            <Button
              type="button"
              variant="outline"
              onClick={() => {
                const nextDraft = createLogDraft();
                setDraft(nextDraft);
                setParams(buildLogParams(nextDraft));
              }}
            >
              重置为最近7天
            </Button>
          </div>
        </CardContent>
      </Card>

      {query.data !== undefined && query.data.list.length === 0 ? (
        <EmptyState title="暂无配置变更记录" description="当前筛选条件下没有匹配的历史变更。" className="min-h-72" />
      ) : null}

      {query.data !== undefined && query.data.list.length > 0 ? (
        <Card>
          <CardHeader>
            <CardTitle>变更历史</CardTitle>
            <CardDescription>展示配置项、变更前后值、操作人和操作 IP。</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <TableContainer>
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>时间</TableHead>
                    <TableHead>配置项</TableHead>
                    <TableHead>变更前</TableHead>
                    <TableHead>变更后</TableHead>
                    <TableHead>操作人</TableHead>
                    <TableHead>IP</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {query.data.list.map((item) => (
                    <TableRow key={item.id}>
                      <TableCell>{formatDateTime(item.changed_at)}</TableCell>
                      <TableCell>{item.config_group}.{item.config_key}</TableCell>
                      <TableCell>{item.old_value ?? "—"}</TableCell>
                      <TableCell>{item.new_value}</TableCell>
                      <TableCell>{item.changed_by_name}</TableCell>
                      <TableCell>{item.ip}</TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </TableContainer>

            <Pagination
              page={query.data.pagination.page}
              totalPages={query.data.pagination.total_pages}
              total={query.data.pagination.total}
              onPageChange={(page) => setParams((current) => ({ ...current, page }))}
            />
          </CardContent>
        </Card>
      ) : null}
    </div>
  );
}
