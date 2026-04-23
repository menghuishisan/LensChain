"use client";

// useSchoolSSO.ts
// 模块02 SSO 配置 hook，封装配置读取、保存、测试和启停。

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import {
  enableSchoolSso,
  getSchoolSsoConfig,
  testSchoolSsoConfig,
  updateSchoolSsoConfig,
} from "@/services/school";

/**
 * schoolSsoConfigQueryKey 本校 SSO 配置 Query key。
 */
export const schoolSsoConfigQueryKey = ["school", "sso-config"] as const;

/**
 * useSchoolSsoConfig 查询本校 SSO 配置。
 */
export function useSchoolSsoConfig() {
  return useQuery({
    queryKey: schoolSsoConfigQueryKey,
    queryFn: getSchoolSsoConfig,
  });
}

/**
 * useUpdateSchoolSsoConfigMutation 保存 SSO 配置，成功后刷新配置并要求重新测试。
 */
export function useUpdateSchoolSsoConfigMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: updateSchoolSsoConfig,
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: schoolSsoConfigQueryKey });
    },
  });
}

/**
 * useTestSchoolSsoConfigMutation 测试 SSO 连接，成功后刷新配置。
 */
export function useTestSchoolSsoConfigMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: testSchoolSsoConfig,
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: schoolSsoConfigQueryKey });
    },
  });
}

/**
 * useEnableSchoolSsoMutation 启用或禁用 SSO，成功后刷新配置。
 */
export function useEnableSchoolSsoMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: enableSchoolSso,
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: schoolSsoConfigQueryKey });
    },
  });
}
