"use client";

// useCourses.ts
// 模块03课程 hook，封装课程列表、详情、生命周期、选课、课表、共享和统计缓存策略。

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import {
  archiveCourse,
  addStudentToCourse,
  batchAddStudentsToCourse,
  cloneCourse,
  createCourse,
  deleteCourse,
  endCourse,
  exportCourseStatistics,
  getAssignmentStatistics,
  getCourse,
  getCourseOverview,
  getCourseSchedule,
  getMyProgress,
  getMySchedule,
  getSharedCourse,
  joinCourse,
  listCourseStudents,
  listCourses,
  listMyCourses,
  listSharedCourses,
  listStudentsProgress,
  publishCourse,
  refreshInviteCode,
  removeStudentFromCourse,
  setCourseSchedule,
  setCourseShare,
  updateCourse,
} from "@/services/course";
import type { ID } from "@/types/api";
import type { CourseListParams, CourseStudentListParams, MyCourseListParams } from "@/types/course";

/**
 * coursesQueryKey 课程列表 Query key。
 */
export function coursesQueryKey(params: CourseListParams) {
  return ["course", "list", params] as const;
}

/**
 * courseDetailQueryKey 课程详情 Query key。
 */
export function courseDetailQueryKey(courseID: ID) {
  return ["course", "detail", courseID] as const;
}

/**
 * useCourses 查询教师课程列表。
 */
export function useCourses(params: CourseListParams) {
  return useQuery({ queryKey: coursesQueryKey(params), queryFn: () => listCourses(params) });
}

/**
 * useCourse 查询课程详情。
 */
export function useCourse(courseID: ID) {
  return useQuery({ queryKey: courseDetailQueryKey(courseID), queryFn: () => getCourse(courseID), enabled: courseID.length > 0 });
}

/**
 * useCreateCourseMutation 创建课程，成功后刷新课程列表。
 */
export function useCreateCourseMutation() {
  const queryClient = useQueryClient();
  return useMutation({ mutationFn: createCourse, onSuccess: () => void queryClient.invalidateQueries({ queryKey: ["course"] }) });
}

/**
 * useUpdateCourseMutation 编辑课程，成功后刷新课程详情和列表。
 */
export function useUpdateCourseMutation(courseID: ID) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (payload: Parameters<typeof updateCourse>[1]) => updateCourse(courseID, payload),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["course"] });
      void queryClient.invalidateQueries({ queryKey: courseDetailQueryKey(courseID) });
    },
  });
}

/**
 * useCourseLifecycleMutations 课程发布、结束、归档、删除、克隆、共享和邀请码操作。
 */
export function useCourseLifecycleMutations(courseID: ID) {
  const queryClient = useQueryClient();
  const refreshCourse = () => {
    void queryClient.invalidateQueries({ queryKey: ["course"] });
    void queryClient.invalidateQueries({ queryKey: courseDetailQueryKey(courseID) });
  };

  return {
    publish: useMutation({ mutationFn: () => publishCourse(courseID), onSuccess: refreshCourse }),
    end: useMutation({ mutationFn: () => endCourse(courseID), onSuccess: refreshCourse }),
    archive: useMutation({ mutationFn: () => archiveCourse(courseID), onSuccess: refreshCourse }),
    remove: useMutation({ mutationFn: () => deleteCourse(courseID), onSuccess: refreshCourse }),
    clone: useMutation({ mutationFn: () => cloneCourse(courseID), onSuccess: () => void queryClient.invalidateQueries({ queryKey: ["course"] }) }),
    share: useMutation({ mutationFn: (isShared: boolean) => setCourseShare(courseID, isShared), onSuccess: refreshCourse }),
    refreshInvite: useMutation({ mutationFn: () => refreshInviteCode(courseID), onSuccess: refreshCourse }),
  };
}

/**
 * useJoinCourseMutation 通过邀请码加入课程，成功后刷新我的课程。
 */
export function useJoinCourseMutation() {
  const queryClient = useQueryClient();
  return useMutation({ mutationFn: joinCourse, onSuccess: () => void queryClient.invalidateQueries({ queryKey: ["course", "my"] }) });
}

/**
 * useMyCourses 查询学生我的课程。
 */
export function useMyCourses(params: MyCourseListParams) {
  return useQuery({ queryKey: ["course", "my", params], queryFn: () => listMyCourses(params) });
}

/**
 * useCourseStudents 查询课程学生列表。
 */
export function useCourseStudents(courseID: ID, params: CourseStudentListParams) {
  return useQuery({ queryKey: ["course", "students", courseID, params], queryFn: () => listCourseStudents(courseID, params) });
}

/**
 * useRemoveCourseStudentMutation 移除学生，成功后刷新学生列表和进度。
 */
export function useRemoveCourseStudentMutation(courseID: ID) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (studentID: ID) => removeStudentFromCourse(courseID, studentID),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["course", "students", courseID] });
      void queryClient.invalidateQueries({ queryKey: ["course", "students-progress", courseID] });
    },
  });
}

/**
 * useAddCourseStudentMutation 添加单个学生，成功后刷新学生列表。
 */
export function useAddCourseStudentMutation(courseID: ID) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (studentID: ID) => addStudentToCourse(courseID, studentID),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: ["course", "students", courseID] }),
  });
}

/**
 * useBatchAddCourseStudentsMutation 批量添加学生，成功后刷新学生列表。
 */
export function useBatchAddCourseStudentsMutation(courseID: ID) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (studentIDs: ID[]) => batchAddStudentsToCourse(courseID, studentIDs),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: ["course", "students", courseID] }),
  });
}

/**
 * useCourseProgress 查询当前学生课程进度。
 */
export function useCourseProgress(courseID: ID) {
  return useQuery({ queryKey: ["course", "my-progress", courseID], queryFn: () => getMyProgress(courseID), enabled: courseID.length > 0 });
}

/**
 * useStudentsProgress 查询教师视角学生进度。
 */
export function useStudentsProgress(courseID: ID, params: CourseStudentListParams) {
  return useQuery({ queryKey: ["course", "students-progress", courseID, params], queryFn: () => listStudentsProgress(courseID, params) });
}

/**
 * useCourseSchedule 查询课程表。
 */
export function useCourseSchedule(courseID: ID) {
  return useQuery({ queryKey: ["course", "schedule", courseID], queryFn: () => getCourseSchedule(courseID), enabled: courseID.length > 0 });
}

/**
 * useSetCourseScheduleMutation 设置课程表，成功后刷新课程表和我的课程表。
 */
export function useSetCourseScheduleMutation(courseID: ID) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (payload: Parameters<typeof setCourseSchedule>[1]) => setCourseSchedule(courseID, payload),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["course", "schedule", courseID] });
      void queryClient.invalidateQueries({ queryKey: ["course", "my-schedule"] });
    },
  });
}

/**
 * useMySchedule 查询我的周课程表。
 */
export function useMySchedule() {
  return useQuery({ queryKey: ["course", "my-schedule"], queryFn: getMySchedule });
}

/**
 * useSharedCourses 查询共享课程库。
 */
export function useSharedCourses(params: CourseListParams) {
  return useQuery({ queryKey: ["course", "shared", params], queryFn: () => listSharedCourses(params) });
}

/**
 * useSharedCourse 查询共享课程详情。
 */
export function useSharedCourse(courseID: ID) {
  return useQuery({ queryKey: ["course", "shared", courseID], queryFn: () => getSharedCourse(courseID), enabled: courseID.length > 0 });
}

/**
 * useCourseStatistics 查询课程整体统计和作业统计。
 */
export function useCourseStatistics(courseID: ID) {
  const overview = useQuery({ queryKey: ["course", "statistics", "overview", courseID], queryFn: () => getCourseOverview(courseID), enabled: courseID.length > 0 });
  const assignments = useQuery({ queryKey: ["course", "statistics", "assignments", courseID], queryFn: () => getAssignmentStatistics(courseID), enabled: courseID.length > 0 });
  return { overview, assignments };
}

/**
 * useExportCourseStatisticsMutation 下载课程统计报告。
 */
export function useExportCourseStatisticsMutation(courseID: ID) {
  return useMutation({
    mutationFn: () => exportCourseStatistics(courseID),
    onSuccess: (result) => downloadBlob(result.blob, result.filename ?? "课程统计报告.xlsx"),
  });
}

function downloadBlob(blob: Blob, filename: string) {
  const url = URL.createObjectURL(blob);
  const anchor = document.createElement("a");
  anchor.href = url;
  anchor.download = filename;
  anchor.click();
  URL.revokeObjectURL(url);
}
