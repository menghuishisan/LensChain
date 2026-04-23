"use client";

// SubmissionReviewPanel.tsx
// 模块03提交批改面板，逐题录入分数和评语，提交后刷新提交与成绩缓存。

import { useEffect, useRef, useState } from "react";
import React from "react";

import { Button } from "@/components/ui/Button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/Card";
import { ErrorState } from "@/components/ui/ErrorState";
import { Input } from "@/components/ui/Input";
import { LoadingState } from "@/components/ui/LoadingState";
import { Textarea } from "@/components/ui/Textarea";
import { useToast } from "@/components/ui/Toast";
import { useGradeSubmissionMutation, useSubmission } from "@/hooks/useAssignments";

/**
 * SubmissionReviewPanel 组件属性。
 */
export interface SubmissionReviewPanelProps {
  submissionID: string;
}

/**
 * SubmissionReviewPanel 作业批改组件。
 */
export function SubmissionReviewPanel({ submissionID }: SubmissionReviewPanelProps) {
  const query = useSubmission(submissionID);
  const mutation = useGradeSubmissionMutation(submissionID);
  const { showToast } = useToast();
  const initializedSubmissionIDRef = useRef<string | null>(null);
  const [teacherComment, setTeacherComment] = useState("");
  const [scores, setScores] = useState<Record<string, number>>({});
  const [comments, setComments] = useState<Record<string, string>>({});

  useEffect(() => {
    if (!query.data) {
      return;
    }
    if (initializedSubmissionIDRef.current === query.data.id) {
      return;
    }
    initializedSubmissionIDRef.current = query.data.id;
    setTeacherComment(query.data.teacher_comment ?? "");
    setScores(
      Object.fromEntries(
        query.data.answers.map((answer) => [answer.question_id, answer.score ?? 0]),
      ),
    );
    setComments(
      Object.fromEntries(
        query.data.answers.map((answer) => [answer.question_id, answer.teacher_comment ?? ""]),
      ),
    );
  }, [query.data]);

  if (query.isLoading) return <LoadingState />;
  if (query.isError) return <ErrorState description={query.error.message} />;
  if (!query.data) return null;

  return (
    <Card>
      <CardHeader>
        <CardTitle>批改提交：{query.data.student_name}</CardTitle>
        <CardDescription>给分不得超过题目分值，后端会再次校验。</CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="grid gap-3 md:grid-cols-4">
          <div className="rounded-lg border border-border bg-muted/50 p-3 text-sm text-muted-foreground">第 {query.data.submission_no} 次提交</div>
          <div className="rounded-lg border border-border bg-muted/50 p-3 text-sm text-muted-foreground">{query.data.status_text}</div>
          <div className="rounded-lg border border-border bg-muted/50 p-3 text-sm text-muted-foreground">{query.data.is_late ? `迟交 ${query.data.late_days ?? 0} 天` : "按时提交"}</div>
          <div className="rounded-lg border border-border bg-muted/50 p-3 text-sm text-muted-foreground">扣分后 {query.data.score_after_deduction ?? query.data.total_score ?? 0}</div>
        </div>
        {query.data.is_late ? (
          <div className="rounded-lg border border-amber-500/30 bg-amber-500/10 p-3 text-sm text-amber-700">
            原始得分 {query.data.score_before_deduction ?? "-"}，扣分后 {query.data.score_after_deduction ?? "-"}。
          </div>
        ) : null}
        {query.data.answers.map((answer) => (
          <div key={answer.id} className="rounded-xl border border-border p-4">
            <p className="font-semibold">{answer.question_title}</p>
            <p className="mt-2 whitespace-pre-wrap rounded-lg bg-muted/50 p-3 text-sm">{answer.answer_content ?? answer.answer_file_url ?? "未作答"}</p>
            <div className="mt-3 grid gap-3 md:grid-cols-[8rem_1fr]">
              <Input type="number" placeholder="得分" value={scores[answer.question_id] ?? answer.score ?? ""} onChange={(event) => setScores((current) => ({ ...current, [answer.question_id]: Number(event.target.value) }))} />
              <Input placeholder="单题评语" value={comments[answer.question_id] ?? answer.teacher_comment ?? ""} onChange={(event) => setComments((current) => ({ ...current, [answer.question_id]: event.target.value }))} />
            </div>
          </div>
        ))}
        <Textarea placeholder="总体评语" value={teacherComment} onChange={(event) => setTeacherComment(event.target.value)} />
        <Button
          isLoading={mutation.isPending}
          onClick={() =>
            mutation.mutate(
              {
                teacher_comment: teacherComment,
                answers: query.data.answers.map((answer) => ({
                  question_id: answer.question_id,
                  score: scores[answer.question_id] ?? answer.score ?? 0,
                  teacher_comment: comments[answer.question_id] ?? answer.teacher_comment,
                })),
              },
              { onSuccess: () => showToast({ title: "批改已提交", variant: "success" }) },
            )
          }
        >
          提交批改
        </Button>
      </CardContent>
    </Card>
  );
}
