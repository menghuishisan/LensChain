"use client";

// useSchoolApplications.ts
// 模块02入驻申请 hook，封装公开申请、验证码、状态查询、重新申请和超管审核。

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import {
  approveSchoolApplication,
  getSchoolApplicationReapplyDetail,
  getSchoolApplication,
  listSchoolApplications,
  querySchoolApplications,
  reapplySchoolApplication,
  rejectSchoolApplication,
  sendSchoolApplicationSmsCode,
  submitSchoolApplication,
} from "@/services/school";
import type { ID } from "@/types/api";
import type { AdminSchoolApplicationListParams, QuerySchoolApplicationsParams, ReapplySchoolApplicationRequest, SchoolApplicationReapplyDetailParams } from "@/types/school";

/**
 * schoolApplicationsQueryKey 入驻申请列表 Query key。
 */
export function schoolApplicationsQueryKey(params: AdminSchoolApplicationListParams) {
  return ["school", "applications", params] as const;
}

/**
 * schoolApplicationDetailQueryKey 入驻申请详情 Query key。
 */
export function schoolApplicationDetailQueryKey(id: ID) {
  return ["school", "applications", id] as const;
}

/**
 * useSubmitSchoolApplicationMutation 提交公开入驻申请。
 */
export function useSubmitSchoolApplicationMutation() {
  return useMutation({ mutationFn: submitSchoolApplication });
}

/**
 * useSendSchoolApplicationSmsMutation 发送申请查询验证码。
 */
export function useSendSchoolApplicationSmsMutation() {
  return useMutation({ mutationFn: sendSchoolApplicationSmsCode });
}

/**
 * useQuerySchoolApplications 按手机号和验证码查询公开申请状态。
 */
export function useQuerySchoolApplications(params: QuerySchoolApplicationsParams | null) {
  return useQuery({
    queryKey: ["school", "application-query", params],
    queryFn: () => {
      if (params === null) {
        throw new Error("缺少查询参数");
      }
      return querySchoolApplications(params);
    },
    enabled: params !== null,
  });
}

/**
 * useSchoolApplicationReapplyDetail 查询被拒申请完整字段，用于重新申请预填。
 */
export function useSchoolApplicationReapplyDetail(id: ID | null, params: SchoolApplicationReapplyDetailParams | null) {
  return useQuery({
    queryKey: ["school", "application-reapply-detail", id, params],
    queryFn: () => {
      if (id === null || params === null) {
        throw new Error("缺少重新申请查询参数");
      }
      return getSchoolApplicationReapplyDetail(id, params);
    },
    enabled: id !== null && params !== null,
  });
}

/**
 * useReapplySchoolApplicationMutation 提交被拒申请的重新申请。
 */
export function useReapplySchoolApplicationMutation(id: ID) {
  return useMutation({
    mutationFn: (payload: ReapplySchoolApplicationRequest) => reapplySchoolApplication(id, payload),
  });
}

/**
 * useSchoolApplications 查询超管入驻申请列表。
 */
export function useSchoolApplications(params: AdminSchoolApplicationListParams) {
  return useQuery({
    queryKey: schoolApplicationsQueryKey(params),
    queryFn: () => listSchoolApplications(params),
  });
}

/**
 * useSchoolApplication 查询超管入驻申请详情。
 */
export function useSchoolApplication(id: ID) {
  return useQuery({
    queryKey: schoolApplicationDetailQueryKey(id),
    queryFn: () => getSchoolApplication(id),
    enabled: id.length > 0,
  });
}

/**
 * useApproveSchoolApplicationMutation 审核通过入驻申请，成功后刷新申请和学校列表。
 */
export function useApproveSchoolApplicationMutation(id: ID) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: approveSchoolApplication.bind(null, id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["school", "applications"] });
      void queryClient.invalidateQueries({ queryKey: schoolApplicationDetailQueryKey(id) });
      void queryClient.invalidateQueries({ queryKey: ["school", "schools"] });
    },
  });
}

/**
 * useRejectSchoolApplicationMutation 审核拒绝入驻申请，成功后刷新申请缓存。
 */
export function useRejectSchoolApplicationMutation(id: ID) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: rejectSchoolApplication.bind(null, id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["school", "applications"] });
      void queryClient.invalidateQueries({ queryKey: schoolApplicationDetailQueryKey(id) });
    },
  });
}
