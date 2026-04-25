"use client";

// CoursePanels.tsx
// 模块03课程页面级业务组件，组合课程、内容、选课、作业、成绩、公告、讨论和统计 hooks。

import Link from "next/link";
import React from "react";
import { useEffect, useRef, useState } from "react";

import { AssignmentEditor } from "@/components/business/AssignmentEditor";
import { CourseCard } from "@/components/business/CourseCard";
import { DiscussionThread } from "@/components/business/DiscussionThread";
import { LessonContentEditor } from "@/components/business/LessonContentEditor";
import { SubmissionReviewPanel } from "@/components/business/SubmissionReviewPanel";
import { Badge } from "@/components/ui/Badge";
import { Button, buttonClassName } from "@/components/ui/Button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/Card";
import { EmptyState } from "@/components/ui/EmptyState";
import { ErrorState } from "@/components/ui/ErrorState";
import { Input } from "@/components/ui/Input";
import { LoadingState } from "@/components/ui/LoadingState";
import { Pagination } from "@/components/ui/Pagination";
import { Textarea } from "@/components/ui/Textarea";
import { useToast } from "@/components/ui/Toast";
import { buildAssignmentAnswersPayload, persistAssignmentDraftLocal, useAssignmentAutosave } from "@/hooks/useAssignmentAutosave";
import { useAssignment, useAssignmentDraft, useAssignments, useCourseGrades, useGradeMutations, useMySubmissions, useSaveAssignmentDraftMutation, useSubmitAssignmentMutation, useSubmissions } from "@/hooks/useAssignments";
import { useAddCourseStudentMutation, useBatchAddCourseStudentsMutation, useCourse, useCourseLifecycleMutations, useCourseProgress, useCourses, useJoinCourseMutation, useMyCourses, useMySchedule, useRemoveCourseStudentMutation, useSharedCourse, useCourseStudents, useCourseStatistics, useExportCourseStatisticsMutation } from "@/hooks/useCourses";
import { useCourseChapters, useCourseContentMutations, useLesson, useUploadCourseFileMutation } from "@/hooks/useCourseContent";
import { useAnnouncementMutations, useAnnouncements, useDiscussionMutations, useDiscussions, useEvaluations, useEvaluationMutations } from "@/hooks/useDiscussions";
import { buildLessonProgressPayload, buildLessonUnloadProgressPayload, getLessonResumeSecond, shouldReportVideoProgress } from "@/hooks/useLessonVideoProgress";
import { safeMarkdownText, validateCourseAttachment } from "@/lib/content-safety";
import { formatDateTime } from "@/lib/format";
import { resolveAdjacentLessons } from "@/lib/course-navigation";
import { buildWeeklyScheduleGrid } from "@/lib/schedule-grid";
import type { ID } from "@/types/api";
import type { SubmissionListParams } from "@/types/courseAssignment";

/**
 * TeacherCourseListPanel 教师课程列表组件。
 */
export function TeacherCourseListPanel() {
  const [page, setPage] = useState(1);
  const query = useCourses({ page, page_size: 20 });
  if (query.isLoading) return <LoadingState />;
  if (query.isError) return <ErrorState description={query.error.message} />;
  const list = query.data?.list ?? [];
  return (
    <div className="space-y-5">
      <div className="flex items-center justify-between"><h2 className="font-display text-3xl font-semibold">我的课程</h2><Link className={buttonClassName()} href="/teacher/courses/create">创建课程</Link></div>
      {list.length === 0 ? <EmptyState title="暂无课程" description="创建课程后可继续维护章节、作业和学生。" /> : <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">{list.map((item) => <CourseCard key={item.id} course={item} href={`/teacher/courses/${item.id}`} />)}</div>}
      {query.data?.pagination ? <Pagination page={query.data.pagination.page} totalPages={query.data.pagination.total_pages} total={query.data.pagination.total} onPageChange={setPage} /> : null}
    </div>
  );
}

/**
 * StudentCourseListPanel 学生我的课程组件。
 */
export function StudentCourseListPanel() {
  const [page, setPage] = useState(1);
  const query = useMyCourses({ page, page_size: 20 });
  if (query.isLoading) return <LoadingState />;
  if (query.isError) return <ErrorState description={query.error.message} />;
  const list = query.data?.list ?? [];
  return (
    <div className="space-y-5">
      <div className="flex items-center justify-between"><h2 className="font-display text-3xl font-semibold">我的课程</h2><Link className={buttonClassName({ variant: "outline" })} href="/student/courses/join">加入课程</Link></div>
      {list.length === 0 ? <EmptyState title="暂无课程" description="输入教师提供的邀请码加入课程。" /> : <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">{list.map((item) => <CourseCard key={item.id} course={item} href={`/student/courses/${item.id}`} />)}</div>}
      {query.data?.pagination ? <Pagination page={query.data.pagination.page} totalPages={query.data.pagination.total_pages} total={query.data.pagination.total} onPageChange={setPage} /> : null}
    </div>
  );
}

/**
 * JoinCoursePanel 学生邀请码加入课程组件。
 */
export function JoinCoursePanel() {
  const [inviteCode, setInviteCode] = useState("");
  const mutation = useJoinCourseMutation();
  const { showToast } = useToast();
  return (
    <Card>
      <CardHeader><CardTitle>加入课程</CardTitle><CardDescription>请输入教师提供的6位邀请码。</CardDescription></CardHeader>
      <CardContent className="flex gap-3">
        <Input value={inviteCode} onChange={(event) => setInviteCode(event.target.value.toUpperCase())} maxLength={6} placeholder="A3B7K9" />
        <Button disabled={inviteCode.length !== 6} isLoading={mutation.isPending} onClick={() => mutation.mutate({ invite_code: inviteCode }, { onSuccess: () => showToast({ title: "加入课程成功", variant: "success" }), onError: (error) => showToast({ title: "加入失败", description: error.message, variant: "destructive" }) })}>加入</Button>
      </CardContent>
    </Card>
  );
}

/**
 * CourseContentManagerPanel 课程章节与课时管理组件。
 */
export function CourseContentManagerPanel({ courseID }: { courseID: ID }) {
  const query = useCourseChapters(courseID);
  const mutations = useCourseContentMutations(courseID);
  const [chapterTitle, setChapterTitle] = useState("");
  if (query.isLoading) return <LoadingState />;
  if (query.isError) return <ErrorState description={query.error.message} />;
  return (
    <Card>
      <CardHeader><CardTitle>课程内容管理</CardTitle><CardDescription>创建章节和课时，排序请求需提交完整ID数组。</CardDescription></CardHeader>
      <CardContent className="space-y-4">
        <div className="flex gap-2"><Input placeholder="章节标题" value={chapterTitle} onChange={(event) => setChapterTitle(event.target.value)} /><Button disabled={!chapterTitle.trim()} onClick={() => mutations.createChapter.mutate({ title: chapterTitle }, { onSuccess: () => setChapterTitle("") })}>添加章节</Button></div>
        {(query.data ?? []).map((chapter, index, chapters) => (
          <ChapterContentBlock key={chapter.id} courseID={courseID} chapter={chapter} index={index} chapters={chapters} />
        ))}
      </CardContent>
    </Card>
  );
}

/**
 * StudentCourseHomePanel 学生课程学习主页。
 */
export function StudentCourseHomePanel({ courseID }: { courseID: ID }) {
  const course = useCourse(courseID);
  const progress = useCourseProgress(courseID);
  if (course.isLoading || progress.isLoading) return <LoadingState />;
  if (course.isError) return <ErrorState description={course.error.message} />;
  const lessonProgressByID = new Map((progress.data?.lessons ?? []).map((lesson) => [lesson.lesson_id, lesson]));
  return (
    <div className="space-y-5">
      <Card><CardHeader><CardTitle>{course.data?.title}</CardTitle><CardDescription>{course.data?.teacher_name}</CardDescription></CardHeader><CardContent className="space-y-4"><div className="h-3 rounded-full bg-muted"><div className="h-full rounded-full bg-primary" style={{ width: `${progress.data?.progress ?? 0}%` }} /></div><p className="mt-2 text-sm text-muted-foreground">{progress.data?.completed_count ?? 0}/{progress.data?.total_lessons ?? 0} 课时 · {progress.data?.total_study_hours ?? 0}小时</p><div className="flex flex-wrap gap-2"><Link className={buttonClassName({ variant: "outline" })} href={`#course-content`}>内容</Link><Link className={buttonClassName({ variant: "outline" })} href={`/student/courses/${courseID}/assignments`}>作业</Link><Link className={buttonClassName({ variant: "outline" })} href={`/courses/${courseID}/discussions`}>讨论</Link><Link className={buttonClassName({ variant: "outline" })} href={`/courses/${courseID}/announcements`}>公告</Link><Link className={buttonClassName({ variant: "outline" })} href={`/student/courses/${courseID}/grades`}>成绩</Link></div></CardContent></Card>
      {course.data?.chapters.map((chapter) => <Card key={chapter.id}><CardHeader><CardTitle>{chapter.title}</CardTitle></CardHeader><CardContent id="course-content" className="grid gap-2">{chapter.lessons.map((lesson) => { const lessonProgress = lessonProgressByID.get(lesson.id); return <Link className="rounded-lg bg-muted/60 p-3" key={lesson.id} href={`/student/lessons/${lesson.id}`}><div className="flex items-center justify-between gap-3"><span>{lesson.title} · {lesson.content_type_text}</span><Badge variant={lessonProgress?.status_text === "已完成" ? "success" : "outline"}>{lessonProgress?.status_text ?? "未开始"}</Badge></div></Link>; })}</CardContent></Card>)}
    </div>
  );
}

/**
 * StudentLessonPanel 学生课时学习页。
 */
export function StudentLessonPanel({ lessonID }: { lessonID: ID }) {
  const lesson = useLesson(lessonID);
  const chapterTree = useCourseChapters(lesson.data?.course_id ?? "");
  const courseProgress = useCourseProgress(lesson.data?.course_id ?? "");
  const mutations = useCourseContentMutations(lesson.data?.course_id ?? "", lesson.data?.chapter_id, lessonID);
  const videoRef = useRef<HTMLVideoElement | null>(null);
  const lastReportedVideoSecondRef = useRef(0);
  const [speed, setSpeed] = useState(1);
  const lessonProgress = courseProgress.data?.lessons.find((item) => item.lesson_id === lessonID);
  useEffect(() => {
    const videoElement = videoRef.current;
    return () => {
      const currentVideoSecond = Math.floor(videoElement?.currentTime ?? 0);
      const payload = buildLessonUnloadProgressPayload(currentVideoSecond, lastReportedVideoSecondRef.current, lesson.data?.video_duration);
      if (payload) {
        mutations.updateProgress.mutate(payload);
      }
    };
  }, [lesson.data?.video_duration, mutations.updateProgress]);
  useEffect(() => {
    const videoElement = videoRef.current;
    if (!videoElement || !lesson.data?.video_url) {
      return;
    }

    const resumeSecond = getLessonResumeSecond(lessonProgress?.video_progress, lesson.data.video_duration);
    if (resumeSecond <= 0) {
      return;
    }

    const setResumeTime = () => {
      videoElement.currentTime = resumeSecond;
    };

    videoElement.addEventListener("loadedmetadata", setResumeTime, { once: true });
    return () => videoElement.removeEventListener("loadedmetadata", setResumeTime);
  }, [lesson.data?.video_duration, lesson.data?.video_url, lessonProgress?.video_progress]);
  if (lesson.isLoading) return <LoadingState />;
  if (lesson.isError) return <ErrorState description={lesson.error.message} />;
  if (!lesson.data) return <EmptyState title="课时不存在" description="该课时可能已被删除或无权访问。" />;
  const adjacentLessons = chapterTree.data ? resolveAdjacentLessons(chapterTree.data, lessonID) : { previousLesson: null, nextLesson: null };
  return (
    <Card>
      <CardHeader>
        <CardTitle>{lesson.data.title}</CardTitle>
        <CardDescription>{lesson.data.content_type_text} · 预计 {lesson.data.estimated_minutes ?? "未设置"} 分钟</CardDescription>
      </CardHeader>
      <CardContent className="space-y-5">
        {lesson.data.video_url ? (
          <div className="space-y-3">
          <video ref={videoRef} className="w-full rounded-xl bg-black" controls src={lesson.data.video_url} onTimeUpdate={(event) => {
            const current = Math.floor(event.currentTarget.currentTime);
            if (shouldReportVideoProgress(current, lastReportedVideoSecondRef.current)) {
              lastReportedVideoSecondRef.current = current;
              mutations.updateProgress.mutate(buildLessonProgressPayload(current, lesson.data.video_duration));
            }
          }} />
          <div className="flex gap-2">
            {[0.5, 1, 1.5, 2].map((item) => (
              <Button key={item} size="sm" variant={speed === item ? "primary" : "outline"} onClick={() => { setSpeed(item); if (videoRef.current) videoRef.current.playbackRate = item; }}>{item}x</Button>
            ))}
          </div>
          </div>
        ) : null}
        {lesson.data.content ? <pre className="whitespace-pre-wrap rounded-xl bg-muted/60 p-4 text-sm">{safeMarkdownText(lesson.data.content)}</pre> : null}
        {lesson.data.experiment_id ? <Link className={buttonClassName({ variant: "outline" })} href={`/student/experiments/${lesson.data.experiment_id}/launch`}>启动实验环境</Link> : null}
        <div className="grid gap-2">
          {lesson.data.attachments.map((attachment) => (
            <a key={attachment.id} className="rounded-lg border border-border p-3 text-sm text-primary hover:bg-muted" href={attachment.file_url}>
              {attachment.file_name}
            </a>
          ))}
        </div>
        <Button onClick={() => mutations.updateProgress.mutate({ status: 3, video_progress: lesson.data?.video_duration ?? undefined, study_duration_increment: 0 })}>标记为已完成</Button>
        <div className="flex flex-wrap items-center justify-between gap-3 border-t border-border pt-4">
          {adjacentLessons.previousLesson ? (
            <Link className={buttonClassName({ variant: "outline" })} href={`/student/lessons/${adjacentLessons.previousLesson.id}`}>
              ← 上一课时：{adjacentLessons.previousLesson.title}
            </Link>
          ) : <span className="text-sm text-muted-foreground">当前已是第一课时</span>}
          {adjacentLessons.nextLesson ? (
            <Link className={buttonClassName({ variant: "outline" })} href={`/student/lessons/${adjacentLessons.nextLesson.id}`}>
              下一课时：{adjacentLessons.nextLesson.title} →
            </Link>
          ) : <span className="text-sm text-muted-foreground">当前已是最后一课时</span>}
        </div>
      </CardContent>
    </Card>
  );
}

function ChapterContentBlock({ courseID, chapter, index, chapters }: { courseID: ID; chapter: NonNullable<ReturnType<typeof useCourseChapters>["data"]>[number]; index: number; chapters: NonNullable<ReturnType<typeof useCourseChapters>["data"]> }) {
  const mutations = useCourseContentMutations(courseID, chapter.id);
  const [lessonTitle, setLessonTitle] = useState("");
  const [lessonContentType, setLessonContentType] = useState<1 | 2 | 3 | 4>(2);
  const [estimatedMinutes, setEstimatedMinutes] = useState("");
  const moveChapter = (direction: -1 | 1) => {
    const next = [...chapters];
    const targetIndex = index + direction;
    if (targetIndex < 0 || targetIndex >= next.length) return;
    [next[index], next[targetIndex]] = [next[targetIndex], next[index]];
    mutations.sortChapters.mutate(next.map((item) => item.id));
  };
  const moveLesson = (lessonIndex: number, direction: -1 | 1) => {
    const nextLessons = [...chapter.lessons];
    const targetIndex = lessonIndex + direction;
    if (targetIndex < 0 || targetIndex >= nextLessons.length) return;
    [nextLessons[lessonIndex], nextLessons[targetIndex]] = [nextLessons[targetIndex], nextLessons[lessonIndex]];
    mutations.sortLessons.mutate(nextLessons.map((item) => item.id));
  };

  return (
    <div className="rounded-xl border border-border p-4">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <h3 className="font-display text-xl font-semibold">{chapter.title}</h3>
        <div className="flex gap-2">
          <Button size="sm" variant="outline" disabled={index === 0} onClick={() => moveChapter(-1)}>上移</Button>
          <Button size="sm" variant="outline" disabled={index === chapters.length - 1} onClick={() => moveChapter(1)}>下移</Button>
          <Button size="sm" variant="outline" onClick={() => mutations.deleteChapter.mutate(chapter.id)}>删除章节</Button>
        </div>
      </div>
      <div className="mt-3 grid gap-2">
        {chapter.lessons.map((lesson, lessonIndex) => (
          <div key={lesson.id} className="flex items-center justify-between gap-2 rounded-lg bg-muted/60 p-3 text-sm">
            <Link className="hover:text-primary" href={`/teacher/lessons/${lesson.id}/edit`}>{lesson.title} · {lesson.content_type_text}</Link>
            <span className="flex gap-2">
              <Button size="sm" variant="outline" disabled={lessonIndex === 0} onClick={() => moveLesson(lessonIndex, -1)}>上移</Button>
              <Button size="sm" variant="outline" disabled={lessonIndex === chapter.lessons.length - 1} onClick={() => moveLesson(lessonIndex, 1)}>下移</Button>
              <Button size="sm" variant="outline" onClick={() => mutations.deleteLesson.mutate(lesson.id)}>删除</Button>
            </span>
          </div>
        ))}
      </div>
      <div className="mt-4 grid gap-3 md:grid-cols-[1fr_10rem_8rem_auto]">
        <Input placeholder="课时标题" value={lessonTitle} onChange={(event) => setLessonTitle(event.target.value)} />
        <select className="h-10 rounded-lg border border-input bg-background px-3 text-sm" value={lessonContentType} onChange={(event) => setLessonContentType(Number(event.target.value) as 1 | 2 | 3 | 4)}>
          <option value={1}>视频</option>
          <option value={2}>图文</option>
          <option value={3}>附件</option>
          <option value={4}>实验</option>
        </select>
        <Input placeholder="预计分钟" type="number" value={estimatedMinutes} onChange={(event) => setEstimatedMinutes(event.target.value)} />
        <Button
          className="mt-0"
          size="sm"
          disabled={!lessonTitle.trim()}
          onClick={() =>
            mutations.createLesson.mutate(
              {
                title: lessonTitle,
                content_type: lessonContentType,
                estimated_minutes: estimatedMinutes ? Number(estimatedMinutes) : null,
              },
              {
                onSuccess: () => {
                  setLessonTitle("");
                  setLessonContentType(2);
                  setEstimatedMinutes("");
                },
              },
            )
          }
        >
          添加课时
        </Button>
      </div>
    </div>
  );
}

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
  if (query.isLoading) return <LoadingState />;
  if (query.isError) return <ErrorState description={query.error.message} />;
  return (
    <Card>
      <CardHeader><CardTitle>学生管理</CardTitle><CardDescription>查看课程学生、加入方式和学习进度。</CardDescription></CardHeader>
      <CardContent className="space-y-3">
        <div className="grid gap-2 md:grid-cols-[1fr_auto]">
          <Input placeholder="学生ID" value={studentID} onChange={(event) => setStudentID(event.target.value)} />
          <Button disabled={!studentID.trim()} isLoading={addMutation.isPending} onClick={() => addMutation.mutate(studentID, { onSuccess: () => setStudentID("") })}>添加学生</Button>
        </div>
        <div className="grid gap-2 md:grid-cols-[1fr_auto]">
          <Input placeholder="批量学生ID，用逗号分隔" value={batchIDs} onChange={(event) => setBatchIDs(event.target.value)} />
          <Button variant="outline" disabled={!batchIDs.trim()} isLoading={batchMutation.isPending} onClick={() => batchMutation.mutate(batchIDs.split(",").map((id) => id.trim()).filter(Boolean), { onSuccess: () => setBatchIDs("") })}>批量添加</Button>
        </div>
        {(query.data?.list ?? []).map((student) => (
          <div key={student.id} className="flex items-center justify-between rounded-xl border border-border p-4">
            <div><p className="font-semibold">{student.name}</p><p className="text-sm text-muted-foreground">{student.student_no ?? "无学号"} · {student.join_method_text} · 进度 {student.progress}%</p></div>
            <Button variant="outline" size="sm" onClick={() => removeMutation.mutate(student.id)}>移除</Button>
          </div>
        ))}
        {query.data?.pagination ? <Pagination page={query.data.pagination.page} totalPages={query.data.pagination.total_pages} total={query.data.pagination.total} onPageChange={setPage} /> : null}
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
    onAutosaved: () => setHasUnsavedChanges(false),
  });
  if (assignment.isLoading) return <LoadingState />;
  if (assignment.isError) return <ErrorState description={assignment.error.message} />;
  const payload = buildAssignmentAnswersPayload(answers);
  const persistLocal = () => persistAssignmentDraftLocal(assignmentID, answers);
  return (
    <Card><CardHeader><CardTitle>{assignment.data?.title}</CardTitle><CardDescription>草稿会保存到 localStorage 和服务端；网络失败时本地草稿保留。</CardDescription></CardHeader><CardContent className="space-y-4">
      {hasUnsavedChanges ? <p className="text-sm text-amber-600">有未保存的作答，系统将每 60 秒自动保存。</p> : null}
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
      <div className="flex gap-3"><Button variant="outline" onClick={() => { persistLocal(); saveDraft.mutate(payload, { onSuccess: () => { setHasUnsavedChanges(false); showToast({ title: "草稿已保存", variant: "success" }); } }); }}>保存草稿</Button><Button onClick={() => { persistLocal(); submit.mutate(payload, { onSuccess: () => { setHasUnsavedChanges(false); showToast({ title: "作业已提交", variant: "success" }); } }); }}>提交作业</Button></div>
      <div className="rounded-xl border border-border p-4">
        <p className="font-semibold">提交历史</p>
        {(submissions.data?.submissions ?? []).map((item) => <p key={item.id} className="mt-2 text-sm">第{item.submission_no}次 · {item.status_text} · {item.total_score ?? "待评分"} · {formatDateTime(item.submitted_at)}</p>)}
      </div>
    </CardContent></Card>
  );
}

/**
 * DiscussionListPanel 课程讨论区列表组件。
 */
export function DiscussionListPanel({ courseID }: { courseID: ID }) {
  const [page, setPage] = useState(1);
  const query = useDiscussions(courseID, { page, page_size: 20 });
  const mutations = useDiscussionMutations(courseID);
  const [title, setTitle] = useState("");
  const [content, setContent] = useState("");
  const pinnedDiscussions = (query.data?.list ?? []).filter((item) => item.is_pinned);
  const normalDiscussions = (query.data?.list ?? []).filter((item) => !item.is_pinned);
  return (
    <Card><CardHeader><CardTitle>课程讨论区</CardTitle></CardHeader><CardContent className="space-y-4"><div className="grid gap-2"><Input placeholder="标题" value={title} onChange={(event) => setTitle(event.target.value)} /><Textarea placeholder="内容，Markdown纯文本安全渲染" value={content} onChange={(event) => setContent(event.target.value)} /><Button disabled={!title || !content} onClick={() => mutations.create.mutate({ title, content }, { onSuccess: () => { setTitle(""); setContent(""); } })}>发新帖</Button></div>{pinnedDiscussions.length > 0 ? <div className="space-y-3"><p className="text-sm font-semibold text-muted-foreground">置顶帖子</p>{pinnedDiscussions.map((item) => <div key={item.id} className="rounded-xl border border-border bg-muted/40 p-4"><Link className="font-semibold hover:text-primary" href={`/discussions/${item.id}`}>置顶 · {item.title}</Link><p className="mt-1 text-sm text-muted-foreground">回复{item.reply_count} · 赞{item.like_count}</p></div>)}</div> : null}{normalDiscussions.map((item) => <div key={item.id} className="rounded-xl border border-border p-4"><Link className="font-semibold hover:text-primary" href={`/discussions/${item.id}`}>{item.title}</Link><p className="mt-1 text-sm text-muted-foreground">回复{item.reply_count} · 赞{item.like_count}</p></div>)}{query.data?.pagination ? <Pagination page={query.data.pagination.page} totalPages={query.data.pagination.total_pages} total={query.data.pagination.total} onPageChange={setPage} /> : null}</CardContent></Card>
  );
}

/**
 * AnnouncementPanel 课程公告组件。
 */
export function AnnouncementPanel({ courseID }: { courseID: ID }) {
  const [page, setPage] = useState(1);
  const query = useAnnouncements(courseID, { page, page_size: 20 });
  const mutations = useAnnouncementMutations(courseID);
  const [title, setTitle] = useState("");
  const [content, setContent] = useState("");
  const pinnedAnnouncements = (query.data?.list ?? []).filter((item) => item.is_pinned);
  const normalAnnouncements = (query.data?.list ?? []).filter((item) => !item.is_pinned);
  return (
    <Card><CardHeader><CardTitle>课程公告</CardTitle></CardHeader><CardContent className="space-y-4"><div className="grid gap-2"><Input placeholder="公告标题" value={title} onChange={(event) => setTitle(event.target.value)} /><Textarea placeholder="公告内容" value={content} onChange={(event) => setContent(event.target.value)} /><Button disabled={!title || !content} onClick={() => mutations.create.mutate({ title, content }, { onSuccess: () => { setTitle(""); setContent(""); } })}>发布公告</Button></div>{pinnedAnnouncements.length > 0 ? <div className="space-y-3"><p className="text-sm font-semibold text-muted-foreground">置顶公告</p>{pinnedAnnouncements.map((item) => <div key={item.id} className="rounded-xl border border-border bg-muted/40 p-4"><div className="flex justify-between gap-3"><p className="font-semibold">置顶 · {item.title}</p><div className="flex gap-2"><Button size="sm" variant="outline" onClick={() => mutations.pin.mutate({ id: item.id, isPinned: !item.is_pinned })}>取消置顶</Button><Button size="sm" variant="destructive" onClick={() => mutations.remove.mutate(item.id)}>删除</Button></div></div><pre className="mt-2 whitespace-pre-wrap text-sm">{safeMarkdownText(item.content)}</pre></div>)}</div> : null}{normalAnnouncements.map((item) => <div key={item.id} className="rounded-xl border border-border p-4"><div className="flex justify-between gap-3"><p className="font-semibold">{item.title}</p><div className="flex gap-2"><Button size="sm" variant="outline" onClick={() => mutations.pin.mutate({ id: item.id, isPinned: !item.is_pinned })}>置顶</Button><Button size="sm" variant="destructive" onClick={() => mutations.remove.mutate(item.id)}>删除</Button></div></div><pre className="mt-2 whitespace-pre-wrap text-sm">{safeMarkdownText(item.content)}</pre></div>)}{query.data?.pagination ? <Pagination page={query.data.pagination.page} totalPages={query.data.pagination.total_pages} total={query.data.pagination.total} onPageChange={setPage} /> : null}</CardContent></Card>
  );
}

/**
 * EvaluationPanel 课程评价组件。
 */
export function EvaluationPanel({ courseID, role }: { courseID: ID; role: "teacher" | "student" }) {
  const query = useEvaluations(courseID, { page: 1, page_size: 20 });
  const mutations = useEvaluationMutations(courseID);
  const [rating, setRating] = useState(5);
  const [comment, setComment] = useState("");
  return (
    <Card><CardHeader><CardTitle>课程评价</CardTitle><CardDescription>课程结束后学生可评价，教师可查看统计。</CardDescription></CardHeader><CardContent className="space-y-4">{role === "student" ? <div className="grid gap-2"><Input type="number" min={1} max={5} value={rating} onChange={(event) => setRating(Number(event.target.value))} /><Textarea value={comment} onChange={(event) => setComment(event.target.value)} /><Button onClick={() => mutations.create.mutate({ rating, comment })}>提交评价</Button></div> : null}<div className="rounded-xl bg-muted/60 p-4">平均评分：{query.data?.summary.avg_rating ?? 0} · 共{query.data?.summary.total_count ?? 0}条</div>{query.data?.items.map((item) => <div key={item.id} className="rounded-xl border border-border p-4">{item.student_name} · {item.rating}星<p className="mt-2 text-sm">{item.comment}</p></div>)}</CardContent></Card>
  );
}

/**
 * SharedCourseDetailPanel 共享课程详情组件。
 */
export function SharedCourseDetailPanel({ courseID }: { courseID: ID }) {
  const query = useSharedCourse(courseID);
  const lifecycle = useCourseLifecycleMutations(courseID);
  if (query.isLoading) return <LoadingState />;
  if (query.isError) return <ErrorState description={query.error.message} />;
  if (!query.data) return null;
  return <Card><CardHeader><CardTitle>{query.data.title}</CardTitle><CardDescription>{query.data.teacher_name} · {query.data.school_name} · 评分 {query.data.rating}</CardDescription></CardHeader><CardContent className="space-y-4"><p>{query.data.description}</p>{query.data.chapters.map((chapter) => <div key={chapter.id} className="rounded-xl border border-border p-4"><p className="font-semibold">{chapter.title}</p>{chapter.lessons.map((lesson) => <p key={lesson.id} className="mt-2 text-sm text-muted-foreground">{lesson.title} · {lesson.content_type_text}</p>)}</div>)}<Button onClick={() => lifecycle.clone.mutate()}>克隆为我的课程</Button></CardContent></Card>;
}

/**
 * SchedulePanel 我的课程表组件。
 */
export function SchedulePanel() {
  const query = useMySchedule();
  if (query.isLoading) return <LoadingState />;
  const weeklyGrid = buildWeeklyScheduleGrid(query.data?.schedules ?? []);
  return <Card><CardHeader><CardTitle>我的课程表</CardTitle></CardHeader><CardContent className="grid gap-4 lg:grid-cols-7">{weeklyGrid.map((day) => <div key={day.dayOfWeek} className="rounded-xl border border-border bg-muted/25 p-3"><p className="font-semibold">{day.dayLabel}</p><div className="mt-3 grid gap-3">{day.items.length === 0 ? <p className="text-sm text-muted-foreground">暂无课程</p> : day.items.map((item) => <Link key={`${item.course_id}-${item.start_time}`} href={`/student/courses/${item.course_id}`} className="rounded-lg border border-border bg-background p-3"><p className="font-semibold">{item.course_title}</p><p className="mt-1 text-sm text-muted-foreground">{item.start_time}-{item.end_time}</p><p className="mt-1 text-xs text-muted-foreground">{item.location ?? "地点待定"}</p></Link>)}</div></div>)}</CardContent></Card>;
}

function Stat({ title, value }: { title: string; value: string | number }) {
  return <Card><CardHeader><CardDescription>{title}</CardDescription><CardTitle>{value}</CardTitle></CardHeader></Card>;
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
