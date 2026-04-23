"use client";

// useCtfBattle.ts
// 模块05攻防赛 hook，封装分组、当前回合、攻击、防守和 Token 相关缓存。

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import {
  autoAssignCtfAdGroups,
  createCtfAdGroup,
  getCurrentAdRound,
  listAdAttacks,
  listAdDefenses,
  listAdGroups,
  listCtfCompetitionTokenLedger,
  submitAdAttack,
  submitAdDefense,
} from "@/services/ctf";
import type { ID, QueryParams } from "@/types/api";

/**
 * useAdGroups 查询竞赛攻防分组。
 */
export function useAdGroups(competitionID: ID) {
  return useQuery({ queryKey: ["ctf", "ad-groups", competitionID], queryFn: () => listAdGroups(competitionID), enabled: competitionID.length > 0 });
}

/**
 * useCtfCompetitionTokenLedger 查询竞赛级 Token 流水。
 */
export function useCtfCompetitionTokenLedger(competitionID: ID, params: QueryParams = {}) {
  return useQuery({ queryKey: ["ctf", "competition-token-ledger", competitionID, params], queryFn: () => listCtfCompetitionTokenLedger(competitionID, params), enabled: competitionID.length > 0 });
}

/**
 * useCurrentAdRound 查询当前回合状态。
 */
export function useCurrentAdRound(groupID: ID) {
  return useQuery({ queryKey: ["ctf", "ad-current-round", groupID], queryFn: () => getCurrentAdRound(groupID), enabled: groupID.length > 0, refetchInterval: 10000 });
}

/**
 * useAdAttacks 查询本回合攻击记录。
 */
export function useAdAttacks(roundID: ID, params: QueryParams = {}) {
  return useQuery({ queryKey: ["ctf", "ad-attacks", roundID, params], queryFn: () => listAdAttacks(roundID, params), enabled: roundID.length > 0 });
}

/**
 * useAdDefenses 查询本回合防守记录。
 */
export function useAdDefenses(roundID: ID, params: QueryParams = {}) {
  return useQuery({ queryKey: ["ctf", "ad-defenses", roundID, params], queryFn: () => listAdDefenses(roundID, params), enabled: roundID.length > 0 });
}

/**
 * useAdBattleMutations 提交攻击和防守补丁，明确以后端返回成功/失败结果为准。
 */
export function useAdBattleMutations(roundID: ID, competitionID?: ID) {
  const queryClient = useQueryClient();
  const refreshBattleState = () => {
    void queryClient.invalidateQueries({ queryKey: ["ctf", "ad-attacks", roundID] });
    void queryClient.invalidateQueries({ queryKey: ["ctf", "ad-defenses", roundID] });
    if (competitionID) {
      void queryClient.invalidateQueries({ queryKey: ["ctf", "leaderboard", competitionID] });
      void queryClient.invalidateQueries({ queryKey: ["ctf", "ad-groups", competitionID] });
      void queryClient.invalidateQueries({ queryKey: ["ctf", "competition-token-ledger", competitionID] });
    }
  };

  return {
    createGroup: useMutation({ mutationFn: (payload: Parameters<typeof createCtfAdGroup>[1]) => createCtfAdGroup(competitionID ?? "", payload), onSuccess: refreshBattleState }),
    autoAssignGroups: useMutation({ mutationFn: (payload: Parameters<typeof autoAssignCtfAdGroups>[1]) => autoAssignCtfAdGroups(competitionID ?? "", payload), onSuccess: refreshBattleState }),
    attack: useMutation({ mutationFn: (payload: Parameters<typeof submitAdAttack>[1]) => submitAdAttack(roundID, payload), onSuccess: refreshBattleState }),
    defense: useMutation({ mutationFn: (payload: Parameters<typeof submitAdDefense>[1]) => submitAdDefense(roundID, payload), onSuccess: refreshBattleState }),
  };
}
