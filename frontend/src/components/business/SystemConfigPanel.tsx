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
    return <LoadingState title="正在加载平台设置" description="正在读取平台基础设置、安全规则和数据保障选项。" />;
  }

  if (query.isError && query.data === undefined) {
    return <ErrorState title="平台设置加载失败" description={query.error.message} />;
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
                平台设置
              </div>
              <div>
                <h2 className="font-display text-3xl font-semibold tracking-tight">平台名称、默认规则与安全设置</h2>
                <p className="mt-2 max-w-2xl text-sm leading-7 text-slate-300">
                  每项设置单独保存，避免多人同时调整时相互覆盖。
                </p>
              </div>
            </div>
            <div className="flex flex-wrap items-center gap-3">
              <Link href="/super/system/configs/change-logs">
                <Button type="button" variant="outline" className="border-white/20 bg-white/10 text-white hover:bg-white/15">
                  <FileClock className="h-4 w-4" />
                  调整记录
                </Button>
              </Link>
              <Link href="/super/system/backups">
                <Button type="button" variant="outline" className="border-white/20 bg-white/10 text-white hover:bg-white/15">
                  数据保障
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
            <CardDescription>设置按条目分别保存，避免影响同组其他内容。</CardDescription>
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
                        当前为{getValueTypeText(item.value_type)}内容，最近更新于 {formatDateTime(item.updated_at)}。
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
                          title="确认保存设置"
                          description={`将“${item.description || item.key}”从「${item.value || "空"}」更新为「${draftValue || "空"}」。`}
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
                                  showToast({ title: "设置已保存", description: `${item.description || item.key} 已更新。`, variant: "success" });
                                  if (group.group === "platform" && item.key === "name" && draftValue.trim() !== "") {
                                    document.title = `${draftValue} | 平台设置`;
                                  }
                                },
                                onError: (error) => {
                                  showToast({ title: "保存失败", description: error.message, variant: "destructive" });
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
          <p className="mt-1 text-sm text-muted-foreground">启用状态会按当前设置同步保存。</p>
        </div>
      </label>
    );
  }

  return (
    <FormField label="设置内容" description={item.is_sensitive ? "敏感内容不会直接显示在页面中。" : undefined}>
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
    return <LoadingState title="正在加载调整记录" description="正在读取平台设置的历史调整情况。" />;
  }

  if (query.isError && query.data === undefined) {
    return <ErrorState title="调整记录加载失败" description={query.error.message} />;
  }

  return (
    <div className="space-y-6">
      <Card className="overflow-hidden border-none bg-[linear-gradient(135deg,#0f172a_0%,#1d4d4f_45%,#312e81_100%)] text-white shadow-[0_32px_90px_-40px_rgba(15,23,42,0.72)]">
        <CardContent className="p-6 md:p-8">
          <div className="flex flex-col gap-5 lg:flex-row lg:items-end lg:justify-between">
            <div className="space-y-3">
              <Link href="/super/system/configs" className="inline-flex items-center gap-2 text-sm font-semibold text-slate-200">
                <ArrowLeft className="h-4 w-4" />
                返回平台设置
              </Link>
              <div>
                <h2 className="font-display text-3xl font-semibold tracking-tight">设置调整记录</h2>
                <p className="mt-2 max-w-2xl text-sm leading-7 text-slate-300">
                  查看平台设置的前后变化、操作人和调整时间，便于追溯和协作。
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
          <CardDescription>可按设置类别、关键字和时间范围查看历史调整。</CardDescription>
        </CardHeader>
        <CardContent className="space-y-5">
          <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
            <FormField label="设置类别">
              <Select value={draft.configGroup} onValueChange={(value) => setDraft((current) => ({ ...current, configGroup: value as ConfigLogFilterDraft["configGroup"] }))}>
                <SelectTrigger>
                  <SelectValue placeholder="全部类别" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">全部类别</SelectItem>
                  <SelectItem value="platform">platform</SelectItem>
                  <SelectItem value="storage">storage</SelectItem>
                  <SelectItem value="security">security</SelectItem>
                  <SelectItem value="backup">backup</SelectItem>
                </SelectContent>
              </Select>
            </FormField>
            <FormField label="设置关键字">
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
              查看记录
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
              重置为最近 7 天
            </Button>
          </div>
        </CardContent>
      </Card>

      {query.data !== undefined && query.data.list.length === 0 ? (
        <EmptyState title="暂无调整记录" description="当前筛选条件下还没有可查看的历史记录。" className="min-h-72" />
      ) : null}

      {query.data !== undefined && query.data.list.length > 0 ? (
        <Card>
          <CardHeader>
            <CardTitle>历史记录</CardTitle>
            <CardDescription>查看设置内容的前后变化、操作人和来源信息。</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <TableContainer>
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>时间</TableHead>
                    <TableHead>设置内容</TableHead>
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
