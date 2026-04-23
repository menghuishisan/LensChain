// courseAssignment.ts
// 模块03作业 service：作业、题目、草稿、提交、批改和课程成绩。

import { apiClient } from "@/lib/api-client";
import type { ID } from "@/types/api";
import type {
  AddQuestionRequest,
  AssignmentAnswersRequest,
  AssignmentDetail,
  AssignmentDraft,
  AssignmentListParams,
  AssignmentListResponse,
  CreateAssignmentRequest,
  GradeConfigItem,
  GradeConfigResponse,
  GradeSubmissionRequest,
  GradeSummaryResponse,
  MySubmissionsResponse,
  SaveAssignmentDraftResponse,
  SubmissionDetail,
  SubmissionListParams,
  SubmissionListResponse,
  SubmitAssignmentResponse,
  UpdateAssignmentRequest,
  UpdateQuestionRequest,
} from "@/types/courseAssignment";

/**
 * createAssignment 对应 POST /api/v1/courses/:id/assignments，用于创建作业。
 */
export function createAssignment(courseID: ID, payload: CreateAssignmentRequest) {
  return apiClient.post<{ id: ID }>(`/courses/${courseID}/assignments`, payload);
}

/**
 * listAssignments 对应 GET /api/v1/courses/:id/assignments，用于作业列表。
 */
export function listAssignments(courseID: ID, params: AssignmentListParams) {
  return apiClient.get<AssignmentListResponse>(`/courses/${courseID}/assignments`, { query: params });
}

/**
 * getAssignment 对应 GET /api/v1/assignments/:id，用于作业详情。
 */
export function getAssignment(assignmentID: ID) {
  return apiClient.get<AssignmentDetail>(`/assignments/${assignmentID}`);
}

/**
 * updateAssignment 对应 PUT /api/v1/assignments/:id，用于编辑作业。
 */
export function updateAssignment(assignmentID: ID, payload: UpdateAssignmentRequest) {
  return apiClient.put<null>(`/assignments/${assignmentID}`, payload);
}

/**
 * deleteAssignment 对应 DELETE /api/v1/assignments/:id，用于删除作业。
 */
export function deleteAssignment(assignmentID: ID) {
  return apiClient.delete<null>(`/assignments/${assignmentID}`);
}

/**
 * publishAssignment 对应 POST /api/v1/assignments/:id/publish，用于发布作业。
 */
export function publishAssignment(assignmentID: ID) {
  return apiClient.post<null>(`/assignments/${assignmentID}/publish`);
}

/**
 * addQuestion 对应 POST /api/v1/assignments/:id/questions，用于添加题目。
 */
export function addQuestion(assignmentID: ID, payload: AddQuestionRequest) {
  return apiClient.post<{ id: ID }>(`/assignments/${assignmentID}/questions`, payload);
}

/**
 * updateQuestion 对应 PUT /api/v1/assignment-questions/:id，用于编辑题目。
 */
export function updateQuestion(questionID: ID, payload: UpdateQuestionRequest) {
  return apiClient.put<null>(`/assignment-questions/${questionID}`, payload);
}

/**
 * deleteQuestion 对应 DELETE /api/v1/assignment-questions/:id，用于删除题目。
 */
export function deleteQuestion(questionID: ID) {
  return apiClient.delete<null>(`/assignment-questions/${questionID}`);
}

/**
 * saveAssignmentDraft 对应 PUT /api/v1/assignments/:id/draft，用于保存作答草稿。
 */
export function saveAssignmentDraft(assignmentID: ID, payload: AssignmentAnswersRequest) {
  return apiClient.put<SaveAssignmentDraftResponse>(`/assignments/${assignmentID}/draft`, payload);
}

/**
 * getAssignmentDraft 对应 GET /api/v1/assignments/:id/draft，用于读取作答草稿。
 */
export function getAssignmentDraft(assignmentID: ID) {
  return apiClient.get<AssignmentDraft | null>(`/assignments/${assignmentID}/draft`);
}

/**
 * submitAssignment 对应 POST /api/v1/assignments/:id/submit，用于正式提交作业。
 */
export function submitAssignment(assignmentID: ID, payload: AssignmentAnswersRequest) {
  return apiClient.post<SubmitAssignmentResponse>(`/assignments/${assignmentID}/submit`, payload);
}

/**
 * listMySubmissions 对应 GET /api/v1/assignments/:id/my-submissions，用于我的提交记录。
 */
export function listMySubmissions(assignmentID: ID) {
  return apiClient.get<MySubmissionsResponse>(`/assignments/${assignmentID}/my-submissions`);
}

/**
 * listSubmissions 对应 GET /api/v1/assignments/:id/submissions，用于教师查看提交列表。
 */
export function listSubmissions(assignmentID: ID, params: SubmissionListParams) {
  return apiClient.get<SubmissionListResponse>(`/assignments/${assignmentID}/submissions`, { query: params });
}

/**
 * getSubmission 对应 GET /api/v1/submissions/:id，用于提交详情。
 */
export function getSubmission(submissionID: ID) {
  return apiClient.get<SubmissionDetail>(`/submissions/${submissionID}`);
}

/**
 * gradeSubmission 对应 POST /api/v1/submissions/:id/grade，用于教师批改提交。
 */
export function gradeSubmission(submissionID: ID, payload: GradeSubmissionRequest) {
  return apiClient.post<null>(`/submissions/${submissionID}/grade`, payload);
}

/**
 * setGradeConfig 对应 PUT /api/v1/courses/:id/grade-config，用于设置成绩权重。
 */
export function setGradeConfig(courseID: ID, items: GradeConfigItem[]) {
  return apiClient.put<null>(`/courses/${courseID}/grade-config`, { items });
}

/**
 * getGradeConfig 对应 GET /api/v1/courses/:id/grade-config，用于获取成绩权重。
 */
export function getGradeConfig(courseID: ID) {
  return apiClient.get<GradeConfigResponse>(`/courses/${courseID}/grade-config`);
}

/**
 * getGradeSummary 对应 GET /api/v1/courses/:id/grades，用于教师成绩汇总。
 */
export function getGradeSummary(courseID: ID) {
  return apiClient.get<GradeSummaryResponse>(`/courses/${courseID}/grades`);
}

/**
 * adjustGrade 对应 PATCH /api/v1/courses/:id/grades/:student_id，用于手动调分。
 */
export function adjustGrade(courseID: ID, studentID: ID, payload: { final_score: number; reason: string }) {
  return apiClient.patch<null>(`/courses/${courseID}/grades/${studentID}`, payload);
}

/**
 * exportGrades 对应 GET /api/v1/courses/:id/grades/export，用于下载成绩单。
 */
export function exportGrades(courseID: ID) {
  return apiClient.download(`/courses/${courseID}/grades/export`);
}

/**
 * getMyGrades 对应 GET /api/v1/courses/:id/my-grades，用于学生查看课程成绩。
 */
export function getMyGrades(courseID: ID) {
  return apiClient.get<Omit<GradeSummaryResponse, "students"> & { scores: Record<ID, number>; weighted_total: number; final_score: number; is_adjusted: boolean }>(`/courses/${courseID}/my-grades`);
}
