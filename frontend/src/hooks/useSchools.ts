"use client";

// useSchools.ts
// 模块02学校管理 hook，封装超管学校CRUD、授权有效期和生命周期操作。

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import {
  cancelSchool,
  createSchool,
  freezeSchool,
  getSchool,
  getSchoolLicense,
  getSchoolProfile,
  listSchools,
  restoreSchool,
  unfreezeSchool,
  updateSchool,
  updateSchoolLicense,
  updateSchoolProfile,
} from "@/services/school";
import type { ID } from "@/types/api";
import type { SchoolListParams } from "@/types/school";

/**
 * schoolsQueryKey 学校列表 Query key。
 */
export function schoolsQueryKey(params: SchoolListParams) {
  return ["school", "schools", params] as const;
}

/**
 * schoolDetailQueryKey 学校详情 Query key。
 */
export function schoolDetailQueryKey(id: ID) {
  return ["school", "schools", id] as const;
}

/**
 * schoolProfileQueryKey 本校资料 Query key。
 */
export const schoolProfileQueryKey = ["school", "profile"] as const;

/**
 * schoolLicenseQueryKey 本校授权 Query key。
 */
export const schoolLicenseQueryKey = ["school", "license"] as const;

/**
 * useSchools 查询超管学校列表。
 */
export function useSchools(params: SchoolListParams) {
  return useQuery({
    queryKey: schoolsQueryKey(params),
    queryFn: () => listSchools(params),
  });
}

/**
 * useSchool 查询超管学校详情。
 */
export function useSchool(id: ID) {
  return useQuery({
    queryKey: schoolDetailQueryKey(id),
    queryFn: () => getSchool(id),
    enabled: id.length > 0,
  });
}

/**
 * useCreateSchoolMutation 超管后台直接创建学校。
 */
export function useCreateSchoolMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: createSchool,
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["school", "schools"] });
    },
  });
}

/**
 * useUpdateSchoolMutation 超管编辑学校信息。
 */
export function useUpdateSchoolMutation(id: ID) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (payload: Parameters<typeof updateSchool>[1]) => updateSchool(id, payload),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["school", "schools"] });
      void queryClient.invalidateQueries({ queryKey: schoolDetailQueryKey(id) });
    },
  });
}

/**
 * useUpdateSchoolLicenseMutation 设置学校授权有效期。
 */
export function useUpdateSchoolLicenseMutation(id: ID) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (payload: Parameters<typeof updateSchoolLicense>[1]) => updateSchoolLicense(id, payload),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["school", "schools"] });
      void queryClient.invalidateQueries({ queryKey: schoolDetailQueryKey(id) });
    },
  });
}

/**
 * useSchoolLifecycleMutations 学校冻结、解冻、注销、恢复生命周期操作。
 */
export function useSchoolLifecycleMutations() {
  const queryClient = useQueryClient();
  const invalidateSchools = () => {
    void queryClient.invalidateQueries({ queryKey: ["school", "schools"] });
  };

  return {
    freeze: useMutation({ mutationFn: ({ id, reason }: { id: ID; reason: string }) => freezeSchool(id, { reason }), onSuccess: invalidateSchools }),
    unfreeze: useMutation({ mutationFn: unfreezeSchool, onSuccess: invalidateSchools }),
    cancel: useMutation({ mutationFn: cancelSchool, onSuccess: invalidateSchools }),
    restore: useMutation({ mutationFn: restoreSchool, onSuccess: invalidateSchools }),
  };
}

/**
 * useSchoolProfile 查询校管本校资料。
 */
export function useSchoolProfile() {
  return useQuery({
    queryKey: schoolProfileQueryKey,
    queryFn: getSchoolProfile,
  });
}

/**
 * useUpdateSchoolProfileMutation 校管编辑本校资料，成功后刷新本校缓存。
 */
export function useUpdateSchoolProfileMutation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: updateSchoolProfile,
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: schoolProfileQueryKey });
    },
  });
}

/**
 * useSchoolLicense 查询校管授权状态。
 */
export function useSchoolLicense() {
  return useQuery({
    queryKey: schoolLicenseQueryKey,
    queryFn: getSchoolLicense,
  });
}
