// grade.ts
// 模块06评测与成绩 service：学期、等级映射、审核、GPA、学习概览、申诉、预警、成绩单和分析接口。

import { apiClient } from "@/lib/api-client";
import type { ID, PaginatedData, QueryParams } from "@/types/api";
import type {
  AcademicWarningDetail,
  AcademicWarningListParams,
  AcademicWarningListResponse,
  ApproveGradeAppealRequest,
  CourseGradeAnalytics,
  CreateGradeAppealRequest,
  GPAResponse,
  GenerateTranscriptRequest,
  GradeAppealDetail,
  GradeAppealItem,
  GradeAppealListParams,
  GradeLevelConfig,
  GradeReviewDetail,
  GradeReviewItem,
  GradeReviewListParams,
  GradeReviewStatusResponse,
  GradeSemester,
  GradeSemesterListResponse,
  HandleAcademicWarningRequest,
  LearningOverview,
  PlatformGradeAnalytics,
  RejectGradeAppealRequest,
  ReviewHandleRequest,
  SchoolGradeAnalytics,
  SemesterGradesResponse,
  SemesterListParams,
  SemesterRequest,
  SubmitGradeReviewRequest,
  TranscriptListParams,
  TranscriptListResponse,
  TranscriptResponse,
  UnlockGradeReviewRequest,
  UpdateWarningConfigRequest,
  WarningConfig,
} from "@/types/grade";

/**
 * createSemester 对应 POST /api/v1/grades/semesters，用于创建学期。
 */
export function createSemester(payload: SemesterRequest) {
  return apiClient.post<GradeSemester>("/grades/semesters", payload);
}

/**
 * listSemesters 对应 GET /api/v1/grades/semesters，用于学期列表。
 */
export function listSemesters(params: SemesterListParams) {
  return apiClient.get<GradeSemesterListResponse>("/grades/semesters", { query: params });
}

/**
 * updateSemester 对应 PUT /api/v1/grades/semesters/:id，用于更新学期。
 */
export function updateSemester(semesterID: ID, payload: SemesterRequest) {
  return apiClient.put<null>(`/grades/semesters/${semesterID}`, payload);
}

/**
 * deleteSemester 对应 DELETE /api/v1/grades/semesters/:id，用于删除学期。
 */
export function deleteSemester(semesterID: ID) {
  return apiClient.delete<null>(`/grades/semesters/${semesterID}`);
}

/**
 * setCurrentSemester 对应 PATCH /api/v1/grades/semesters/:id/set-current，用于设为当前学期。
 */
export function setCurrentSemester(semesterID: ID) {
  return apiClient.patch<null>(`/grades/semesters/${semesterID}/set-current`);
}

/**
 * getLevelConfigs 对应 GET /api/v1/grades/level-configs，用于获取等级映射配置。
 */
export function getLevelConfigs() {
  return apiClient.get<GradeLevelConfig>("/grades/level-configs", { query: {} });
}

/**
 * updateLevelConfigs 对应 PUT /api/v1/grades/level-configs，用于更新等级映射配置。
 */
export function updateLevelConfigs(levels: GradeLevelConfig["levels"]) {
  return apiClient.put<null>("/grades/level-configs", { levels });
}

/**
 * resetDefaultLevelConfigs 对应 POST /api/v1/grades/level-configs/reset-default，用于恢复默认等级映射。
 */
export function resetDefaultLevelConfigs() {
  return apiClient.post<GradeLevelConfig>("/grades/level-configs/reset-default");
}

/**
 * submitGradeReview 对应 POST /api/v1/grades/reviews，用于教师提交成绩审核。
 */
export function submitGradeReview(payload: SubmitGradeReviewRequest) {
  return apiClient.post<GradeReviewStatusResponse>("/grades/reviews", payload);
}

/**
 * listGradeReviews 对应 GET /api/v1/grades/reviews，用于审核列表。
 */
export function listGradeReviews(params: GradeReviewListParams) {
  return apiClient.get<PaginatedData<GradeReviewItem>>("/grades/reviews", { query: params });
}

/**
 * getGradeReview 对应 GET /api/v1/grades/reviews/:id，用于审核详情。
 */
export function getGradeReview(reviewID: ID) {
  return apiClient.get<GradeReviewDetail>(`/grades/reviews/${reviewID}`);
}

/**
 * approveGradeReview 对应 POST /api/v1/grades/reviews/:id/approve，用于审核通过。
 */
export function approveGradeReview(reviewID: ID, payload: ReviewHandleRequest) {
  return apiClient.post<null>(`/grades/reviews/${reviewID}/approve`, payload);
}

/**
 * rejectGradeReview 对应 POST /api/v1/grades/reviews/:id/reject，用于审核驳回。
 */
export function rejectGradeReview(reviewID: ID, payload: ReviewHandleRequest) {
  return apiClient.post<null>(`/grades/reviews/${reviewID}/reject`, payload);
}

/**
 * unlockGradeReview 对应 POST /api/v1/grades/reviews/:id/unlock，用于解锁成绩。
 */
export function unlockGradeReview(reviewID: ID, payload: UnlockGradeReviewRequest) {
  return apiClient.post<null>(`/grades/reviews/${reviewID}/unlock`, payload);
}

/**
 * getStudentSemesterGrades 对应 GET /api/v1/grades/students/:id/semester-grades，用于查看学生学期成绩。
 */
export function getStudentSemesterGrades(studentID: ID, params: QueryParams = {}) {
  return apiClient.get<SemesterGradesResponse>(`/grades/students/${studentID}/semester-grades`, { query: params });
}

/**
 * getStudentGPA 对应 GET /api/v1/grades/students/:id/gpa，用于查看学生GPA。
 */
export function getStudentGPA(studentID: ID) {
  return apiClient.get<GPAResponse>(`/grades/students/${studentID}/gpa`, { query: {} });
}

/**
 * getMySemesterGrades 对应 GET /api/v1/grades/my/semester-grades，用于学生查看我的学期成绩。
 */
export function getMySemesterGrades(params: QueryParams = {}) {
  return apiClient.get<SemesterGradesResponse>("/grades/my/semester-grades", { query: params });
}

/**
 * getMyGPA 对应 GET /api/v1/grades/my/gpa，用于学生查看我的GPA。
 */
export function getMyGPA() {
  return apiClient.get<GPAResponse>("/grades/my/gpa", { query: {} });
}

/**
 * getMyLearningOverview 对应 GET /api/v1/grades/my/learning-overview，用于个人中心学习概览。
 * 注意：该接口属于模块06聚合层，不可写进 auth service。
 */
export function getMyLearningOverview() {
  return apiClient.get<LearningOverview>("/grades/my/learning-overview", { query: {} });
}

/**
 * createGradeAppeal 对应 POST /api/v1/grades/appeals，用于学生提交成绩申诉。
 */
export function createGradeAppeal(payload: CreateGradeAppealRequest) {
  return apiClient.post<GradeAppealItem>("/grades/appeals", payload);
}

/**
 * listGradeAppeals 对应 GET /api/v1/grades/appeals，用于申诉列表。
 */
export function listGradeAppeals(params: GradeAppealListParams) {
  return apiClient.get<PaginatedData<GradeAppealItem>>("/grades/appeals", { query: params });
}

/**
 * getGradeAppeal 对应 GET /api/v1/grades/appeals/:id，用于申诉详情。
 */
export function getGradeAppeal(appealID: ID) {
  return apiClient.get<GradeAppealDetail>(`/grades/appeals/${appealID}`);
}

/**
 * approveGradeAppeal 对应 POST /api/v1/grades/appeals/:id/approve，用于教师同意申诉。
 */
export function approveGradeAppeal(appealID: ID, payload: ApproveGradeAppealRequest) {
  return apiClient.post<null>(`/grades/appeals/${appealID}/approve`, payload);
}

/**
 * rejectGradeAppeal 对应 POST /api/v1/grades/appeals/:id/reject，用于教师驳回申诉。
 */
export function rejectGradeAppeal(appealID: ID, payload: RejectGradeAppealRequest) {
  return apiClient.post<null>(`/grades/appeals/${appealID}/reject`, payload);
}

/**
 * listAcademicWarnings 对应 GET /api/v1/grades/warnings，用于学业预警列表。
 */
export function listAcademicWarnings(params: AcademicWarningListParams) {
  return apiClient.get<AcademicWarningListResponse>("/grades/warnings", { query: params });
}

/**
 * getAcademicWarning 对应 GET /api/v1/grades/warnings/:id，用于预警详情。
 */
export function getAcademicWarning(warningID: ID) {
  return apiClient.get<AcademicWarningDetail>(`/grades/warnings/${warningID}`);
}

/**
 * handleAcademicWarning 对应 POST /api/v1/grades/warnings/:id/handle，用于处理预警。
 */
export function handleAcademicWarning(warningID: ID, payload: HandleAcademicWarningRequest) {
  return apiClient.post<null>(`/grades/warnings/${warningID}/handle`, payload);
}

/**
 * getWarningConfig 对应 GET /api/v1/grades/warning-configs，用于获取预警配置。
 */
export function getWarningConfig() {
  return apiClient.get<WarningConfig>("/grades/warning-configs", { query: {} });
}

/**
 * updateWarningConfig 对应 PUT /api/v1/grades/warning-configs，用于更新预警配置。
 */
export function updateWarningConfig(payload: UpdateWarningConfigRequest) {
  return apiClient.put<null>("/grades/warning-configs", payload);
}

/**
 * generateTranscript 对应 POST /api/v1/grades/transcripts/generate，用于生成成绩单。
 */
export function generateTranscript(payload: GenerateTranscriptRequest) {
  return apiClient.post<TranscriptResponse>("/grades/transcripts/generate", payload);
}

/**
 * listTranscripts 对应 GET /api/v1/grades/transcripts，用于成绩单列表。
 */
export function listTranscripts(params: TranscriptListParams = {}) {
  return apiClient.get<TranscriptListResponse>("/grades/transcripts", { query: params });
}

/**
 * downloadTranscript 对应 GET /api/v1/grades/transcripts/:id/download，用于下载成绩单。
 * 成绩单下载必须走后端 URL，不自行拼接对象存储地址。
 */
export function downloadTranscript(transcriptID: ID) {
  return apiClient.download(`/grades/transcripts/${transcriptID}/download`);
}

/**
 * getCourseGradeAnalytics 对应 GET /api/v1/grades/analytics/course/:id，用于课程成绩分析。
 */
export function getCourseGradeAnalytics(courseID: ID, params: QueryParams = {}) {
  return apiClient.get<CourseGradeAnalytics>(`/grades/analytics/course/${courseID}`, { query: params });
}

/**
 * getSchoolGradeAnalytics 对应 GET /api/v1/grades/analytics/school，用于全校成绩分析。
 */
export function getSchoolGradeAnalytics(params: QueryParams = {}) {
  return apiClient.get<SchoolGradeAnalytics>("/grades/analytics/school", { query: params });
}

/**
 * getPlatformGradeAnalytics 对应 GET /api/v1/grades/analytics/platform，用于平台成绩总览。
 */
export function getPlatformGradeAnalytics() {
  return apiClient.get<PlatformGradeAnalytics>("/grades/analytics/platform", { query: {} });
}
