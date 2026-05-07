"use client";

// CourseListPanels.tsx
// 课程列表相关组件：教师课程列表、学生课程列表、加入课程、共享课程详情。

import Link from "next/link";
import { useState } from "react";

import { CourseCard } from "@/components/business/CourseCard";
import { Button, buttonClassName } from "@/components/ui/Button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/Card";
import { EmptyState } from "@/components/ui/EmptyState";
import { ErrorState } from "@/components/ui/ErrorState";
import { Input } from "@/components/ui/Input";
import { LoadingState } from "@/components/ui/LoadingState";
import { Pagination } from "@/components/ui/Pagination";
import { useToast } from "@/components/ui/Toast";
import {
  useCourseLifecycleMutations, useCourses, useJoinCourseMutation,
  useMyCourses, useSharedCourse,
} from "@/hooks/useCourses";
import type { ID } from "@/types/api";

/**
 * TeacherCourseListPanel 教师课程列表组件。
 */
export function TeacherCourseListPanel() {
  const [page, setPage] = useState(1);
  const query = useCourses({ page, page_size: 20 });
  if (query.isLoading) return <LoadingState variant="grid" />;
  if (query.isError) return <ErrorState description={query.error.message} />;
  const list = query.data?.list ?? [];
  return (
    <div className="space-y-5">
      <div className="flex items-center justify-between">
        <h2 className="font-display text-3xl font-semibold">我的课程</h2>
        <div className="flex gap-2">
          <Link className={buttonClassName()} href="/teacher/courses/create">创建课程</Link>
          <Link className={buttonClassName({ variant: "outline" })} href="/shared-courses">共享课程库</Link>
        </div>
      </div>
      {list.length === 0 ? (
        <EmptyState title="暂无课程" description="创建课程后可继续维护章节、作业和学生。" />
      ) : (
        <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
          {list.map((item) => <CourseCard key={item.id} course={item} href={`/teacher/courses/${item.id}`} />)}
        </div>
      )}
      {query.data?.pagination ? (
        <Pagination page={query.data.pagination.page} totalPages={query.data.pagination.total_pages} total={query.data.pagination.total} onPageChange={setPage} />
      ) : null}
    </div>
  );
}

/**
 * StudentCourseListPanel 学生我的课程组件。
 */
export function StudentCourseListPanel() {
  const [page, setPage] = useState(1);
  const query = useMyCourses({ page, page_size: 20 });
  if (query.isLoading) return <LoadingState variant="grid" />;
  if (query.isError) return <ErrorState description={query.error.message} />;
  const list = query.data?.list ?? [];
  return (
    <div className="space-y-5">
      <div className="flex items-center justify-between">
        <h2 className="font-display text-3xl font-semibold">我的课程</h2>
        <div className="flex gap-2">
          <Link className={buttonClassName({ variant: "outline" })} href="/student/schedule">课程表</Link>
          <Link className={buttonClassName({ variant: "outline" })} href="/student/courses/join">加入课程</Link>
        </div>
      </div>
      {list.length === 0 ? (
        <EmptyState title="暂无课程" description="输入教师提供的邀请码加入课程。" />
      ) : (
        <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
          {list.map((item) => <CourseCard key={item.id} course={item} href={`/student/courses/${item.id}`} />)}
        </div>
      )}
      {query.data?.pagination ? (
        <Pagination page={query.data.pagination.page} totalPages={query.data.pagination.total_pages} total={query.data.pagination.total} onPageChange={setPage} />
      ) : null}
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
      <CardHeader>
        <CardTitle>加入课程</CardTitle>
        <CardDescription>请输入教师提供的6位邀请码。</CardDescription>
      </CardHeader>
      <CardContent className="flex gap-3">
        <Input value={inviteCode} onChange={(event) => setInviteCode(event.target.value.toUpperCase())} maxLength={6} placeholder="A3B7K9" />
        <Button
          disabled={inviteCode.length !== 6}
          isLoading={mutation.isPending}
          onClick={() => mutation.mutate({ invite_code: inviteCode }, {
            onSuccess: () => showToast({ title: "加入课程成功", variant: "success" }),
            onError: (error) => showToast({ title: "加入失败", description: error.message, variant: "destructive" }),
          })}
        >
          加入
        </Button>
      </CardContent>
    </Card>
  );
}

/**
 * SharedCourseDetailPanel 共享课程详情组件。
 */
export function SharedCourseDetailPanel({ courseID }: { courseID: ID }) {
  const query = useSharedCourse(courseID);
  const lifecycle = useCourseLifecycleMutations(courseID);
  if (query.isLoading) return <LoadingState variant="hero" />;
  if (query.isError) return <ErrorState description={query.error.message} />;
  if (!query.data) return null;
  return (
    <Card>
      <CardHeader>
        <CardTitle>{query.data.title}</CardTitle>
        <CardDescription>{query.data.teacher_name} · {query.data.school_name} · 评分 {query.data.rating}</CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <p>{query.data.description}</p>
        {query.data.chapters.map((chapter) => (
          <div key={chapter.id} className="rounded-xl border border-border p-4">
            <p className="font-semibold">{chapter.title}</p>
            {chapter.lessons.map((lesson) => (
              <p key={lesson.id} className="mt-2 text-sm text-muted-foreground">{lesson.title} · {lesson.content_type_text}</p>
            ))}
          </div>
        ))}
        <Button onClick={() => lifecycle.clone.mutate()}>克隆为我的课程</Button>
      </CardContent>
    </Card>
  );
}
