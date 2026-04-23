"use client";

// useSystemConfigs.ts
// 模块08全局配置 hook，封装配置分组读取、变更记录和配置更新缓存策略。

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import {
  batchUpdateSystemConfigs,
  getConfigGroup,
  getSystemConfigs,
  listConfigChangeLogs,
  updateSystemConfig,
} from "@/services/system";
import type {
  BatchUpdateSystemConfigsRequest,
  SystemConfigChangeLogListParams,
  SystemConfigGroup,
  UpdateSystemConfigRequest,
} from "@/types/system";

/**
 * systemConfigsQueryKey 全部系统配置 Query key。
 */
export const systemConfigsQueryKey = ["system", "configs"] as const;

/**
 * systemConfigGroupQueryKey 单个配置分组 Query key。
 */
export function systemConfigGroupQueryKey(group: SystemConfigGroup) {
  return ["system", "configs", "group", group] as const;
}

/**
 * systemConfigChangeLogsQueryKey 配置变更记录 Query key。
 */
export function systemConfigChangeLogsQueryKey(params: SystemConfigChangeLogListParams) {
  return ["system", "configs", "change-logs", params] as const;
}

/**
 * useSystemConfigs 查询全部系统配置分组。
 */
export function useSystemConfigs() {
  return useQuery({
    queryKey: systemConfigsQueryKey,
    queryFn: getSystemConfigs,
  });
}

/**
 * useSystemConfigGroup 查询单个系统配置分组。
 */
export function useSystemConfigGroup(group: SystemConfigGroup, enabled = true) {
  return useQuery({
    queryKey: systemConfigGroupQueryKey(group),
    queryFn: () => getConfigGroup(group),
    enabled,
  });
}

/**
 * useSystemConfigChangeLogs 查询配置变更记录。
 */
export function useSystemConfigChangeLogs(params: SystemConfigChangeLogListParams, enabled = true) {
  return useQuery({
    queryKey: systemConfigChangeLogsQueryKey(params),
    queryFn: () => listConfigChangeLogs(params),
    enabled,
  });
}

/**
 * useSystemConfigMutations 更新系统配置，并统一处理关联缓存刷新。
 */
export function useSystemConfigMutations() {
  const queryClient = useQueryClient();

  const refreshRelatedCaches = (group: SystemConfigGroup) => {
    // 配置更新既会影响配置分组，也会写入 config_change_logs；
    // backup 分组还会映射到独立备份配置接口，因此要一起失效。
    void queryClient.invalidateQueries({ queryKey: ["system", "configs"] });
    void queryClient.invalidateQueries({ queryKey: systemConfigGroupQueryKey(group) });
    void queryClient.invalidateQueries({ queryKey: ["system", "configs", "change-logs"] });

    if (group === "backup") {
      void queryClient.invalidateQueries({ queryKey: ["system", "backups"] });
      void queryClient.invalidateQueries({ queryKey: ["system", "backups", "config"] });
    }
  };

  return {
    updateOne: useMutation({
      mutationFn: ({
        group,
        key,
        payload,
      }: {
        group: SystemConfigGroup;
        key: string;
        payload: UpdateSystemConfigRequest;
      }) => updateSystemConfig(group, key, payload),
      onSuccess: (_, variables) => refreshRelatedCaches(variables.group),
    }),
    updateGroup: useMutation({
      mutationFn: ({
        group,
        payload,
      }: {
        group: SystemConfigGroup;
        payload: BatchUpdateSystemConfigsRequest;
      }) => batchUpdateSystemConfigs(group, payload),
      onSuccess: (_, variables) => refreshRelatedCaches(variables.group),
    }),
  };
}
