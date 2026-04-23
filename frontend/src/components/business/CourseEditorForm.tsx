"use client";

// CourseEditorForm.tsx
// 模块03课程创建/编辑表单，维护课程基础信息、时间、学分和人数上限。

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";

import { Button } from "@/components/ui/Button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/Card";
import { FormField } from "@/components/ui/FormField";
import { Input } from "@/components/ui/Input";
import { Textarea } from "@/components/ui/Textarea";
import { useToast } from "@/components/ui/Toast";
import { useCourse, useCreateCourseMutation, useUpdateCourseMutation } from "@/hooks/useCourses";
import type { CourseDifficulty, CourseType, CreateCourseRequest } from "@/types/course";

const EMPTY_COURSE_FORM: CreateCourseRequest = {
  title: "",
  description: "",
  cover_url: "",
  course_type: 3,
  difficulty: 2,
  topic: "区块链基础",
  credits: 3,
  semester_id: "",
  start_at: "",
  end_at: "",
  max_students: null,
};

/**
 * CourseEditorForm 组件属性。
 */
export interface CourseEditorFormProps {
  courseID?: string;
}

/**
 * CourseEditorForm 课程编辑表单组件。
 */
export function CourseEditorForm({ courseID }: CourseEditorFormProps) {
  const router = useRouter();
  const { showToast } = useToast();
  const isEdit = Boolean(courseID);
  const courseQuery = useCourse(courseID ?? "");
  const createMutation = useCreateCourseMutation();
  const updateMutation = useUpdateCourseMutation(courseID ?? "");
  const [form, setForm] = useState<CreateCourseRequest>(EMPTY_COURSE_FORM);

  useEffect(() => {
    if (courseQuery.data === undefined) {
      return;
    }
    setForm({
      title: courseQuery.data.title,
      description: courseQuery.data.description,
      cover_url: courseQuery.data.cover_url,
      course_type: courseQuery.data.course_type,
      difficulty: courseQuery.data.difficulty,
      topic: courseQuery.data.topic,
      credits: courseQuery.data.credits,
      semester_id: courseQuery.data.semester_id,
      start_at: courseQuery.data.start_at,
      end_at: courseQuery.data.end_at,
      max_students: courseQuery.data.max_students,
    });
  }, [courseQuery.data]);

  const canSubmit = form.title.trim().length > 0 && form.topic.trim().length > 0;

  return (
    <Card>
      <CardHeader>
        <CardTitle>{isEdit ? "编辑课程" : "创建课程"}</CardTitle>
        <CardDescription>维护课程基本信息、分类、难度、时间和学分。</CardDescription>
      </CardHeader>
      <CardContent>
        <form
          className="grid gap-4 md:grid-cols-2"
          onSubmit={(event) => {
            event.preventDefault();
            if (!canSubmit) {
              return;
            }
            if (isEdit && courseID) {
              updateMutation.mutate(form, {
                onSuccess: () => showToast({ title: "课程已更新", variant: "success" }),
                onError: (error) => showToast({ title: "保存失败", description: error.message, variant: "destructive" }),
              });
              return;
            }
            createMutation.mutate(form, {
              onSuccess: (created) => {
                showToast({ title: "课程已创建", variant: "success" });
                router.push(`/teacher/courses/${created.id}`);
              },
              onError: (error) => showToast({ title: "创建失败", description: error.message, variant: "destructive" }),
            });
          }}
        >
          <FormField label="课程名称" required><Input value={form.title} onChange={(event) => setForm((current) => ({ ...current, title: event.target.value }))} /></FormField>
          <FormField label="主题分类" required><Input value={form.topic} onChange={(event) => setForm((current) => ({ ...current, topic: event.target.value }))} /></FormField>
          <FormField label="课程类型" required>
            <select className="h-10 rounded-lg border border-input bg-background px-3 text-sm" value={form.course_type} onChange={(event) => setForm((current) => ({ ...current, course_type: Number(event.target.value) as CourseType }))}>
              <option value={1}>理论课</option><option value={2}>实验课</option><option value={3}>混合课</option><option value={4}>项目实战</option>
            </select>
          </FormField>
          <FormField label="难度" required>
            <select className="h-10 rounded-lg border border-input bg-background px-3 text-sm" value={form.difficulty} onChange={(event) => setForm((current) => ({ ...current, difficulty: Number(event.target.value) as CourseDifficulty }))}>
              <option value={1}>入门</option><option value={2}>进阶</option><option value={3}>高级</option><option value={4}>研究</option>
            </select>
          </FormField>
          <FormField label="封面URL"><Input value={form.cover_url ?? ""} onChange={(event) => setForm((current) => ({ ...current, cover_url: event.target.value }))} /></FormField>
          <FormField label="学分"><Input type="number" step="0.5" value={form.credits ?? ""} onChange={(event) => setForm((current) => ({ ...current, credits: event.target.value ? Number(event.target.value) : null }))} /></FormField>
          <FormField label="学期ID"><Input value={form.semester_id ?? ""} onChange={(event) => setForm((current) => ({ ...current, semester_id: event.target.value }))} /></FormField>
          <FormField label="人数上限"><Input type="number" value={form.max_students ?? ""} onChange={(event) => setForm((current) => ({ ...current, max_students: event.target.value ? Number(event.target.value) : null }))} /></FormField>
          <FormField label="开始时间"><Input type="datetime-local" onChange={(event) => setForm((current) => ({ ...current, start_at: event.target.value ? new Date(event.target.value).toISOString() : null }))} /></FormField>
          <FormField label="结束时间"><Input type="datetime-local" onChange={(event) => setForm((current) => ({ ...current, end_at: event.target.value ? new Date(event.target.value).toISOString() : null }))} /></FormField>
          <FormField label="课程描述" className="md:col-span-2"><Textarea value={form.description ?? ""} onChange={(event) => setForm((current) => ({ ...current, description: event.target.value }))} /></FormField>
          <div className="md:col-span-2">
            <Button type="submit" disabled={!canSubmit} isLoading={createMutation.isPending || updateMutation.isPending}>{isEdit ? "保存课程" : "创建课程"}</Button>
          </div>
        </form>
      </CardContent>
    </Card>
  );
}
