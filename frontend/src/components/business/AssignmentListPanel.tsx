"use client";

// AssignmentListPanel.tsx
// 模块03 P-06/P-24 作业列表页，区分教师端管理视角和学生端作答视角。

import React, { useMemo, useState } from "react";
import Link from "next/link";

import { Badge } from "@/components/ui/Badge";
import { Button, buttonClassName } from "@/components/ui/Button";
import { ConfirmDialog } from "@/components/ui/ConfirmDialog";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/Card";
import { EmptyState } from "@/components/ui/EmptyState";
import { ErrorState } from "@/components/ui/ErrorState";
import { LoadingState } from "@/components/ui/LoadingState";
import { Pagination } from "@/components/ui/Pagination";
import { useAssignment, useAssignmentMutations, useAssignments, useMySubmissions, useSubmissions } from "@/hooks/useAssignments";
import { formatDateTime } from "@/lib/format";
import type { ID } from "@/types/api";
import type { AssignmentListItem, MySubmissionsResponse, SubmissionListResponse } from "@/types/courseAssignment";

export interface AssignmentListPanelProps {
  courseID: ID;
  role: "teacher" | "student";
}

// AssignmentListPanel 根据角色展示教师作业管理页或学生作业列表页。
export function AssignmentListPanel({ courseID, role }: AssignmentListPanelProps) {
  const [page, setPage] = useState(1);
  const query = useAssignments(courseID, { page, page_size: 20 });

  if (query.isLoading) {
    return <LoadingState variant="hero" />;
  }

  if (query.isError) {
    return <ErrorState description={query.error.message} />;
  }

  const list = query.data?.list ?? [];

  return (
    <div className="space-y-5">
      <div className="rounded-3xl border border-border/70 bg-[linear-gradient(135deg,hsl(var(--primary)/0.85),hsl(var(--primary)/0.65))] p-6 text-primary-foreground">
        <div className="flex flex-wrap items-start justify-between gap-4">
          <div>
            <p className="text-sm text-primary-foreground/75">{role === "teacher" ? "课程作业管理" : "课程作业列表"}</p>
            <h1 className="mt-2 font-display text-3xl font-semibold">{role === "teacher" ? "课程作业管理" : "课程作业列表"}</h1>
            <p className="mt-3 max-w-3xl text-sm leading-6 text-primary-foreground/80">
              {role === "teacher"
                ? "查看作业发布状态、提交概况与可执行操作，快速进入编辑或批改链路。"
                : "查看当前课程下可作答作业、截止状态、剩余提交次数和最近一次提交结果。"}
            </p>
          </div>
          {role === "teacher" ? (
            <Link className={buttonClassName()} href={`/teacher/assignments/new/edit?course_id=${courseID}`}>
              创建作业
            </Link>
          ) : null}
        </div>
      </div>

      {list.length === 0 ? (
        <EmptyState
          title={role === "teacher" ? "暂无作业" : "暂无可见作业"}
          description={role === "teacher" ? "创建第一份作业后可继续编辑题目、发布并查看提交情况。" : "教师发布作业后，这里会显示可作答或已结束的课程作业。"}
        />
      ) : (
        <div className="grid gap-4">
          {list.map((item) =>
            role === "teacher" ? (
              <TeacherAssignmentCard key={item.id} assignment={item} courseID={courseID} />
            ) : (
              <StudentAssignmentCard key={item.id} assignment={item} />
            ),
          )}
        </div>
      )}

      {query.data?.pagination ? (
        <Pagination page={query.data.pagination.page} totalPages={query.data.pagination.total_pages} total={query.data.pagination.total} onPageChange={setPage} />
      ) : null}
    </div>
  );
}

interface TeacherAssignmentCardProps {
  assignment: AssignmentListItem;
  courseID: ID;
}

function TeacherAssignmentCard({ assignment, courseID }: TeacherAssignmentCardProps) {
  const detail = useAssignment(assignment.id);
  const submissions = useSubmissions(assignment.id, { page: 1, page_size: 20 });
  const mutations = useAssignmentMutations(courseID, assignment.id);
  const hasSubmission = (submissions.data?.list?.length ?? 0) > 0;
  const [showSubmissions, setShowSubmissions] = useState(false);
  const questionCount = detail.data?.questions.length ?? 0;

  return (
    <Card>
      <CardHeader>
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div className="space-y-2">
            <CardTitle>{assignment.title}</CardTitle>
            <CardDescription>
              {assignment.assignment_type_text} · 截止 {assignment.deadline_at ? formatDateTime(assignment.deadline_at) : "未设置"}
            </CardDescription>
          </div>
          <div className="flex flex-wrap gap-2">
            <Badge variant={assignment.is_published ? "success" : "outline"}>{assignment.is_published ? "已发布" : "未发布"}</Badge>
            <Badge variant="outline">{assignment.total_score} 分</Badge>
          </div>
        </div>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="grid gap-3 md:grid-cols-4">
          <MetricCard label="提交概况" value={`${assignment.submit_count}/${assignment.total_students}`} hint="已提交 / 总学生数" />
          <MetricCard label="题目数量" value={detail.isLoading ? "加载中" : `${questionCount}`} hint="发布前至少 1 题" />
          <MetricCard label="最近批改" value={hasSubmission ? "已有提交" : "暂无提交"} hint={hasSubmission ? `已有 ${submissions.data?.pagination.total ?? submissions.data?.list.length ?? 0} 份提交记录` : "当前还没有学生提交"} />
          <MetricCard label="操作限制" value={hasSubmission ? "不可删除" : "可删除"} hint={hasSubmission ? "已有提交记录的作业不可删除" : "草稿或未有提交时可删除"} />
        </div>

        <div className="rounded-2xl border border-border/70 bg-muted/25 p-4 text-sm text-muted-foreground">
          {assignment.is_published ? "作业已发布，学生端可见并可进入作答页。" : "作业当前仍为草稿，教师可继续补题或直接发布。"}
          {questionCount === 0 ? " 当前题目数为 0，需至少添加一道题后才能发布。" : ` 当前已配置 ${questionCount} 道题目。`}
        </div>

        <div className="flex flex-wrap gap-3">
          <Link className={buttonClassName({ variant: "outline" })} href={`/teacher/assignments/${assignment.id}/edit?course_id=${courseID}`}>
            编辑作业
          </Link>
          {!assignment.is_published ? (
            <Button disabled={questionCount === 0} onClick={() => mutations.publishAssignment.mutate()}>
              发布作业
            </Button>
          ) : (
            <Link className={buttonClassName({ variant: "outline" })} href={`/teacher/assignments/${assignment.id}/edit?course_id=${courseID}`}>
              查看详情
            </Link>
          )}
          <ConfirmDialog
            title="删除作业"
            description="删除后该作业及其所有提交记录将无法恢复，确定继续吗？"
            confirmText="删除"
            onConfirm={() => mutations.deleteAssignment.mutate()}
            trigger={<Button variant="destructive" disabled={hasSubmission}>删除作业</Button>}
          />
          {hasSubmission ? <Button variant="outline" onClick={() => setShowSubmissions((v) => !v)}>{showSubmissions ? "收起提交" : "查看提交"}</Button> : null}
        </div>
        {showSubmissions && submissions.data?.list ? (
          <div className="rounded-xl border border-border p-4 space-y-2">
            <p className="text-sm font-semibold">提交列表（{submissions.data.list.length} 份）</p>
            {submissions.data.list.map((sub) => (
              <div key={sub.id} className="flex items-center justify-between rounded-lg bg-muted/60 p-3 text-sm">
                <span>{sub.student_name ?? sub.student_id} · 第{sub.submission_no}次提交 · {sub.status_text}{sub.total_score != null ? ` · ${sub.total_score}分` : ""}</span>
                <Link className={buttonClassName({ variant: "outline", size: "sm" })} href={`/teacher/submissions/${sub.id}/grade`}>批改</Link>
              </div>
            ))}
          </div>
        ) : null}
      </CardContent>
    </Card>
  );
}

interface StudentAssignmentCardProps {
  assignment: AssignmentListItem;
}

function StudentAssignmentCard({ assignment }: StudentAssignmentCardProps) {
  const detail = useAssignment(assignment.id);
  const submissions = useMySubmissions(assignment.id);
  const latestSubmission = submissions.data?.submissions[0] ?? null;
  const remainingSubmissions = useMemo(() => {
    if (!detail.data) {
      return null;
    }
    return Math.max(detail.data.max_submissions - (submissions.data?.submissions.length ?? 0), 0);
  }, [detail.data, submissions.data?.submissions.length]);
  const availability = resolveStudentAssignmentAvailability(assignment.deadline_at);

  return (
    <Card>
      <CardHeader>
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div className="space-y-2">
            <CardTitle>{assignment.title}</CardTitle>
            <CardDescription>
              {assignment.assignment_type_text} · 截止 {assignment.deadline_at ? formatDateTime(assignment.deadline_at) : "未设置"}
            </CardDescription>
          </div>
          <div className="flex flex-wrap gap-2">
            <Badge variant={availability === "进行中" ? "outline" : "secondary"}>{availability}</Badge>
            <Badge variant="outline">{assignment.total_score} 分</Badge>
          </div>
        </div>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="grid gap-3 md:grid-cols-3">
          <MetricCard
            label="剩余提交次数"
            value={remainingSubmissions === null || !detail.data ? "加载中" : `${remainingSubmissions}/${detail.data.max_submissions}`}
            hint={detail.data?.late_policy_text ?? "读取迟交策略中"}
          />
          <MetricCard
            label="最近一次提交"
            value={latestSubmission ? latestSubmission.status_text : "尚未提交"}
            hint={latestSubmission?.submitted_at ? formatDateTime(latestSubmission.submitted_at) : "可进入作答页开始答题"}
          />
          <MetricCard label="最近得分" value={latestSubmission?.total_score ?? "-"} hint={latestSubmission ? `第 ${latestSubmission.submission_no} 次提交` : "暂无评分结果"} />
        </div>

        <div className="rounded-2xl border border-border/70 bg-muted/25 p-4 text-sm text-muted-foreground">
          <p>剩余提交次数 {remainingSubmissions === null || !detail.data ? "加载中" : `${remainingSubmissions}/${detail.data.max_submissions}`}</p>
          <p className="mt-1">最近一次提交：{latestSubmission ? latestSubmission.status_text : "暂无"}{latestSubmission?.total_score !== null && latestSubmission?.total_score !== undefined ? ` · 得分 ${latestSubmission.total_score}` : ""}</p>
        </div>

        <div className="flex flex-wrap gap-3">
          <Link className={buttonClassName()} href={`/student/assignments/${assignment.id}`}>
            进入作答
          </Link>
          <span className="text-sm text-muted-foreground">
            {availability === "进行中" ? "当前可继续作答或查看提交历史。" : "作业已截止，可查看历史作答与批改结果。"}
          </span>
        </div>
      </CardContent>
    </Card>
  );
}

interface MetricCardProps {
  label: string;
  value: string | number;
  hint: string;
}

function MetricCard({ label, value, hint }: MetricCardProps) {
  return (
    <div className="rounded-2xl border border-border/70 bg-muted/25 p-4">
      <p className="text-sm text-muted-foreground">{label}</p>
      <p className="mt-2 text-lg font-semibold">{value}</p>
      <p className="mt-1 text-xs text-muted-foreground">{hint}</p>
    </div>
  );
}

function resolveStudentAssignmentAvailability(deadlineAt: string | null) {
  if (!deadlineAt) {
    return "进行中";
  }

  return new Date(deadlineAt).getTime() > Date.now() ? "进行中" : "已截止";
}
