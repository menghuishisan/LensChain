/**
 * useSimInteraction.ts — InteractionSchema 拉取 + 失效订阅（06.2 §5.1）。
 *
 * 数据来源：HTTP GET /api/v1/experiment-instances/:id/sim-scenes/:code/interaction-schema
 * 浏览器缓存 24h；WS schema_invalidated 信号触发立即作废 + 重拉。
 */

import { useEffect, useMemo } from 'react';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import {
  mapInteractionDefinition,
  type InteractionDefinition,
  type InteractionSchema,
  type SimPanel,
} from '@lenschain/sim-engine-renderers';
import { getInteractionSchema } from '@/services/experiment';
import type { ID } from '@/types/api';
import type { SimInteractionSchema } from '@/types/experiment';

const SCHEMA_QUERY_KEY = (instanceID: ID, sceneCode: string) =>
  ['sim-interaction-schema', instanceID, sceneCode] as const;

/** 24 小时缓存（06.2 §5.1）。 */
const CACHE_24H = 24 * 60 * 60 * 1000;

/**
 * useSimInteraction 拉取并映射单个场景的交互 schema。
 *
 * 返回值 schema 已经是 camelCase 形态（InteractionSchema），可直接喂给 SimInteractionForm。
 */
export interface UseSimInteractionResult {
  schema: InteractionSchema | null;
  isLoading: boolean;
  error: Error | null;
  refetch: () => void;
}

export function useSimInteraction(instanceID: ID, sceneCode: string): UseSimInteractionResult {
  const enabled = Boolean(instanceID) && Boolean(sceneCode);

  const query = useQuery({
    queryKey: SCHEMA_QUERY_KEY(instanceID, sceneCode),
    queryFn: async () => {
      const raw = await getInteractionSchema(instanceID, sceneCode);
      // 服务层 SimInteractionSchema 与渲染器 InteractionDefinition 结构等价（snake_case），
      // 直接当作后者用 mapper 转 camelCase。
      return mapInteractionDefinition(raw as unknown as InteractionDefinition);
    },
    enabled,
    staleTime: CACHE_24H,
    gcTime: CACHE_24H,
  });

  return useMemo<UseSimInteractionResult>(
    () => ({
      schema: query.data ?? null,
      isLoading: query.isLoading,
      error: query.error as Error | null,
      refetch: () => { void query.refetch(); },
    }),
    [query.data, query.isLoading, query.error, query.refetch],
  );
}

/**
 * useSimSchemaInvalidation 订阅 SimPanel 的 schema_invalidated 事件，
 * 收到后立即作废对应场景的 schema 查询缓存（迫使下次读取重拉）。
 *
 * 必须在 SimEnginePanel 内挂载一次；不挂载就不会响应教师重新发布事件。
 */
export function useSimSchemaInvalidation(
  panel: SimPanel | null,
  instanceID: ID,
): void {
  const queryClient = useQueryClient();

  useEffect(() => {
    if (!panel) return;
    const unsubscribe = panel.onSchemaInvalidated((sceneCode: string) => {
      void queryClient.invalidateQueries({
        queryKey: SCHEMA_QUERY_KEY(instanceID, sceneCode),
      });
    });
    return unsubscribe;
  }, [panel, instanceID, queryClient]);
}

/** 便利：根据原始 SimInteractionSchema 同步映射成 camelCase。供测试 / 离线使用。 */
export function toCamelCaseSchema(raw: SimInteractionSchema): InteractionSchema {
  return mapInteractionDefinition(raw as unknown as InteractionDefinition);
}
