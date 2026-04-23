// notification.ts
// 模块07通知与消息类型定义：收件箱、公告、偏好、模板、统计和实时推送。

import type { ID, PaginatedData, QueryParams } from "@/types/api";

/** 通知分类：1系统 2课程 3实验 4竞赛 5成绩。 */
export type NotificationCategory = 1 | 2 | 3 | 4 | 5;

/** 公告状态：1草稿 2已发布 3已下架。 */
export type AnnouncementStatus = 1 | 2 | 3;

/** 收件箱查询参数。 */
export interface NotificationInboxListParams extends QueryParams {
  page?: number;
  page_size?: number;
  category?: NotificationCategory;
  is_read?: boolean;
  keyword?: string;
}

/** 收件箱消息项。 */
export interface NotificationInboxItem {
  id: ID;
  category: NotificationCategory;
  category_text: string;
  title: string;
  content: string;
  source_module: string;
  source_type: string | null;
  source_id: string | null;
  is_read: boolean;
  read_at: string | null;
  created_at: string;
}

/** 未读分类统计。 */
export interface NotificationUnreadByCategory {
  system: number;
  course: number;
  experiment: number;
  competition: number;
  grade: number;
}

/** 收件箱列表响应。 */
export interface NotificationInboxListResponse extends PaginatedData<NotificationInboxItem> {
  unread_count: number;
}

/** 未读计数响应。 */
export interface NotificationUnreadCount {
  total: number;
  by_category: NotificationUnreadByCategory;
}

/** 创建系统公告请求。 */
export interface CreateSystemAnnouncementRequest {
  title: string;
  content: string;
  scheduled_at?: string | null;
}

/** 更新系统公告请求。 */
export interface UpdateSystemAnnouncementRequest {
  title?: string;
  content?: string;
  scheduled_at?: string | null;
  is_pinned?: boolean;
}

/** 公告列表查询参数。 */
export interface NotificationAnnouncementListParams extends QueryParams {
  page?: number;
  page_size?: number;
  status?: AnnouncementStatus;
}

/** 系统公告项。 */
export interface SystemAnnouncementItem {
  id: ID;
  title: string;
  content: string;
  is_pinned: boolean;
  is_read?: boolean | null;
  status?: AnnouncementStatus | null;
  status_text?: string | null;
  published_at: string | null;
  created_at?: string | null;
}

/** 公告详情。 */
export interface SystemAnnouncementDetail extends SystemAnnouncementItem {
  scheduled_at?: string | null;
  unpublished_at?: string | null;
  updated_at?: string | null;
}

/** 公告列表响应。 */
export type SystemAnnouncementListResponse = PaginatedData<SystemAnnouncementItem>;

/** 发送定向通知请求。 */
export interface SendNotificationRequest {
  title: string;
  content: string;
  target_type: "all_school" | "course" | "user" | "users";
  target_id: string;
  category: NotificationCategory;
}

/** 通知偏好项。 */
export interface NotificationPreferenceItem {
  category: NotificationCategory;
  category_text: string;
  is_enabled: boolean;
  is_forced: boolean;
}

/** 通知偏好响应。 */
export interface NotificationPreferencesResponse {
  preferences: NotificationPreferenceItem[];
}

/** 更新通知偏好项。 */
export interface UpdateNotificationPreferenceItem {
  category: NotificationCategory;
  is_enabled: boolean;
}

/** 模板变量定义。 */
export interface NotificationTemplateVariable {
  name: string;
  description: string;
  required: boolean;
}

/** 模板列表项。 */
export interface NotificationTemplateListItem {
  id: ID;
  event_type: string;
  category: NotificationCategory;
  category_text: string;
  title_template: string;
  content_template: string;
  variables: NotificationTemplateVariable[];
  is_enabled: boolean;
}

/** 模板详情。 */
export interface NotificationTemplateDetail extends NotificationTemplateListItem {
  created_at?: string | null;
  updated_at?: string | null;
}

/** 模板列表响应。 */
export interface NotificationTemplateListResponse {
  list: NotificationTemplateListItem[];
}

/** 更新模板请求。 */
export interface UpdateNotificationTemplateRequest {
  title_template: string;
  content_template: string;
  is_enabled: boolean;
}

/** 模板预览响应。 */
export interface PreviewNotificationTemplateResponse {
  title: string;
  content: string;
}

/** 消息统计查询参数。 */
export interface NotificationStatisticsParams extends QueryParams {
  date_from?: string;
  date_to?: string;
}

/** 分类统计项。 */
export interface NotificationCategoryStat {
  category: string;
  sent: number;
  read: number;
  read_rate: number;
}

/** 每日消息趋势。 */
export interface NotificationDailyTrendItem {
  date: string;
  sent: number;
  read: number;
}

/** 消息统计响应。 */
export interface NotificationStatistics {
  total_sent: number;
  total_read: number;
  read_rate: number;
  by_category: NotificationCategoryStat[];
  daily_trend: NotificationDailyTrendItem[];
}

/** 通知 WebSocket 消息。 */
export interface NotificationRealtimeMessage {
  type: "unread_count_update" | "new_notification" | "new_announcement" | "pong";
  channel?: string | null;
  data?: unknown;
}
