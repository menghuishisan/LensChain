"use client";

// useAnnouncements.ts
// 模块07公告 hook，封装列表、详情、创建、编辑、发布、下架和删除缓存。

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import {
  createNotificationAnnouncement,
  deleteNotificationAnnouncement,
  getNotificationAnnouncement,
  listNotificationAnnouncements,
  publishNotificationAnnouncement,
  unpublishNotificationAnnouncement,
  updateNotificationAnnouncement,
} from "@/services/notification";
import type { ID } from "@/types/api";
import type { CreateSystemAnnouncementRequest, NotificationAnnouncementListParams, UpdateSystemAnnouncementRequest } from "@/types/notification";

/**
 * useAnnouncements 查询公告列表。
 */
export function useAnnouncements(params: NotificationAnnouncementListParams = {}) {
  return useQuery({ queryKey: ["notification", "announcements", params], queryFn: () => listNotificationAnnouncements(params) });
}

/**
 * useAnnouncement 查询公告详情。
 */
export function useAnnouncement(announcementID: ID) {
  return useQuery({ queryKey: ["notification", "announcement", announcementID], queryFn: () => getNotificationAnnouncement(announcementID), enabled: announcementID.length > 0 });
}

/**
 * useAnnouncementMutations 创建、编辑、发布、下架和删除公告。
 */
export function useAnnouncementMutations(announcementID?: ID) {
  const queryClient = useQueryClient();
  const refreshAnnouncements = () => {
    void queryClient.invalidateQueries({ queryKey: ["notification", "announcements"] });
    if (announcementID) {
      void queryClient.invalidateQueries({ queryKey: ["notification", "announcement", announcementID] });
    }
  };

  return {
    create: useMutation({ mutationFn: (payload: CreateSystemAnnouncementRequest) => createNotificationAnnouncement(payload), onSuccess: refreshAnnouncements }),
    update: useMutation({ mutationFn: (payload: UpdateSystemAnnouncementRequest) => updateNotificationAnnouncement(announcementID ?? "", payload), onSuccess: refreshAnnouncements }),
    publish: useMutation({ mutationFn: () => publishNotificationAnnouncement(announcementID ?? ""), onSuccess: refreshAnnouncements }),
    unpublish: useMutation({ mutationFn: () => unpublishNotificationAnnouncement(announcementID ?? ""), onSuccess: refreshAnnouncements }),
    remove: useMutation({ mutationFn: () => deleteNotificationAnnouncement(announcementID ?? ""), onSuccess: refreshAnnouncements }),
  };
}
