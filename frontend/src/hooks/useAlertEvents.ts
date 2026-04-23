"use client";

// useAlertEvents.ts
// 模块08告警事件 hook，封装事件列表、详情和处理/忽略后的跨面板刷新策略。

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { getAlertEvent, handleAlertEvent, ignoreAlertEvent, listAlertEvents } from "@/services/system";
import type { ID } from "@/types/api";
import type { HandleSystemAlertEventRequest, SystemAlertEventListParams } from "@/types/system";

import {
  systemDashboardHealthQueryKey,
  systemDashboardRealtimeQueryKey,
  systemDashboardResourcesQueryKey,
} from "@/hooks/useSystemDashboard";

/**
 * alertEventsQueryKey 告警事件列表 Query key。
 */
export function alertEventsQueryKey(params: SystemAlertEventListParams) {
  return ["system", "alert-events", params] as const;
}

/**
 * alertEventDetailQueryKey 告警事件详情 Query key。
 */
export function alertEventDetailQueryKey(eventID: ID) {
  return ["system", "alert-events", "detail", eventID] as const;
}

/**
 * useAlertEvents 查询告警事件列表，并按文档要求每 60 秒自动刷新。
 */
export function useAlertEvents(params: SystemAlertEventListParams, enabled = true) {
  return useQuery({
    queryKey: alertEventsQueryKey(params),
    queryFn: () => listAlertEvents(params),
    enabled,
    refetchInterval: 60 * 1000,
  });
}

/**
 * useAlertEvent 查询单条告警事件详情。
 */
export function useAlertEvent(eventID: ID) {
  return useQuery({
    queryKey: alertEventDetailQueryKey(eventID),
    queryFn: () => getAlertEvent(eventID),
    enabled: eventID.length > 0,
  });
}

/**
 * useAlertEventMutations 处理或忽略告警，并刷新列表、详情与仪表盘。
 */
export function useAlertEventMutations(eventID?: ID) {
  const queryClient = useQueryClient();

  const refreshEventAndDashboard = () => {
    // 告警状态变化会直接影响事件列表、事件详情以及仪表盘中的待处理数量和最近告警。
    void queryClient.invalidateQueries({ queryKey: ["system", "alert-events"] });
    if (eventID !== undefined && eventID.length > 0) {
      void queryClient.invalidateQueries({ queryKey: alertEventDetailQueryKey(eventID) });
    }
    void queryClient.invalidateQueries({ queryKey: systemDashboardHealthQueryKey });
    void queryClient.invalidateQueries({ queryKey: systemDashboardResourcesQueryKey });
    void queryClient.invalidateQueries({ queryKey: systemDashboardRealtimeQueryKey });
  };

  return {
    handle: useMutation({
      mutationFn: (payload: HandleSystemAlertEventRequest) => handleAlertEvent(eventID ?? "", payload),
      onSuccess: refreshEventAndDashboard,
    }),
    ignore: useMutation({
      mutationFn: (payload: HandleSystemAlertEventRequest) => ignoreAlertEvent(eventID ?? "", payload),
      onSuccess: refreshEventAndDashboard,
    }),
  };
}
