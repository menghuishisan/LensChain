"use client";

// CourseStatisticsPanel.tsx
// 模块03 P-11 课程统计页，展示课程概览、学习进度分布、作业统计和导出入口。

import React from "react";

import { Badge } from "@/components/ui/Badge";
import { Button } from "@/components/ui/Button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/Card";
import { ErrorState } from "@/components/ui/ErrorState";
import { LoadingState } from "@/components/ui/LoadingState";
import { useCourseStatistics, useExportCourseStatisticsMutation } from "@/hooks/useCourses";
import type { ID } from "@/types/api";

// CourseStatisticsPanel 展示课程整体统计和作业层面的分布信息。
export function CourseStatisticsPanel({ courseID }: { courseID: ID }) {
  const stats = useCourseStatistics(courseID);
  const exportMutation = useExportCourseStatisticsMutation(courseID);

  if (stats.overview.isLoading || stats.assignments.isLoading) {
    return <LoadingState />;
  }

  if (stats.overview.isError) {
    return <ErrorState description={stats.overview.error.message} />;
  }

  if (stats.assignments.isError) {
    return <ErrorState description={stats.assignments.error.message} />;
  }

  const overview = stats.overview.data;
  const assignments = stats.assignments.data?.assignments ?? [];

  return (
    <div className="space-y-5">
      <div className="rounded-3xl border border-border/70 bg-[linear-gradient(135deg,hsl(182_34%_14%),hsl(34_52%_28%))] p-6 text-primary-foreground">
        <p className="text-sm text-primary-foreground/75">课程统计</p>
        <h1 className="mt-2 font-display text-3xl font-semibold">查看课程整体数据与作业统计</h1>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>课程概览</CardTitle>
          <CardDescription>学生规模、学习进度、完课率、平均分和活跃度总览。</CardDescription>
        </CardHeader>
        <CardContent className="grid gap-3 md:grid-cols-3 xl:grid-cols-6">
          <MetricCard label="学生数" value={overview?.student_count ?? 0} />
          <MetricCard label="课时数" value={overview?.lesson_count ?? 0} />
          <MetricCard label="作业数" value={overview?.assignment_count ?? 0} />
          <MetricCard label="完课率" value={`${overview?.completion_rate ?? 0}%`} />
          <MetricCard label="平均分" value={overview?.avg_score ?? 0} />
          <MetricCard label="活跃度" value={`${overview?.activity_rate ?? 0}%`} />
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>学习进度分布</CardTitle>
          <CardDescription>按未开始、进行中、已完成划分学生学习状态。</CardDescription>
        </CardHeader>
        <CardContent className="grid gap-3 md:grid-cols-3">
          <ProgressCard label={`未开始 ${overview?.progress_distribution.not_started_rate ?? 0}%`} />
          <ProgressCard label={`进行中 ${overview?.progress_distribution.in_progress_rate ?? 0}%`} />
          <ProgressCard label={`已完成 ${overview?.progress_distribution.completed_rate ?? 0}%`} />
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>作业统计</CardTitle>
          <CardDescription>查看每份作业的提交率、均分和分数分布。</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          {assignments.map((assignment) => (
            <div key={assignment.id} className="rounded-2xl border border-border/70 bg-muted/25 p-4">
              <div className="flex flex-wrap items-center justify-between gap-3">
                <div>
                  <p className="font-medium text-foreground">{assignment.title}</p>
                  <p className="mt-1 text-sm text-muted-foreground">
                    提交 {assignment.submit_count}/{assignment.total_students} · 平均分 {assignment.avg_score} · 最高分 {assignment.max_score} · 最低分 {assignment.min_score}
                  </p>
                </div>
                <Badge variant="outline">提交率 {assignment.submit_rate}%</Badge>
              </div>
              <div className="mt-4">
                <p className="text-sm font-medium text-foreground">分数分布</p>
                <div className="mt-2 flex flex-wrap gap-2">
                  {assignment.score_distribution.map((item) => (
                    <Badge key={`${assignment.id}-${item.range}`} variant="secondary">
                      {item.range}：{item.count}
                    </Badge>
                  ))}
                </div>
              </div>
            </div>
          ))}
          <Button variant="outline" onClick={() => exportMutation.mutate()}>
            导出统计报告
          </Button>
        </CardContent>
      </Card>
    </div>
  );
}

function MetricCard({ label, value }: { label: string; value: string | number }) {
  return (
    <div className="rounded-2xl border border-border/70 bg-muted/25 p-4">
      <p className="text-sm text-muted-foreground">{label}</p>
      <p className="mt-2 text-lg font-semibold">{value}</p>
    </div>
  );
}

function ProgressCard({ label }: { label: string }) {
  return (
    <div className="rounded-2xl border border-border/70 bg-muted/25 p-4">
      <p className="text-base font-semibold">{label}</p>
    </div>
  );
}
