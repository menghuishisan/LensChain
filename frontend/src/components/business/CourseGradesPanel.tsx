"use client";

// CourseGradesPanel.tsx
// 模块03 P-10 成绩管理页，面向教师和学生展示权重、成绩汇总和导出入口。

import React, { useEffect, useMemo, useState } from "react";

import { Badge } from "@/components/ui/Badge";
import { Button } from "@/components/ui/Button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/Card";
import { ErrorState } from "@/components/ui/ErrorState";
import { Input } from "@/components/ui/Input";
import { LoadingState } from "@/components/ui/LoadingState";
import { useAssignments, useCourseGrades, useGradeMutations } from "@/hooks/useAssignments";
import type { ID } from "@/types/api";

export interface CourseGradesPanelProps {
  courseID: ID;
  role: "teacher" | "student";
}

// CourseGradesPanel 根据角色展示教师成绩管理页或学生成绩查看页。
export function CourseGradesPanel({ courseID, role }: CourseGradesPanelProps) {
  const grades = useCourseGrades(courseID);
  const assignments = useAssignments(courseID, { page: 1, page_size: 100 });
  const mutations = useGradeMutations(courseID);
  const [weights, setWeights] = useState<Record<string, number>>({});

  useEffect(() => {
    if (grades.config?.data?.items?.length) {
      setWeights(Object.fromEntries(grades.config.data.items.map((item) => [item.assignment_id, item.weight])));
      return;
    }

    if (assignments.data?.list?.length && Object.keys(weights).length === 0) {
      setWeights(Object.fromEntries(assignments.data.list.map((item) => [item.id, 0])));
    }
  }, [assignments.data?.list, grades.config?.data?.items, weights]);

  if (grades.summary.isLoading || grades.mine.isLoading || assignments.isLoading) {
    return <LoadingState />;
  }

  if (grades.summary.isError) {
    return <ErrorState description={grades.summary.error.message} />;
  }

  if (grades.mine.isError) {
    return <ErrorState description={grades.mine.error.message} />;
  }

  if (assignments.isError) {
    return <ErrorState description={assignments.error.message} />;
  }

  const totalWeight = Object.values(weights).reduce((sum, value) => sum + value, 0);

  if (role === "student") {
    const mine = grades.mine.data;
    return (
      <div className="space-y-5">
        <div className="rounded-3xl border border-border/70 bg-[linear-gradient(135deg,hsl(182_34%_14%),hsl(28_46%_28%))] p-6 text-primary-foreground">
          <p className="text-sm text-primary-foreground/75">课程成绩</p>
          <h1 className="mt-2 font-display text-3xl font-semibold">查看加权总分与各作业成绩</h1>
        </div>

        <Card>
          <CardHeader>
            <CardTitle>我的成绩</CardTitle>
            <CardDescription>最终成绩、加权总分和各作业得分一览。</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="grid gap-3 md:grid-cols-3">
              <MetricCard label="最终成绩" value={mine?.final_score ?? "暂无"} />
              <MetricCard label="加权总分" value={mine?.weighted_total ?? "暂无"} />
              <MetricCard label="成绩状态" value={mine?.is_adjusted ? "已调整" : "自动计算"} />
            </div>
            <div className="grid gap-3">
              {(assignments.data?.list ?? []).map((assignment) => (
                <div key={assignment.id} className="flex items-center justify-between rounded-2xl border border-border/70 bg-muted/25 p-4 text-sm">
                  <div>
                    <p className="font-medium text-foreground">{assignment.title}</p>
                    <p className="mt-1 text-muted-foreground">总分 {assignment.total_score}</p>
                  </div>
                  <span className="text-base font-semibold">{mine?.scores?.[assignment.id] ?? "-"}</span>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      </div>
    );
  }

  const students = grades.summary.data?.students ?? [];

  return (
    <div className="space-y-5">
      <div className="rounded-3xl border border-border/70 bg-[linear-gradient(135deg,hsl(182_34%_14%),hsl(28_46%_28%))] p-6 text-primary-foreground">
        <p className="text-sm text-primary-foreground/75">成绩管理</p>
        <h1 className="mt-2 font-display text-3xl font-semibold">配置权重、查看汇总并导出成绩单</h1>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>成绩权重配置</CardTitle>
          <CardDescription>各作业权重总和必须为 100%，保存后系统按最新配置重新计算总成绩。</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid gap-3">
            {(assignments.data?.list ?? []).map((assignment) => (
              <div key={assignment.id} className="grid gap-3 rounded-2xl border border-border/70 bg-muted/25 p-4 md:grid-cols-[1fr_8rem_auto]">
                <div>
                  <p className="font-medium text-foreground">{assignment.title}</p>
                  <p className="mt-1 text-sm text-muted-foreground">总分 {assignment.total_score}</p>
                </div>
                <Input
                  aria-label={`${assignment.title}权重`}
                  type="number"
                  value={weights[assignment.id] ?? 0}
                  onChange={(event) => setWeights((current) => ({ ...current, [assignment.id]: Number(event.target.value) }))}
                />
                <div className="flex items-center text-sm text-muted-foreground">%</div>
              </div>
            ))}
          </div>
          <div className="flex flex-wrap items-center gap-3">
            <Badge variant={totalWeight === 100 ? "success" : "outline"}>当前总权重 {totalWeight}%</Badge>
            {totalWeight !== 100 ? <span className="text-sm text-destructive">权重总和必须为 100%</span> : null}
          </div>
          <div className="flex flex-wrap gap-3">
            <Button
              disabled={totalWeight !== 100}
              onClick={() =>
                mutations.setConfig.mutate((assignments.data?.list ?? []).map((item) => ({ assignment_id: item.id, name: item.title, weight: weights[item.id] ?? 0 })))
              }
            >
              保存权重
            </Button>
            <Button variant="outline" onClick={() => mutations.exportGrades.mutate()}>
              导出成绩单
            </Button>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>成绩汇总</CardTitle>
          <CardDescription>查看各学生加权总分、最终成绩和调整状态。</CardDescription>
        </CardHeader>
        <CardContent className="space-y-3">
          {students.map((student) => (
            <div key={student.student_id} className="grid gap-3 rounded-2xl border border-border/70 bg-muted/25 p-4 md:grid-cols-[1.2fr_1fr_1fr_auto]">
              <div>
                <p className="font-medium text-foreground">{student.student_name}</p>
                <p className="mt-1 text-sm text-muted-foreground">{student.student_no ?? "学号未提供"}</p>
              </div>
              <MetricInline label="加权总分" value={student.weighted_total} />
              <MetricInline label="最终成绩" value={student.final_score} />
              {student.is_adjusted ? <Badge variant="outline">已调整</Badge> : <Badge variant="success">自动计算</Badge>}
            </div>
          ))}
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

function MetricInline({ label, value }: { label: string; value: string | number }) {
  return (
    <div>
      <p className="text-sm text-muted-foreground">{label}</p>
      <p className="mt-1 text-base font-semibold">{value}</p>
    </div>
  );
}
