// course.ts
// 模块03课程与教学 service：课程、章节、课时、选课、进度、课表、共享和统计接口。

import { apiClient } from "@/lib/api-client";
import type { ID } from "@/types/api";
import type {
  AssignmentStatsResponse,
  ChapterWithLessons,
  CourseDetail,
  CourseListParams,
  CourseListResponse,
  CourseOverviewStats,
  CourseStudentListParams,
  CourseStudentListResponse,
  CreateChapterRequest,
  CreateCourseRequest,
  CreateCourseResponse,
  CreateLessonRequest,
  JoinCourseRequest,
  LessonDetail,
  MyCourseListParams,
  MyCourseListResponse,
  MyProgressResponse,
  MyScheduleResponse,
  ReorderIDsRequest,
  ScheduleItemResponse,
  SetCourseScheduleRequest,
  SharedCourseDetail,
  SharedCourseListResponse,
  StudentsProgressResponse,
  UpdateChapterRequest,
  UpdateCourseRequest,
  UpdateLessonRequest,
  UpdateProgressRequest,
  UploadAttachmentRequest,
  CourseFilePurpose,
  UploadCourseFileResponse,
} from "@/types/course";

/**
 * createCourse 对应 POST /api/v1/courses，用于教师创建课程。
 */
export function createCourse(payload: CreateCourseRequest) {
  return apiClient.post<CreateCourseResponse>("/courses", payload);
}

/**
 * listCourses 对应 GET /api/v1/courses，用于教师课程列表。
 */
export function listCourses(params: CourseListParams) {
  return apiClient.get<CourseListResponse>("/courses", { query: params });
}

/**
 * getCourse 对应 GET /api/v1/courses/:id，用于课程详情。
 */
export function getCourse(id: ID) {
  return apiClient.get<CourseDetail>(`/courses/${id}`);
}

/**
 * updateCourse 对应 PUT /api/v1/courses/:id，用于编辑课程信息。
 */
export function updateCourse(id: ID, payload: UpdateCourseRequest) {
  return apiClient.put<null>(`/courses/${id}`, payload);
}

/**
 * deleteCourse 对应 DELETE /api/v1/courses/:id，用于删除草稿课程。
 */
export function deleteCourse(id: ID) {
  return apiClient.delete<null>(`/courses/${id}`);
}

/**
 * publishCourse 对应 POST /api/v1/courses/:id/publish，用于发布课程。
 */
export function publishCourse(id: ID) {
  return apiClient.post<null>(`/courses/${id}/publish`);
}

/**
 * endCourse 对应 POST /api/v1/courses/:id/end，用于结束课程。
 */
export function endCourse(id: ID) {
  return apiClient.post<null>(`/courses/${id}/end`);
}

/**
 * archiveCourse 对应 POST /api/v1/courses/:id/archive，用于归档课程。
 */
export function archiveCourse(id: ID) {
  return apiClient.post<null>(`/courses/${id}/archive`);
}

/**
 * cloneCourse 对应 POST /api/v1/courses/:id/clone，用于克隆课程。
 */
export function cloneCourse(id: ID) {
  return apiClient.post<{ id: ID }>(`/courses/${id}/clone`);
}

/**
 * setCourseShare 对应 PATCH /api/v1/courses/:id/share，用于设置共享状态。
 */
export function setCourseShare(id: ID, isShared: boolean) {
  return apiClient.patch<null>(`/courses/${id}/share`, { is_shared: isShared });
}

/**
 * refreshInviteCode 对应 POST /api/v1/courses/:id/invite-code/refresh，用于刷新邀请码。
 */
export function refreshInviteCode(id: ID) {
  return apiClient.post<{ invite_code: string }>(`/courses/${id}/invite-code/refresh`);
}

/**
 * listChapters 对应 GET /api/v1/courses/:id/chapters，用于课程目录树。
 */
export function listChapters(courseID: ID) {
  return apiClient.get<ChapterWithLessons[]>(`/courses/${courseID}/chapters`);
}

/**
 * createChapter 对应 POST /api/v1/courses/:id/chapters，用于创建章节。
 */
export function createChapter(courseID: ID, payload: CreateChapterRequest) {
  return apiClient.post<{ id: ID }>(`/courses/${courseID}/chapters`, payload);
}

/**
 * updateChapter 对应 PUT /api/v1/chapters/:id，用于编辑章节。
 */
export function updateChapter(chapterID: ID, payload: UpdateChapterRequest) {
  return apiClient.put<null>(`/chapters/${chapterID}`, payload);
}

/**
 * deleteChapter 对应 DELETE /api/v1/chapters/:id，用于删除章节。
 */
export function deleteChapter(chapterID: ID) {
  return apiClient.delete<null>(`/chapters/${chapterID}`);
}

/**
 * sortChapters 对应 PUT /api/v1/courses/:id/chapters/sort，用于章节排序。
 */
export function sortChapters(courseID: ID, payload: ReorderIDsRequest) {
  return apiClient.put<null>(`/courses/${courseID}/chapters/sort`, payload);
}

/**
 * createLesson 对应 POST /api/v1/chapters/:id/lessons，用于创建课时。
 */
export function createLesson(chapterID: ID, payload: CreateLessonRequest) {
  return apiClient.post<{ id: ID }>(`/chapters/${chapterID}/lessons`, payload);
}

/**
 * getLesson 对应 GET /api/v1/lessons/:id，用于课时详情。
 */
export function getLesson(lessonID: ID) {
  return apiClient.get<LessonDetail>(`/lessons/${lessonID}`);
}

/**
 * updateLesson 对应 PUT /api/v1/lessons/:id，用于编辑课时。
 */
export function updateLesson(lessonID: ID, payload: UpdateLessonRequest) {
  return apiClient.put<null>(`/lessons/${lessonID}`, payload);
}

/**
 * deleteLesson 对应 DELETE /api/v1/lessons/:id，用于删除课时。
 */
export function deleteLesson(lessonID: ID) {
  return apiClient.delete<null>(`/lessons/${lessonID}`);
}

/**
 * sortLessons 对应 PUT /api/v1/chapters/:id/lessons/sort，用于课时排序。
 */
export function sortLessons(chapterID: ID, payload: ReorderIDsRequest) {
  return apiClient.put<null>(`/chapters/${chapterID}/lessons/sort`, payload);
}

/**
 * uploadLessonAttachment 对应 POST /api/v1/lessons/:id/attachments，用于保存课时附件元数据。
 */
export function uploadLessonAttachment(lessonID: ID, payload: UploadAttachmentRequest) {
  return apiClient.post<{ id: ID }>(`/lessons/${lessonID}/attachments`, payload);
}

/**
 * uploadCourseFile 对应 POST /api/v1/course-files/upload，用于真实上传课程文件到对象存储。
 */
export function uploadCourseFile(file: File, purpose: CourseFilePurpose, onUploadProgress?: (progress: number) => void) {
  const formData = new FormData();
  formData.append("file", file);
  formData.append("purpose", purpose);
  return apiClient.upload<UploadCourseFileResponse>("/course-files/upload", formData, { onUploadProgress });
}

/**
 * deleteLessonAttachment 对应 DELETE /api/v1/lesson-attachments/:id，用于删除课时附件。
 */
export function deleteLessonAttachment(attachmentID: ID) {
  return apiClient.delete<null>(`/lesson-attachments/${attachmentID}`);
}

/**
 * joinCourse 对应 POST /api/v1/courses/join，用于学生通过邀请码加入课程。
 */
export function joinCourse(payload: JoinCourseRequest) {
  return apiClient.post<null>("/courses/join", payload);
}

/**
 * addStudentToCourse 对应 POST /api/v1/courses/:id/students，用于教师添加学生。
 */
export function addStudentToCourse(courseID: ID, studentID: ID) {
  return apiClient.post<null>(`/courses/${courseID}/students`, { student_id: studentID });
}

/**
 * batchAddStudentsToCourse 对应 POST /api/v1/courses/:id/students/batch，用于批量添加学生。
 */
export function batchAddStudentsToCourse(courseID: ID, studentIDs: ID[]) {
  return apiClient.post<null>(`/courses/${courseID}/students/batch`, { student_ids: studentIDs });
}

/**
 * removeStudentFromCourse 对应 DELETE /api/v1/courses/:id/students/:student_id，用于移除学生。
 */
export function removeStudentFromCourse(courseID: ID, studentID: ID) {
  return apiClient.delete<null>(`/courses/${courseID}/students/${studentID}`);
}

/**
 * listCourseStudents 对应 GET /api/v1/courses/:id/students，用于课程学生列表。
 */
export function listCourseStudents(courseID: ID, params: CourseStudentListParams) {
  return apiClient.get<CourseStudentListResponse>(`/courses/${courseID}/students`, { query: params });
}

/**
 * updateLessonProgress 对应 POST /api/v1/lessons/:id/progress，用于更新学习进度。
 */
export function updateLessonProgress(lessonID: ID, payload: UpdateProgressRequest) {
  return apiClient.post<null>(`/lessons/${lessonID}/progress`, payload);
}

/**
 * getMyProgress 对应 GET /api/v1/courses/:id/my-progress，用于我的课程学习进度。
 */
export function getMyProgress(courseID: ID) {
  return apiClient.get<MyProgressResponse>(`/courses/${courseID}/my-progress`);
}

/**
 * listStudentsProgress 对应 GET /api/v1/courses/:id/students-progress，用于教师查看学生进度。
 */
export function listStudentsProgress(courseID: ID, params: CourseStudentListParams) {
  return apiClient.get<StudentsProgressResponse>(`/courses/${courseID}/students-progress`, { query: params });
}

/**
 * setCourseSchedule 对应 PUT /api/v1/courses/:id/schedules，用于设置课程表。
 */
export function setCourseSchedule(courseID: ID, payload: SetCourseScheduleRequest) {
  return apiClient.put<null>(`/courses/${courseID}/schedules`, payload);
}

/**
 * getCourseSchedule 对应 GET /api/v1/courses/:id/schedules，用于获取课程表。
 */
export function getCourseSchedule(courseID: ID) {
  return apiClient.get<ScheduleItemResponse[]>(`/courses/${courseID}/schedules`);
}

/**
 * getMySchedule 对应 GET /api/v1/my-schedule，用于我的周课程表。
 */
export function getMySchedule() {
  return apiClient.get<MyScheduleResponse>("/my-schedule");
}

/**
 * listSharedCourses 对应 GET /api/v1/shared-courses，用于共享课程库。
 */
export function listSharedCourses(params: CourseListParams) {
  return apiClient.get<SharedCourseListResponse>("/shared-courses", { query: params });
}

/**
 * getSharedCourse 对应 GET /api/v1/shared-courses/:id，用于共享课程详情。
 */
export function getSharedCourse(id: ID) {
  return apiClient.get<SharedCourseDetail>(`/shared-courses/${id}`);
}

/**
 * listMyCourses 对应 GET /api/v1/my-courses，用于学生我的课程列表。
 */
export function listMyCourses(params: MyCourseListParams) {
  return apiClient.get<MyCourseListResponse>("/my-courses", { query: params });
}

/**
 * getCourseOverview 对应 GET /api/v1/courses/:id/statistics/overview，用于课程整体统计。
 */
export function getCourseOverview(courseID: ID) {
  return apiClient.get<CourseOverviewStats>(`/courses/${courseID}/statistics/overview`);
}

/**
 * getAssignmentStatistics 对应 GET /api/v1/courses/:id/statistics/assignments，用于作业统计。
 */
export function getAssignmentStatistics(courseID: ID) {
  return apiClient.get<AssignmentStatsResponse>(`/courses/${courseID}/statistics/assignments`);
}

/**
 * exportCourseStatistics 对应 GET /api/v1/courses/:id/statistics/export，用于下载统计报告。
 */
export function exportCourseStatistics(courseID: ID) {
  return apiClient.download(`/courses/${courseID}/statistics/export`);
}
