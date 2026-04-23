// notification.ts
// 模块07通知与消息 service：收件箱、公告、偏好、模板、统计和定向通知接口。

import { apiClient } from "@/lib/api-client";
import type { ID, QueryParams } from "@/types/api";
import type {
  CreateSystemAnnouncementRequest,
  NotificationAnnouncementListParams,
  NotificationInboxItem,
  NotificationInboxListParams,
  NotificationInboxListResponse,
  NotificationPreferencesResponse,
  NotificationStatistics,
  NotificationStatisticsParams,
  NotificationTemplateDetail,
  NotificationTemplateListResponse,
  NotificationUnreadCount,
  PreviewNotificationTemplateResponse,
  SendNotificationRequest,
  SystemAnnouncementDetail,
  SystemAnnouncementListResponse,
  UpdateNotificationPreferenceItem,
  UpdateNotificationTemplateRequest,
  UpdateSystemAnnouncementRequest,
} from "@/types/notification";

/**
 * listInbox 对应 GET /api/v1/notifications/inbox，用于收件箱列表。
 */
export function listInbox(params: NotificationInboxListParams) {
  return apiClient.get<NotificationInboxListResponse>("/notifications/inbox", { query: params });
}

/**
 * getInboxDetail 对应 GET /api/v1/notifications/inbox/:id，用于消息详情。
 */
export function getInboxDetail(messageID: ID) {
  return apiClient.get<NotificationInboxItem>(`/notifications/inbox/${messageID}`);
}

/**
 * markInboxRead 对应 PATCH /api/v1/notifications/inbox/:id/read，用于单条标记已读。
 */
export function markInboxRead(messageID: ID) {
  return apiClient.patch<null>(`/notifications/inbox/${messageID}/read`);
}

/**
 * batchReadInbox 对应 POST /api/v1/notifications/inbox/batch-read，用于批量标记已读。
 */
export function batchReadInbox(ids: ID[]) {
  return apiClient.post<null>("/notifications/inbox/batch-read", { ids });
}

/**
 * readAllInbox 对应 POST /api/v1/notifications/inbox/read-all，用于全部标记已读。
 */
export function readAllInbox() {
  return apiClient.post<null>("/notifications/inbox/read-all");
}

/**
 * deleteInboxMessage 对应 DELETE /api/v1/notifications/inbox/:id，用于删除消息。
 */
export function deleteInboxMessage(messageID: ID) {
  return apiClient.delete<null>(`/notifications/inbox/${messageID}`);
}

/**
 * getUnreadCount 对应 GET /api/v1/notifications/inbox/unread-count，用于未读数。
 */
export function getUnreadCount() {
  return apiClient.get<NotificationUnreadCount>("/notifications/inbox/unread-count", { query: {} });
}

/**
 * createNotificationAnnouncement 对应 POST /api/v1/notifications/announcements，用于创建系统公告。
 */
export function createNotificationAnnouncement(payload: CreateSystemAnnouncementRequest) {
  return apiClient.post<SystemAnnouncementDetail>("/notifications/announcements", payload);
}

/**
 * listNotificationAnnouncements 对应 GET /api/v1/notifications/announcements，用于系统公告列表。
 */
export function listNotificationAnnouncements(params: NotificationAnnouncementListParams = {}) {
  return apiClient.get<SystemAnnouncementListResponse>("/notifications/announcements", { query: params });
}

/**
 * getNotificationAnnouncement 对应 GET /api/v1/notifications/announcements/:id，用于系统公告详情。
 */
export function getNotificationAnnouncement(announcementID: ID) {
  return apiClient.get<SystemAnnouncementDetail>(`/notifications/announcements/${announcementID}`);
}

/**
 * updateNotificationAnnouncement 对应 PUT /api/v1/notifications/announcements/:id，用于编辑系统公告。
 */
export function updateNotificationAnnouncement(announcementID: ID, payload: UpdateSystemAnnouncementRequest) {
  return apiClient.put<null>(`/notifications/announcements/${announcementID}`, payload);
}

/**
 * publishNotificationAnnouncement 对应 POST /api/v1/notifications/announcements/:id/publish，用于发布系统公告。
 */
export function publishNotificationAnnouncement(announcementID: ID) {
  return apiClient.post<null>(`/notifications/announcements/${announcementID}/publish`);
}

/**
 * unpublishNotificationAnnouncement 对应 POST /api/v1/notifications/announcements/:id/unpublish，用于下架系统公告。
 */
export function unpublishNotificationAnnouncement(announcementID: ID) {
  return apiClient.post<null>(`/notifications/announcements/${announcementID}/unpublish`);
}

/**
 * deleteNotificationAnnouncement 对应 DELETE /api/v1/notifications/announcements/:id，用于删除系统公告。
 */
export function deleteNotificationAnnouncement(announcementID: ID) {
  return apiClient.delete<null>(`/notifications/announcements/${announcementID}`);
}

/**
 * sendDirectNotification 对应 POST /api/v1/notifications/send，用于定向通知。
 */
export function sendDirectNotification(payload: SendNotificationRequest) {
  return apiClient.post<null>("/notifications/send", payload);
}

/**
 * getNotificationPreferences 对应 GET /api/v1/notifications/preferences，用于通知偏好。
 */
export function getNotificationPreferences() {
  return apiClient.get<NotificationPreferencesResponse>("/notifications/preferences", { query: {} });
}

/**
 * updateNotificationPreferences 对应 PUT /api/v1/notifications/preferences，用于更新通知偏好。
 */
export function updateNotificationPreferences(preferences: UpdateNotificationPreferenceItem[]) {
  return apiClient.put<NotificationPreferencesResponse>("/notifications/preferences", { preferences });
}

/**
 * listNotificationTemplates 对应 GET /api/v1/notifications/templates，用于模板列表。
 */
export function listNotificationTemplates() {
  return apiClient.get<NotificationTemplateListResponse>("/notifications/templates", { query: {} });
}

/**
 * getNotificationTemplate 对应 GET /api/v1/notifications/templates/:id，用于模板详情。
 */
export function getNotificationTemplate(templateID: ID) {
  return apiClient.get<NotificationTemplateDetail>(`/notifications/templates/${templateID}`);
}

/**
 * updateNotificationTemplate 对应 PUT /api/v1/notifications/templates/:id，用于更新模板。
 */
export function updateNotificationTemplate(templateID: ID, payload: UpdateNotificationTemplateRequest) {
  return apiClient.put<null>(`/notifications/templates/${templateID}`, payload);
}

/**
 * previewNotificationTemplate 对应 POST /api/v1/notifications/templates/:id/preview，用于预览模板。
 */
export function previewNotificationTemplate(templateID: ID, params: Record<string, unknown>) {
  return apiClient.post<PreviewNotificationTemplateResponse>(`/notifications/templates/${templateID}/preview`, { params });
}

/**
 * getNotificationStatistics 对应 GET /api/v1/notifications/statistics，用于消息统计。
 */
export function getNotificationStatistics(params: NotificationStatisticsParams = {}) {
  return apiClient.get<NotificationStatistics>("/notifications/statistics", { query: params });
}
