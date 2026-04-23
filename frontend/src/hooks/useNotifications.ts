"use client";

// useNotifications.ts
// 模块07收件箱 hook，统一管理消息列表、未读计数、批量已读、全部已读和删除缓存。

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import {
  batchReadInbox,
  deleteInboxMessage,
  getInboxDetail,
  getUnreadCount,
  listInbox,
  markInboxRead,
  readAllInbox,
} from "@/services/notification";
import type { ID } from "@/types/api";
import type { NotificationInboxListParams } from "@/types/notification";

/**
 * useNotifications 查询收件箱列表。
 */
export function useNotifications(params: NotificationInboxListParams) {
  return useQuery({ queryKey: ["notification", "inbox", params], queryFn: () => listInbox(params) });
}

/**
 * useNotification 查询消息详情。
 */
export function useNotification(messageID: ID) {
  return useQuery({ queryKey: ["notification", "detail", messageID], queryFn: () => getInboxDetail(messageID), enabled: messageID.length > 0 });
}

/**
 * useUnreadCount 查询未读数。
 */
export function useUnreadCount() {
  return useQuery({ queryKey: ["notification", "unread-count"], queryFn: getUnreadCount });
}

/**
 * useNotificationMutations 标记已读、批量已读、全部已读和删除消息。
 */
export function useNotificationMutations(messageID?: ID) {
  const queryClient = useQueryClient();
  const refreshInbox = () => {
    void queryClient.invalidateQueries({ queryKey: ["notification", "inbox"] });
    void queryClient.invalidateQueries({ queryKey: ["notification", "unread-count"] });
    if (messageID) {
      void queryClient.invalidateQueries({ queryKey: ["notification", "detail", messageID] });
    }
  };

  return {
    markRead: useMutation({ mutationFn: () => markInboxRead(messageID ?? ""), onSuccess: refreshInbox }),
    batchRead: useMutation({ mutationFn: batchReadInbox, onSuccess: refreshInbox }),
    readAll: useMutation({ mutationFn: readAllInbox, onSuccess: refreshInbox }),
    remove: useMutation({ mutationFn: () => deleteInboxMessage(messageID ?? ""), onSuccess: refreshInbox }),
  };
}
