"use client";

// useCtfCompetitions.ts
// 模块05竞赛 hook：竞赛生命周期、题目配置、公告、排行榜、监控、统计和结果缓存。

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import {
  addCompetitionChallenges,
  archiveCtfCompetition,
  createCtfAnnouncement,
  createCtfCompetition,
  deleteCtfCompetition,
  getCtfAdminOverview,
  getCtfCompetition,
  getCtfCompetitionMonitor,
  getCtfCompetitionResults,
  getCtfCompetitionStatistics,
  getCtfResourceQuota,
  getCtfFinalLeaderboard,
  getCtfLeaderboardHistory,
  getCtfLeaderboard,
  listCompetitionChallenges,
  listCtfAnnouncements,
  listCtfCompetitions,
  publishCtfCompetition,
  removeCompetitionChallenge,
  sortCompetitionChallenges,
  terminateCtfCompetition,
  updateCtfResourceQuota,
  updateCtfCompetition,
} from "@/services/ctf";
import type { ID, QueryParams } from "@/types/api";
import type { CtfCompetitionListParams } from "@/types/ctf";

/**
 * ctfCompetitionQueryKey 竞赛详情 Query key。
 */
export function ctfCompetitionQueryKey(competitionID: ID) {
  return ["ctf", "competition", competitionID] as const;
}

/**
 * useCtfCompetitions 查询 CTF 竞赛列表。
 */
export function useCtfCompetitions(params: CtfCompetitionListParams) {
  return useQuery({ queryKey: ["ctf", "competitions", params], queryFn: () => listCtfCompetitions(params) });
}

/**
 * useCtfCompetition 查询竞赛详情。
 */
export function useCtfCompetition(competitionID: ID) {
  return useQuery({ queryKey: ctfCompetitionQueryKey(competitionID), queryFn: () => getCtfCompetition(competitionID), enabled: competitionID.length > 0 });
}

/**
 * useCtfCompetitionMutations 管理竞赛创建、编辑、发布、归档、删除和强制终止。
 */
export function useCtfCompetitionMutations(competitionID?: ID) {
  const queryClient = useQueryClient();
  const refreshCompetition = () => {
    void queryClient.invalidateQueries({ queryKey: ["ctf", "competitions"] });
    void queryClient.invalidateQueries({ queryKey: ["ctf", "admin-overview"] });
    if (competitionID) {
      void queryClient.invalidateQueries({ queryKey: ctfCompetitionQueryKey(competitionID) });
      void queryClient.invalidateQueries({ queryKey: ["ctf", "competition-challenges", competitionID] });
    }
  };

  return {
    create: useMutation({ mutationFn: createCtfCompetition, onSuccess: refreshCompetition }),
    update: useMutation({ mutationFn: (payload: Parameters<typeof updateCtfCompetition>[1]) => updateCtfCompetition(competitionID ?? "", payload), onSuccess: refreshCompetition }),
    remove: useMutation({ mutationFn: () => deleteCtfCompetition(competitionID ?? ""), onSuccess: refreshCompetition }),
    publish: useMutation({ mutationFn: () => publishCtfCompetition(competitionID ?? ""), onSuccess: refreshCompetition }),
    archive: useMutation({ mutationFn: () => archiveCtfCompetition(competitionID ?? ""), onSuccess: refreshCompetition }),
    terminate: useMutation({ mutationFn: (reason: string) => terminateCtfCompetition(competitionID ?? "", reason), onSuccess: refreshCompetition }),
    addChallenges: useMutation({ mutationFn: (challengeIDs: ID[]) => addCompetitionChallenges(competitionID ?? "", challengeIDs), onSuccess: refreshCompetition }),
    removeChallenge: useMutation({ mutationFn: removeCompetitionChallenge, onSuccess: refreshCompetition }),
    sortChallenges: useMutation({ mutationFn: (items: Array<{ id: ID; sort_order: number }>) => sortCompetitionChallenges(competitionID ?? "", items), onSuccess: refreshCompetition }),
  };
}

/**
 * useCtfCompetitionChallenges 查询竞赛题目列表，选手视角不包含源码和断言敏感信息。
 */
export function useCtfCompetitionChallenges(competitionID: ID) {
  return useQuery({ queryKey: ["ctf", "competition-challenges", competitionID], queryFn: () => listCompetitionChallenges(competitionID), enabled: competitionID.length > 0 });
}

/**
 * useCtfLeaderboard 查询实时排行榜快照。
 */
export function useCtfLeaderboard(competitionID: ID, params: QueryParams = {}) {
  return useQuery({ queryKey: ["ctf", "leaderboard", competitionID, params], queryFn: () => getCtfLeaderboard(competitionID, params), enabled: competitionID.length > 0 });
}

/**
 * useCtfFinalLeaderboard 查询竞赛最终排名。
 */
export function useCtfFinalLeaderboard(competitionID: ID) {
  return useQuery({ queryKey: ["ctf", "leaderboard-final", competitionID], queryFn: () => getCtfFinalLeaderboard(competitionID), enabled: competitionID.length > 0 });
}

/**
 * useCtfLeaderboardHistory 查询排行榜历史快照。
 */
export function useCtfLeaderboardHistory(competitionID: ID, params: QueryParams = {}) {
  return useQuery({ queryKey: ["ctf", "leaderboard-history", competitionID, params], queryFn: () => getCtfLeaderboardHistory(competitionID, params), enabled: competitionID.length > 0 });
}

/**
 * useCtfAnnouncements 查询竞赛公告列表。
 */
export function useCtfAnnouncements(competitionID: ID) {
  return useQuery({ queryKey: ["ctf", "announcements", competitionID], queryFn: () => listCtfAnnouncements(competitionID), enabled: competitionID.length > 0 });
}

/**
 * useCtfAnnouncementMutations 发布公告，成功后刷新公告和监控数据。
 */
export function useCtfAnnouncementMutations(competitionID: ID) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (payload: Parameters<typeof createCtfAnnouncement>[1]) => createCtfAnnouncement(competitionID, payload),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["ctf", "announcements", competitionID] });
      void queryClient.invalidateQueries({ queryKey: ["ctf", "monitor", competitionID] });
    },
  });
}

/**
 * useCtfCompetitionMonitor 查询竞赛运行监控。
 */
export function useCtfCompetitionMonitor(competitionID: ID) {
  return useQuery({ queryKey: ["ctf", "monitor", competitionID], queryFn: () => getCtfCompetitionMonitor(competitionID), enabled: competitionID.length > 0 });
}

/**
 * useCtfCompetitionStatistics 查询竞赛统计数据。
 */
export function useCtfCompetitionStatistics(competitionID: ID) {
  return useQuery({ queryKey: ["ctf", "statistics", competitionID], queryFn: () => getCtfCompetitionStatistics(competitionID), enabled: competitionID.length > 0 });
}

/**
 * useCtfCompetitionResults 查询竞赛最终结果。
 */
export function useCtfCompetitionResults(competitionID: ID) {
  return useQuery({ queryKey: ["ctf", "results", competitionID], queryFn: () => getCtfCompetitionResults(competitionID), enabled: competitionID.length > 0 });
}

/**
 * useCtfAdminOverview 查询全平台竞赛概览。
 */
export function useCtfAdminOverview() {
  return useQuery({ queryKey: ["ctf", "admin-overview"], queryFn: getCtfAdminOverview });
}

/**
 * useCtfResourceQuota 查询指定竞赛的 CTF 资源配额。
 */
export function useCtfResourceQuota(competitionID: ID) {
  return useQuery({ queryKey: ["ctf", "resource-quota", competitionID], queryFn: () => getCtfResourceQuota(competitionID), enabled: competitionID.length > 0 });
}

/**
 * useCtfResourceQuotaMutation 设置竞赛资源配额，成功后刷新配额和监控。
 */
export function useCtfResourceQuotaMutation(competitionID: ID) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (payload: Parameters<typeof updateCtfResourceQuota>[1]) => updateCtfResourceQuota(competitionID, payload),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["ctf", "resource-quota", competitionID] });
      void queryClient.invalidateQueries({ queryKey: ["ctf", "monitor", competitionID] });
    },
  });
}
