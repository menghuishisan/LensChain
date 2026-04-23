"use client";

// useSystemStatistics.ts
// 模块08平台统计 hook，封装总览、趋势和学校排行查询。

import { useQuery } from "@tanstack/react-query";

import { getStatisticsOverview, getStatisticsTrend, getSystemSchoolRanking } from "@/services/system";
import type { SystemStatisticsTrendParams } from "@/types/system";

/**
 * systemStatisticsOverviewQueryKey 平台统计总览 Query key。
 */
export const systemStatisticsOverviewQueryKey = ["system", "statistics", "overview"] as const;

/**
 * systemStatisticsTrendQueryKey 平台趋势 Query key。
 */
export function systemStatisticsTrendQueryKey(params: SystemStatisticsTrendParams) {
  return ["system", "statistics", "trend", params] as const;
}

/**
 * systemSchoolRankingQueryKey 学校活跃度排行 Query key。
 */
export const systemSchoolRankingQueryKey = ["system", "statistics", "schools"] as const;

/**
 * useSystemStatisticsOverview 查询平台统计总览。
 */
export function useSystemStatisticsOverview() {
  return useQuery({
    queryKey: systemStatisticsOverviewQueryKey,
    queryFn: getStatisticsOverview,
  });
}

/**
 * useSystemStatisticsTrend 查询平台趋势数据。
 */
export function useSystemStatisticsTrend(params: SystemStatisticsTrendParams) {
  return useQuery({
    queryKey: systemStatisticsTrendQueryKey(params),
    queryFn: () => getStatisticsTrend(params),
  });
}

/**
 * useSystemSchoolRanking 查询学校活跃度排行。
 */
export function useSystemSchoolRanking() {
  return useQuery({
    queryKey: systemSchoolRankingQueryKey,
    queryFn: getSystemSchoolRanking,
  });
}
