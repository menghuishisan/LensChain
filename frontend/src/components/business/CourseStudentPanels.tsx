"use client";

// CourseStudentPanels.tsx
// 课程学生管理与作业作答组件。

import { CheckCircle2, CircleAlert, Loader2 } from "lucide-react";
import { useEffect, useState } from "react";

import { Button } from "@/components/ui/Button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/Card";
import { ErrorState } from "@/components/ui/ErrorState";
import { Input } from "@/components/ui/Input";
import { LoadingState } from "@/components/ui/LoadingState";
import { Pagination } from "@/components/ui/Pagination";
import { Textarea } from "@/components/ui/Textarea";
import { useToast } from "@/components/ui/Toast";
import { buildAssignmentAnswersPayload, persistAssignmentDraftLocal, useAssignmentAutosave } from "@/hooks/useAssignmentAutosave";
import {
  useAssignment, useAssignmentDraft,
  useMySubmissions, useSaveAssignmentDraftMutation, useSubmitAssignmentMutation,
} from "@/hooks/useAssignments";
import {
  useAddCourseStudentMutation, useBatchAddCourseStudentsMutation,
  useRemoveCourseStudentMutation, useCourseStudents,
} from "@/hooks/useCourses";
import { useUploadCourseFileMutation } from "@/hooks/useCourseContent";
import { validateCourseAttachment } from "@/lib/content-safety";
import { formatDateTime } from "@/lib/format";
import type { ID } from "@/types/api";

/**
 * CourseStudentsPanel 课程学生管理组件。
 */
export function CourseStudentsPanel({ courseID }: { courseID: ID }) {
  const [page, setPage] = useState(1);
  const [studentID, setStudentID] = useState("");
  const [batchIDs, setBatchIDs] = useState("");
  const query = useCourseStudents(courseID, { page, page_size: 20 });
  const addMutation = useAddCourseStudentMutation(courseID);
  const batchMutation = useBatchAddCourseStudentsMutation(courseID);
  const removeMutation = useRemoveCourseStudentMutation(courseID);
  if (query.isLoading) return <LoadingState variant="table" />;
  if (query.isError) return <ErrorState description={query.error.message} />;
  return (
    <Card>
      <CardHeader>
        <CardTitle>课程学生管理</CardTitle>
        <CardDescription>查看课程学生、加入方式和学习进度。</CardDescription>
      </CardHeader>
      <CardContent className="space-y-3">
        <div className="grid gap-2 md:grid-cols-[1fr_auto]">
          <Input placeholder="学生ID" value={studentID} onChange={(event) => setStudentID(event.target.value)} />
          <Button
            disabled={!studentID.trim()}
            isLoading={addMutation.isPending}
            onClick={() => addMutation.mutate(studentID, { onSuccess: () => setStudentID("") })}
          >
            添加学生
          </Button>
        </div>
        <div className="grid gap-2 md:grid-cols-[1fr_auto]">
          <Input placeholder="批量学生ID，用逗号分隔" value={batchIDs} onChange={(event) => setBatchIDs(event.target.value)} />
          <Button
            variant="outline"
            disabled={!batchIDs.trim()}
            isLoading={batchMutation.isPending}
            onClick={() => batchMutation.mutate(batchIDs.split(",").map((id) => id.trim()).filter(Boolean), { onSuccess: () => setBatchIDs("") })}
          >
            批量添加
          </Button>
        </div>
        {(query.data?.list ?? []).map((student) => (
          <div key={student.id} className="flex items-center justify-between rounded-xl border border-border p-4">
            <div>
              <p className="font-semibold">{student.name}</p>
              <p className="text-sm text-muted-foreground">{student.student_no ?? "无学号"} · {student.join_method_text} · 进度 {student.progress}%</p>
            </div>
            <Button variant="outline" size="sm" onClick={() => removeMutation.mutate(student.id)}>移除</Button>
          </div>
        ))}
        {query.data?.pagination ? (
          <Pagination page={query.data.pagination.page} totalPages={query.data.pagination.total_pages} total={query.data.pagination.total} onPageChange={setPage} />
        ) : null}
      </CardContent>
    </Card>
  );
}

/**
 * AssignmentAnswerPanel 学生作业作答页，支持本地和服务端草稿。
 */
export function AssignmentAnswerPanel({ assignmentID }: { assignmentID: ID }) {
  const assignment = useAssignment(assignmentID);
  const draft = useAssignmentDraft(assignmentID);
  const saveDraft = useSaveAssignmentDraftMutation(assignmentID);
  const submit = useSubmitAssignmentMutation(assignmentID);
  const submissions = useMySubmissions(assignmentID);
  const uploadMutation = useUploadCourseFileMutation();
  const { showToast } = useToast();
  const [answers, setAnswers] = useState<Record<string, string>>({});
  const [hasUnsavedChanges, setHasUnsavedChanges] = useState(false);
  const [lastSavedAt, setLastSavedAt] = useState<Date | null>(null);
  useEffect(() => {
    const localDraft = localStorage.getItem(`assignment-draft:${assignmentID}`);
    if (draft.data?.answers) {
      setAnswers(Object.fromEntries(draft.data.answers.map((item) => [item.question_id, item.answer_content ?? item.answer_file_url ?? ""])));
      setHasUnsavedChanges(false);
    } else if (localDraft) {
      setAnswers(JSON.parse(localDraft) as Record<string, string>);
      setHasUnsavedChanges(false);
    }
  }, [assignmentID, draft.data]);
  useAssignmentAutosave({
    assignmentID,
    answers,
    hasUnsavedChanges,
    onSaveDraft: (draftPayload) => saveDraft.mutate(draftPayload),
    onAutosaved: () => {
      setHasUnsavedChanges(false);
      setLastSavedAt(new Date());
    },
  });
  if (assignment.isLoading) return <LoadingState variant="hero" />;
  if (assignment.isError) return <ErrorState description={assignment.error.message} />;
  const payload = buildAssignmentAnswersPayload(answers);
  const persistLocal = () => persistAssignmentDraftLocal(assignmentID, answers);
  return (
    <Card>
      <CardHeader>
        <CardTitle>{assignment.data?.title}</CardTitle>
        <CardDescription>草稿会保存到 localStorage 和服务端；网络失败时本地草稿保留。</CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
      <div className="flex items-center gap-2 rounded-lg border border-border bg-muted/40 px-3 py-2 text-xs">
        {saveDraft.isPending ? (
          <><Loader2 className="h-4 w-4 animate-spin text-primary" /><span className="text-muted-foreground">正在保存草稿…</span></>
        ) : hasUnsavedChanges ? (
          <><CircleAlert className="h-4 w-4 text-amber-500" /><span className="text-amber-700 dark:text-amber-400">有未保存的作答，系统将每 60 秒自动保存到服务器（本地草稿已即时保留）</span></>
        ) : lastSavedAt ? (
          <><CheckCircle2 className="h-4 w-4 text-emerald-500" /><span className="text-muted-foreground">已自动保存 · {lastSavedAt.toLocaleTimeString()}</span></>
        ) : (
          <><CheckCircle2 className="h-4 w-4 text-muted-foreground" /><span className="text-muted-foreground">草稿与服务器已同步</span></>
        )}
      </div>
      {assignment.data?.questions.map((question) => {
        const questionOptions = parseQuestionOptions(question.options);
        const answerValue = answers[question.id] ?? "";
        return (
          <div key={question.id} className="rounded-xl border border-border p-4">
            <p className="font-semibold">{question.title} · {question.score}分</p>
            {question.question_type === 1 || question.question_type === 3 ? (
              <div className="mt-3 grid gap-2">
                {questionOptions.map((option) => (
                  <label key={option} className="flex items-center gap-2 text-sm">
                    <input
                      type="radio"
                      name={`question-${question.id}`}
                      checked={answerValue === option}
                      onChange={() => {
                        setAnswers((current) => ({ ...current, [question.id]: option }));
                        setHasUnsavedChanges(true);
                      }}
                    />
                    <span>{option}</span>
                  </label>
                ))}
              </div>
            ) : null}
            {question.question_type === 2 ? (
              <div className="mt-3 grid gap-2">
                {questionOptions.map((option) => {
                  const selectedValues = answerValue.length > 0 ? answerValue.split("\n") : [];
                  return (
                    <label key={option} className="flex items-center gap-2 text-sm">
                      <input
                        type="checkbox"
                        checked={selectedValues.includes(option)}
                        onChange={() => {
                          setAnswers((current) => ({ ...current, [question.id]: toggleMultiSelectAnswer(current[question.id] ?? "", option) }));
                          setHasUnsavedChanges(true);
                        }}
                      />
                      <span>{option}</span>
                    </label>
                  );
                })}
              </div>
            ) : null}
            {question.question_type === 4 ? (
              <Input
                className="mt-3"
                placeholder="请输入答案"
                value={answerValue}
                onChange={(event) => {
                  setAnswers((current) => ({ ...current, [question.id]: event.target.value }));
                  setHasUnsavedChanges(true);
                }}
              />
            ) : null}
            {question.question_type === 5 || question.question_type === 6 ? (
              <Textarea
                className="mt-3"
                placeholder={question.question_type === 6 ? "请输入代码或答案" : "请输入回答内容"}
                value={answerValue}
                onChange={(event) => {
                  setAnswers((current) => ({ ...current, [question.id]: event.target.value }));
                  setHasUnsavedChanges(true);
                }}
              />
            ) : null}
            {question.question_type === 7 ? (
              <div className="mt-3 grid gap-3">
                <Textarea
                  placeholder="可补充实验说明或报告摘要"
                  value={answerValue.startsWith("course/") ? "" : answerValue}
                  onChange={(event) => {
                    setAnswers((current) => ({ ...current, [question.id]: event.target.value }));
                    setHasUnsavedChanges(true);
                  }}
                />
                <input
                  aria-label="实验报告上传"
                  className="block text-sm"
                  type="file"
                  accept=".pdf,.doc,.docx,.ppt,.pptx"
                  onChange={(event) => {
                    const file = event.target.files?.[0];
                    if (!file) return;
                    const result = validateCourseAttachment(file, "document");
                    if (!result.isValid) {
                      showToast({ title: "文件不符合要求", description: result.error, variant: "destructive" });
                      return;
                    }
                    uploadMutation.mutate({
                      file,
                      purpose: "assignment_report",
                    }, {
                      onSuccess: (uploaded) => {
                        setAnswers((current) => ({ ...current, [question.id]: uploaded.file_url }));
                        setHasUnsavedChanges(true);
                      },
                    });
                  }}
                />
              </div>
            ) : null}
          </div>
        );
      })}
      <div className="flex gap-3">
        <Button
          variant="outline"
          onClick={() => {
            persistLocal();
            saveDraft.mutate(payload, { onSuccess: () => { setHasUnsavedChanges(false); setLastSavedAt(new Date()); showToast({ title: "草稿已保存", variant: "success" }); } });
          }}
        >
          保存草稿
        </Button>
        <Button
          onClick={() => {
            persistLocal();
            submit.mutate(payload, { onSuccess: () => { setHasUnsavedChanges(false); showToast({ title: "作业已提交", variant: "success" }); } });
          }}
        >
          提交作业
        </Button>
      </div>
      <div className="rounded-xl border border-border p-4">
        <p className="font-semibold">提交历史</p>
        {(submissions.data?.submissions ?? []).map((item) => (
          <p key={item.id} className="mt-2 text-sm">
            第{item.submission_no}次 · {item.status_text} · {item.total_score ?? "待评分"} · {formatDateTime(item.submitted_at)}
          </p>
        ))}
      </div>
    </CardContent>
  </Card>
  );
}

function parseQuestionOptions(options?: string | null) {
  if (!options) {
    return [];
  }
  return options
    .split("\n")
    .map((item) => item.trim())
    .filter((item) => item.length > 0);
}

function toggleMultiSelectAnswer(currentAnswer: string, option: string) {
  const selectedValues = currentAnswer.length > 0 ? currentAnswer.split("\n").filter(Boolean) : [];
  if (selectedValues.includes(option)) {
    return selectedValues.filter((item) => item !== option).join("\n");
  }
  return [...selectedValues, option].join("\n");
}
