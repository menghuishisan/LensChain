"use client";

// AssignmentEditor.tsx
// 模块03作业编辑器，支持作业基础信息、题目增删改和发布作业。

import { useEffect, useRef, useState } from "react";
import React from "react";

import { Button } from "@/components/ui/Button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/Card";
import { FormField } from "@/components/ui/FormField";
import { Input } from "@/components/ui/Input";
import { Textarea } from "@/components/ui/Textarea";
import { useToast } from "@/components/ui/Toast";
import { useAssignment, useAssignmentMutations } from "@/hooks/useAssignments";
import type { AssignmentType, LatePolicy, QuestionType } from "@/types/courseAssignment";

/**
 * AssignmentEditor 组件属性。
 */
export interface AssignmentEditorProps {
  courseID: string;
  assignmentID?: string;
}

/**
 * AssignmentEditor 作业编辑器组件。
 */
export function AssignmentEditor({ courseID, assignmentID }: AssignmentEditorProps) {
  const { showToast } = useToast();
  const assignment = useAssignment(assignmentID ?? "");
  const mutations = useAssignmentMutations(courseID, assignmentID);
  const initializedAssignmentIDRef = useRef<string | null>(null);
  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");
  const [assignmentType, setAssignmentType] = useState<AssignmentType>(1);
  const [deadlineAt, setDeadlineAt] = useState("");
  const [maxSubmissions, setMaxSubmissions] = useState(1);
  const [latePolicy, setLatePolicy] = useState<LatePolicy>(1);
  const [lateDeductionPerDay, setLateDeductionPerDay] = useState(10);
  const [questionTitle, setQuestionTitle] = useState("");
  const [questionType, setQuestionType] = useState<QuestionType>(1);
  const [score, setScore] = useState(10);
  const [optionsText, setOptionsText] = useState("");
  const [correctAnswer, setCorrectAnswer] = useState("");
  const [referenceAnswer, setReferenceAnswer] = useState("");
  const [judgeConfig, setJudgeConfig] = useState("");

  useEffect(() => {
    if (!assignment.data) {
      return;
    }

    if (initializedAssignmentIDRef.current === assignment.data.id) {
      return;
    }
    initializedAssignmentIDRef.current = assignment.data.id;

    setTitle(assignment.data.title);
    setDescription(assignment.data.description ?? "");
    setAssignmentType(assignment.data.assignment_type);
    setDeadlineAt(assignment.data.deadline_at ? assignment.data.deadline_at.slice(0, 16) : "");
    setMaxSubmissions(assignment.data.max_submissions);
    setLatePolicy(assignment.data.late_policy);
    setLateDeductionPerDay(assignment.data.late_deduction_per_day ?? 10);
  }, [assignment.data]);

  const saveAssignment = () => {
    const payload = {
      title,
      description,
      assignment_type: assignmentType,
      deadline_at: deadlineAt ? new Date(deadlineAt).toISOString() : new Date().toISOString(),
      max_submissions: maxSubmissions,
      late_policy: latePolicy,
      late_deduction_per_day: latePolicy === 2 ? lateDeductionPerDay : null,
    };
    if (assignmentID) {
      mutations.updateAssignment.mutate(payload, { onSuccess: () => showToast({ title: "作业已保存", variant: "success" }) });
    } else {
      mutations.createAssignment.mutate(payload, { onSuccess: () => showToast({ title: "作业已创建", variant: "success" }) });
    }
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle>{assignmentID ? "编辑作业" : "创建作业"}</CardTitle>
        <CardDescription>发布前至少添加一道题目，客观题提交后由后端即时批改。</CardDescription>
      </CardHeader>
      <CardContent className="space-y-6">
        <div className="grid gap-4 md:grid-cols-2">
          <FormField id="assignment-title" label="标题" required><Input id="assignment-title" value={title} onChange={(event) => setTitle(event.target.value)} /></FormField>
          <FormField id="assignment-type" label="作业类型" required>
            <select id="assignment-type" className="h-10 rounded-lg border border-input bg-background px-3 text-sm" value={assignmentType} onChange={(event) => setAssignmentType(Number(event.target.value) as AssignmentType)}>
              <option value={1}>普通作业</option><option value={2}>测验</option>
            </select>
          </FormField>
          <FormField id="assignment-deadline" label="截止时间" required><Input id="assignment-deadline" type="datetime-local" value={deadlineAt} onChange={(event) => setDeadlineAt(event.target.value)} /></FormField>
          <FormField id="assignment-max-submissions" label="最大提交次数" required><Input id="assignment-max-submissions" type="number" min={1} value={maxSubmissions} onChange={(event) => setMaxSubmissions(Number(event.target.value))} /></FormField>
          <FormField id="assignment-late-policy" label="迟交策略" required>
            <select id="assignment-late-policy" className="h-10 rounded-lg border border-input bg-background px-3 text-sm" value={latePolicy} onChange={(event) => setLatePolicy(Number(event.target.value) as LatePolicy)}>
              <option value={1}>不允许迟交</option><option value={2}>允许迟交并扣分</option><option value={3}>允许迟交不扣分</option>
            </select>
          </FormField>
          {latePolicy === 2 ? <FormField id="assignment-late-deduction" label="每日扣分比例(%)" required><Input id="assignment-late-deduction" type="number" min={0} max={100} value={lateDeductionPerDay} onChange={(event) => setLateDeductionPerDay(Number(event.target.value))} /></FormField> : null}
          <FormField id="assignment-description" label="说明" className="md:col-span-2"><Textarea id="assignment-description" value={description} onChange={(event) => setDescription(event.target.value)} /></FormField>
        </div>
        <div className="flex gap-3">
          <Button onClick={saveAssignment} isLoading={mutations.createAssignment.isPending || mutations.updateAssignment.isPending}>保存作业</Button>
          {assignmentID ? <Button variant="secondary" onClick={() => mutations.publishAssignment.mutate(undefined, { onSuccess: () => showToast({ title: "作业已发布", variant: "success" }) })}>发布作业</Button> : null}
        </div>
        {assignmentID ? (
          <div className="rounded-xl border border-border p-4">
            <h3 className="font-display text-xl font-semibold">题目管理</h3>
            <div className="mt-4 grid gap-3 md:grid-cols-2">
              <FormField id="question-title" label="题目标题" required>
                <Input id="question-title" placeholder="题目标题" value={questionTitle} onChange={(event) => setQuestionTitle(event.target.value)} />
              </FormField>
              <FormField id="question-type" label="题目类型" required>
                <select id="question-type" className="h-10 rounded-lg border border-input bg-background px-3 text-sm" value={questionType} onChange={(event) => setQuestionType(Number(event.target.value) as QuestionType)}>
                  <option value={1}>单选</option><option value={2}>多选</option><option value={3}>判断</option><option value={4}>填空</option><option value={5}>简答</option><option value={6}>编程</option><option value={7}>实验报告</option>
                </select>
              </FormField>
              <FormField id="question-score" label="题目分值" required>
                <Input id="question-score" type="number" value={score} onChange={(event) => setScore(Number(event.target.value))} />
              </FormField>
              {questionType <= 4 ? (
                <>
                  <FormField id="question-options" label="选项">
                    <Textarea id="question-options" placeholder="选项，使用换行分隔" value={optionsText} onChange={(event) => setOptionsText(event.target.value)} rows={4} />
                  </FormField>
                  <FormField id="question-correct-answer" label="正确答案">
                    <Input id="question-correct-answer" placeholder="正确答案" value={correctAnswer} onChange={(event) => setCorrectAnswer(event.target.value)} />
                  </FormField>
                </>
              ) : null}
              {questionType === 5 || questionType === 7 ? (
                <FormField id="question-reference-answer" label="参考答案/要求">
                  <Textarea id="question-reference-answer" placeholder="参考答案或实验报告要求" value={referenceAnswer} onChange={(event) => setReferenceAnswer(event.target.value)} rows={5} />
                </FormField>
              ) : null}
              {questionType === 6 ? (
                <FormField id="question-judge-config" label="判题配置">
                  <Textarea id="question-judge-config" placeholder="判题配置(JSON或脚本说明)" value={judgeConfig} onChange={(event) => setJudgeConfig(event.target.value)} rows={5} />
                </FormField>
              ) : null}
              <div className="md:col-span-2">
                <Button
                  onClick={() =>
                    mutations.addQuestion.mutate(
                      {
                        question_type: questionType,
                        title: questionTitle,
                        score,
                        options: questionType <= 4 ? optionsText || null : null,
                        correct_answer: questionType <= 4 ? correctAnswer || null : null,
                        reference_answer: questionType === 5 || questionType === 7 ? referenceAnswer || null : null,
                        judge_config: questionType === 6 ? judgeConfig || null : null,
                      },
                      {
                        onSuccess: () => {
                          setQuestionTitle("");
                          setScore(10);
                          setOptionsText("");
                          setCorrectAnswer("");
                          setReferenceAnswer("");
                          setJudgeConfig("");
                        },
                      },
                    )
                  }
                >
                  添加题目
                </Button>
              </div>
            </div>
            <div className="mt-4 grid gap-2">
              {assignment.data?.questions.map((question) => (
                <div key={question.id} className="flex items-center justify-between rounded-lg bg-muted/60 p-3 text-sm">
                  <span>{question.question_type_text} · {question.title} · {question.score}分</span>
                  <Button size="sm" variant="outline" onClick={() => mutations.deleteQuestion.mutate(question.id)}>删除</Button>
                </div>
              ))}
            </div>
          </div>
        ) : null}
      </CardContent>
    </Card>
  );
}
