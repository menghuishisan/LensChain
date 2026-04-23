"use client";

// useCtfChallenges.ts
// 模块05题目 hook：题库、合约、断言、漏洞转化、预验证和审核缓存策略。

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import {
  createChallengeAssertion,
  createChallengeContract,
  createCtfChallenge,
  deleteCtfChallenge,
  generateChallengeFromTemplate,
  getCtfChallenge,
  getCtfChallengeTemplate,
  getCtfVerification,
  importSwcChallenge,
  importExternalVulnerability,
  listChallengeAssertions,
  listChallengeContracts,
  listCtfChallengeTemplates,
  listCtfChallenges,
  listCtfVerifications,
  listPendingCtfChallengeReviews,
  listSwcRegistry,
  reviewCtfChallenge,
  submitCtfChallengeReview,
  updateCtfChallenge,
  verifyCtfChallenge,
} from "@/services/ctf";
import type { ID, QueryParams } from "@/types/api";
import type { CtfChallengeListParams } from "@/types/ctf";

/**
 * ctfChallengeQueryKey 题目详情 Query key。
 */
export function ctfChallengeQueryKey(challengeID: ID) {
  return ["ctf", "challenge", challengeID] as const;
}

/**
 * useCtfChallenges 查询题库列表。
 */
export function useCtfChallenges(params: CtfChallengeListParams) {
  return useQuery({ queryKey: ["ctf", "challenges", params], queryFn: () => listCtfChallenges(params) });
}

/**
 * useCtfChallenge 查询题目详情。
 */
export function useCtfChallenge(challengeID: ID) {
  return useQuery({ queryKey: ctfChallengeQueryKey(challengeID), queryFn: () => getCtfChallenge(challengeID), enabled: challengeID.length > 0 });
}

/**
 * useCtfChallengeMutations 创建、编辑、删除、提交审核和审核题目。
 */
export function useCtfChallengeMutations(challengeID?: ID) {
  const queryClient = useQueryClient();
  const refreshChallenge = () => {
    void queryClient.invalidateQueries({ queryKey: ["ctf", "challenges"] });
    void queryClient.invalidateQueries({ queryKey: ["ctf", "challenge-reviews"] });
    if (challengeID) {
      void queryClient.invalidateQueries({ queryKey: ctfChallengeQueryKey(challengeID) });
      void queryClient.invalidateQueries({ queryKey: ["ctf", "verifications", challengeID] });
    }
  };

  return {
    create: useMutation({ mutationFn: createCtfChallenge, onSuccess: refreshChallenge }),
    update: useMutation({ mutationFn: (payload: Parameters<typeof updateCtfChallenge>[1]) => updateCtfChallenge(challengeID ?? "", payload), onSuccess: refreshChallenge }),
    remove: useMutation({ mutationFn: () => deleteCtfChallenge(challengeID ?? ""), onSuccess: refreshChallenge }),
    submitReview: useMutation({ mutationFn: () => submitCtfChallengeReview(challengeID ?? ""), onSuccess: refreshChallenge }),
    review: useMutation({ mutationFn: (payload: Parameters<typeof reviewCtfChallenge>[1]) => reviewCtfChallenge(challengeID ?? "", payload), onSuccess: refreshChallenge }),
  };
}

/**
 * useCtfChallengeAssets 查询题目合约和断言。
 */
export function useCtfChallengeAssets(challengeID: ID) {
  const contracts = useQuery({ queryKey: ["ctf", "challenge-contracts", challengeID], queryFn: () => listChallengeContracts(challengeID), enabled: challengeID.length > 0 });
  const assertions = useQuery({ queryKey: ["ctf", "challenge-assertions", challengeID], queryFn: () => listChallengeAssertions(challengeID), enabled: challengeID.length > 0 });
  return { contracts, assertions };
}

/**
 * useCtfChallengeAssetMutations 添加合约和断言后刷新题目详情与资产列表。
 */
export function useCtfChallengeAssetMutations(challengeID: ID) {
  const queryClient = useQueryClient();
  const refreshAssets = () => {
    void queryClient.invalidateQueries({ queryKey: ctfChallengeQueryKey(challengeID) });
    void queryClient.invalidateQueries({ queryKey: ["ctf", "challenge-contracts", challengeID] });
    void queryClient.invalidateQueries({ queryKey: ["ctf", "challenge-assertions", challengeID] });
  };

  return {
    createContract: useMutation({ mutationFn: (payload: Parameters<typeof createChallengeContract>[1]) => createChallengeContract(challengeID, payload), onSuccess: refreshAssets }),
    createAssertion: useMutation({ mutationFn: (payload: Parameters<typeof createChallengeAssertion>[1]) => createChallengeAssertion(challengeID, payload), onSuccess: refreshAssets }),
  };
}

/**
 * useSwcRegistry 查询 SWC Registry 来源。
 */
export function useSwcRegistry(params: QueryParams = {}) {
  return useQuery({ queryKey: ["ctf", "swc", params], queryFn: () => listSwcRegistry(params) });
}

/**
 * useCtfChallengeTemplates 查询漏洞参数化模板库。
 */
export function useCtfChallengeTemplates(params: QueryParams = {}) {
  return useQuery({ queryKey: ["ctf", "challenge-templates", params], queryFn: () => listCtfChallengeTemplates(params) });
}

/**
 * useCtfChallengeTemplate 查询参数化模板详情。
 */
export function useCtfChallengeTemplate(templateID: ID) {
  return useQuery({ queryKey: ["ctf", "challenge-template", templateID], queryFn: () => getCtfChallengeTemplate(templateID), enabled: templateID.length > 0 });
}

/**
 * useVulnerabilityConvertMutations 漏洞转化入口，覆盖 SWC 导入和模板生成。
 */
export function useVulnerabilityConvertMutations() {
  const queryClient = useQueryClient();
  return {
    importSwc: useMutation({ mutationFn: importSwcChallenge, onSuccess: () => void queryClient.invalidateQueries({ queryKey: ["ctf", "challenges"] }) }),
    generateFromTemplate: useMutation({ mutationFn: generateChallengeFromTemplate, onSuccess: () => void queryClient.invalidateQueries({ queryKey: ["ctf", "challenges"] }) }),
    importExternal: useMutation({ mutationFn: importExternalVulnerability, onSuccess: () => void queryClient.invalidateQueries({ queryKey: ["ctf", "challenges"] }) }),
  };
}

/**
 * useCtfVerifications 查询题目预验证记录。
 */
export function useCtfVerifications(challengeID: ID) {
  return useQuery({ queryKey: ["ctf", "verifications", challengeID], queryFn: () => listCtfVerifications(challengeID), enabled: challengeID.length > 0 });
}

/**
 * useCtfVerification 查询预验证详情。
 */
export function useCtfVerification(verificationID: ID) {
  return useQuery({ queryKey: ["ctf", "verification", verificationID], queryFn: () => getCtfVerification(verificationID), enabled: verificationID.length > 0 });
}

/**
 * useVerifyCtfChallengeMutation 发起预验证，成功后刷新验证记录。
 */
export function useVerifyCtfChallengeMutation(challengeID: ID) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (payload: Parameters<typeof verifyCtfChallenge>[1]) => verifyCtfChallenge(challengeID, payload),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["ctf", "verifications", challengeID] });
      void queryClient.invalidateQueries({ queryKey: ctfChallengeQueryKey(challengeID) });
    },
  });
}

/**
 * usePendingCtfChallengeReviews 查询待审核题目列表。
 */
export function usePendingCtfChallengeReviews() {
  return useQuery({ queryKey: ["ctf", "challenge-reviews", "pending"], queryFn: listPendingCtfChallengeReviews });
}
