"use client";

// useNotificationTemplates.ts
// 模块07模板 hook，封装模板列表、详情、更新和安全预览缓存。

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import {
  getNotificationPreferences,
  getNotificationStatistics,
  getNotificationTemplate,
  listNotificationTemplates,
  previewNotificationTemplate,
  sendDirectNotification,
  updateNotificationPreferences,
  updateNotificationTemplate,
} from "@/services/notification";
import type { ID } from "@/types/api";
import type {
  NotificationStatisticsParams,
  SendNotificationRequest,
  UpdateNotificationPreferenceItem,
  UpdateNotificationTemplateRequest,
} from "@/types/notification";

/**
 * useNotificationTemplates 查询模板列表。
 */
export function useNotificationTemplates() {
  return useQuery({ queryKey: ["notification", "templates"], queryFn: listNotificationTemplates });
}

/**
 * useNotificationTemplate 查询模板详情。
 */
export function useNotificationTemplate(templateID: ID) {
  return useQuery({ queryKey: ["notification", "template", templateID], queryFn: () => getNotificationTemplate(templateID), enabled: templateID.length > 0 });
}

/**
 * useNotificationTemplateMutations 更新模板和预览模板渲染效果。
 */
export function useNotificationTemplateMutations(templateID?: ID) {
  const queryClient = useQueryClient();
  return {
    update: useMutation({
      mutationFn: (payload: UpdateNotificationTemplateRequest) => updateNotificationTemplate(templateID ?? "", payload),
      onSuccess: () => {
        void queryClient.invalidateQueries({ queryKey: ["notification", "templates"] });
        if (templateID) {
          void queryClient.invalidateQueries({ queryKey: ["notification", "template", templateID] });
        }
      },
    }),
    preview: useMutation({ mutationFn: (params: Record<string, unknown>) => previewNotificationTemplate(templateID ?? "", params) }),
  };
}

/**
 * useNotificationPreferences 查询通知偏好设置。
 */
export function useNotificationPreferences() {
  return useQuery({ queryKey: ["notification", "preferences"], queryFn: getNotificationPreferences });
}

/**
 * useNotificationPreferenceMutation 更新通知偏好设置。
 */
export function useNotificationPreferenceMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (preferences: UpdateNotificationPreferenceItem[]) => updateNotificationPreferences(preferences),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: ["notification", "preferences"] }),
  });
}

/**
 * useNotificationStatistics 查询消息统计。
 */
export function useNotificationStatistics(params: NotificationStatisticsParams = {}) {
  return useQuery({ queryKey: ["notification", "statistics", params], queryFn: () => getNotificationStatistics(params) });
}

/**
 * useDirectNotificationMutation 发送定向通知。
 */
export function useDirectNotificationMutation() {
  return useMutation({ mutationFn: (payload: SendNotificationRequest) => sendDirectNotification(payload) });
}
