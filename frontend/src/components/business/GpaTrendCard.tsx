"use client";

// GpaTrendCard.tsx
// 模块06 GPA 趋势卡片，展示累计 GPA 和按学期变化趋势。

import { TrendingUp } from "lucide-react";

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/Card";
import { EmptyState } from "@/components/ui/EmptyState";
import { ErrorState } from "@/components/ui/ErrorState";
import { LoadingState } from "@/components/ui/LoadingState";
import { useMyGPA } from "@/hooks/useGrades";
import { formatGPA, formatScore } from "@/lib/format";

/**
 * GpaTrendCard GPA 趋势卡片组件。
 */
export function GpaTrendCard() {
  const gpaQuery = useMyGPA();

  if (gpaQuery.isLoading) {
    return <LoadingState title="正在加载GPA" description="读取累计 GPA 与学期趋势。" />;
  }

  if (gpaQuery.isError) {
    return <ErrorState title="GPA加载失败" description={gpaQuery.error.message} />;
  }

  if (!gpaQuery.data) {
    return <EmptyState title="暂无GPA数据" description="审核通过的成绩会纳入 GPA 计算。" />;
  }

  const gpa = gpaQuery.data;

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <TrendingUp className="h-5 w-5 text-primary" />
          GPA总览
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-5">
        <div className="grid gap-3 md:grid-cols-2">
          <MetricBlock title="累计GPA" value={formatGPA(gpa.cumulative_gpa)} />
          <MetricBlock title="累计学分" value={formatScore(gpa.cumulative_credits)} />
        </div>
        <div className="space-y-3">
          {gpa.semester_list.map((item, index) => (
            <div key={item.semester_id} className="rounded-xl border border-border p-4">
              <div className="flex items-center justify-between gap-3">
                <div>
                  <p className="font-semibold">{item.semester_name}</p>
                  <p className="mt-1 text-xs text-muted-foreground">学分 {formatScore(item.credits)}</p>
                </div>
                <span className="font-display text-2xl font-semibold">{formatGPA(item.gpa)}</span>
              </div>
              <div className="mt-3 h-2 rounded-full bg-muted">
                <div className="h-2 rounded-full bg-primary" style={{ width: `${Math.min(100, (item.gpa / 4) * 100)}%` }} />
              </div>
              <p className="mt-2 text-xs text-muted-foreground">趋势点 {index + 1}/{gpa.semester_list.length}</p>
            </div>
          ))}
        </div>
      </CardContent>
    </Card>
  );
}

function MetricBlock({ title, value }: { title: string; value: string }) {
  return (
    <div className="rounded-xl border border-border bg-muted/30 p-4">
      <p className="text-sm text-muted-foreground">{title}</p>
      <p className="mt-1 font-display text-3xl font-semibold">{value}</p>
    </div>
  );
}
