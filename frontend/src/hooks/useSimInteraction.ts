// useSimInteraction.ts
// SimEngine 场景交互 schema 查询 Hook（API §4.2）。
// 通过 TanStack Query 管理 InteractionSchema 的获取、缓存和失效。

import { useCallback, useEffect } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";

import { getInteractionSchema } from "@/services/experiment";
import type { ID } from "@/types/api";
import type { SimActionDef, SimInteractionSchema, SimWSMessage } from "@/types/experiment";

/** 查询键工厂。 */
function interactionSchemaKey(instanceID: ID, sceneCode: string) {
  return ["experiment", "interaction-schema", instanceID, sceneCode] as const;
}

/** useSimInteraction 参数。 */
export interface UseSimInteractionOptions {
  instanceID: ID;
  sceneCode: string;
  enabled?: boolean;
  userRole?: string;
}

/** useSimInteraction 返回值。 */
export interface UseSimInteractionReturn {
  schema: SimInteractionSchema | null;
  actions: SimActionDef[];
  isLoading: boolean;
  error: string | null;
  invalidateSchema: () => void;
}

/**
 * useSimInteraction 获取场景交互 schema 并缓存 24h（API §4.2）。
 * WS schema_invalidated 信号触发 invalidateSchema 后立即重新拉取。
 * userRole 传入后自动过滤仅当前角色可见的 action。
 */
export function useSimInteraction(options: UseSimInteractionOptions): UseSimInteractionReturn {
  const { instanceID, sceneCode, enabled = true, userRole } = options;
  const queryClient = useQueryClient();

  const { data, isLoading, error } = useQuery({
    queryKey: interactionSchemaKey(instanceID, sceneCode),
    queryFn: () => getInteractionSchema(instanceID, sceneCode),
    enabled: enabled && instanceID.length > 0 && sceneCode.length > 0,
    staleTime: 24 * 60 * 60 * 1000,
    gcTime: 24 * 60 * 60 * 1000,
  });

  const invalidateSchema = useCallback(() => {
    void queryClient.invalidateQueries({ queryKey: interactionSchemaKey(instanceID, sceneCode) });
  }, [queryClient, instanceID, sceneCode]);

  const actions = data?.actions
    ? userRole
      ? data.actions.filter((a) => a.roles.includes(userRole))
      : data.actions
    : [];

  return {
    schema: data ?? null,
    actions,
    isLoading,
    error: error ? String(error) : null,
    invalidateSchema,
  };
}

/**
 * useSimSchemaInvalidation 监听 WS 消息，收到 schema_invalidated 时失效所有场景的 schema 缓存。
 * 在 SimEnginePage 层调用一次即可，不需每个场景各调一次。
 */
export function useSimSchemaInvalidation(instanceID: ID, latestWSMessage: SimWSMessage | null) {
  const queryClient = useQueryClient();

  useEffect(() => {
    if (latestWSMessage?.type === "schema_invalidated") {
      void queryClient.invalidateQueries({
        queryKey: ["experiment", "interaction-schema", instanceID],
      });
    }
  }, [queryClient, instanceID, latestWSMessage]);
}
