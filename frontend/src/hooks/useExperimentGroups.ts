"use client";

// useExperimentGroups.ts
// 模块04实验分组 hook，封装分组、成员、进度与组内消息的 Query 缓存。

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import {
  autoAssignExperimentGroups,
  createExperimentGroups,
  deleteExperimentGroup,
  getExperimentGroup,
  getExperimentGroupProgress,
  joinExperimentGroup,
  listExperimentGroupMembers,
  listExperimentGroupMessages,
  listExperimentGroups,
  removeExperimentGroupMember,
  sendExperimentGroupMessage,
  updateExperimentGroup,
} from "@/services/experiment";
import type { ID, QueryParams } from "@/types/api";

/**
 * experimentGroupQueryKey 实验分组详情 Query key。
 */
export function experimentGroupQueryKey(groupID: ID) {
  return ["experiment", "group", groupID] as const;
}

/**
 * useExperimentGroups 查询实验分组列表。
 */
export function useExperimentGroups(params: QueryParams) {
  return useQuery({ queryKey: ["experiment", "groups", params], queryFn: () => listExperimentGroups(params) });
}

/**
 * useExperimentGroup 查询实验分组详情。
 */
export function useExperimentGroup(groupID: ID) {
  return useQuery({ queryKey: experimentGroupQueryKey(groupID), queryFn: () => getExperimentGroup(groupID), enabled: groupID.length > 0 });
}

/**
 * useExperimentGroupMembers 查询实验分组成员。
 */
export function useExperimentGroupMembers(groupID: ID) {
  return useQuery({ queryKey: ["experiment", "group-members", groupID], queryFn: () => listExperimentGroupMembers(groupID), enabled: groupID.length > 0 });
}

/**
 * useExperimentGroupProgress 查询组内协作进度。
 */
export function useExperimentGroupProgress(groupID: ID) {
  return useQuery({ queryKey: ["experiment", "group-progress", groupID], queryFn: () => getExperimentGroupProgress(groupID), enabled: groupID.length > 0 });
}

/**
 * useExperimentGroupMessages 查询组内消息历史。
 */
export function useExperimentGroupMessages(groupID: ID, params: QueryParams = {}) {
  return useQuery({ queryKey: ["experiment", "group-messages", groupID, params], queryFn: () => listExperimentGroupMessages(groupID, params), enabled: groupID.length > 0 });
}

/**
 * useExperimentGroupMutations 实验分组创建、随机分组、加入、编辑、移除成员和发送消息。
 */
export function useExperimentGroupMutations(groupID?: ID) {
  const queryClient = useQueryClient();
  const refreshGroups = () => {
    void queryClient.invalidateQueries({ queryKey: ["experiment", "groups"] });
    if (groupID) {
      void queryClient.invalidateQueries({ queryKey: experimentGroupQueryKey(groupID) });
      void queryClient.invalidateQueries({ queryKey: ["experiment", "group-members", groupID] });
      void queryClient.invalidateQueries({ queryKey: ["experiment", "group-progress", groupID] });
      void queryClient.invalidateQueries({ queryKey: ["experiment", "group-messages", groupID] });
    }
  };

  return {
    create: useMutation({ mutationFn: createExperimentGroups, onSuccess: refreshGroups }),
    autoAssign: useMutation({ mutationFn: autoAssignExperimentGroups, onSuccess: refreshGroups }),
    update: useMutation({ mutationFn: (payload: Parameters<typeof updateExperimentGroup>[1]) => updateExperimentGroup(groupID ?? "", payload), onSuccess: refreshGroups }),
    remove: useMutation({ mutationFn: () => deleteExperimentGroup(groupID ?? ""), onSuccess: refreshGroups }),
    join: useMutation({ mutationFn: (roleID?: ID | null) => joinExperimentGroup(groupID ?? "", roleID), onSuccess: refreshGroups }),
    removeMember: useMutation({ mutationFn: (studentID: ID) => removeExperimentGroupMember(groupID ?? "", studentID), onSuccess: refreshGroups }),
    sendMessage: useMutation({ mutationFn: (content: string) => sendExperimentGroupMessage(groupID ?? "", content), onSuccess: refreshGroups }),
  };
}
