"use client";

// AlertRuleEditor.tsx
// 模块08告警规则组件，覆盖规则列表、创建、编辑、启停和删除。

import { BellRing, Plus, RefreshCcw, Trash2 } from "lucide-react";
import type { ReactNode } from "react";
import { useState } from "react";

import { Button } from "@/components/ui/Button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/Card";
import { ConfirmDialog } from "@/components/ui/ConfirmDialog";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/Dialog";
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
import { useAlertRuleMutations, useAlertRules } from "@/hooks/useAlertRules";
import {
  getSystemAlertLevelClassName,
  SYSTEM_ALERT_LEVEL_OPTIONS,
  SYSTEM_ALERT_TYPE_OPTIONS,
  SYSTEM_EVENT_GROUP_BY_OPTIONS,
  SYSTEM_EVENT_SOURCE_OPTIONS,
  SYSTEM_SERVICE_NAME_OPTIONS,
  SYSTEM_THRESHOLD_METRIC_OPTIONS,
  SYSTEM_THRESHOLD_OPERATOR_OPTIONS,
} from "@/lib/system";
import { cn } from "@/lib/utils";
import type { ID } from "@/types/api";
import type {
  CreateSystemAlertRuleRequest,
  SystemAlertLevel,
  SystemAlertRuleCondition,
  SystemAlertRuleItem,
  SystemAlertType,
  SystemServiceName,
  SystemThresholdMetric,
  SystemThresholdOperator,
  UpdateSystemAlertRuleRequest,
} from "@/types/system";

const LOGIN_ACTION_OPTIONS = [
  { value: 1, label: "登录成功" },
  { value: 2, label: "登录失败" },
  { value: 3, label: "主动登出" },
  { value: 4, label: "被踢下线" },
  { value: 5, label: "账号锁定" },
] as const;

interface RuleEditorState {
  open: boolean;
  mode: "create" | "edit";
  ruleID?: ID;
}

interface RuleFormState {
  name: string;
  description: string;
  alertType: SystemAlertType;
  level: SystemAlertLevel;
  silencePeriod: string;
  thresholdMetric: SystemThresholdMetric;
  thresholdOperator: SystemThresholdOperator;
  thresholdValue: string;
  thresholdDuration: string;
  eventSource: "login_logs";
  eventAction: string;
  eventGroupBy: "ip";
  eventCountThreshold: string;
  eventTimeWindow: string;
  serviceName: SystemServiceName;
  serviceCheckURL: string;
  serviceCheckInterval: string;
  serviceFailThreshold: string;
}

function createDefaultRuleForm(): RuleFormState {
  return {
    name: "",
    description: "",
    alertType: 1,
    level: 3,
    silencePeriod: "1800",
    thresholdMetric: "cpu_usage",
    thresholdOperator: ">",
    thresholdValue: "80",
    thresholdDuration: "300",
    eventSource: "login_logs",
    eventAction: "2",
    eventGroupBy: "ip",
    eventCountThreshold: "10",
    eventTimeWindow: "600",
    serviceName: "postgresql",
    serviceCheckURL: "",
    serviceCheckInterval: "30",
    serviceFailThreshold: "3",
  };
}

function createRuleFormFromItem(item: SystemAlertRuleItem): RuleFormState {
  const form = createDefaultRuleForm();
  form.name = item.name;
  form.description = item.description ?? "";
  form.alertType = item.alert_type;
  form.level = item.level;
  form.silencePeriod = String(item.silence_period);

  if (item.condition.metric !== undefined) {
    form.thresholdMetric = item.condition.metric;
  }
  if (item.condition.operator !== undefined) {
    form.thresholdOperator = item.condition.operator;
  }
  if (item.condition.value !== undefined) {
    form.thresholdValue = String(item.condition.value);
  }
  if (item.condition.duration !== undefined) {
    form.thresholdDuration = String(item.condition.duration);
  }
  if (item.condition.event_source !== undefined) {
    form.eventSource = item.condition.event_source;
  }
  if (item.condition.event_filter?.action !== undefined) {
    form.eventAction = String(item.condition.event_filter.action);
  }
  if (item.condition.group_by !== undefined) {
    form.eventGroupBy = item.condition.group_by;
  }
  if (item.condition.count_threshold !== undefined) {
    form.eventCountThreshold = String(item.condition.count_threshold);
  }
  if (item.condition.time_window !== undefined) {
    form.eventTimeWindow = String(item.condition.time_window);
  }
  if (item.condition.service_name !== undefined) {
    form.serviceName = item.condition.service_name;
  }
  if (item.condition.check_url !== undefined) {
    form.serviceCheckURL = item.condition.check_url;
  }
  if (item.condition.check_interval !== undefined) {
    form.serviceCheckInterval = String(item.condition.check_interval);
  }
  if (item.condition.fail_threshold !== undefined) {
    form.serviceFailThreshold = String(item.condition.fail_threshold);
  }

  return form;
}

function validateRuleForm(form: RuleFormState) {
  if (form.name.trim() === "") {
    return "规则名称不能为空";
  }
  if (Number(form.silencePeriod) < 0) {
    return "静默期不能小于 0";
  }
  if (form.alertType === 1) {
    if (Number(form.thresholdValue) <= 0 || Number(form.thresholdDuration) <= 0) {
      return "阈值告警的阈值和持续时间必须大于 0";
    }
  }
  if (form.alertType === 2) {
    if (Number(form.eventCountThreshold) <= 0 || Number(form.eventTimeWindow) <= 0) {
      return "事件告警的次数阈值和时间窗口必须大于 0";
    }
  }
  if (form.alertType === 3) {
    if (Number(form.serviceFailThreshold) <= 0) {
      return "服务状态告警的失败阈值必须大于 0";
    }
    if (form.serviceCheckInterval !== "" && Number(form.serviceCheckInterval) <= 0) {
      return "服务状态告警的检查间隔必须大于 0";
    }
  }
  return null;
}

function buildAlertRuleCondition(form: RuleFormState): SystemAlertRuleCondition {
  if (form.alertType === 1) {
    return {
      metric: form.thresholdMetric,
      operator: form.thresholdOperator,
      value: Number(form.thresholdValue),
      duration: Number(form.thresholdDuration),
    };
  }

  if (form.alertType === 2) {
    return {
      event_source: form.eventSource,
      event_filter: {
        action: Number(form.eventAction),
      },
      group_by: form.eventGroupBy,
      count_threshold: Number(form.eventCountThreshold),
      time_window: Number(form.eventTimeWindow),
    };
  }

  return {
    service_name: form.serviceName,
    check_url: form.serviceCheckURL.trim() === "" ? undefined : form.serviceCheckURL.trim(),
    check_interval: form.serviceCheckInterval.trim() === "" ? undefined : Number(form.serviceCheckInterval),
    fail_threshold: Number(form.serviceFailThreshold),
  };
}

function buildCreatePayload(form: RuleFormState): CreateSystemAlertRuleRequest {
  return {
    name: form.name.trim(),
    description: form.description.trim() === "" ? null : form.description.trim(),
    alert_type: form.alertType,
    level: form.level,
    silence_period: Number(form.silencePeriod),
    condition: buildAlertRuleCondition(form),
  };
}

function buildUpdatePayload(form: RuleFormState): UpdateSystemAlertRuleRequest {
  return {
    name: form.name.trim(),
    description: form.description.trim() === "" ? null : form.description.trim(),
    alert_type: form.alertType,
    level: form.level,
    silence_period: Number(form.silencePeriod),
    condition: buildAlertRuleCondition(form),
  };
}

/**
 * AlertRuleEditor 告警规则管理组件。
 */
export function AlertRuleEditor() {
  const { showToast } = useToast();
  const [params, setParams] = useState<{ page: number; page_size: number; alert_type?: SystemAlertType; level?: SystemAlertLevel; is_enabled?: boolean }>({
    page: 1,
    page_size: 20,
  });
  const [dialogState, setDialogState] = useState<RuleEditorState>({ open: false, mode: "create" });
  const [form, setForm] = useState<RuleFormState>(createDefaultRuleForm);
  const query = useAlertRules(params);
  const mutations = useAlertRuleMutations();

  if (query.isLoading && query.data === undefined) {
    return <LoadingState title="正在加载告警规则" description="链镜正在读取平台基础设施级别的告警规则清单。" />;
  }

  if (query.isError && query.data === undefined) {
    return <ErrorState title="告警规则加载失败" description={query.error.message} />;
  }

  return (
    <div className="space-y-6">
      <Card className="overflow-hidden border-none bg-[linear-gradient(135deg,#0f172a_0%,#4c1d95_45%,#7f1d1d_100%)] text-white shadow-[0_32px_90px_-40px_rgba(15,23,42,0.72)]">
        <CardContent className="p-6 md:p-8">
          <div className="flex flex-col gap-5 lg:flex-row lg:items-end lg:justify-between">
            <div className="space-y-3">
              <div className="inline-flex items-center gap-2 rounded-full border border-white/12 bg-white/10 px-3 py-1 text-xs font-semibold uppercase tracking-[0.2em] text-fuchsia-100/90">
                <BellRing className="h-3.5 w-3.5" />
                告警规则管理
              </div>
              <div>
                <h2 className="font-display text-3xl font-semibold tracking-tight">阈值、事件、服务状态</h2>
                <p className="mt-2 max-w-2xl text-sm leading-7 text-slate-300">
                  告警规则仅覆盖平台基础设施级别告警；所有枚举值与条件字段均对齐后端文档和 DTO 标准值。
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
                className="bg-white text-slate-950 hover:bg-slate-100"
                onClick={() => {
                  setForm(createDefaultRuleForm());
                  setDialogState({ open: true, mode: "create" });
                }}
              >
                <Plus className="h-4 w-4" />
                创建规则
              </Button>
            </div>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>筛选条件</CardTitle>
          <CardDescription>支持按类型、级别和启用状态筛选规则。</CardDescription>
        </CardHeader>
        <CardContent className="grid gap-4 md:grid-cols-3">
          <FormField label="告警类型">
            <Select
              value={params.alert_type === undefined ? "all" : String(params.alert_type)}
              onValueChange={(value) =>
                setParams((current) => ({
                  ...current,
                  page: 1,
                  alert_type: value === "all" ? undefined : (Number(value) as SystemAlertType),
                }))
              }
            >
              <SelectTrigger>
                <SelectValue placeholder="全部类型" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">全部类型</SelectItem>
                {SYSTEM_ALERT_TYPE_OPTIONS.map((item) => (
                  <SelectItem key={item.value} value={String(item.value)}>
                    {item.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </FormField>
          <FormField label="告警级别">
            <Select
              value={params.level === undefined ? "all" : String(params.level)}
              onValueChange={(value) =>
                setParams((current) => ({
                  ...current,
                  page: 1,
                  level: value === "all" ? undefined : (Number(value) as SystemAlertLevel),
                }))
              }
            >
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
          <FormField label="启用状态">
            <Select
              value={params.is_enabled === undefined ? "all" : String(params.is_enabled)}
              onValueChange={(value) =>
                setParams((current) => ({
                  ...current,
                  page: 1,
                  is_enabled: value === "all" ? undefined : value === "true",
                }))
              }
            >
              <SelectTrigger>
                <SelectValue placeholder="全部状态" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">全部状态</SelectItem>
                <SelectItem value="true">已启用</SelectItem>
                <SelectItem value="false">已禁用</SelectItem>
              </SelectContent>
            </Select>
          </FormField>
        </CardContent>
      </Card>

      {query.data !== undefined && query.data.list.length === 0 ? (
        <EmptyState title="暂无告警规则" description="当前筛选条件下没有匹配的规则，可从右上角创建新规则。" className="min-h-72" />
      ) : null}

      {query.data !== undefined && query.data.list.length > 0 ? (
        <Card>
          <CardHeader>
            <CardTitle>规则列表</CardTitle>
            <CardDescription>支持编辑、启停和删除；启停状态会立即影响后端检测逻辑。</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <TableContainer>
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>规则名称</TableHead>
                    <TableHead>类型</TableHead>
                    <TableHead>级别</TableHead>
                    <TableHead>状态</TableHead>
                    <TableHead>操作</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {query.data.list.map((item) => (
                    <TableRow key={item.id}>
                      <TableCell>
                        <div>
                          <p className="font-semibold text-foreground">{item.name}</p>
                          <p className="mt-1 max-w-md text-sm text-muted-foreground">
                            {item.description ?? summarizeRuleCondition(item)}
                          </p>
                        </div>
                      </TableCell>
                      <TableCell>{item.alert_type_text}</TableCell>
                      <TableCell>
                        <span className={cn("inline-flex rounded-full px-2.5 py-1 text-xs font-semibold", getSystemAlertLevelClassName(item.level))}>
                          {item.level_text}
                        </span>
                      </TableCell>
                      <TableCell>
                        <span className={item.is_enabled ? "inline-flex rounded-full bg-emerald-500/12 px-2.5 py-1 text-xs font-semibold text-emerald-700 dark:text-emerald-300" : "inline-flex rounded-full bg-muted px-2.5 py-1 text-xs font-semibold text-muted-foreground"}>
                          {item.is_enabled ? "已启用" : "已禁用"}
                        </span>
                      </TableCell>
                      <TableCell>
                        <div className="flex flex-wrap gap-2">
                          <Button
                            type="button"
                            size="sm"
                            variant="outline"
                            onClick={() => {
                              setForm(createRuleFormFromItem(item));
                              setDialogState({ open: true, mode: "edit", ruleID: item.id });
                            }}
                          >
                            编辑
                          </Button>
                          <ConfirmDialog
                            title={item.is_enabled ? "确认禁用规则" : "确认启用规则"}
                            description={`规则「${item.name}」${item.is_enabled ? "禁用后将停止检测" : "启用后将恢复检测"}。`}
                            confirmText={item.is_enabled ? "确认禁用" : "确认启用"}
                            confirmVariant={item.is_enabled ? "destructive" : "primary"}
                            trigger={
                              <Button type="button" size="sm" variant="outline">
                                {item.is_enabled ? "禁用" : "启用"}
                              </Button>
                            }
                            onConfirm={() => {
                              mutations.toggle.mutate(
                                {
                                  ruleID: item.id,
                                  payload: { is_enabled: !item.is_enabled },
                                },
                                {
                                  onSuccess: () => {
                                    showToast({ title: `规则已${item.is_enabled ? "禁用" : "启用"}`, variant: "success" });
                                  },
                                  onError: (error) => {
                                    showToast({ title: "状态更新失败", description: error.message, variant: "destructive" });
                                  },
                                },
                              );
                            }}
                          />
                          <ConfirmDialog
                            title="确认删除规则"
                            description={`删除后规则「${item.name}」将不再参与检测。`}
                            confirmText="确认删除"
                            confirmVariant="destructive"
                            trigger={
                              <Button type="button" size="sm" variant="outline" className="text-destructive">
                                <Trash2 className="h-4 w-4" />
                                删除
                              </Button>
                            }
                            onConfirm={() => {
                              mutations.remove.mutate(item.id, {
                                onSuccess: () => {
                                  showToast({ title: "规则已删除", variant: "success" });
                                },
                                onError: (error) => {
                                  showToast({ title: "删除失败", description: error.message, variant: "destructive" });
                                },
                              });
                            }}
                          />
                        </div>
                      </TableCell>
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

      <Dialog open={dialogState.open} onOpenChange={(open) => setDialogState((current) => ({ ...current, open }))}>
        <DialogContent className="max-h-[90vh] max-w-4xl overflow-y-auto">
          <DialogHeader>
            <DialogTitle>{dialogState.mode === "create" ? "创建告警规则" : "编辑告警规则"}</DialogTitle>
            <DialogDescription>
              阈值告警仅允许 <code>cpu_usage</code> / <code>memory_usage</code> / <code>storage_usage</code>；
              服务状态告警仅允许后端定义的标准服务名。
            </DialogDescription>
          </DialogHeader>

          <div className="grid gap-6">
            <div className="grid gap-4 md:grid-cols-2">
              <FormField label="规则名称" required>
                <Input value={form.name} onChange={(event) => setForm((current) => ({ ...current, name: event.target.value }))} />
              </FormField>
              <FormField label="静默期（秒）" required>
                <Input type="number" min={0} value={form.silencePeriod} onChange={(event) => setForm((current) => ({ ...current, silencePeriod: event.target.value }))} />
              </FormField>
            </div>

            <FormField label="描述">
              <Textarea rows={3} value={form.description} onChange={(event) => setForm((current) => ({ ...current, description: event.target.value }))} />
            </FormField>

            <div className="grid gap-4 md:grid-cols-2">
              <FormField label="告警类型" required>
                <Select value={String(form.alertType)} onValueChange={(value) => setForm((current) => ({ ...current, alertType: Number(value) as SystemAlertType }))}>
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {SYSTEM_ALERT_TYPE_OPTIONS.map((item) => (
                      <SelectItem key={item.value} value={String(item.value)}>
                        {item.label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </FormField>
              <FormField label="告警级别" required>
                <Select value={String(form.level)} onValueChange={(value) => setForm((current) => ({ ...current, level: Number(value) as SystemAlertLevel }))}>
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {SYSTEM_ALERT_LEVEL_OPTIONS.map((item) => (
                      <SelectItem key={item.value} value={String(item.value)}>
                        {item.label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </FormField>
            </div>

            {form.alertType === 1 ? (
              <RuleSection title="阈值告警配置">
                <FormField label="监控指标" required>
                  <Select value={form.thresholdMetric} onValueChange={(value) => setForm((current) => ({ ...current, thresholdMetric: value as SystemThresholdMetric }))}>
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      {SYSTEM_THRESHOLD_METRIC_OPTIONS.map((item) => (
                        <SelectItem key={item.value} value={item.value}>
                          {item.label}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </FormField>
                <FormField label="运算符" required>
                  <Select value={form.thresholdOperator} onValueChange={(value) => setForm((current) => ({ ...current, thresholdOperator: value as SystemThresholdOperator }))}>
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      {SYSTEM_THRESHOLD_OPERATOR_OPTIONS.map((item) => (
                        <SelectItem key={item.value} value={item.value}>
                          {item.label}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </FormField>
                <FormField label="阈值" required>
                  <Input type="number" min={0} value={form.thresholdValue} onChange={(event) => setForm((current) => ({ ...current, thresholdValue: event.target.value }))} />
                </FormField>
                <FormField label="持续时间（秒）" required>
                  <Input type="number" min={1} value={form.thresholdDuration} onChange={(event) => setForm((current) => ({ ...current, thresholdDuration: event.target.value }))} />
                </FormField>
              </RuleSection>
            ) : null}

            {form.alertType === 2 ? (
              <RuleSection title="事件告警配置">
                <FormField label="事件来源" required>
                  <Select value={form.eventSource} onValueChange={(value) => setForm((current) => ({ ...current, eventSource: value as "login_logs" }))}>
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      {SYSTEM_EVENT_SOURCE_OPTIONS.map((item) => (
                        <SelectItem key={item.value} value={item.value}>
                          {item.label}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </FormField>
                <FormField label="事件动作（event_filter.action）" required description="使用模块01登录动作标准值。">
                  <Select value={form.eventAction} onValueChange={(value) => setForm((current) => ({ ...current, eventAction: value }))}>
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      {LOGIN_ACTION_OPTIONS.map((item) => (
                        <SelectItem key={item.value} value={String(item.value)}>
                          {item.label}（{item.value}）
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </FormField>
                <FormField label="分组字段" required>
                  <Select value={form.eventGroupBy} onValueChange={(value) => setForm((current) => ({ ...current, eventGroupBy: value as "ip" }))}>
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      {SYSTEM_EVENT_GROUP_BY_OPTIONS.map((item) => (
                        <SelectItem key={item.value} value={item.value}>
                          {item.label}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </FormField>
                <FormField label="次数阈值" required>
                  <Input type="number" min={1} value={form.eventCountThreshold} onChange={(event) => setForm((current) => ({ ...current, eventCountThreshold: event.target.value }))} />
                </FormField>
                <FormField label="时间窗口（秒）" required>
                  <Input type="number" min={1} value={form.eventTimeWindow} onChange={(event) => setForm((current) => ({ ...current, eventTimeWindow: event.target.value }))} />
                </FormField>
              </RuleSection>
            ) : null}

            {form.alertType === 3 ? (
              <RuleSection title="服务状态告警配置">
                <FormField label="服务名称" required>
                  <Select value={form.serviceName} onValueChange={(value) => setForm((current) => ({ ...current, serviceName: value as SystemServiceName }))}>
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      {SYSTEM_SERVICE_NAME_OPTIONS.map((item) => (
                        <SelectItem key={item.value} value={item.value}>
                          {item.label}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </FormField>
                <FormField label="检查地址">
                  <Input value={form.serviceCheckURL} onChange={(event) => setForm((current) => ({ ...current, serviceCheckURL: event.target.value }))} placeholder="例如 localhost:5432" />
                </FormField>
                <FormField label="检查间隔（秒）">
                  <Input type="number" min={1} value={form.serviceCheckInterval} onChange={(event) => setForm((current) => ({ ...current, serviceCheckInterval: event.target.value }))} />
                </FormField>
                <FormField label="失败阈值" required>
                  <Input type="number" min={1} value={form.serviceFailThreshold} onChange={(event) => setForm((current) => ({ ...current, serviceFailThreshold: event.target.value }))} />
                </FormField>
              </RuleSection>
            ) : null}
          </div>

          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => setDialogState((current) => ({ ...current, open: false }))}>
              取消
            </Button>
            <Button
              type="button"
              isLoading={mutations.create.isPending || mutations.update.isPending}
              onClick={() => {
                const validationError = validateRuleForm(form);
                if (validationError !== null) {
                  showToast({ title: "规则配置不合法", description: validationError, variant: "destructive" });
                  return;
                }

                if (dialogState.mode === "create") {
                  mutations.create.mutate(buildCreatePayload(form), {
                    onSuccess: () => {
                      showToast({ title: "规则创建成功", variant: "success" });
                      setDialogState({ open: false, mode: "create" });
                    },
                    onError: (error) => {
                      showToast({ title: "规则创建失败", description: error.message, variant: "destructive" });
                    },
                  });
                  return;
                }

                mutations.update.mutate(
                  {
                    ruleID: dialogState.ruleID ?? "",
                    payload: buildUpdatePayload(form),
                  },
                  {
                    onSuccess: () => {
                      showToast({ title: "规则更新成功", variant: "success" });
                      setDialogState((current) => ({ ...current, open: false }));
                    },
                    onError: (error) => {
                      showToast({ title: "规则更新失败", description: error.message, variant: "destructive" });
                    },
                  },
                );
              }}
            >
              {dialogState.mode === "create" ? "创建规则" : "保存修改"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}

function RuleSection({ title, children }: { title: string; children: ReactNode }) {
  return (
    <section className="grid gap-4 rounded-[1.5rem] border border-border/70 bg-muted/20 p-5 md:grid-cols-2">
      <div className="md:col-span-2">
        <h3 className="font-semibold text-foreground">{title}</h3>
      </div>
      {children}
    </section>
  );
}

function summarizeRuleCondition(item: SystemAlertRuleItem) {
  if (item.alert_type === 1) {
    return `${item.condition.metric} ${item.condition.operator} ${item.condition.value}，持续 ${item.condition.duration} 秒`;
  }
  if (item.alert_type === 2) {
    return `${item.condition.event_source} 按 ${item.condition.group_by} 聚合，${item.condition.time_window} 秒内超过 ${item.condition.count_threshold} 次`;
  }
  return `${item.condition.service_name} 连续失败 ${item.condition.fail_threshold} 次`;
}
