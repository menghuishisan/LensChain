"use client";

// SystemStatisticsPanel.tsx
// 模块08平台统计组件，展示总览指标、趋势图与学校活跃度排行。

import {
  Activity,
  BookOpen,
  Building2,
  FlaskConical,
  RefreshCcw,
  Trophy,
  Users,
} from "lucide-react";
import type { ReactNode } from "react";
import { useState } from "react";

import { Button } from "@/components/ui/Button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/Card";
import { EmptyState } from "@/components/ui/EmptyState";
import { ErrorState } from "@/components/ui/ErrorState";
import { LoadingState } from "@/components/ui/LoadingState";
import { useSystemSchoolRanking, useSystemStatisticsOverview, useSystemStatisticsTrend } from "@/hooks/useSystemStatistics";
import { formatNumber, formatScore } from "@/lib/format";
import { SYSTEM_STATISTICS_METRIC_OPTIONS, SYSTEM_STATISTICS_PERIOD_OPTIONS } from "@/lib/system";
import { cn } from "@/lib/utils";
import type { SystemStatisticsMetric, SystemStatisticsPeriod, SystemStatisticsTrendPoint } from "@/types/system";

/**
 * SystemStatisticsPanel 平台统计组件。
 */
export function SystemStatisticsPanel() {
  const [metric, setMetric] = useState<SystemStatisticsMetric>("active_users");
  const [period, setPeriod] = useState<SystemStatisticsPeriod>("30d");
  const overviewQuery = useSystemStatisticsOverview();
  const trendQuery = useSystemStatisticsTrend({ metric, period });
  const rankingQuery = useSystemSchoolRanking();

  const isInitialLoading =
    overviewQuery.isLoading &&
    trendQuery.isLoading &&
    rankingQuery.isLoading &&
    overviewQuery.data === undefined &&
    trendQuery.data === undefined &&
    rankingQuery.data === undefined;

  const blockingError =
    (overviewQuery.isError && overviewQuery.data === undefined && overviewQuery.error) ||
    (trendQuery.isError && trendQuery.data === undefined && trendQuery.error) ||
    (rankingQuery.isError && rankingQuery.data === undefined && rankingQuery.error) ||
    null;

  if (isInitialLoading) {
    return <LoadingState title="正在加载数据概览" description="正在整理平台趋势和学校使用情况。" />;
  }

  if (blockingError !== null) {
    return (
      <ErrorState
        title="数据概览加载失败"
        description={blockingError.message}
        action={
          <Button
            type="button"
            variant="outline"
            onClick={() => {
              void Promise.all([overviewQuery.refetch(), trendQuery.refetch(), rankingQuery.refetch()]);
            }}
          >
            <RefreshCcw className="h-4 w-4" />
            重新加载
          </Button>
        }
      />
    );
  }

  const overview = overviewQuery.data;
  const trend = trendQuery.data;
  const ranking = rankingQuery.data?.list ?? [];

  return (
    <div className="space-y-6">
      <Card className="overflow-hidden border-none bg-[linear-gradient(135deg,#0f172a_0%,#10293b_45%,#134e4a_100%)] text-white shadow-[0_32px_90px_-38px_rgba(15,23,42,0.7)]">
        <CardContent className="p-6 md:p-8">
          <div className="flex flex-col gap-5 lg:flex-row lg:items-end lg:justify-between">
            <div className="space-y-3">
              <div className="inline-flex items-center gap-2 rounded-full border border-white/12 bg-white/10 px-3 py-1 text-xs font-semibold uppercase tracking-[0.2em] text-cyan-100/90">
                <Activity className="h-3.5 w-3.5" />
                数据概览
              </div>
              <div>
                <h2 className="font-display text-3xl font-semibold tracking-tight">平台趋势与学校活跃度</h2>
                <p className="mt-2 max-w-2xl text-sm leading-7 text-slate-300">
                  通过平台整体趋势和学校使用情况，帮助你判断运行状态和服务重点。
                </p>
              </div>
            </div>
            <Button
              type="button"
              variant="outline"
              className="border-white/20 bg-white/10 text-white hover:bg-white/15"
              isLoading={overviewQuery.isRefetching || trendQuery.isRefetching || rankingQuery.isRefetching}
              onClick={() => {
                void Promise.all([overviewQuery.refetch(), trendQuery.refetch(), rankingQuery.refetch()]);
              }}
            >
              <RefreshCcw className="h-4 w-4" />
              刷新数据
            </Button>
          </div>

          <div className="mt-6 grid gap-3 md:grid-cols-4 xl:grid-cols-5">
            <HeroChip label="今日活跃" value={`${formatNumber(overview?.today.active_users ?? 0)} 人`} />
            <HeroChip label="今日新增" value={`${formatNumber(overview?.today.new_users ?? 0)} 人`} />
            <HeroChip label="实验启动" value={`${formatNumber(overview?.today.experiments_started ?? 0)} 次`} />
            <HeroChip label="今日请求" value={`${formatNumber(overview?.today.api_requests ?? 0)} 次`} />
            <HeroChip label="趋势周期" value={period.toUpperCase()} />
          </div>
        </CardContent>
      </Card>

      <div className="grid gap-4 lg:grid-cols-5">
        <OverviewCard title="总用户数" value={formatNumber(overview?.total_users ?? 0)} icon={<Users className="h-5 w-5" />} />
        <OverviewCard title="总学校数" value={formatNumber(overview?.total_schools ?? 0)} icon={<Building2 className="h-5 w-5" />} />
        <OverviewCard title="总课程数" value={formatNumber(overview?.total_courses ?? 0)} icon={<BookOpen className="h-5 w-5" />} />
        <OverviewCard title="总实验数" value={formatNumber(overview?.total_experiments ?? 0)} icon={<FlaskConical className="h-5 w-5" />} />
        <OverviewCard title="总竞赛数" value={formatNumber(overview?.total_competitions ?? 0)} icon={<Trophy className="h-5 w-5" />} />
      </div>

      <div className="grid gap-6 xl:grid-cols-[1.15fr_0.85fr]">
        <Card>
          <CardHeader className="space-y-4">
            <div className="flex flex-col gap-3 lg:flex-row lg:items-start lg:justify-between">
              <div className="space-y-1">
                <CardTitle>趋势图</CardTitle>
                <CardDescription>按日聚合的平台趋势数据，默认展示最近 30 天活跃用户。</CardDescription>
              </div>
              <div className="flex flex-wrap gap-2">
                {SYSTEM_STATISTICS_PERIOD_OPTIONS.map((item) => (
                  <Button
                    key={item.value}
                    type="button"
                    size="sm"
                    variant={item.value === period ? "primary" : "outline"}
                    onClick={() => setPeriod(item.value)}
                  >
                    {item.label}
                  </Button>
                ))}
              </div>
            </div>
            <div className="flex flex-wrap gap-2">
              {SYSTEM_STATISTICS_METRIC_OPTIONS.map((item) => (
                <Button
                  key={item.value}
                  type="button"
                  size="sm"
                  variant={item.value === metric ? "secondary" : "ghost"}
                  className={cn(item.value === metric ? "bg-secondary text-secondary-foreground" : "text-muted-foreground")}
                  onClick={() => setMetric(item.value)}
                >
                  {item.label}
                </Button>
              ))}
            </div>
          </CardHeader>
          <CardContent className="space-y-4">
            <TrendSummary points={trend?.data_points ?? []} />
            <TrendChart points={trend?.data_points ?? []} metricLabel={SYSTEM_STATISTICS_METRIC_OPTIONS.find((item) => item.value === metric)?.label ?? ""} />
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>学校活跃度排行</CardTitle>
            <CardDescription>按学校活跃用户与活跃度分数排序，帮助识别使用表现突出的学校。</CardDescription>
          </CardHeader>
          <CardContent>
            {ranking.length === 0 ? (
              <EmptyState title="暂无学校排行数据" description="当前还没有可展示的学校使用情况。" className="min-h-64" />
            ) : (
              <div className="space-y-3">
                {ranking.map((item) => (
                  <div key={item.school_id} className="rounded-2xl border border-border/70 bg-muted/20 p-4">
                    <div className="flex items-start justify-between gap-4">
                      <div className="min-w-0">
                        <div className="flex items-center gap-3">
                          <span className="inline-flex h-8 w-8 items-center justify-center rounded-full bg-primary/12 text-sm font-semibold text-primary">
                            {item.rank}
                          </span>
                          <div className="min-w-0">
                            <p className="truncate font-semibold text-foreground">{item.school_name}</p>
                            <p className="mt-1 text-sm text-muted-foreground">
                              活跃用户 {formatNumber(item.active_users)} / 总用户 {formatNumber(item.total_users)}
                            </p>
                          </div>
                        </div>
                      </div>
                      <div className="text-right">
                        <p className="font-display text-2xl font-semibold text-foreground">{formatScore(item.activity_score, 1)}</p>
                        <p className="mt-1 text-xs uppercase tracking-[0.16em] text-muted-foreground">活跃度分</p>
                      </div>
                    </div>
                    <div className="mt-4 h-2.5 overflow-hidden rounded-full bg-muted">
                      <div
                        className="h-full rounded-full bg-gradient-to-r from-primary to-emerald-400"
                        style={{ width: `${Math.max(0, Math.min(item.activity_score, 100))}%` }}
                      />
                    </div>
                  </div>
                ))}
              </div>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  );
}

function OverviewCard({ title, value, icon }: { title: string; value: string; icon: ReactNode }) {
  return (
    <Card>
      <CardContent className="flex items-start justify-between gap-4 p-5">
        <div>
          <p className="text-sm font-medium text-muted-foreground">{title}</p>
          <p className="mt-3 font-display text-3xl font-semibold text-foreground">{value}</p>
        </div>
        <div className="rounded-2xl bg-primary/10 p-3 text-primary">{icon}</div>
      </CardContent>
    </Card>
  );
}

function HeroChip({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-2xl border border-white/12 bg-white/8 px-4 py-3">
      <p className="text-xs uppercase tracking-[0.18em] text-slate-400">{label}</p>
      <p className="mt-1 text-sm font-semibold text-white">{value}</p>
    </div>
  );
}

function TrendSummary({ points }: { points: SystemStatisticsTrendPoint[] }) {
  if (points.length === 0) {
    return null;
  }

  const values = points.map((point) => point.value);
  const latest = values[values.length - 1] ?? 0;
  const peak = Math.max(...values);
  const average = values.reduce((sum, value) => sum + value, 0) / values.length;

  return (
    <div className="grid gap-3 md:grid-cols-3">
      <SummaryPill label="最新值" value={formatNumber(latest)} />
      <SummaryPill label="峰值" value={formatNumber(peak)} />
      <SummaryPill label="均值" value={formatScore(average, 1)} />
    </div>
  );
}

function SummaryPill({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-2xl border border-border/70 bg-muted/20 px-4 py-3">
      <p className="text-xs uppercase tracking-[0.18em] text-muted-foreground">{label}</p>
      <p className="mt-2 font-display text-2xl font-semibold">{value}</p>
    </div>
  );
}

function TrendChart({
  points,
  metricLabel,
}: {
  points: SystemStatisticsTrendPoint[];
  metricLabel: string;
}) {
  if (points.length === 0) {
    return <EmptyState title="暂无趋势数据" description="当前选择的指标和周期还没有可展示的数据点。" className="min-h-72" />;
  }

  const width = 760;
  const height = 320;
  const paddingX = 24;
  const paddingTop = 24;
  const paddingBottom = 48;
  const values = points.map((point) => point.value);
  const maxValue = Math.max(...values, 1);
  const minValue = Math.min(...values, 0);
  const range = Math.max(maxValue - minValue, 1);
  const stepX = points.length === 1 ? 0 : (width - paddingX * 2) / (points.length - 1);

  const chartPoints = points.map((point, index) => {
    const x = paddingX + stepX * index;
    const y = paddingTop + ((maxValue - point.value) / range) * (height - paddingTop - paddingBottom);
    return { x, y, point };
  });

  const linePath = chartPoints.map((item) => `${item.x},${item.y}`).join(" ");
  const areaPath = `${linePath} ${chartPoints[chartPoints.length - 1]?.x ?? paddingX},${height - paddingBottom} ${chartPoints[0]?.x ?? paddingX},${height - paddingBottom}`;

  return (
    <div className="overflow-hidden rounded-[1.75rem] border border-border bg-[linear-gradient(180deg,rgba(15,23,42,0.03),rgba(15,23,42,0.08))] p-4">
      <div className="mb-3 flex items-center justify-between gap-3">
        <div>
          <p className="text-sm font-semibold text-foreground">{metricLabel}趋势</p>
          <p className="mt-1 text-sm text-muted-foreground">按日聚合展示平台趋势数据。</p>
        </div>
        <span className="rounded-full border border-primary/20 bg-primary/10 px-3 py-1 text-xs font-semibold text-primary">
          {points.length} 个数据点
        </span>
      </div>

      <svg viewBox={`0 0 ${width} ${height}`} className="w-full">
        {[0, 0.25, 0.5, 0.75, 1].map((ratio) => {
          const y = paddingTop + (height - paddingTop - paddingBottom) * ratio;
          const labelValue = Math.round(maxValue - range * ratio);
          return (
            <g key={ratio}>
              <line x1={paddingX} y1={y} x2={width - paddingX} y2={y} stroke="rgba(148, 163, 184, 0.18)" strokeDasharray="4 6" />
              <text x={4} y={y + 4} fontSize="11" fill="rgba(100, 116, 139, 1)">
                {labelValue}
              </text>
            </g>
          );
        })}

        <defs>
          <linearGradient id="system-trend-fill" x1="0" x2="0" y1="0" y2="1">
            <stop offset="0%" stopColor="rgba(13, 148, 136, 0.32)" />
            <stop offset="100%" stopColor="rgba(13, 148, 136, 0.02)" />
          </linearGradient>
        </defs>

        <polygon points={areaPath} fill="url(#system-trend-fill)" />
        <polyline fill="none" stroke="rgb(13 148 136)" strokeWidth="3" strokeLinejoin="round" strokeLinecap="round" points={linePath} />

        {chartPoints.map((item) => (
          <g key={item.point.date}>
            <circle cx={item.x} cy={item.y} r="4" fill="white" stroke="rgb(13 148 136)" strokeWidth="3" />
          </g>
        ))}

        {chartPoints.map((item, index) => {
          const shouldShow = index === 0 || index === chartPoints.length - 1 || index % Math.max(1, Math.floor(chartPoints.length / 6)) === 0;
          if (!shouldShow) {
            return null;
          }

          return (
            <text
              key={`${item.point.date}-label`}
              x={item.x}
              y={height - 16}
              textAnchor="middle"
              fontSize="11"
              fill="rgba(100, 116, 139, 1)"
            >
              {item.point.date.slice(5)}
            </text>
          );
        })}
      </svg>
    </div>
  );
}
