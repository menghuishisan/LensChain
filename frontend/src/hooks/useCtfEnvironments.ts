"use client";

// useCtfEnvironments.ts
// 模块05题目环境 hook，封装环境启动、详情、重置、销毁和提交记录缓存。

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import {
  destroyCtfChallengeEnvironment,
  forceDestroyCtfChallengeEnvironment,
  getCtfChallengeEnvironment,
  listCtfCompetitionEnvironments,
  listCtfSubmissions,
  listMyCtfEnvironments,
  resetCtfChallengeEnvironment,
  startCtfChallengeEnvironment,
  submitCtfChallenge,
} from "@/services/ctf";
import type { ID, QueryParams } from "@/types/api";
import type { CtfEnvironmentListParams } from "@/types/ctf";

/**
 * useCtfChallengeEnvironment 查询题目环境详情。
 */
export function useCtfChallengeEnvironment(environmentID: ID) {
  return useQuery({ queryKey: ["ctf", "environment", environmentID], queryFn: () => getCtfChallengeEnvironment(environmentID), enabled: environmentID.length > 0, refetchInterval: 5000 });
}

/**
 * useMyCtfEnvironments 查询我的所有题目环境。
 */
export function useMyCtfEnvironments(competitionID: ID) {
  return useQuery({ queryKey: ["ctf", "my-environments", competitionID], queryFn: () => listMyCtfEnvironments(competitionID), enabled: competitionID.length > 0 });
}

/**
 * useCtfCompetitionEnvironments 查询竞赛环境资源列表。
 */
export function useCtfCompetitionEnvironments(competitionID: ID, params: CtfEnvironmentListParams = {}) {
  return useQuery({ queryKey: ["ctf", "competition-environments", competitionID, params], queryFn: () => listCtfCompetitionEnvironments(competitionID, params), enabled: competitionID.length > 0 });
}

/**
 * useCtfEnvironmentMutations 启动、重置和销毁题目环境。
 */
export function useCtfEnvironmentMutations(competitionID: ID, challengeID?: ID, environmentID?: ID) {
  const queryClient = useQueryClient();
  const refreshEnvironmentState = () => {
    void queryClient.invalidateQueries({ queryKey: ["ctf", "my-environments", competitionID] });
    void queryClient.invalidateQueries({ queryKey: ["ctf", "competition-challenges", competitionID] });
    if (environmentID) {
      void queryClient.invalidateQueries({ queryKey: ["ctf", "environment", environmentID] });
    }
    void queryClient.invalidateQueries({ queryKey: ["ctf", "competition-environments", competitionID] });
  };

  return {
    start: useMutation({ mutationFn: () => startCtfChallengeEnvironment(competitionID, challengeID ?? ""), onSuccess: refreshEnvironmentState }),
    reset: useMutation({ mutationFn: () => resetCtfChallengeEnvironment(environmentID ?? ""), onSuccess: refreshEnvironmentState }),
    destroy: useMutation({ mutationFn: () => destroyCtfChallengeEnvironment(environmentID ?? ""), onSuccess: refreshEnvironmentState }),
    forceDestroy: useMutation({ mutationFn: (reason?: string) => forceDestroyCtfChallengeEnvironment(environmentID ?? "", reason), onSuccess: refreshEnvironmentState }),
  };
}

/**
 * useCtfSubmissions 查询团队提交记录。
 */
export function useCtfSubmissions(competitionID: ID, params: QueryParams = {}) {
  return useQuery({ queryKey: ["ctf", "submissions", competitionID, params], queryFn: () => listCtfSubmissions(competitionID, params), enabled: competitionID.length > 0 });
}

/**
 * useSubmitCtfChallengeMutation 提交 Flag 或攻击交易，成功后刷新提交、排行榜和题目列表。
 */
export function useSubmitCtfChallengeMutation(competitionID: ID) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (payload: Parameters<typeof submitCtfChallenge>[1]) => submitCtfChallenge(competitionID, payload),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["ctf", "submissions", competitionID] });
      void queryClient.invalidateQueries({ queryKey: ["ctf", "leaderboard", competitionID] });
      void queryClient.invalidateQueries({ queryKey: ["ctf", "competition-challenges", competitionID] });
    },
  });
}
