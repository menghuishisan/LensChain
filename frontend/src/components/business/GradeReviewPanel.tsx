"use client";

// GradeReviewPanel.tsx
// 模块06成绩审核组件，支持教师提交审核和学校管理员审核详情处理。

import { useState } from "react";

import { Badge } from "@/components/ui/Badge";
import { Button } from "@/components/ui/Button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/Card";
import { EmptyState } from "@/components/ui/EmptyState";
import { FormField } from "@/components/ui/FormField";
import { Input } from "@/components/ui/Input";
import { Textarea } from "@/components/ui/Textarea";
import { useGradeReview, useGradeReviewMutations, useGradeReviews } from "@/hooks/useGradeReviews";
import { getGradeReviewStatusVariant } from "@/lib/grade";
import { formatDateTime } from "@/lib/format";
import type { ID } from "@/types/api";

/**
 * GradeReviewPanel 组件属性。
 */
export interface GradeReviewPanelProps {
  mode: "teacher" | "admin";
  reviewID?: ID;
}

/**
 * GradeReviewPanel 成绩审核组件。
 */
export function GradeReviewPanel({ mode, reviewID }: GradeReviewPanelProps) {
  const reviewsQuery = useGradeReviews({ page: 1, page_size: 20 });
  const reviewQuery = useGradeReview(reviewID ?? "");
  const mutations = useGradeReviewMutations(reviewID);
  const [courseID, setCourseID] = useState("");
  const [semesterID, setSemesterID] = useState("");
  const [note, setNote] = useState("");
  const [comment, setComment] = useState("");

  if (mode === "teacher" && !reviewID) {
    return (
      <div className="space-y-5">
        <Card>
          <CardHeader>
            <CardTitle>成绩审核提交</CardTitle>
          </CardHeader>
          <CardContent className="grid gap-4 md:grid-cols-[1fr_1fr_1fr_auto]">
            <FormField label="课程ID">
              <Input value={courseID} onChange={(event) => setCourseID(event.target.value)} />
            </FormField>
            <FormField label="学期ID">
              <Input value={semesterID} onChange={(event) => setSemesterID(event.target.value)} />
            </FormField>
            <FormField label="提交备注">
              <Input value={note} onChange={(event) => setNote(event.target.value)} />
            </FormField>
            <Button className="self-end" disabled={!courseID || !semesterID} onClick={() => mutations.submit.mutate({ course_id: courseID, semester_id: semesterID, submit_note: note })} isLoading={mutations.submit.isPending}>
              提交审核
            </Button>
          </CardContent>
        </Card>
        <ReviewListBlock items={reviewsQuery.data?.list ?? []} />
      </div>
    );
  }

  if (!reviewQuery.data) {
    return <EmptyState title="暂无审核详情" description="请选择待审核记录查看明细。" />;
  }

  const review = reviewQuery.data;

  return (
    <Card>
      <CardHeader>
        <CardTitle>{review.course_name} · 成绩审核详情</CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="flex flex-wrap gap-2">
          <Badge variant={getGradeReviewStatusVariant(review.status)}>{review.status_text}</Badge>
          <Badge variant="outline">{review.semester_name}</Badge>
          {review.is_locked ? <Badge variant="success">已锁定</Badge> : <Badge variant="outline">未锁定</Badge>}
        </div>
        <div className="grid gap-3 md:grid-cols-2">
          <MetaField label="提交教师" value={review.submitted_by_name} />
          <MetaField label="提交时间" value={formatDateTime(review.submitted_at)} />
          <MetaField label="审核教师" value={review.reviewed_by_name ?? "—"} />
          <MetaField label="审核时间" value={formatDateTime(review.reviewed_at)} />
        </div>
        <div className="rounded-xl border border-border p-4 text-sm text-muted-foreground whitespace-pre-wrap">{review.submit_note ?? "无教师备注"}</div>
        <FormField label="审核意见">
          <Textarea value={comment} onChange={(event) => setComment(event.target.value)} rows={5} />
        </FormField>
        <div className="flex flex-wrap gap-2">
          <Button onClick={() => mutations.approve.mutate({ review_comment: comment })} isLoading={mutations.approve.isPending}>通过</Button>
          <Button variant="destructive" onClick={() => mutations.reject.mutate({ review_comment: comment })} isLoading={mutations.reject.isPending}>驳回</Button>
          <Button variant="outline" onClick={() => mutations.unlock.mutate({ unlock_reason: comment || "管理员解锁" })} isLoading={mutations.unlock.isPending}>解锁</Button>
        </div>
      </CardContent>
    </Card>
  );
}

function ReviewListBlock({ items }: { items: Array<{ id: string; course_name: string; semester_name: string; status: number; status_text: string }> }) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>审核列表</CardTitle>
      </CardHeader>
      <CardContent className="space-y-2">
        {items.length === 0 ? <EmptyState title="暂无审核记录" description="教师提交或学校管理员审核后显示记录。" /> : null}
        {items.map((item) => (
          <div key={item.id} className="rounded-xl border border-border p-4">
            <div className="flex items-center justify-between gap-3">
              <div>
                <p className="font-semibold">{item.course_name}</p>
                <p className="mt-1 text-sm text-muted-foreground">{item.semester_name}</p>
              </div>
              <Badge variant={getGradeReviewStatusVariant(item.status as 1 | 2 | 3 | 4)}>{item.status_text}</Badge>
            </div>
          </div>
        ))}
      </CardContent>
    </Card>
  );
}

function MetaField({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-xl border border-border bg-muted/25 p-4">
      <p className="text-xs text-muted-foreground">{label}</p>
      <p className="mt-1 text-sm font-semibold">{value}</p>
    </div>
  );
}
