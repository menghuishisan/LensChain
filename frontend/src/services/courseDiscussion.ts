// courseDiscussion.ts
// 模块03讨论、公告、评价 service。

import { apiClient } from "@/lib/api-client";
import type { ID, PaginatedData, QueryParams } from "@/types/api";
import type {
  AnnouncementItem,
  CreateAnnouncementRequest,
  CreateDiscussionRequest,
  CreateReplyRequest,
  DiscussionDetail,
  DiscussionListItem,
  EvaluationListResponse,
} from "@/types/courseAssignment";

/**
 * createAnnouncement 对应 POST /api/v1/courses/:id/announcements，用于发布公告。
 */
export function createAnnouncement(courseID: ID, payload: CreateAnnouncementRequest) {
  return apiClient.post<{ id: ID }>(`/courses/${courseID}/announcements`, payload);
}

/**
 * listAnnouncements 对应 GET /api/v1/courses/:id/announcements，用于公告列表。
 */
export function listAnnouncements(courseID: ID, params: QueryParams) {
  return apiClient.get<PaginatedData<AnnouncementItem>>(`/courses/${courseID}/announcements`, { query: params });
}

/**
 * updateAnnouncement 对应 PUT /api/v1/announcements/:id，用于编辑公告。
 */
export function updateAnnouncement(announcementID: ID, payload: Partial<CreateAnnouncementRequest>) {
  return apiClient.put<null>(`/announcements/${announcementID}`, payload);
}

/**
 * pinAnnouncement 对应 PATCH /api/v1/announcements/:id/pin，用于置顶公告。
 */
export function pinAnnouncement(announcementID: ID, isPinned: boolean) {
  return apiClient.patch<null>(`/announcements/${announcementID}/pin`, { is_pinned: isPinned });
}

/**
 * deleteAnnouncement 对应 DELETE /api/v1/announcements/:id，用于删除公告。
 */
export function deleteAnnouncement(announcementID: ID) {
  return apiClient.delete<null>(`/announcements/${announcementID}`);
}

/**
 * createDiscussion 对应 POST /api/v1/courses/:id/discussions，用于发帖。
 */
export function createDiscussion(courseID: ID, payload: CreateDiscussionRequest) {
  return apiClient.post<{ id: ID }>(`/courses/${courseID}/discussions`, payload);
}

/**
 * listDiscussions 对应 GET /api/v1/courses/:id/discussions，用于讨论列表。
 */
export function listDiscussions(courseID: ID, params: QueryParams) {
  return apiClient.get<PaginatedData<DiscussionListItem>>(`/courses/${courseID}/discussions`, { query: params });
}

/**
 * getDiscussion 对应 GET /api/v1/discussions/:id，用于帖子详情。
 */
export function getDiscussion(discussionID: ID) {
  return apiClient.get<DiscussionDetail>(`/discussions/${discussionID}`);
}

/**
 * deleteDiscussion 对应 DELETE /api/v1/discussions/:id，用于删除帖子。
 */
export function deleteDiscussion(discussionID: ID) {
  return apiClient.delete<null>(`/discussions/${discussionID}`);
}

/**
 * pinDiscussion 对应 PATCH /api/v1/discussions/:id/pin，用于置顶帖子。
 */
export function pinDiscussion(discussionID: ID, isPinned: boolean) {
  return apiClient.patch<null>(`/discussions/${discussionID}/pin`, { is_pinned: isPinned });
}

/**
 * createDiscussionReply 对应 POST /api/v1/discussions/:id/replies，用于回复帖子。
 */
export function createDiscussionReply(discussionID: ID, payload: CreateReplyRequest) {
  return apiClient.post<{ id: ID }>(`/discussions/${discussionID}/replies`, payload);
}

/**
 * deleteDiscussionReply 对应 DELETE /api/v1/discussion-replies/:id，用于删除回复。
 */
export function deleteDiscussionReply(replyID: ID) {
  return apiClient.delete<null>(`/discussion-replies/${replyID}`);
}

/**
 * likeDiscussion 对应 POST /api/v1/discussions/:id/like，用于点赞。
 */
export function likeDiscussion(discussionID: ID) {
  return apiClient.post<{ liked: true }>(`/discussions/${discussionID}/like`);
}

/**
 * unlikeDiscussion 对应 DELETE /api/v1/discussions/:id/like，用于取消点赞。
 */
export function unlikeDiscussion(discussionID: ID) {
  return apiClient.delete<{ liked: false }>(`/discussions/${discussionID}/like`);
}

/**
 * createEvaluation 对应 POST /api/v1/courses/:id/evaluations，用于提交课程评价。
 */
export function createEvaluation(courseID: ID, payload: { rating: number; comment?: string | null }) {
  return apiClient.post<{ id: ID }>(`/courses/${courseID}/evaluations`, payload);
}

/**
 * listEvaluations 对应 GET /api/v1/courses/:id/evaluations，用于课程评价列表。
 */
export function listEvaluations(courseID: ID, params: QueryParams) {
  return apiClient.get<EvaluationListResponse>(`/courses/${courseID}/evaluations`, { query: params });
}

/**
 * updateEvaluation 对应 PUT /api/v1/course-evaluations/:id，用于修改评价。
 */
export function updateEvaluation(evaluationID: ID, payload: { rating?: number; comment?: string | null }) {
  return apiClient.put<null>(`/course-evaluations/${evaluationID}`, payload);
}
