"use client";

// LessonContentEditor.tsx
// 模块03课时内容编辑组件，支持视频、图文、附件、实验类型和附件元数据保存。

import { useEffect, useRef, useState } from "react";
import React from "react";

import { Button } from "@/components/ui/Button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/Card";
import { FormField } from "@/components/ui/FormField";
import { Input } from "@/components/ui/Input";
import { Textarea } from "@/components/ui/Textarea";
import { useToast } from "@/components/ui/Toast";
import { useCourseContentMutations, useLesson, useUploadCourseFileMutation } from "@/hooks/useCourseContent";
import { validateCourseAttachment } from "@/lib/content-safety";
import type { LessonContentType, UpdateLessonRequest } from "@/types/course";

/**
 * LessonContentEditor 组件属性。
 */
export interface LessonContentEditorProps {
  lessonID: string;
}

/**
 * LessonContentEditor 课时内容编辑组件。
 */
export function LessonContentEditor({ lessonID }: LessonContentEditorProps) {
  const query = useLesson(lessonID);
  const mutations = useCourseContentMutations(query.data?.course_id ?? "", query.data?.chapter_id, lessonID);
  const uploadMutation = useUploadCourseFileMutation();
  const { showToast } = useToast();
  const initializedLessonIDRef = useRef<string | null>(null);
  const [form, setForm] = useState<UpdateLessonRequest>({});
  const [uploadError, setUploadError] = useState<string | null>(null);
  const [uploadProgress, setUploadProgress] = useState(0);

  useEffect(() => {
    if (!query.data) {
      return;
    }

    if (initializedLessonIDRef.current === query.data.id) {
      return;
    }
    initializedLessonIDRef.current = query.data.id;

    setForm({
      title: query.data.title,
      content_type: query.data.content_type,
      content: query.data.content,
      video_url: query.data.video_url,
      video_duration: query.data.video_duration,
      experiment_id: query.data.experiment_id,
      estimated_minutes: query.data.estimated_minutes,
    });
  }, [query.data]);

  return (
    <Card>
      <CardHeader><CardTitle>课时内容编辑</CardTitle><CardDescription>编辑课时内容并保存附件元数据，实际对象存储上传由后续存储能力提供。</CardDescription></CardHeader>
      <CardContent>
        <form className="grid gap-4" onSubmit={(event) => { event.preventDefault(); mutations.updateLesson.mutate(form, { onSuccess: () => showToast({ title: "课时已保存", variant: "success" }) }); }}>
          <FormField id="lesson-title" label="课时标题" required><Input id="lesson-title" value={form.title ?? ""} onChange={(event) => setForm((current) => ({ ...current, title: event.target.value }))} /></FormField>
          <FormField id="lesson-content-type" label="内容类型" required>
            <select id="lesson-content-type" className="h-10 rounded-lg border border-input bg-background px-3 text-sm" value={form.content_type ?? 2} onChange={(event) => setForm((current) => ({ ...current, content_type: Number(event.target.value) as LessonContentType }))}>
              <option value={1}>视频</option><option value={2}>图文</option><option value={3}>附件</option><option value={4}>实验</option>
            </select>
          </FormField>
          {form.content_type === 2 ? <FormField id="lesson-markdown-content" label="图文/Markdown内容"><Textarea id="lesson-markdown-content" rows={8} value={form.content ?? ""} onChange={(event) => setForm((current) => ({ ...current, content: event.target.value }))} /></FormField> : null}
          {form.content_type === 1 ? <FormField id="lesson-video-url" label="视频URL"><Input id="lesson-video-url" value={form.video_url ?? ""} onChange={(event) => setForm((current) => ({ ...current, video_url: event.target.value }))} /></FormField> : null}
          {form.content_type === 1 ? <FormField id="lesson-video-duration" label="视频时长（秒）"><Input id="lesson-video-duration" type="number" value={form.video_duration ?? ""} onChange={(event) => setForm((current) => ({ ...current, video_duration: event.target.value ? Number(event.target.value) : null }))} /></FormField> : null}
          {form.content_type === 4 ? <FormField id="lesson-experiment-id" label="实验模板ID"><Input id="lesson-experiment-id" value={form.experiment_id ?? ""} onChange={(event) => setForm((current) => ({ ...current, experiment_id: event.target.value }))} /></FormField> : null}
          <FormField id="lesson-estimated-minutes" label="预计学习分钟"><Input id="lesson-estimated-minutes" type="number" value={form.estimated_minutes ?? ""} onChange={(event) => setForm((current) => ({ ...current, estimated_minutes: event.target.value ? Number(event.target.value) : null }))} /></FormField>
          <div className="rounded-xl border border-dashed border-border p-4">
            <p className="font-semibold">附件校验与展示</p>
            <input
              className="mt-3 block text-sm"
              type="file"
              onChange={(event) => {
                const file = event.target.files?.[0];
                if (!file) return;
                const result = validateCourseAttachment(file, file.type.startsWith("video/") ? "video" : "document");
                setUploadError(result.error ?? null);
                setUploadProgress(0);
                if (!result.isValid) return;
                uploadMutation.mutate(
                  { file, purpose: "lesson_attachment", onProgress: setUploadProgress },
                  {
                    onSuccess: (uploaded) => {
                      mutations.uploadAttachment.mutate({
                        file_name: uploaded.file_name,
                        file_url: uploaded.file_url,
                        file_size: uploaded.file_size,
                        file_type: uploaded.file_type,
                      });
                    },
                  },
                );
              }}
            />
            {uploadError ? <p className="mt-2 text-sm text-destructive">{uploadError}</p> : null}
            {uploadProgress > 0 ? <div className="mt-3 h-2 rounded-full bg-muted"><div className="h-full rounded-full bg-primary" style={{ width: `${uploadProgress}%` }} /></div> : null}
            {query.data?.attachments.map((attachment) => (
              <a key={attachment.id} className="mt-2 block text-sm text-primary hover:underline" href={attachment.file_url}>{attachment.file_name}</a>
            ))}
          </div>
          <Button type="submit" isLoading={mutations.updateLesson.isPending}>保存课时</Button>
        </form>
      </CardContent>
    </Card>
  );
}
