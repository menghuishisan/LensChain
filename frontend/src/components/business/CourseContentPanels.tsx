"use client";

// CourseContentPanels.tsx
// 课程内容相关组件：章节管理、学生学习主页、课时学习、课程表。

import Link from "next/link";
import { useEffect, useRef, useState } from "react";

import { Badge } from "@/components/ui/Badge";
import { Button, buttonClassName } from "@/components/ui/Button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/Card";
import { EmptyState } from "@/components/ui/EmptyState";
import { ErrorState } from "@/components/ui/ErrorState";
import { Input } from "@/components/ui/Input";
import { LoadingState } from "@/components/ui/LoadingState";
import { useCourse, useCourseProgress, useMySchedule } from "@/hooks/useCourses";
import { useCourseChapters, useCourseContentMutations, useLesson } from "@/hooks/useCourseContent";
import { buildLessonProgressPayload, buildLessonUnloadProgressPayload, getLessonResumeSecond, shouldReportVideoProgress } from "@/hooks/useLessonVideoProgress";
import { safeMarkdownText } from "@/lib/content-safety";
import { formatDateTime, formatHours } from "@/lib/format";
import { resolveAdjacentLessons } from "@/lib/course-navigation";
import { buildWeeklyScheduleGrid } from "@/lib/schedule-grid";
import type { ID } from "@/types/api";

/**
 * CourseContentManagerPanel 课程章节与课时管理组件。
 */
export function CourseContentManagerPanel({ courseID }: { courseID: ID }) {
  const query = useCourseChapters(courseID);
  const mutations = useCourseContentMutations(courseID);
  const [chapterTitle, setChapterTitle] = useState("");
  if (query.isLoading) return <LoadingState variant="list" />;
  if (query.isError) return <ErrorState description={query.error.message} />;
  return (
    <Card>
      <CardHeader>
        <CardTitle>课程内容管理</CardTitle>
        <CardDescription>创建章节和课时，排序请求需提交完整ID数组。</CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="flex gap-2">
          <Input placeholder="章节标题" value={chapterTitle} onChange={(event) => setChapterTitle(event.target.value)} />
          <Button
            disabled={!chapterTitle.trim()}
            onClick={() => mutations.createChapter.mutate({ title: chapterTitle }, { onSuccess: () => setChapterTitle("") })}
          >
            添加章节
          </Button>
        </div>
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
  if (course.isLoading || progress.isLoading) return <LoadingState variant="hero" />;
  if (course.isError) return <ErrorState description={course.error.message} />;
  const lessonProgressByID = new Map((progress.data?.lessons ?? []).map((lesson) => [lesson.lesson_id, lesson]));
  return (
    <div className="space-y-5">
      <Card>
        <CardHeader>
          <CardTitle>{course.data?.title}</CardTitle>
          <CardDescription>{course.data?.teacher_name}</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="h-3 rounded-full bg-muted">
            <div className="h-full rounded-full bg-primary" style={{ width: `${progress.data?.progress ?? 0}%` }} />
          </div>
          <p className="mt-2 text-sm text-muted-foreground">
            {progress.data?.completed_count ?? 0}/{progress.data?.total_lessons ?? 0} 课时 · {formatHours(progress.data?.total_study_hours ?? 0)}
          </p>
          <div className="flex flex-wrap gap-2">
            <Link className={buttonClassName({ variant: "outline" })} href="#course-content">内容</Link>
            <Link className={buttonClassName({ variant: "outline" })} href={`/student/courses/${courseID}/assignments`}>作业</Link>
            <Link className={buttonClassName({ variant: "outline" })} href={`/courses/${courseID}/discussions`}>讨论</Link>
            <Link className={buttonClassName({ variant: "outline" })} href={`/courses/${courseID}/announcements`}>公告</Link>
            <Link className={buttonClassName({ variant: "outline" })} href={`/courses/${courseID}/evaluations`}>评价</Link>
            <Link className={buttonClassName({ variant: "outline" })} href={`/student/courses/${courseID}/grades`}>成绩</Link>
          </div>
        </CardContent>
      </Card>
      {course.data?.chapters.map((chapter) => (
        <Card key={chapter.id}>
          <CardHeader>
            <CardTitle>{chapter.title}</CardTitle>
          </CardHeader>
          <CardContent id="course-content" className="grid gap-2">
            {chapter.lessons.map((lesson) => {
              const lessonProgress = lessonProgressByID.get(lesson.id);
              return (
                <Link className="rounded-lg bg-muted/60 p-3" key={lesson.id} href={`/student/lessons/${lesson.id}`}>
                  <div className="flex items-center justify-between gap-3">
                    <span>{lesson.title} · {lesson.content_type_text}</span>
                    <Badge variant={lessonProgress?.status_text === "已完成" ? "success" : "outline"}>
                      {lessonProgress?.status_text ?? "未开始"}
                    </Badge>
                  </div>
                </Link>
              );
            })}
          </CardContent>
        </Card>
      ))}
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
  if (lesson.isLoading) return <LoadingState variant="hero" />;
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
              <Button
                key={item}
                size="sm"
                variant={speed === item ? "primary" : "outline"}
                onClick={() => { setSpeed(item); if (videoRef.current) videoRef.current.playbackRate = item; }}
              >
                {item}x
              </Button>
            ))}
          </div>
          </div>
        ) : null}
        {lesson.data.content ? (
          <pre className="whitespace-pre-wrap rounded-xl bg-muted/60 p-4 text-sm">{safeMarkdownText(lesson.data.content)}</pre>
        ) : null}
        {lesson.data.experiment_id ? (
          <Link className={buttonClassName({ variant: "outline" })} href={`/student/experiments/${lesson.data.experiment_id}/launch?course_id=${lesson.data.course_id}`}>
            启动实验环境
          </Link>
        ) : null}
        <div className="grid gap-2">
          {lesson.data.attachments.map((attachment) => (
            <a key={attachment.id} className="rounded-lg border border-border p-3 text-sm text-primary hover:bg-muted" href={attachment.file_url}>
              {attachment.file_name}
            </a>
          ))}
        </div>
        <Button
          onClick={() => mutations.updateProgress.mutate({ status: 3, video_progress: lesson.data?.video_duration ?? undefined, study_duration_increment: 0 })}
        >
          标记为已完成
        </Button>
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

function ChapterContentBlock({ courseID, chapter, index, chapters }: {
  courseID: ID;
  chapter: NonNullable<ReturnType<typeof useCourseChapters>["data"]>[number];
  index: number;
  chapters: NonNullable<ReturnType<typeof useCourseChapters>["data"]>;
}) {
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
        <select
          className="h-10 rounded-lg border border-input bg-background px-3 text-sm"
          value={lessonContentType}
          onChange={(event) => setLessonContentType(Number(event.target.value) as 1 | 2 | 3 | 4)}
        >
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
 * SchedulePanel 我的课程表组件。
 */
export function SchedulePanel() {
  const query = useMySchedule();
  if (query.isLoading) return <LoadingState variant="grid" />;
  const weeklyGrid = buildWeeklyScheduleGrid(query.data?.schedules ?? []);
  return (
    <Card>
      <CardHeader>
        <CardTitle>我的课程表</CardTitle>
      </CardHeader>
      <CardContent className="grid gap-4 lg:grid-cols-7">
        {weeklyGrid.map((day) => (
          <div key={day.dayOfWeek} className="rounded-xl border border-border bg-muted/25 p-3">
            <p className="font-semibold">{day.dayLabel}</p>
            <div className="mt-3 grid gap-3">
              {day.items.length === 0 ? (
                <p className="text-sm text-muted-foreground">暂无课程</p>
              ) : (
                day.items.map((item) => (
                  <Link key={`${item.course_id}-${item.start_time}`} href={`/student/courses/${item.course_id}`} className="rounded-lg border border-border bg-background p-3">
                    <p className="font-semibold">{item.course_title}</p>
                    <p className="mt-1 text-sm text-muted-foreground">{item.start_time}-{item.end_time}</p>
                    <p className="mt-1 text-xs text-muted-foreground">{item.location ?? "地点待定"}</p>
                  </Link>
                ))
              )}
            </div>
          </div>
        ))}
      </CardContent>
    </Card>
  );
}
