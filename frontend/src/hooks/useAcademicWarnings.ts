"use client";

// useAcademicWarnings.ts
// 模块06学业预警 hook，封装预警列表、详情、处理和预警配置缓存策略。

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import {
  getAcademicWarning,
  getWarningConfig,
  handleAcademicWarning,
  listAcademicWarnings,
  updateWarningConfig,
} from "@/services/grade";
import type { ID } from "@/types/api";
import type { AcademicWarningListParams, HandleAcademicWarningRequest, UpdateWarningConfigRequest } from "@/types/grade";

/**
 * useAcademicWarnings 查询学业预警列表。
 */
export function useAcademicWarnings(params: AcademicWarningListParams) {
  return useQuery({ queryKey: ["grade", "warnings", params], queryFn: () => listAcademicWarnings(params) });
}

/**
 * useAcademicWarning 查询学业预警详情。
 */
export function useAcademicWarning(warningID: ID) {
  return useQuery({ queryKey: ["grade", "warning", warningID], queryFn: () => getAcademicWarning(warningID), enabled: warningID.length > 0 });
}

/**
 * useAcademicWarningMutations 处理学业预警和更新预警配置。
 */
export function useAcademicWarningMutations(warningID?: ID) {
  const queryClient = useQueryClient();
  return {
    handle: useMutation({
      mutationFn: (payload: HandleAcademicWarningRequest) => handleAcademicWarning(warningID ?? "", payload),
      onSuccess: () => {
        void queryClient.invalidateQueries({ queryKey: ["grade", "warnings"] });
        if (warningID) {
          void queryClient.invalidateQueries({ queryKey: ["grade", "warning", warningID] });
        }
      },
    }),
    updateConfig: useMutation({
      mutationFn: (payload: UpdateWarningConfigRequest) => updateWarningConfig(payload),
      onSuccess: () => void queryClient.invalidateQueries({ queryKey: ["grade", "warning-config"] }),
    }),
  };
}

/**
 * useWarningConfig 查询学业预警配置。
 */
export function useWarningConfig() {
  return useQuery({ queryKey: ["grade", "warning-config"], queryFn: getWarningConfig });
}
