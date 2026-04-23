// courseAssignment.ts
// 模块03作业、提交、批改、讨论、公告、评价和成绩类型定义。

import type { ID, PaginatedData, QueryParams } from "@/types/api";

/**
 * 作业类型：1作业 2测验。
 */
export type AssignmentType = 1 | 2;

/**
 * 题目类型：1单选 2多选 3判断 4填空 5简答 6编程 7实验报告。
 */
export type QuestionType = 1 | 2 | 3 | 4 | 5 | 6 | 7;

/**
 * 迟交策略：1不允许 2扣分 3不扣分。
 */
export type LatePolicy = 1 | 2 | 3;

/**
 * 提交状态：1已提交 2待批改 3已批改。
 */
export type SubmissionStatus = 1 | 2 | 3;

/**
 * 创建作业请求。
 */
export interface CreateAssignmentRequest {
  title: string;
  description?: string | null;
  chapter_id?: ID | null;
  assignment_type: AssignmentType;
  deadline_at: string;
  max_submissions?: number | null;
  late_policy: LatePolicy;
  late_deduction_per_day?: number | null;
}

/**
 * 编辑作业请求。
 */
export type UpdateAssignmentRequest = Partial<CreateAssignmentRequest>;

/**
 * 作业列表查询参数。
 */
export interface AssignmentListParams extends QueryParams {
  page?: number;
  page_size?: number;
  assignment_type?: AssignmentType;
}

/**
 * 作业列表项。
 */
export interface AssignmentListItem {
  id: ID;
  title: string;
  assignment_type: AssignmentType;
  assignment_type_text: string;
  total_score: number;
  deadline_at: string | null;
  is_published: boolean;
  submit_count: number;
  total_students: number;
  sort_order: number;
}

/**
 * 作业列表响应。
 */
export type AssignmentListResponse = PaginatedData<AssignmentListItem>;

/**
 * 题目详情。
 */
export interface QuestionDetailItem {
  id: ID;
  question_type: QuestionType;
  question_type_text: string;
  title: string;
  options: string | null;
  correct_answer: string | null;
  reference_answer: string | null;
  score: number;
  judge_config: string | null;
  sort_order: number;
}

/**
 * 作业详情。
 */
export interface AssignmentDetail {
  id: ID;
  course_id: ID;
  chapter_id: ID | null;
  title: string;
  description: string | null;
  assignment_type: AssignmentType;
  assignment_type_text: string;
  total_score: number;
  deadline_at: string | null;
  max_submissions: number;
  late_policy: LatePolicy;
  late_policy_text: string;
  late_deduction_per_day: number | null;
  is_published: boolean;
  questions: QuestionDetailItem[];
}

/**
 * 添加题目请求。
 */
export interface AddQuestionRequest {
  question_type: QuestionType;
  title: string;
  options?: string | null;
  correct_answer?: string | null;
  reference_answer?: string | null;
  score: number;
  judge_config?: string | null;
}

/**
 * 编辑题目请求。
 */
export type UpdateQuestionRequest = Partial<AddQuestionRequest>;

/**
 * 作答答案项。
 */
export interface SubmitAnswerRequest {
  question_id: ID;
  answer_content?: string | null;
  answer_file_url?: string | null;
}

/**
 * 草稿/提交请求。
 */
export interface AssignmentAnswersRequest {
  answers: SubmitAnswerRequest[];
}

/**
 * 保存草稿响应。
 */
export interface SaveAssignmentDraftResponse {
  assignment_id: ID;
  saved_at: string;
  answer_count: number;
}

/**
 * 作业草稿。
 */
export interface AssignmentDraft {
  assignment_id: ID;
  saved_at: string;
  answers: SubmitAnswerRequest[];
}

/**
 * 提交作业响应。
 */
export interface SubmitAssignmentResponse {
  submission_id: ID;
  submission_no: number;
  remaining_submissions: number;
  is_late: boolean;
  instant_feedback: {
    auto_graded_score: number;
    auto_graded_total: number;
    details: Array<{ question_id: ID; is_correct?: boolean; score?: number; status?: string }>;
  };
}

/**
 * 提交列表查询参数。
 */
export interface SubmissionListParams extends QueryParams {
  page?: number;
  page_size?: number;
  status?: SubmissionStatus;
  keyword?: string;
}

/**
 * 提交列表项。
 */
export interface SubmissionListItem {
  id: ID;
  student_id: ID;
  student_name: string;
  student_no: string | null;
  submission_no: number;
  status: SubmissionStatus;
  status_text: string;
  total_score: number | null;
  is_late: boolean;
  submitted_at: string;
}

/**
 * 提交列表响应。
 */
export type SubmissionListResponse = PaginatedData<SubmissionListItem>;

/**
 * 提交详情。
 */
export interface SubmissionDetail extends SubmissionListItem {
  assignment_id: ID;
  late_days: number | null;
  score_before_deduction: number | null;
  score_after_deduction: number | null;
  teacher_comment: string | null;
  graded_at: string | null;
  answers: Array<{
    id: ID;
    question_id: ID;
    question_title: string;
    question_type: QuestionType;
    answer_content: string | null;
    answer_file_url: string | null;
    is_correct: boolean | null;
    score: number | null;
    teacher_comment: string | null;
    auto_judge_result: string | null;
  }>;
}

/**
 * 批改提交请求。
 */
export interface GradeSubmissionRequest {
  teacher_comment?: string | null;
  answers: Array<{
    question_id: ID;
    score: number;
    teacher_comment?: string | null;
  }>;
}

/**
 * 我的提交列表响应。
 */
export interface MySubmissionsResponse {
  submissions: Array<{
    id: ID;
    submission_no: number;
    status: SubmissionStatus;
    status_text: string;
    total_score: number | null;
    is_late: boolean;
    submitted_at: string;
  }>;
}

/**
 * 公告创建请求。
 */
export interface CreateAnnouncementRequest {
  title: string;
  content: string;
}

/**
 * 公告列表项。
 */
export interface AnnouncementItem {
  id: ID;
  title: string;
  content: string;
  is_pinned: boolean;
  teacher_name: string;
  created_at: string;
  updated_at: string;
}

/**
 * 讨论创建请求。
 */
export interface CreateDiscussionRequest {
  title: string;
  content: string;
}

/**
 * 讨论列表项。
 */
export interface DiscussionListItem {
  id: ID;
  title: string;
  author_id: ID;
  author_name: string;
  is_pinned: boolean;
  reply_count: number;
  like_count: number;
  is_liked: boolean;
  last_replied_at: string | null;
  created_at: string;
}

/**
 * 讨论详情。
 */
export interface DiscussionDetail extends DiscussionListItem {
  course_id: ID;
  content: string;
  replies: Array<{
    id: ID;
    author_id: ID;
    author_name: string;
    content: string;
    reply_to_id: ID | null;
    reply_to_name: string | null;
    created_at: string;
  }>;
}

/**
 * 回复讨论请求。
 */
export interface CreateReplyRequest {
  content: string;
  reply_to_id?: ID | null;
}

/**
 * 课程评价列表响应。
 */
export interface EvaluationListResponse {
  summary: {
    avg_rating: number;
    total_count: number;
    distribution: [number, number, number, number, number];
  };
  items: Array<{
    id: ID;
    student_id: ID;
    student_name: string;
    rating: number;
    comment: string | null;
    created_at: string;
  }>;
  pagination: {
    page: number;
    page_size: number;
    total: number;
    total_pages: number;
  };
}

/**
 * 成绩权重项。
 */
export interface GradeConfigItem {
  assignment_id: ID;
  name: string;
  weight: number;
}

/**
 * 成绩配置响应。
 */
export interface GradeConfigResponse {
  items: GradeConfigItem[];
}

/**
 * 成绩汇总响应。
 */
export interface GradeSummaryResponse {
  grade_config: GradeConfigResponse;
  students: Array<{
    student_id: ID;
    student_name: string;
    student_no: string | null;
    scores: Record<ID, number>;
    weighted_total: number;
    final_score: number;
    is_adjusted: boolean;
  }>;
}
