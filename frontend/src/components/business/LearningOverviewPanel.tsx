"use client";

// LearningOverviewPanel.tsx
// 模块06学习概览组件，专门展示 grade 聚合层返回的课程、实验、竞赛和学习时长。

import { BookOpen, Clock3, FlaskConical, Trophy } from "lucide-react";

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/Card";
import { EmptyState } from "@/components/ui/EmptyState";
import { ErrorState } from "@/components/ui/ErrorState";
import { LoadingState } from "@/components/ui/LoadingState";
import { useMyLearningOverview } from "@/hooks/useGrades";
import { formatNumber } from "@/lib/format";

/**
 * LearningOverviewPanel 学习概览组件。
 */
export function LearningOverviewPanel() {
  const overviewQuery = useMyLearningOverview();

  if (overviewQuery.isLoading) {
    return <LoadingState title="正在加载学习概览" description="正在整理你的课程、实验、竞赛和学习时长。" />;
  }

  if (overviewQuery.isError) {
    return <ErrorState title="学习概览加载失败" description={overviewQuery.error.message} />;
  }

  if (!overviewQuery.data) {
    return <EmptyState title="暂无学习概览数据" description="当产生课程、实验或竞赛记录后，这里会自动显示你的学习概况。" />;
  }

  const overview = overviewQuery.data;

  return (
    <div className="grid gap-3 sm:grid-cols-2">
      <OverviewMetric icon={BookOpen} title="课程数" value={formatNumber(overview.course_count)} accent="text-sky-600" />
      <OverviewMetric icon={FlaskConical} title="实验数" value={formatNumber(overview.experiment_count)} accent="text-emerald-600" />
      <OverviewMetric icon={Trophy} title="竞赛数" value={formatNumber(overview.competition_count)} accent="text-amber-600" />
      <OverviewMetric icon={Clock3} title="总学习时长" value={`${overview.total_study_hours} 小时`} accent="text-primary" />
    </div>
  );
}

function OverviewMetric({ icon: Icon, title, value, accent }: { icon: typeof BookOpen; title: string; value: string; accent: string }) {
  return (
    <Card>
      <CardHeader className="pb-3">
        <CardTitle className="flex items-center gap-2 text-base">
          <Icon className={`h-4 w-4 ${accent}`} />
          {title}
        </CardTitle>
      </CardHeader>
      <CardContent>
        <p className="font-display text-3xl font-semibold">{value}</p>
      </CardContent>
    </Card>
  );
}
