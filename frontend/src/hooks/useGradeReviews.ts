"use client";

// useGradeReviews.ts
// 模块06成绩审核 hook，封装教师提交、学校管理员审核通过/驳回/解锁和详情查询。

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import {
  approveGradeReview,
  getGradeReview,
  listGradeReviews,
  rejectGradeReview,
  submitGradeReview,
  unlockGradeReview,
} from "@/services/grade";
import type { ID } from "@/types/api";
import type { GradeReviewListParams, ReviewHandleRequest, SubmitGradeReviewRequest, UnlockGradeReviewRequest } from "@/types/grade";

/**
 * useGradeReviews 查询成绩审核列表。
 */
export function useGradeReviews(params: GradeReviewListParams) {
  return useQuery({ queryKey: ["grade", "reviews", params], queryFn: () => listGradeReviews(params) });
}

/**
 * useGradeReview 查询成绩审核详情。
 */
export function useGradeReview(reviewID: ID) {
  return useQuery({ queryKey: ["grade", "review", reviewID], queryFn: () => getGradeReview(reviewID), enabled: reviewID.length > 0 });
}

/**
 * useGradeReviewMutations 成绩审核提交、通过、驳回和解锁。
 */
export function useGradeReviewMutations(reviewID?: ID) {
  const queryClient = useQueryClient();
  const refreshReviews = () => {
    void queryClient.invalidateQueries({ queryKey: ["grade", "reviews"] });
    void queryClient.invalidateQueries({ queryKey: ["grade", "my-semester-grades"] });
    void queryClient.invalidateQueries({ queryKey: ["grade", "my-gpa"] });
    if (reviewID) {
      void queryClient.invalidateQueries({ queryKey: ["grade", "review", reviewID] });
    }
  };

  return {
    submit: useMutation({ mutationFn: (payload: SubmitGradeReviewRequest) => submitGradeReview(payload), onSuccess: refreshReviews }),
    approve: useMutation({ mutationFn: (payload: ReviewHandleRequest) => approveGradeReview(reviewID ?? "", payload), onSuccess: refreshReviews }),
    reject: useMutation({ mutationFn: (payload: ReviewHandleRequest) => rejectGradeReview(reviewID ?? "", payload), onSuccess: refreshReviews }),
    unlock: useMutation({ mutationFn: (payload: UnlockGradeReviewRequest) => unlockGradeReview(reviewID ?? "", payload), onSuccess: refreshReviews }),
  };
}
