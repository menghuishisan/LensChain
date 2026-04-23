// grade.ts
// 模块06评测与成绩类型定义：学期、等级映射、成绩审核、GPA、学习概览、申诉、预警、成绩单和分析。

import type { ID, PaginatedData, QueryParams } from "@/types/api";

/** 成绩审核状态：1未提交 2待审核 3已通过 4已驳回。 */
export type GradeReviewStatus = 1 | 2 | 3 | 4;

/** 成绩申诉状态：1待处理 2已同意 3已驳回。 */
export type GradeAppealStatus = 1 | 2 | 3;

/** 学业预警状态：1待处理 2已处理 3已解除。 */
export type AcademicWarningStatus = 1 | 2 | 3;

/** 学业预警类型：1低GPA 2连续挂科。 */
export type AcademicWarningType = 1 | 2;

/** 学期请求。 */
export interface SemesterRequest {
  name: string;
  code: string;
  start_date: string;
  end_date: string;
}

/** 学期列表查询参数。 */
export interface SemesterListParams extends QueryParams {
  page?: number;
  page_size?: number;
  sort_by?: string;
  sort_order?: "asc" | "desc";
}

/** 学期审核汇总。 */
export interface ReviewStatusSummaryCounts {
  not_submitted: number;
  pending: number;
  approved: number;
  rejected: number;
}

/** 学期信息。 */
export interface GradeSemester {
  id: ID;
  school_id?: ID | null;
  name: string;
  code: string;
  start_date: string;
  end_date: string;
  is_current: boolean;
  created_at?: string | null;
  course_count?: number | null;
  review_status_summary?: ReviewStatusSummaryCounts | null;
}

/** 学期列表响应。 */
export type GradeSemesterListResponse = PaginatedData<GradeSemester>;

/** 等级映射项。 */
export interface GradeLevelItem {
  id?: ID | null;
  level_name: string;
  min_score: number;
  max_score: number;
  gpa_point: number;
  sort_order?: number | null;
}

/** 等级映射配置。 */
export interface GradeLevelConfig {
  school_id: ID;
  levels: GradeLevelItem[];
}

/** 成绩审核提交请求。 */
export interface SubmitGradeReviewRequest {
  course_id: ID;
  semester_id: ID;
  submit_note?: string | null;
}

/** 成绩审核列表查询参数。 */
export interface GradeReviewListParams extends QueryParams {
  page?: number;
  page_size?: number;
  status?: GradeReviewStatus;
  semester_id?: ID;
  course_id?: ID;
}

/** 成绩审核列表项。 */
export interface GradeReviewItem {
  id: ID;
  course_id: ID;
  course_name: string;
  semester_id: ID;
  semester_name: string;
  submitted_by: ID;
  submitted_by_name: string;
  status: GradeReviewStatus;
  status_text: string;
  submitted_at: string | null;
  reviewed_at: string | null;
  is_locked: boolean;
}

/** 成绩审核详情。 */
export interface GradeReviewDetail {
  id: ID;
  course_id: ID;
  course_name: string;
  semester_id: ID;
  semester_name: string;
  submitted_by: ID;
  submitted_by_name: string;
  status: GradeReviewStatus;
  status_text: string;
  submit_note: string | null;
  submitted_at: string | null;
  reviewed_by: ID | null;
  reviewed_by_name: string | null;
  reviewed_at: string | null;
  review_comment: string | null;
  is_locked: boolean;
  locked_at: string | null;
  unlocked_by: ID | null;
  unlocked_at: string | null;
  unlock_reason: string | null;
  grade_rows?: Array<{ grade_id: ID; student_id: ID; student_name: string; student_no: string; credits: number; final_score: number; grade_level: string; gpa_point: number; is_adjusted: boolean }>;
  distribution?: Record<string, number>;
}

/** 成绩审核状态响应。 */
export interface GradeReviewStatusResponse {
  id: ID;
  course_id: ID;
  semester_id: ID;
  status: GradeReviewStatus;
  status_text: string;
  submitted_at?: string | null;
}

/** 审核处理请求。 */
export interface ReviewHandleRequest {
  review_comment: string;
}

/** 解锁成绩请求。 */
export interface UnlockGradeReviewRequest {
  unlock_reason: string;
}

/** 学期信息。 */
export interface GradeSemesterInfo {
  id: ID;
  name: string;
  code: string;
}

/** 学期成绩项。 */
export interface SemesterGradeItem {
  grade_id: ID;
  course_id: ID;
  course_name: string;
  teacher_name: string;
  credits: number;
  final_score: number;
  grade_level: string;
  gpa_point: number;
  is_adjusted: boolean;
  review_status: string;
  review_status_text: string;
}

/** 学期成绩汇总。 */
export interface SemesterGradeSummary {
  total_credits: number;
  semester_gpa: number;
  course_count: number;
  passed_count: number;
  failed_count: number;
}

/** 学期成绩响应。 */
export interface SemesterGradesResponse {
  semester: GradeSemesterInfo | null;
  grades: SemesterGradeItem[];
  summary: SemesterGradeSummary | null;
}

/** GPA 学期项。 */
export interface GPASemesterItem {
  semester_id: ID;
  semester_name: string;
  gpa: number;
  credits: number;
}

/** GPA 响应。 */
export interface GPAResponse {
  cumulative_gpa: number;
  cumulative_credits: number;
  semester_list: GPASemesterItem[];
  gpa_trend: number[];
}

/** 学习概览响应。 */
export interface LearningOverview {
  course_count: number;
  experiment_count: number;
  competition_count: number;
  total_study_hours: number;
}

/** 成绩申诉创建请求。 */
export interface CreateGradeAppealRequest {
  grade_id: ID;
  appeal_reason: string;
}

/** 成绩申诉列表查询参数。 */
export interface GradeAppealListParams extends QueryParams {
  page?: number;
  page_size?: number;
  status?: GradeAppealStatus;
  course_id?: ID;
}

/** 成绩申诉列表项。 */
export interface GradeAppealItem {
  id: ID;
  student_id: ID;
  student_name: string;
  course_id: ID;
  course_name: string;
  semester_id: ID;
  semester_name: string;
  original_score: number;
  status: GradeAppealStatus;
  status_text: string;
  created_at: string;
}

/** 成绩申诉详情。 */
export interface GradeAppealDetail {
  id: ID;
  student_id: ID;
  student_name: string;
  course_id: ID;
  course_name: string;
  semester_id: ID;
  semester_name: string;
  grade_id: ID;
  original_score: number;
  appeal_reason: string;
  status: GradeAppealStatus;
  status_text: string;
  handled_by: ID | null;
  handled_by_name: string | null;
  handled_at: string | null;
  new_score: number | null;
  handle_comment: string | null;
  created_at: string;
  updated_at: string;
}

/** 同意申诉请求。 */
export interface ApproveGradeAppealRequest {
  new_score: number;
  handle_comment: string;
}

/** 驳回申诉请求。 */
export interface RejectGradeAppealRequest {
  handle_comment: string;
}

/** 学业预警课程项。 */
export interface AcademicWarningCourseScore {
  course_id: ID;
  course_name: string;
  score: number;
  grade: string;
  credits: number;
}

/** 学业预警挂科课程项。 */
export interface AcademicWarningFailedCourse {
  course_id: ID;
  course_name: string;
  score: number;
  semester: string;
}

/** 学业预警列表查询参数。 */
export interface AcademicWarningListParams extends QueryParams {
  page?: number;
  page_size?: number;
  semester_id?: ID;
  warning_type?: AcademicWarningType;
  status?: AcademicWarningStatus;
  keyword?: string;
}

/** 学业预警详情内容。 */
export interface AcademicWarningDetailData {
  current_gpa?: number | null;
  fail_count?: number | null;
  threshold: number;
  semester_courses?: AcademicWarningCourseScore[];
  failed_courses?: AcademicWarningFailedCourse[];
}

/** 学业预警列表项。 */
export interface AcademicWarningItem {
  id: ID;
  student_id: ID;
  student_name: string;
  student_no: string;
  semester_name: string;
  warning_type: AcademicWarningType;
  warning_type_text: string;
  detail: AcademicWarningDetailData;
  status: AcademicWarningStatus;
  status_text: string;
  created_at: string;
}

/** 学业预警详情。 */
export interface AcademicWarningDetail {
  id: ID;
  student_id: ID;
  student_name: string;
  student_no: string;
  semester_id: ID;
  semester_name: string;
  warning_type: AcademicWarningType;
  warning_type_text: string;
  detail: AcademicWarningDetailData;
  status: AcademicWarningStatus;
  status_text: string;
  handled_by: ID | null;
  handled_by_name: string | null;
  handled_at: string | null;
  handle_note: string | null;
  created_at: string;
  updated_at: string;
}

/** 学业预警列表响应。 */
export type AcademicWarningListResponse = PaginatedData<AcademicWarningItem>;

/** 处理预警请求。 */
export interface HandleAcademicWarningRequest {
  handle_note: string;
}

/** 预警配置。 */
export interface WarningConfig {
  school_id: ID;
  gpa_threshold: number;
  fail_count_threshold: number;
  is_enabled: boolean;
}

/** 更新预警配置请求。 */
export interface UpdateWarningConfigRequest {
  gpa_threshold: number;
  fail_count_threshold: number;
  is_enabled: boolean;
}

/** 生成成绩单请求。 */
export interface GenerateTranscriptRequest {
  student_id?: ID | null;
  semester_ids: ID[];
}

/** 成绩单响应。 */
export interface TranscriptResponse {
  id: ID;
  file_url: string;
  generated_at: string;
  student_id?: ID | null;
  file_size?: number | null;
}

/** 成绩单列表查询参数。 */
export interface TranscriptListParams extends QueryParams {
  page?: number;
  page_size?: number;
  student_id?: ID;
}

/** 成绩单列表项。 */
export interface TranscriptListItem {
  id: ID;
  student_id: ID;
  student_name: string;
  file_url: string;
  file_size: number;
  include_semesters: string[];
  generated_at: string;
  expires_at: string | null;
}

/** 成绩单列表响应。 */
export type TranscriptListResponse = PaginatedData<TranscriptListItem>;

/** 分布项。 */
export interface ScoreDistributionItem {
  range: string;
  count: number;
}

/** 课程表现项。 */
export interface CoursePerformanceItem {
  course_name: string;
  average_score: number;
  pass_rate: number;
}

/** 课程成绩分析。 */
export interface CourseGradeAnalytics {
  course_id: ID;
  course_name: string;
  semester_name: string;
  student_count: number;
  average_score: number;
  median_score: number;
  max_score: number;
  min_score: number;
  pass_rate: number;
  grade_distribution: Record<string, number>;
  score_distribution: ScoreDistributionItem[];
}

/** 全校成绩分析。 */
export interface SchoolGradeAnalytics {
  semester_name: string;
  total_students: number;
  total_courses: number;
  average_gpa: number;
  gpa_distribution: ScoreDistributionItem[];
  fail_rate: number;
  warning_count: number;
  top_courses: CoursePerformanceItem[];
  bottom_courses: CoursePerformanceItem[];
}

/** 平台成绩总览。 */
export interface PlatformGradeAnalytics {
  total_schools: number;
  total_students: number;
  platform_average_gpa: number;
  school_comparison: Array<{ school_name: string; student_count: number; average_gpa: number }>;
}
