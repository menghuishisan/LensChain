"use client";

// useCtfTeams.ts
// 模块05团队和报名 hook，集中处理组队、邀请码、报名和团队缓存失效。

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import {
  createCtfTeam,
  disbandCtfTeam,
  getCtfTeam,
  getCtfTeamChain,
  getCtfTeamTokenLedger,
  getMyCtfRegistration,
  joinCtfTeam,
  leaveCtfTeam,
  listCtfTeams,
  registerCtfCompetition,
  updateCtfTeam,
} from "@/services/ctf";
import type { ID, QueryParams } from "@/types/api";

/**
 * useCtfTeams 查询竞赛团队列表。
 */
export function useCtfTeams(competitionID: ID) {
  return useQuery({ queryKey: ["ctf", "teams", competitionID], queryFn: () => listCtfTeams(competitionID), enabled: competitionID.length > 0 });
}

/**
 * useCtfTeam 查询团队详情。
 */
export function useCtfTeam(teamID: ID) {
  return useQuery({ queryKey: ["ctf", "team", teamID], queryFn: () => getCtfTeam(teamID), enabled: teamID.length > 0 });
}

/**
 * useCtfTeamTokenLedger 查询团队 Token 流水。
 */
export function useCtfTeamTokenLedger(teamID: ID, params: QueryParams = {}) {
  return useQuery({ queryKey: ["ctf", "team-token-ledger", teamID, params], queryFn: () => getCtfTeamTokenLedger(teamID, params), enabled: teamID.length > 0 });
}

/**
 * useCtfTeamChain 查询攻防赛队伍链连接信息。
 */
export function useCtfTeamChain(teamID: ID) {
  return useQuery({ queryKey: ["ctf", "team-chain", teamID], queryFn: () => getCtfTeamChain(teamID), enabled: teamID.length > 0, refetchInterval: 10000 });
}

/**
 * useMyCtfRegistration 查询当前用户报名状态。
 */
export function useMyCtfRegistration(competitionID: ID) {
  return useQuery({ queryKey: ["ctf", "my-registration", competitionID], queryFn: () => getMyCtfRegistration(competitionID), enabled: competitionID.length > 0 });
}

/**
 * useCtfTeamMutations 创建队伍、加入队伍和报名竞赛。
 */
export function useCtfTeamMutations(competitionID: ID) {
  const queryClient = useQueryClient();
  const refreshTeamState = () => {
    void queryClient.invalidateQueries({ queryKey: ["ctf", "teams", competitionID] });
    void queryClient.invalidateQueries({ queryKey: ["ctf", "my-registration", competitionID] });
    void queryClient.invalidateQueries({ queryKey: ["ctf", "competition", competitionID] });
  };

  return {
    createTeam: useMutation({ mutationFn: (name: string) => createCtfTeam(competitionID, { name }), onSuccess: refreshTeamState }),
    joinTeam: useMutation({ mutationFn: joinCtfTeam, onSuccess: refreshTeamState }),
    register: useMutation({ mutationFn: (teamID?: ID | null) => registerCtfCompetition(competitionID, teamID), onSuccess: refreshTeamState }),
    updateTeam: useMutation({ mutationFn: ({ teamID, name }: { teamID: ID; name: string }) => updateCtfTeam(teamID, name), onSuccess: refreshTeamState }),
    disbandTeam: useMutation({ mutationFn: disbandCtfTeam, onSuccess: refreshTeamState }),
    leaveTeam: useMutation({ mutationFn: leaveCtfTeam, onSuccess: refreshTeamState }),
  };
}
