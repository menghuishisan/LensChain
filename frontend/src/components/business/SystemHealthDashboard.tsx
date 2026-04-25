"use client";

// SystemHealthDashboard.tsx
// 模块08运维仪表盘组件，展示健康状态、资源使用、实时指标与最近告警。

import {
  Activity,
  AlertTriangle,
  ArrowRight,
  Boxes,
  DatabaseZap,
  Gauge,
  RefreshCcw,
  ServerCog,
  Users,
} from "lucide-react";
import Link from "next/link";
import type { ReactNode } from "react";

import { Button } from "@/components/ui/Button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/Card";
import { EmptyState } from "@/components/ui/EmptyState";
import { ErrorState } from "@/components/ui/ErrorState";
import { LoadingState } from "@/components/ui/LoadingState";
import { useSystemDashboard } from "@/hooks/useSystemDashboard";
import { formatDateTime, formatNumber, formatScore } from "@/lib/format";
import {
  getSystemAlertLevelClassName,
  getSystemStatusClassName,
  getSystemUsageBarClassName,
  getSystemUsageTone,
} from "@/lib/system";
import { cn } from "@/lib/utils";

/**
 * SystemHealthDashboard 运维仪表盘组件。
 */
export function SystemHealthDashboard() {
  const { healthQuery, resourcesQuery, realtimeQuery } = useSystemDashboard();
  const isInitialLoading =
    healthQuery.isLoading &&
    resourcesQuery.isLoading &&
    realtimeQuery.isLoading &&
    healthQuery.data === undefined &&
    resourcesQuery.data === undefined &&
    realtimeQuery.data === undefined;

  const blockingError =
    (healthQuery.isError && healthQuery.data === undefined && healthQuery.error) ||
    (resourcesQuery.isError && resourcesQuery.data === undefined && resourcesQuery.error) ||
    (realtimeQuery.isError && realtimeQuery.data === undefined && realtimeQuery.error) ||
    null;

  if (isInitialLoading) {
    return <LoadingState title="正在加载运行总览" description="正在汇总平台状态、资源使用和运行提醒。" />;
  }

  if (blockingError !== null) {
    return (
      <ErrorState
        title="运行总览加载失败"
        description={blockingError.message}
        action={
          <Button
            type="button"
            variant="outline"
            onClick={() => {
              void Promise.all([healthQuery.refetch(), resourcesQuery.refetch(), realtimeQuery.refetch()]);
            }}
          >
            <RefreshCcw className="h-4 w-4" />
            重新加载
          </Button>
        }
      />
    );
  }

  const health = healthQuery.data;
  const resources = resourcesQuery.data;
  const realtime = realtimeQuery.data;
  const isRefreshing = healthQuery.isRefetching || resourcesQuery.isRefetching || realtimeQuery.isRefetching;

  return (
    <div className="space-y-6">
      <Card className="relative overflow-hidden border-none bg-slate-950 text-white shadow-[0_30px_80px_-35px_rgba(15,23,42,0.7)]">
        <div className="absolute inset-0 bg-[radial-gradient(circle_at_top_left,rgba(45,212,191,0.22),transparent_38%),radial-gradient(circle_at_bottom_right,rgba(249,115,22,0.18),transparent_34%)]" />
        <CardContent className="relative p-6 md:p-8">
          <div className="flex flex-col gap-5 lg:flex-row lg:items-end lg:justify-between">
            <div className="max-w-2xl space-y-3">
              <div className="inline-flex items-center gap-2 rounded-full border border-white/12 bg-white/6 px-3 py-1 text-xs font-semibold tracking-[0.2em] text-emerald-200/85 uppercase">
                <Gauge className="h-3.5 w-3.5" />
                运行总览
              </div>
              <div>
                <h2 className="font-display text-3xl font-semibold tracking-tight">平台健康总览</h2>
                <p className="mt-2 text-sm leading-7 text-slate-300">
                  统一查看核心服务、资源占用和平台提醒，帮助快速判断当前运行情况。
                </p>
              </div>
            </div>
            <div className="flex flex-wrap items-center gap-3">
              <div className="rounded-2xl border border-white/12 bg-white/8 px-4 py-3">
                <p className="text-xs uppercase tracking-[0.18em] text-slate-400">最后刷新</p>
                <p className="mt-1 text-sm font-semibold text-white">
                  {formatDateTime(health?.last_check_at ?? new Date())}
                </p>
              </div>
              <Button
                type="button"
                variant="outline"
                className="border-white/20 bg-white/10 text-white hover:bg-white/15"
                isLoading={isRefreshing}
                onClick={() => {
                  void Promise.all([healthQuery.refetch(), resourcesQuery.refetch(), realtimeQuery.refetch()]);
                }}
              >
                <RefreshCcw className="h-4 w-4" />
                立即刷新
              </Button>
            </div>
          </div>
          <div className="mt-6 flex flex-wrap gap-3">
            <SystemStatusPill label="整体状态" value={health?.overall_status ?? "unknown"} />
            <SystemStatusPill label="待处理告警" value={`${realtime?.pending_alerts ?? 0} 条`} />
            <SystemStatusPill label="在线用户" value={`${formatNumber(realtime?.online_users ?? 0)} 人`} />
          </div>
        </CardContent>
      </Card>

      <div className="grid gap-4 lg:grid-cols-4">
        <DashboardMetricCard
          title="在线用户"
          value={formatNumber(realtime?.online_users ?? 0)}
          description="当前会话缓存推导的实时在线用户数"
          icon={<Users className="h-5 w-5" />}
        />
        <DashboardMetricCard
          title="活跃实验"
          value={formatNumber(realtime?.active_experiments ?? 0)}
          description="当前正在进行的实验数量"
          icon={<DatabaseZap className="h-5 w-5" />}
        />
        <DashboardMetricCard
          title="进行中竞赛"
          value={formatNumber(realtime?.active_competitions ?? 0)}
          description="当前处于运行态的竞赛数量"
          icon={<Boxes className="h-5 w-5" />}
        />
        <DashboardMetricCard
          title="待处理告警"
          value={formatNumber(realtime?.pending_alerts ?? 0)}
          description="来自告警事件列表中的待处理总数"
          accent="danger"
          icon={<AlertTriangle className="h-5 w-5" />}
        />
      </div>

      <div className="grid gap-6 xl:grid-cols-[1.2fr_0.8fr]">
        <Card>
          <CardHeader className="flex flex-row items-start justify-between gap-4">
            <div className="space-y-1">
              <CardTitle>服务状态</CardTitle>
              <CardDescription>绿色表示正常，黄色表示未知，红色表示不可达或异常。</CardDescription>
            </div>
          </CardHeader>
          <CardContent className="grid gap-3 md:grid-cols-2">
            {(health?.services ?? []).map((service) => (
              <div
                key={service.name}
                className="rounded-2xl border border-border/70 bg-muted/25 p-4 transition hover:border-border hover:bg-muted/35"
              >
                <div className="flex items-start justify-between gap-3">
                  <div>
                    <p className="font-semibold text-foreground">{service.name}</p>
                    <p className="mt-1 text-xs uppercase tracking-[0.16em] text-muted-foreground">
                      响应 {formatNumber(service.latency_ms)} ms
                    </p>
                  </div>
                  <span
                    className={cn(
                      "inline-flex rounded-full px-2.5 py-1 text-xs font-semibold",
                      getSystemStatusClassName(service.status),
                    )}
                  >
                    {service.status === "healthy" ? "正常" : service.status === "unknown" ? "未知" : "异常"}
                  </span>
                </div>
                <div className="mt-4 grid gap-2 text-sm text-muted-foreground">
                  {service.connections !== undefined ? (
                    <p>连接 {service.connections.active}/{service.connections.max}</p>
                  ) : null}
                  {service.nodes !== undefined ? <p>节点 {service.nodes}</p> : null}
                  {service.pods_running !== undefined ? <p>运行 Pod {service.pods_running}</p> : null}
                  {service.memory_used_mb !== undefined ? <p>内存 {formatNumber(service.memory_used_mb)} MB</p> : null}
                  {service.messages_in_queue !== undefined ? <p>消息积压 {formatNumber(service.messages_in_queue)}</p> : null}
                  {service.storage_used_gb !== undefined ? <p>存储 {formatScore(service.storage_used_gb, 1)} GB</p> : null}
                  {service.uptime !== undefined ? <p>运行时长 {service.uptime}</p> : null}
                </div>
              </div>
            ))}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>资源使用</CardTitle>
            <CardDescription>按 CPU、内存、存储和集群视角观察平台资源负载。</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <ResourceUsageCard
              title="CPU"
              percent={resources?.cpu.usage_percent ?? 0}
              detail={`${formatScore(resources?.cpu.cores_used ?? 0, 1)} / ${formatNumber(resources?.cpu.cores_total ?? 0)} cores`}
            />
            <ResourceUsageCard
              title="内存"
              percent={resources?.memory.usage_percent ?? 0}
              detail={`${formatNumber(resources?.memory.used_gb ?? 0)} / ${formatNumber(resources?.memory.total_gb ?? 0)} GB`}
            />
            <ResourceUsageCard
              title="存储"
              percent={resources?.storage.usage_percent ?? 0}
              detail={`${formatNumber(resources?.storage.used_gb ?? 0)} / ${formatNumber(resources?.storage.total_gb ?? 0)} GB`}
            />
            <div className="rounded-2xl border border-border/70 bg-slate-950 px-4 py-4 text-white">
              <div className="flex items-center justify-between gap-3">
                <div>
                  <p className="text-sm font-semibold">K8s 集群</p>
                  <p className="mt-1 text-xs uppercase tracking-[0.18em] text-slate-400">集群摘要</p>
                </div>
                <ServerCog className="h-5 w-5 text-teal-300" />
              </div>
              <div className="mt-4 grid grid-cols-2 gap-3 text-sm">
                <div className="rounded-xl bg-white/8 p-3">
                  <p className="text-slate-400">节点</p>
                  <p className="mt-1 font-semibold">{formatNumber(resources?.k8s.nodes ?? 0)}</p>
                </div>
                <div className="rounded-xl bg-white/8 p-3">
                  <p className="text-slate-400">总 Pod</p>
                  <p className="mt-1 font-semibold">{formatNumber(resources?.k8s.pods_total ?? 0)}</p>
                </div>
                <div className="rounded-xl bg-white/8 p-3">
                  <p className="text-slate-400">运行中</p>
                  <p className="mt-1 font-semibold">{formatNumber(resources?.k8s.pods_running ?? 0)}</p>
                </div>
                <div className="rounded-xl bg-white/8 p-3">
                  <p className="text-slate-400">Pending</p>
                  <p className="mt-1 font-semibold">{formatNumber(resources?.k8s.pods_pending ?? 0)}</p>
                </div>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>

      <Card>
        <CardHeader className="flex flex-row items-start justify-between gap-4">
          <div className="space-y-1">
            <CardTitle>最近告警</CardTitle>
            <CardDescription>展示最近需要关注的运行提醒，可直接进入详情页继续处理。</CardDescription>
          </div>
          <Link href="/super/system/alert-events" className="inline-flex items-center gap-1 text-sm font-semibold text-primary">
            查看全部
            <ArrowRight className="h-4 w-4" />
          </Link>
        </CardHeader>
        <CardContent>
          {(realtime?.recent_alerts ?? []).length === 0 ? (
            <EmptyState title="暂无最近告警" description="当前没有新的待处理告警，平台运行平稳。" className="min-h-44" />
          ) : (
            <div className="space-y-3">
              {(realtime?.recent_alerts ?? []).map((alert) => (
                <Link
                  key={alert.id}
                  href={`/super/system/alert-events?selected=${alert.id}`}
                  className="flex flex-col gap-3 rounded-2xl border border-border/70 bg-muted/20 p-4 transition hover:border-primary/25 hover:bg-muted/35 md:flex-row md:items-center md:justify-between"
                >
                  <div className="min-w-0">
                    <div className="flex flex-wrap items-center gap-2">
                      <span
                        className={cn(
                          "inline-flex rounded-full px-2.5 py-1 text-xs font-semibold",
                          getSystemAlertLevelClassName(alert.level),
                        )}
                      >
                        {alert.level === 1 ? "信息" : alert.level === 2 ? "警告" : alert.level === 3 ? "严重" : "紧急"}
                      </span>
                      <p className="truncate font-semibold text-foreground">{alert.title}</p>
                    </div>
                    <p className="mt-2 text-sm text-muted-foreground">
                      触发时间 {formatDateTime(alert.triggered_at)}
                    </p>
                  </div>
                  <ArrowRight className="h-4 w-4 shrink-0 text-muted-foreground" />
                </Link>
              ))}
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}

function DashboardMetricCard({
  title,
  value,
  description,
  icon,
  accent = "default",
}: {
  title: string;
  value: string;
  description: string;
  icon: ReactNode;
  accent?: "default" | "danger";
}) {
  return (
    <Card className={cn("border-border/80", accent === "danger" ? "border-destructive/25 bg-destructive/5" : "")}>
      <CardContent className="p-5">
        <div className="flex items-start justify-between gap-4">
          <div>
            <p className="text-sm font-medium text-muted-foreground">{title}</p>
            <p className="mt-3 font-display text-3xl font-semibold">{value}</p>
            <p className="mt-2 text-sm leading-6 text-muted-foreground">{description}</p>
          </div>
          <div
            className={cn(
              "rounded-2xl p-3",
              accent === "danger" ? "bg-destructive/10 text-destructive" : "bg-primary/10 text-primary",
            )}
          >
            {icon}
          </div>
        </div>
      </CardContent>
    </Card>
  );
}

function ResourceUsageCard({ title, percent, detail }: { title: string; percent: number; detail: string }) {
  const tone = getSystemUsageTone(percent);

  return (
    <div className="rounded-2xl border border-border/70 bg-muted/20 p-4">
      <div className="flex items-center justify-between gap-3">
        <div>
          <p className="text-sm font-semibold text-foreground">{title}</p>
          <p className="mt-1 text-xs uppercase tracking-[0.18em] text-muted-foreground">
            {tone === "danger" ? "高风险" : tone === "warning" ? "需要关注" : "运行稳定"}
          </p>
        </div>
        <p className="font-display text-2xl font-semibold">{formatScore(percent, 1)}%</p>
      </div>
      <div className="mt-4 h-3 overflow-hidden rounded-full bg-muted">
        <div
          className={cn("h-full rounded-full transition-[width]", getSystemUsageBarClassName(percent))}
          style={{ width: `${Math.max(0, Math.min(percent, 100))}%` }}
        />
      </div>
      <p className="mt-3 text-sm text-muted-foreground">{detail}</p>
    </div>
  );
}

function SystemStatusPill({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-full border border-white/12 bg-white/8 px-4 py-2">
      <span className="text-xs uppercase tracking-[0.18em] text-slate-400">{label}</span>
      <span className="ml-3 text-sm font-semibold text-white">{value}</span>
    </div>
  );
}
