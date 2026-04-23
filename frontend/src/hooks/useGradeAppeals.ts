"use client";

// useGradeAppeals.ts
// 模块06成绩申诉 hook，封装学生申诉、教师处理和申诉详情缓存策略。

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import {
  approveGradeAppeal,
  createGradeAppeal,
  getGradeAppeal,
  listGradeAppeals,
  rejectGradeAppeal,
} from "@/services/grade";
import type { ID } from "@/types/api";
import type { ApproveGradeAppealRequest, CreateGradeAppealRequest, GradeAppealListParams, RejectGradeAppealRequest } from "@/types/grade";

/**
 * useGradeAppeals 查询成绩申诉列表。
 */
export function useGradeAppeals(params: GradeAppealListParams) {
  return useQuery({ queryKey: ["grade", "appeals", params], queryFn: () => listGradeAppeals(params) });
}

/**
 * useGradeAppeal 查询成绩申诉详情。
 */
export function useGradeAppeal(appealID: ID) {
  return useQuery({ queryKey: ["grade", "appeal", appealID], queryFn: () => getGradeAppeal(appealID), enabled: appealID.length > 0 });
}

/**
 * useGradeAppealMutations 提交、同意和驳回成绩申诉。
 */
export function useGradeAppealMutations(appealID?: ID) {
  const queryClient = useQueryClient();
  const refreshAppeals = () => {
    void queryClient.invalidateQueries({ queryKey: ["grade", "appeals"] });
    void queryClient.invalidateQueries({ queryKey: ["grade", "my-semester-grades"] });
    void queryClient.invalidateQueries({ queryKey: ["grade", "my-gpa"] });
    if (appealID) {
      void queryClient.invalidateQueries({ queryKey: ["grade", "appeal", appealID] });
    }
  };

  return {
    create: useMutation({ mutationFn: (payload: CreateGradeAppealRequest) => createGradeAppeal(payload), onSuccess: refreshAppeals }),
    approve: useMutation({ mutationFn: (payload: ApproveGradeAppealRequest) => approveGradeAppeal(appealID ?? "", payload), onSuccess: refreshAppeals }),
    reject: useMutation({ mutationFn: (payload: RejectGradeAppealRequest) => rejectGradeAppeal(appealID ?? "", payload), onSuccess: refreshAppeals }),
  };
}
