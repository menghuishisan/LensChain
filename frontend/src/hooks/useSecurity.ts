"use client";

// useSecurity.ts
// 模块01安全策略 hook，封装安全策略读取和保存。

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { getSecurityPolicy, updateSecurityPolicy } from "@/services/auth";

/**
 * securityPolicyQueryKey 安全策略 Query key。
 */
export const securityPolicyQueryKey = ["auth", "security-policy"] as const;

/**
 * useSecurityPolicy 查询 GET /security-policies。
 */
export function useSecurityPolicy() {
  return useQuery({
    queryKey: securityPolicyQueryKey,
    queryFn: getSecurityPolicy,
  });
}

/**
 * useUpdateSecurityPolicyMutation 调用 PUT /security-policies，成功后刷新安全策略缓存。
 */
export function useUpdateSecurityPolicyMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: updateSecurityPolicy,
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: securityPolicyQueryKey });
    },
  });
}
