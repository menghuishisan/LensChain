"use client";

// CourseOverviewPanel.tsx
// 模块03 P-03 课程管理主页，聚合课程概览、快捷导航和最近更新信息。

import React from "react";
import Link from "next/link";

import { Badge } from "@/components/ui/Badge";
import { buttonClassName, Button } from "@/components/ui/Button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/Card";
import { ErrorState } from "@/components/ui/ErrorState";
import { LoadingState } from "@/components/ui/LoadingState";
import { formatDateTime, formatHours, formatPercent, formatScore } from "@/lib/format";
import { useCourse, useCourseLifecycleMutations, useCourseStatistics } from "@/hooks/useCourses";
import type { ID } from "@/types/api";

interface StatCardProps {
  title: string;
  value: string | number;
  hint?: string;
}

function StatCard({ title, value, hint }: StatCardProps) {
  return (
    <Card>
      <CardHeader>
        <CardDescription>{title}</CardDescription>
        <CardTitle>{value}</CardTitle>
        {hint ? <p className="text-xs text-muted-foreground">{hint}</p> : null}
      </CardHeader>
    </Card>
  );
}

// CourseOverviewPanel 展示课程管理主页的概览、导航入口和最近更新摘要。
export function CourseOverviewPanel({ courseID }: { courseID: ID }) {
  const query = useCourse(courseID);
  const lifecycle = useCourseLifecycleMutations(courseID);
  const stats = useCourseStatistics(courseID);

  if (query.isLoading || stats.overview.isLoading) {
    return <LoadingState variant="hero" />;
  }

  if (query.isError) {
    return <ErrorState description={query.error.message} />;
  }

  if (stats.overview.isError) {
    return <ErrorState description={stats.overview.error.message} />;
  }

  if (!query.data || !stats.overview.data) {
    return null;
  }

  const course = query.data;
  const overview = stats.overview.data;
  const totalLessons = overview.lesson_count ?? course.chapters.reduce((count, chapter) => count + chapter.lessons.length, 0);
  const totalAssignments = overview.assignment_count ?? 0;

  return (
    <div className="space-y-6">
      <div className="rounded-3xl border border-border/70 bg-[radial-gradient(circle_at_top_left,hsl(var(--primary)/0.15),transparent_22rem),linear-gradient(135deg,hsl(220_40%_8%/0.88),hsl(var(--primary)/0.35))] p-6 text-primary-foreground shadow-[0_20px_60px_rgba(0,0,0,0.18)]">
        <div className="flex flex-wrap items-start justify-between gap-4">
          <div className="space-y-3">
            <Link className="inline-flex items-center text-sm text-primary-foreground/80 transition hover:text-primary-foreground" href="/teacher/courses">
              ← 返回课程列表
            </Link>
            <div className="flex flex-wrap items-center gap-3">
              <h1 className="font-display text-3xl font-semibold">{course.title}</h1>
              <Badge variant="secondary">{course.status_text}</Badge>
            </div>
            <p className="max-w-3xl text-sm leading-6 text-primary-foreground/80">
              {course.description?.trim() || "当前课程尚未补充课程简介，可在课程设置中继续完善。"}
            </p>
            <div className="flex flex-wrap gap-2 text-xs text-primary-foreground/80">
              <span>邀请码：{course.invite_code ?? "仅教师可见"}</span>
              <span>教师：{course.teacher_name}</span>
              <span>主题：{course.topic}</span>
              <span>最近更新：{formatDateTime(course.updated_at)}</span>
            </div>
          </div>
          <div className="flex flex-wrap gap-2">
            <Button variant="secondary" onClick={() => lifecycle.publish.mutate()}>
              发布课程
            </Button>
            <Button variant="outline" onClick={() => lifecycle.end.mutate()}>
              结束课程
            </Button>
            <Button variant="outline" onClick={() => lifecycle.archive.mutate()}>
              归档课程
            </Button>
            <Button variant="outline" onClick={() => lifecycle.refreshInvite.mutate()}>
              刷新邀请码
            </Button>
          </div>
        </div>
      </div>

      <div className="grid gap-6 lg:grid-cols-[14rem_minmax(0,1fr)]">
        <nav className="space-y-4">
          <div>
            <p className="mb-2 px-1 text-xs font-semibold uppercase tracking-wider text-muted-foreground">课程管理</p>
            <div className="grid gap-1.5">
              <Link className={buttonClassName({ variant: "outline" })} href={`/teacher/courses/${courseID}`}>
                概览
              </Link>
              <Link className={buttonClassName({ variant: "outline" })} href={`/teacher/courses/${courseID}/content`}>
                内容
              </Link>
              <Link className={buttonClassName({ variant: "outline" })} href={`/teacher/courses/${courseID}/students`}>
                学生
              </Link>
              <Link className={buttonClassName({ variant: "outline" })} href={`/teacher/courses/${courseID}/settings`}>
                设置
              </Link>
            </div>
          </div>
          <div>
            <p className="mb-2 px-1 text-xs font-semibold uppercase tracking-wider text-muted-foreground">教学工具</p>
            <div className="grid gap-1.5">
              <Link className={buttonClassName({ variant: "outline" })} href={`/teacher/courses/${courseID}/assignments`}>
                作业
              </Link>
              <Link className={buttonClassName({ variant: "outline" })} href={`/courses/${courseID}/discussions`}>
                讨论
              </Link>
              <Link className={buttonClassName({ variant: "outline" })} href={`/courses/${courseID}/announcements`}>
                公告
              </Link>
              <Link className={buttonClassName({ variant: "outline" })} href={`/courses/${courseID}/evaluations`}>
                评价
              </Link>
            </div>
          </div>
          <div>
            <p className="mb-2 px-1 text-xs font-semibold uppercase tracking-wider text-muted-foreground">数据分析</p>
            <div className="grid gap-1.5">
              <Link className={buttonClassName({ variant: "outline" })} href={`/teacher/courses/${courseID}/grades`}>
                成绩
              </Link>
              <Link className={buttonClassName({ variant: "outline" })} href={`/teacher/courses/${courseID}/statistics`}>
                统计
              </Link>
              <Link className={buttonClassName({ variant: "outline" })} href={`/teacher/courses/${courseID}/experiment-monitor`}>
                实验监控
              </Link>
              <Link className={buttonClassName({ variant: "outline" })} href={`/teacher/courses/${courseID}/experiment-statistics`}>
                实验统计
              </Link>
            </div>
          </div>
        </nav>

        <div className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle>课程概览</CardTitle>
              <CardDescription>课程运行状态、教学规模和关键指标。</CardDescription>
            </CardHeader>
            <CardContent className="grid gap-4 md:grid-cols-3 xl:grid-cols-6">
              <StatCard title="学生" value={course.student_count} hint="当前已加入课程人数" />
              <StatCard title="课时" value={totalLessons} hint="按章节树统计课时总数" />
              <StatCard title="作业" value={totalAssignments} hint="课程当前作业数量" />
              <StatCard title="完课率" value={formatPercent(overview.completion_rate)} hint="已完成课时的学生占比" />
              <StatCard title="平均分" value={formatScore(overview.avg_score)} hint="课程当前加权平均分" />
              <StatCard title="活跃度" value={formatPercent(overview.activity_rate)} hint="最近学习与互动活跃水平" />
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>最近更新</CardTitle>
              <CardDescription>基于现有课程详情与统计接口整理的最新课程状态。</CardDescription>
            </CardHeader>
            <CardContent className="space-y-3 text-sm text-muted-foreground">
              <div className="rounded-2xl border border-border/70 bg-muted/30 p-4">
                <p className="font-medium text-foreground">最近一次课程更新</p>
                <p className="mt-1">课程信息最后更新于 {formatDateTime(course.updated_at)}，当前状态为“{course.status_text}”。</p>
              </div>
              <div className="rounded-2xl border border-border/70 bg-muted/30 p-4">
                <p className="font-medium text-foreground">最新教学规模</p>
                <p className="mt-1">当前共 {course.student_count} 名学生，已配置 {totalLessons} 个课时和 {totalAssignments} 份作业。</p>
              </div>
              <div className="rounded-2xl border border-border/70 bg-muted/30 p-4">
                <p className="font-medium text-foreground">最近进度摘要</p>
                <p className="mt-1">平均学习进度 {formatPercent(overview.avg_progress)} ，累计学习时长 {formatHours(overview.total_study_hours)} ，完课率 {formatPercent(overview.completion_rate)} 。</p>
              </div>
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  );
}
