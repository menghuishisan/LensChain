"use client";

// useUsers.ts
// 模块01用户管理 hook，封装用户列表、详情、创建、更新、状态变更和删除缓存策略。

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import {
  batchDeleteUsers,
  createSuperAdmin,
  createUser,
  deleteUser,
  getUser,
  listUsers,
  resetUserPassword,
  unlockUser,
  updateUser,
  updateUserStatus,
} from "@/services/auth";
import type { ID } from "@/types/api";
import type { UserListParams } from "@/types/auth";

/**
 * usersQueryKey 用户列表 Query key。
 */
export function usersQueryKey(params: UserListParams) {
  return ["auth", "users", params] as const;
}

/**
 * userDetailQueryKey 用户详情 Query key。
 */
export function userDetailQueryKey(id: ID) {
  return ["auth", "users", id] as const;
}

/**
 * useUsers 查询 GET /users。
 */
export function useUsers(params: UserListParams) {
  return useQuery({
    queryKey: usersQueryKey(params),
    queryFn: () => listUsers(params),
  });
}

/**
 * useUser 查询 GET /users/:id。
 */
export function useUser(id: ID) {
  return useQuery({
    queryKey: userDetailQueryKey(id),
    queryFn: () => getUser(id),
    enabled: id.length > 0,
  });
}

/**
 * useCreateUserMutation 调用 POST /users，成功后刷新用户列表。
 */
export function useCreateUserMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: createUser,
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["auth", "users"] });
    },
  });
}

/**
 * useCreateSuperAdminMutation 调用 POST /users/super-admins，成功后刷新用户列表。
 */
export function useCreateSuperAdminMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: createSuperAdmin,
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["auth", "users"] });
    },
  });
}

/**
 * useUpdateUserMutation 调用 PUT /users/:id，成功后刷新用户列表和详情。
 */
export function useUpdateUserMutation(id: ID) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (payload: Parameters<typeof updateUser>[1]) => updateUser(id, payload),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["auth", "users"] });
      void queryClient.invalidateQueries({ queryKey: userDetailQueryKey(id) });
    },
  });
}

/**
 * useDeleteUserMutation 调用 DELETE /users/:id，成功后刷新用户列表。
 */
export function useDeleteUserMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: deleteUser,
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["auth", "users"] });
    },
  });
}

/**
 * useUpdateUserStatusMutation 调用 PATCH /users/:id/status，成功后刷新用户缓存。
 */
export function useUpdateUserStatusMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, payload }: { id: ID; payload: Parameters<typeof updateUserStatus>[1] }) => updateUserStatus(id, payload),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["auth", "users"] });
    },
  });
}

/**
 * useResetUserPasswordMutation 调用 POST /users/:id/reset-password，成功后刷新用户缓存。
 */
export function useResetUserPasswordMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, payload }: { id: ID; payload: Parameters<typeof resetUserPassword>[1] }) => resetUserPassword(id, payload),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["auth", "users"] });
    },
  });
}

/**
 * useUnlockUserMutation 调用 POST /users/:id/unlock，成功后刷新用户缓存。
 */
export function useUnlockUserMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: unlockUser,
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["auth", "users"] });
    },
  });
}

/**
 * useBatchDeleteUsersMutation 调用 POST /users/batch-delete，成功后刷新用户列表。
 */
export function useBatchDeleteUsersMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: batchDeleteUsers,
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["auth", "users"] });
    },
  });
}
