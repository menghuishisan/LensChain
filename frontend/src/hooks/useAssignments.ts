"use client";

// useAssignments.ts
// 模块03作业 hook，封装作业、题目、草稿、提交、批改、成绩 Query 缓存策略。

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import {
  addQuestion,
  adjustGrade,
  createAssignment,
  deleteAssignment,
  deleteQuestion,
  exportGrades,
  getAssignment,
  getAssignmentDraft,
  getGradeConfig,
  getGradeSummary,
  getMyGrades,
  getSubmission,
  gradeSubmission,
  listAssignments,
  listMySubmissions,
  listSubmissions,
  publishAssignment,
  saveAssignmentDraft,
  setGradeConfig,
  submitAssignment,
  updateAssignment,
  updateQuestion,
} from "@/services/courseAssignment";
import type { ID } from "@/types/api";
import type { AssignmentAnswersRequest, AssignmentListParams, SubmissionListParams } from "@/types/courseAssignment";

/**
 * useAssignments 查询课程作业列表。
 */
export function useAssignments(courseID: ID, params: AssignmentListParams) {
  return useQuery({ queryKey: ["course", "assignments", courseID, params], queryFn: () => listAssignments(courseID, params), enabled: courseID.length > 0 });
}

/**
 * useAssignment 查询作业详情。
 */
export function useAssignment(assignmentID: ID) {
  return useQuery({ queryKey: ["course", "assignment", assignmentID], queryFn: () => getAssignment(assignmentID), enabled: assignmentID.length > 0 });
}

/**
 * useAssignmentMutations 作业和题目管理 mutation。
 */
export function useAssignmentMutations(courseID: ID, assignmentID?: ID) {
  const queryClient = useQueryClient();
  const refreshAssignments = () => {
    void queryClient.invalidateQueries({ queryKey: ["course", "assignments", courseID] });
    if (assignmentID) {
      void queryClient.invalidateQueries({ queryKey: ["course", "assignment", assignmentID] });
    }
  };

  return {
    createAssignment: useMutation({ mutationFn: (payload: Parameters<typeof createAssignment>[1]) => createAssignment(courseID, payload), onSuccess: refreshAssignments }),
    updateAssignment: useMutation({ mutationFn: (payload: Parameters<typeof updateAssignment>[1]) => updateAssignment(assignmentID ?? "", payload), onSuccess: refreshAssignments }),
    deleteAssignment: useMutation({ mutationFn: () => deleteAssignment(assignmentID ?? ""), onSuccess: refreshAssignments }),
    publishAssignment: useMutation({ mutationFn: () => publishAssignment(assignmentID ?? ""), onSuccess: refreshAssignments }),
    addQuestion: useMutation({ mutationFn: (payload: Parameters<typeof addQuestion>[1]) => addQuestion(assignmentID ?? "", payload), onSuccess: refreshAssignments }),
    updateQuestion: useMutation({ mutationFn: ({ questionID, payload }: { questionID: ID; payload: Parameters<typeof updateQuestion>[1] }) => updateQuestion(questionID, payload), onSuccess: refreshAssignments }),
    deleteQuestion: useMutation({ mutationFn: deleteQuestion, onSuccess: refreshAssignments }),
  };
}

/**
 * useAssignmentDraft 查询服务端作业草稿。
 */
export function useAssignmentDraft(assignmentID: ID) {
  return useQuery({ queryKey: ["course", "assignment-draft", assignmentID], queryFn: () => getAssignmentDraft(assignmentID), enabled: assignmentID.length > 0 });
}

/**
 * useSaveAssignmentDraftMutation 保存草稿，成功后刷新草稿缓存。
 */
export function useSaveAssignmentDraftMutation(assignmentID: ID) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (payload: AssignmentAnswersRequest) => saveAssignmentDraft(assignmentID, payload),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: ["course", "assignment-draft", assignmentID] }),
  });
}

/**
 * useSubmitAssignmentMutation 提交作业，成功后刷新提交记录、作业详情和草稿缓存。
 */
export function useSubmitAssignmentMutation(assignmentID: ID) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (payload: AssignmentAnswersRequest) => submitAssignment(assignmentID, payload),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["course", "my-submissions", assignmentID] });
      void queryClient.invalidateQueries({ queryKey: ["course", "assignment", assignmentID] });
      void queryClient.invalidateQueries({ queryKey: ["course", "assignment-draft", assignmentID] });
    },
  });
}

/**
 * useMySubmissions 查询我的提交记录。
 */
export function useMySubmissions(assignmentID: ID) {
  return useQuery({ queryKey: ["course", "my-submissions", assignmentID], queryFn: () => listMySubmissions(assignmentID), enabled: assignmentID.length > 0 });
}

/**
 * useSubmissions 查询教师视角提交列表。
 */
export function useSubmissions(assignmentID: ID, params: SubmissionListParams) {
  return useQuery({ queryKey: ["course", "submissions", assignmentID, params], queryFn: () => listSubmissions(assignmentID, params), enabled: assignmentID.length > 0 });
}

/**
 * useSubmission 查询提交详情。
 */
export function useSubmission(submissionID: ID) {
  return useQuery({ queryKey: ["course", "submission", submissionID], queryFn: () => getSubmission(submissionID), enabled: submissionID.length > 0 });
}

/**
 * useGradeSubmissionMutation 批改提交，成功后刷新提交详情、提交列表和成绩汇总。
 */
export function useGradeSubmissionMutation(submissionID: ID) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (payload: Parameters<typeof gradeSubmission>[1]) => gradeSubmission(submissionID, payload),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["course", "submission", submissionID] });
      void queryClient.invalidateQueries({ queryKey: ["course", "submissions"] });
      void queryClient.invalidateQueries({ queryKey: ["course", "grades"] });
    },
  });
}

/**
 * useCourseGrades 查询课程成绩配置、汇总和我的成绩。
 */
export function useCourseGrades(courseID: ID) {
  const config = useQuery({ queryKey: ["course", "grade-config", courseID], queryFn: () => getGradeConfig(courseID), enabled: courseID.length > 0 });
  const summary = useQuery({ queryKey: ["course", "grades", courseID], queryFn: () => getGradeSummary(courseID), enabled: courseID.length > 0 });
  const mine = useQuery({ queryKey: ["course", "my-grades", courseID], queryFn: () => getMyGrades(courseID), enabled: courseID.length > 0 });
  return { config, summary, mine };
}

/**
 * useGradeMutations 成绩配置、调分和导出 mutation。
 */
export function useGradeMutations(courseID: ID) {
  const queryClient = useQueryClient();
  return {
    setConfig: useMutation({
      mutationFn: (items: Parameters<typeof setGradeConfig>[1]) => setGradeConfig(courseID, items),
      onSuccess: () => {
        void queryClient.invalidateQueries({ queryKey: ["course", "grade-config", courseID] });
        void queryClient.invalidateQueries({ queryKey: ["course", "grades", courseID] });
      },
    }),
    adjust: useMutation({
      mutationFn: ({ studentID, payload }: { studentID: ID; payload: { final_score: number; reason: string } }) => adjustGrade(courseID, studentID, payload),
      onSuccess: () => void queryClient.invalidateQueries({ queryKey: ["course", "grades", courseID] }),
    }),
    exportGrades: useMutation({
      mutationFn: () => exportGrades(courseID),
      onSuccess: (result) => downloadBlob(result.blob, result.filename ?? "课程成绩单.xlsx"),
    }),
  };
}

function downloadBlob(blob: Blob, filename: string) {
  const url = URL.createObjectURL(blob);
  const anchor = document.createElement("a");
  anchor.href = url;
  anchor.download = filename;
  anchor.click();
  URL.revokeObjectURL(url);
}
