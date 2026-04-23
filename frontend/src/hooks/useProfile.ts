"use client";

// useProfile.ts
// 模块01个人中心 hook，仅读写 /profile 基础资料，不读取模块06学习概览。

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { getProfile, updateProfile } from "@/services/auth";

/**
 * profileQueryKey 个人基础资料 Query key。
 */
export const profileQueryKey = ["auth", "profile"] as const;

/**
 * useProfile 查询 GET /profile，仅返回个人基础资料。
 */
export function useProfile() {
  return useQuery({
    queryKey: profileQueryKey,
    queryFn: getProfile,
  });
}

/**
 * useUpdateProfileMutation 调用 PUT /profile，成功后刷新个人资料缓存。
 */
export function useUpdateProfileMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: updateProfile,
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: profileQueryKey });
    },
  });
}
