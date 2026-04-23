"use client";

// useGrades.ts
// 模块06成绩 hook，封装学期、等级映射、我的成绩、GPA、学习概览、成绩单和分析缓存策略。

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import {
  createSemester,
  deleteSemester,
  downloadTranscript,
  generateTranscript,
  getCourseGradeAnalytics,
  getLevelConfigs,
  getMyGPA,
  getMyLearningOverview,
  getMySemesterGrades,
  getPlatformGradeAnalytics,
  getSchoolGradeAnalytics,
  getStudentGPA,
  getStudentSemesterGrades,
  listSemesters,
  listTranscripts,
  resetDefaultLevelConfigs,
  setCurrentSemester,
  updateLevelConfigs,
  updateSemester,
} from "@/services/grade";
import { downloadBlob } from "@/lib/grade";
import type { ID, QueryParams } from "@/types/api";
import type { GenerateTranscriptRequest, SemesterListParams, SemesterRequest, TranscriptListParams } from "@/types/grade";

/**
 * useSemesters 查询学期列表。
 */
export function useSemesters(params: SemesterListParams) {
  return useQuery({ queryKey: ["grade", "semesters", params], queryFn: () => listSemesters(params) });
}

/**
 * useSemesterMutations 创建、编辑、删除和设为当前学期。
 */
export function useSemesterMutations() {
  const queryClient = useQueryClient();
  const refreshSemesters = () => void queryClient.invalidateQueries({ queryKey: ["grade", "semesters"] });

  return {
    create: useMutation({ mutationFn: (payload: SemesterRequest) => createSemester(payload), onSuccess: refreshSemesters }),
    update: useMutation({ mutationFn: ({ semesterID, payload }: { semesterID: ID; payload: SemesterRequest }) => updateSemester(semesterID, payload), onSuccess: refreshSemesters }),
    remove: useMutation({ mutationFn: deleteSemester, onSuccess: refreshSemesters }),
    setCurrent: useMutation({ mutationFn: setCurrentSemester, onSuccess: refreshSemesters }),
  };
}

/**
 * useLevelConfigs 查询等级映射配置。
 */
export function useLevelConfigs() {
  return useQuery({ queryKey: ["grade", "level-configs"], queryFn: getLevelConfigs });
}

/**
 * useLevelConfigMutations 更新和重置等级映射配置。
 */
export function useLevelConfigMutations() {
  const queryClient = useQueryClient();
  return {
    update: useMutation({
      mutationFn: updateLevelConfigs,
      onSuccess: () => {
        void queryClient.invalidateQueries({ queryKey: ["grade", "level-configs"] });
        void queryClient.invalidateQueries({ queryKey: ["grade", "gpa"] });
      },
    }),
    resetDefault: useMutation({
      mutationFn: resetDefaultLevelConfigs,
      onSuccess: () => {
        void queryClient.invalidateQueries({ queryKey: ["grade", "level-configs"] });
        void queryClient.invalidateQueries({ queryKey: ["grade", "gpa"] });
      },
    }),
  };
}

/**
 * useMySemesterGrades 查询学生我的学期成绩。
 */
export function useMySemesterGrades(params: QueryParams = {}) {
  return useQuery({ queryKey: ["grade", "my-semester-grades", params], queryFn: () => getMySemesterGrades(params) });
}

/**
 * useStudentSemesterGrades 查询指定学生学期成绩。
 */
export function useStudentSemesterGrades(studentID: ID, params: QueryParams = {}) {
  return useQuery({ queryKey: ["grade", "student-semester-grades", studentID, params], queryFn: () => getStudentSemesterGrades(studentID, params), enabled: studentID.length > 0 });
}

/**
 * useMyGPA 查询学生我的 GPA。
 */
export function useMyGPA() {
  return useQuery({ queryKey: ["grade", "my-gpa"], queryFn: getMyGPA });
}

/**
 * useStudentGPA 查询指定学生 GPA。
 */
export function useStudentGPA(studentID: ID) {
  return useQuery({ queryKey: ["grade", "student-gpa", studentID], queryFn: () => getStudentGPA(studentID), enabled: studentID.length > 0 });
}

/**
 * useMyLearningOverview 查询个人中心学习概览。
 * 注意：该数据明确来自模块06 grade service，不走 auth service。
 */
export function useMyLearningOverview() {
  return useQuery({ queryKey: ["grade", "my-learning-overview"], queryFn: getMyLearningOverview });
}

/**
 * useTranscripts 查询成绩单列表。
 */
export function useTranscripts(params: TranscriptListParams = {}) {
  return useQuery({ queryKey: ["grade", "transcripts", params], queryFn: () => listTranscripts(params) });
}

/**
 * useTranscriptMutations 生成和下载成绩单。
 */
export function useTranscriptMutations() {
  const queryClient = useQueryClient();
  return {
    generate: useMutation({
      mutationFn: (payload: GenerateTranscriptRequest) => generateTranscript(payload),
      onSuccess: () => void queryClient.invalidateQueries({ queryKey: ["grade", "transcripts"] }),
    }),
    download: useMutation({
      mutationFn: (transcriptID: ID) => downloadTranscript(transcriptID),
      onSuccess: (result) => downloadBlob(result.blob, result.filename ?? "成绩单.pdf"),
    }),
  };
}

/**
 * useCourseGradeAnalytics 查询课程成绩分析。
 */
export function useCourseGradeAnalytics(courseID: ID, params: QueryParams = {}) {
  return useQuery({ queryKey: ["grade", "analytics", "course", courseID, params], queryFn: () => getCourseGradeAnalytics(courseID, params), enabled: courseID.length > 0 });
}

/**
 * useSchoolGradeAnalytics 查询全校成绩分析。
 */
export function useSchoolGradeAnalytics(params: QueryParams = {}) {
  return useQuery({ queryKey: ["grade", "analytics", "school", params], queryFn: () => getSchoolGradeAnalytics(params) });
}

/**
 * usePlatformGradeAnalytics 查询平台成绩总览。
 */
export function usePlatformGradeAnalytics() {
  return useQuery({ queryKey: ["grade", "analytics", "platform"], queryFn: getPlatformGradeAnalytics });
}
