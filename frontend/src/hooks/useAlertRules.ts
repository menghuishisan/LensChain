"use client";

// useAlertRules.ts
// 模块08告警规则 hook，封装规则列表、详情与增删改启停缓存失效。

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import {
  createAlertRule,
  deleteAlertRule,
  getAlertRule,
  listAlertRules,
  toggleAlertRule,
  updateAlertRule,
} from "@/services/system";
import type { ID } from "@/types/api";
import type {
  CreateSystemAlertRuleRequest,
  SystemAlertRuleListParams,
  ToggleSystemAlertRuleRequest,
  UpdateSystemAlertRuleRequest,
} from "@/types/system";

/**
 * alertRulesQueryKey 告警规则列表 Query key。
 */
export function alertRulesQueryKey(params: SystemAlertRuleListParams) {
  return ["system", "alert-rules", params] as const;
}

/**
 * alertRuleDetailQueryKey 告警规则详情 Query key。
 */
export function alertRuleDetailQueryKey(ruleID: ID) {
  return ["system", "alert-rules", "detail", ruleID] as const;
}

/**
 * useAlertRules 查询告警规则列表。
 */
export function useAlertRules(params: SystemAlertRuleListParams) {
  return useQuery({
    queryKey: alertRulesQueryKey(params),
    queryFn: () => listAlertRules(params),
  });
}

/**
 * useAlertRule 查询告警规则详情。
 */
export function useAlertRule(ruleID: ID) {
  return useQuery({
    queryKey: alertRuleDetailQueryKey(ruleID),
    queryFn: () => getAlertRule(ruleID),
    enabled: ruleID.length > 0,
  });
}

/**
 * useAlertRuleMutations 管理告警规则创建、编辑、启停和删除。
 */
export function useAlertRuleMutations() {
  const queryClient = useQueryClient();

  const refreshRules = (ruleID?: ID) => {
    void queryClient.invalidateQueries({ queryKey: ["system", "alert-rules"] });
    if (ruleID !== undefined && ruleID.length > 0) {
      void queryClient.invalidateQueries({ queryKey: alertRuleDetailQueryKey(ruleID) });
    }
  };

  return {
    create: useMutation({
      mutationFn: (payload: CreateSystemAlertRuleRequest) => createAlertRule(payload),
      onSuccess: () => refreshRules(),
    }),
    update: useMutation({
      mutationFn: ({
        ruleID,
        payload,
      }: {
        ruleID: ID;
        payload: UpdateSystemAlertRuleRequest;
      }) => updateAlertRule(ruleID, payload),
      onSuccess: (_, variables) => refreshRules(variables.ruleID),
    }),
    toggle: useMutation({
      mutationFn: ({
        ruleID,
        payload,
      }: {
        ruleID: ID;
        payload: ToggleSystemAlertRuleRequest;
      }) => toggleAlertRule(ruleID, payload),
      onSuccess: (_, variables) => refreshRules(variables.ruleID),
    }),
    remove: useMutation({
      mutationFn: (ruleID: ID) => deleteAlertRule(ruleID),
      onSuccess: (_, ruleID) => refreshRules(ruleID),
    }),
  };
}
