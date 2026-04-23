"use client";

// useBackups.ts
// 模块08数据备份 hook，封装备份列表轮询、备份配置与下载/触发/更新操作。

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { downloadBackup, getBackupConfig, listBackups, triggerBackup, updateBackupConfig } from "@/services/system";
import type { SystemBackupListParams, UpdateSystemBackupConfigRequest } from "@/types/system";

/**
 * backupsQueryKey 备份列表 Query key。
 */
export function backupsQueryKey(params: SystemBackupListParams) {
  return ["system", "backups", "list", params] as const;
}

/**
 * backupConfigQueryKey 备份配置 Query key。
 */
export const backupConfigQueryKey = ["system", "backups", "config"] as const;

/**
 * useBackups 查询备份列表。
 * 若存在进行中的备份，则每 10 秒轮询一次，确保状态、文件大小与错误信息及时更新。
 */
export function useBackups(params: SystemBackupListParams) {
  return useQuery({
    queryKey: backupsQueryKey(params),
    queryFn: () => listBackups(params),
    refetchInterval: (query) =>
      query.state.data?.list.some((item) => item.status === 1) === true ? 10 * 1000 : false,
  });
}

/**
 * useBackupConfig 查询备份配置。
 */
export function useBackupConfig() {
  return useQuery({
    queryKey: backupConfigQueryKey,
    queryFn: getBackupConfig,
  });
}

/**
 * useBackupMutations 管理手动备份、下载和备份配置更新。
 */
export function useBackupMutations() {
  const queryClient = useQueryClient();

  const refreshBackups = () => {
    // 备份配置底层写入 system_configs.backup 分组，因此需要同步刷新备份配置、
    // 备份列表、系统配置列表以及配置变更记录。
    void queryClient.invalidateQueries({ queryKey: ["system", "backups"] });
    void queryClient.invalidateQueries({ queryKey: backupConfigQueryKey });
    void queryClient.invalidateQueries({ queryKey: ["system", "configs"] });
    void queryClient.invalidateQueries({ queryKey: ["system", "configs", "change-logs"] });
  };

  return {
    trigger: useMutation({
      mutationFn: triggerBackup,
      onSuccess: refreshBackups,
    }),
    download: useMutation({
      mutationFn: downloadBackup,
    }),
    updateConfig: useMutation({
      mutationFn: (payload: UpdateSystemBackupConfigRequest) => updateBackupConfig(payload),
      onSuccess: refreshBackups,
    }),
  };
}
