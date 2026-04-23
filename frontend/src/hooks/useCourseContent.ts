"use client";

// useCourseContent.ts
// 模块03课程内容 hook，封装章节、课时、附件和学习进度 mutation。

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import {
  createChapter,
  createLesson,
  deleteChapter,
  deleteLesson,
  deleteLessonAttachment,
  getLesson,
  listChapters,
  sortChapters,
  sortLessons,
  updateChapter,
  updateLesson,
  updateLessonProgress,
  uploadCourseFile,
  uploadLessonAttachment,
} from "@/services/course";
import type { ID } from "@/types/api";

/**
 * useCourseChapters 查询课程章节课时目录树。
 */
export function useCourseChapters(courseID: ID) {
  return useQuery({ queryKey: ["course", "chapters", courseID], queryFn: () => listChapters(courseID), enabled: courseID.length > 0 });
}

/**
 * useLesson 查询课时详情。
 */
export function useLesson(lessonID: ID) {
  return useQuery({ queryKey: ["course", "lesson", lessonID], queryFn: () => getLesson(lessonID), enabled: lessonID.length > 0 });
}

/**
 * useCourseContentMutations 章节、课时、附件相关 mutation，成功后刷新目录和课时详情。
 */
export function useCourseContentMutations(courseID: ID, chapterID?: ID, lessonID?: ID) {
  const queryClient = useQueryClient();
  const refreshContent = () => {
    void queryClient.invalidateQueries({ queryKey: ["course", "chapters", courseID] });
    if (lessonID) {
      void queryClient.invalidateQueries({ queryKey: ["course", "lesson", lessonID] });
    }
  };

  return {
    createChapter: useMutation({ mutationFn: (payload: Parameters<typeof createChapter>[1]) => createChapter(courseID, payload), onSuccess: refreshContent }),
    updateChapter: useMutation({ mutationFn: ({ id, payload }: { id: ID; payload: Parameters<typeof updateChapter>[1] }) => updateChapter(id, payload), onSuccess: refreshContent }),
    deleteChapter: useMutation({ mutationFn: deleteChapter, onSuccess: refreshContent }),
    sortChapters: useMutation({ mutationFn: (ids: ID[]) => sortChapters(courseID, { ids }), onSuccess: refreshContent }),
    createLesson: useMutation({ mutationFn: (payload: Parameters<typeof createLesson>[1]) => createLesson(chapterID ?? "", payload), onSuccess: refreshContent }),
    updateLesson: useMutation({ mutationFn: (payload: Parameters<typeof updateLesson>[1]) => updateLesson(lessonID ?? "", payload), onSuccess: refreshContent }),
    deleteLesson: useMutation({ mutationFn: deleteLesson, onSuccess: refreshContent }),
    sortLessons: useMutation({ mutationFn: (ids: ID[]) => sortLessons(chapterID ?? "", { ids }), onSuccess: refreshContent }),
    uploadAttachment: useMutation({ mutationFn: (payload: Parameters<typeof uploadLessonAttachment>[1]) => uploadLessonAttachment(lessonID ?? "", payload), onSuccess: refreshContent }),
    deleteAttachment: useMutation({ mutationFn: deleteLessonAttachment, onSuccess: refreshContent }),
    updateProgress: useMutation({
      mutationFn: (payload: Parameters<typeof updateLessonProgress>[1]) => updateLessonProgress(lessonID ?? "", payload),
      onSuccess: () => {
        refreshContent();
        void queryClient.invalidateQueries({ queryKey: ["course", "my-progress"] });
      },
    }),
  };
}

/**
 * useUploadCourseFileMutation 上传课程文件并暴露真实上传进度。
 */
export function useUploadCourseFileMutation() {
  return useMutation({
    mutationFn: ({ file, purpose, onProgress }: { file: File; purpose: Parameters<typeof uploadCourseFile>[1]; onProgress?: (progress: number) => void }) =>
      uploadCourseFile(file, purpose, onProgress),
  });
}
