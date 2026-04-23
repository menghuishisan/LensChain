"use client";

// useSystemDashboard.ts
// 模块08运维仪表盘 hook，封装健康状态、资源使用和实时指标的自动刷新查询。

import { useQuery } from "@tanstack/react-query";

import { getDashboardHealth, getDashboardRealtime, getDashboardResources } from "@/services/system";

/**
 * systemDashboardHealthQueryKey 健康状态 Query key。
 */
export const systemDashboardHealthQueryKey = ["system", "dashboard", "health"] as const;

/**
 * systemDashboardResourcesQueryKey 资源使用 Query key。
 */
export const systemDashboardResourcesQueryKey = ["system", "dashboard", "resources"] as const;

/**
 * systemDashboardRealtimeQueryKey 实时指标 Query key。
 */
export const systemDashboardRealtimeQueryKey = ["system", "dashboard", "realtime"] as const;

/**
 * useSystemDashboard 查询仪表盘三类数据，并按文档要求每 30 秒自动刷新。
 */
export function useSystemDashboard() {
  const healthQuery = useQuery({
    queryKey: systemDashboardHealthQueryKey,
    queryFn: getDashboardHealth,
    refetchInterval: 30 * 1000,
  });

  const resourcesQuery = useQuery({
    queryKey: systemDashboardResourcesQueryKey,
    queryFn: getDashboardResources,
    refetchInterval: 30 * 1000,
  });

  const realtimeQuery = useQuery({
    queryKey: systemDashboardRealtimeQueryKey,
    queryFn: getDashboardRealtime,
    refetchInterval: 30 * 1000,
  });

  return {
    healthQuery,
    resourcesQuery,
    realtimeQuery,
  };
}
