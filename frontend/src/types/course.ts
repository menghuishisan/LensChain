// course.ts
// 模块03课程与教学类型定义：课程、章节、课时、选课、进度、课表、共享、统计。

import type { ID, PaginatedData, QueryParams } from "@/types/api";

/**
 * 课程状态：1草稿 2已发布 3进行中 4已结束 5已归档。
 */
export type CourseStatus = 1 | 2 | 3 | 4 | 5;

/**
 * 课程类型：1理论课 2实验课 3混合课 4项目实战。
 */
export type CourseType = 1 | 2 | 3 | 4;

/**
 * 课程难度：1入门 2进阶 3高级 4研究。
 */
export type CourseDifficulty = 1 | 2 | 3 | 4;

/**
 * 课时内容类型：1视频 2图文 3附件 4实验。
 */
export type LessonContentType = 1 | 2 | 3 | 4;

/**
 * 学习进度状态：1未开始 2进行中 3已完成。
 */
export type LearningStatus = 1 | 2 | 3;

/**
 * 课程创建请求。
 */
export interface CreateCourseRequest {
  title: string;
  description?: string | null;
  cover_url?: string | null;
  course_type: CourseType;
  difficulty: CourseDifficulty;
  topic: string;
  credits?: number | null;
  semester_id?: ID | null;
  start_at?: string | null;
  end_at?: string | null;
  max_students?: number | null;
}

/**
 * 课程创建响应。
 */
export interface CreateCourseResponse {
  id: ID;
  title: string;
  status: CourseStatus;
  status_text: string;
  invite_code: string;
  cover_url: string | null;
}

/**
 * 课程更新请求。
 */
export type UpdateCourseRequest = Partial<CreateCourseRequest>;

/**
 * 课程列表查询参数。
 */
export interface CourseListParams extends QueryParams {
  page?: number;
  page_size?: number;
  keyword?: string;
  status?: CourseStatus;
  course_type?: CourseType;
}

/**
 * 教师课程列表项。
 */
export interface CourseListItem {
  id: ID;
  title: string;
  cover_url: string | null;
  course_type: CourseType;
  course_type_text: string;
  difficulty: CourseDifficulty;
  difficulty_text: string;
  topic: string;
  status: CourseStatus;
  status_text: string;
  is_shared: boolean;
  student_count: number;
  start_at: string | null;
  end_at: string | null;
  created_at: string;
}

/**
 * 章节含课时。
 */
export interface ChapterWithLessons {
  id: ID;
  title: string;
  description: string | null;
  sort_order: number;
  lessons: LessonListItem[];
}

/**
 * 课时列表项。
 */
export interface LessonListItem {
  id: ID;
  title: string;
  content_type: LessonContentType;
  content_type_text: string;
  video_duration: number | null;
  experiment_id: ID | null;
  estimated_minutes: number | null;
  sort_order: number;
}

/**
 * 课程详情。
 */
export interface CourseDetail extends CourseListItem {
  description: string | null;
  invite_code?: string | null;
  credits: number | null;
  semester_id: ID | null;
  max_students: number | null;
  teacher_id: ID;
  teacher_name: string;
  updated_at: string;
  chapters: ChapterWithLessons[];
}

/**
 * 课程列表响应。
 */
export type CourseListResponse = PaginatedData<CourseListItem>;

/**
 * 章节创建请求。
 */
export interface CreateChapterRequest {
  title: string;
  description?: string | null;
}

/**
 * 章节更新请求。
 */
export type UpdateChapterRequest = Partial<CreateChapterRequest>;

/**
 * 课时创建请求。
 */
export interface CreateLessonRequest {
  title: string;
  content_type: LessonContentType;
  content?: string | null;
  video_url?: string | null;
  video_duration?: number | null;
  experiment_id?: ID | null;
  estimated_minutes?: number | null;
}

/**
 * 课时更新请求。
 */
export type UpdateLessonRequest = Partial<CreateLessonRequest>;

/**
 * 课时附件。
 */
export interface LessonAttachmentItem {
  id: ID;
  file_name: string;
  file_url: string;
  file_size: number;
  file_type: string;
}

/**
 * 课时详情。
 */
export interface LessonDetail extends LessonListItem {
  chapter_id: ID;
  course_id: ID;
  content: string | null;
  video_url: string | null;
  attachments: LessonAttachmentItem[];
}

/**
 * 上传课时附件请求。
 */
export interface UploadAttachmentRequest {
  file_name: string;
  file_url: string;
  file_size: number;
  file_type: string;
}

/**
 * 课程文件用途。
 */
export type CourseFilePurpose = "lesson_attachment" | "assignment_report";

/**
 * 课程文件上传响应。
 */
export interface UploadCourseFileResponse {
  file_name: string;
  file_url: string;
  download_url: string;
  file_size: number;
  file_type: string;
}

/**
 * 顺序重排请求。
 */
export interface ReorderIDsRequest {
  ids: ID[];
}

/**
 * 邀请码加入课程请求。
 */
export interface JoinCourseRequest {
  invite_code: string;
}

/**
 * 课程学生列表查询参数。
 */
export interface CourseStudentListParams extends QueryParams {
  page?: number;
  page_size?: number;
  keyword?: string;
}

/**
 * 已选课学生项。
 */
export interface EnrolledStudentItem {
  id: ID;
  name: string;
  student_no: string | null;
  college: string | null;
  major: string | null;
  class_name: string | null;
  join_method: number;
  join_method_text: string;
  joined_at: string;
  progress: number;
}

/**
 * 课程学生列表响应。
 */
export type CourseStudentListResponse = PaginatedData<EnrolledStudentItem>;

/**
 * 课程表项请求。
 */
export interface ScheduleItemRequest {
  day_of_week: number;
  start_time: string;
  end_time: string;
  location?: string | null;
}

/**
 * 设置课程表请求。
 */
export interface SetCourseScheduleRequest {
  schedules: ScheduleItemRequest[];
}

/**
 * 课程表项响应。
 */
export interface ScheduleItemResponse extends ScheduleItemRequest {
  id: ID;
}

/**
 * 我的周课程表响应。
 */
export interface MyScheduleResponse {
  schedules: Array<{
    course_id: ID;
    course_title: string;
    teacher_name: string;
    day_of_week: number;
    start_time: string;
    end_time: string;
    location: string | null;
  }>;
}

/**
 * 更新学习进度请求。
 */
export interface UpdateProgressRequest {
  status: LearningStatus;
  video_progress?: number | null;
  study_duration_increment: number;
}

/**
 * 我的课程学习进度。
 */
export interface MyProgressResponse {
  course_id: ID;
  total_lessons: number;
  completed_count: number;
  progress: number;
  total_study_hours: number;
  lessons: Array<{
    lesson_id: ID;
    lesson_title: string;
    chapter_title: string;
    status: LearningStatus;
    status_text: string;
    video_progress: number;
    video_duration: number | null;
    study_duration: number;
    completed_at: string | null;
    last_accessed_at: string | null;
  }>;
}

/**
 * 学生学习进度列表响应。
 */
export type StudentsProgressResponse = PaginatedData<{
  student_id: ID;
  student_name: string;
  student_no: string | null;
  completed_count: number;
  total_lessons: number;
  progress: number;
  total_study_hours: number;
  last_accessed_at: string | null;
}>;

/**
 * 共享课程列表项。
 */
export interface SharedCourseItem {
  id: ID;
  title: string;
  description: string | null;
  cover_url: string | null;
  course_type: CourseType;
  course_type_text: string;
  difficulty: CourseDifficulty;
  difficulty_text: string;
  topic: string;
  teacher_name: string;
  school_name: string;
  student_count: number;
  rating: number;
}

/**
 * 共享课程列表响应。
 */
export type SharedCourseListResponse = PaginatedData<SharedCourseItem>;

/**
 * 共享课程详情。
 */
export interface SharedCourseDetail extends SharedCourseItem {
  status: CourseStatus;
  status_text: string;
  credits: number | null;
  start_at: string | null;
  end_at: string | null;
  max_students: number | null;
  chapters: ChapterWithLessons[];
}

/**
 * 我的课程列表查询参数。
 */
export interface MyCourseListParams extends QueryParams {
  page?: number;
  page_size?: number;
  status?: 2 | 3 | 4;
}

/**
 * 我的课程列表项。
 */
export interface MyCourseItem {
  id: ID;
  title: string;
  cover_url: string | null;
  course_type: CourseType;
  course_type_text: string;
  teacher_name: string;
  status: CourseStatus;
  status_text: string;
  progress: number;
  joined_at: string;
}

/**
 * 我的课程列表响应。
 */
export type MyCourseListResponse = PaginatedData<MyCourseItem>;

/**
 * 课程统计概览。
 */
export interface CourseOverviewStats {
  student_count: number;
  lesson_count: number;
  assignment_count: number;
  avg_progress: number;
  avg_score: number;
  completion_rate: number;
  activity_rate: number;
  total_study_hours: number;
  progress_distribution: {
    not_started_rate: number;
    in_progress_rate: number;
    completed_rate: number;
  };
}

/**
 * 作业统计响应。
 */
export interface AssignmentStatsResponse {
  assignments: Array<{
    id: ID;
    title: string;
    submit_count: number;
    total_students: number;
    submit_rate: number;
    avg_score: number;
    max_score: number;
    min_score: number;
    score_distribution: Array<{ range: string; count: number }>;
  }>;
}
