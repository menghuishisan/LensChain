"use client";

// useToolProxyCookie.ts
// 工具反代 cookie 预热 hook：iframe 渲染之前先签发 HttpOnly cookie，并按 25 分钟节奏刷新
// （后端 cookie 默认 30 分钟过期，留 5 分钟缓冲）。
//
// 设计要点：
//   - enabled=false 时不会发起请求，适合"该 toolKind 不存在"或"实例未运行"场景；
//   - refetchInterval 用于自动续签，避免 iframe 长时间打开后 cookie 过期导致 401 大面积刷新；
//   - retry 设为 1：第一次失败可能是网络抖动，但反复重试无意义（业务校验失败不会因重试通过）。

import { useQuery } from "@tanstack/react-query";

import type { ID } from "@/types/api";
import { ensureToolProxyCookie, type ToolProxyCookie } from "@/services/experimentToolProxy";

const REFRESH_INTERVAL_MS = 25 * 60 * 1000;

/**
 * useToolProxyCookie 预热并维持工具反代 cookie。
 *
 * @param instanceID 实验实例 ID
 * @param toolKind   工具类型（terminal / ide / desktop / explorer / monitor 等）
 * @param enabled    是否启用（实例未就绪 / 该 toolKind 不存在时传 false）
 */
export function useToolProxyCookie(instanceID: ID, toolKind: string, enabled = true) {
  return useQuery<ToolProxyCookie>({
    queryKey: ["experiment", "tool-proxy-cookie", instanceID, toolKind],
    queryFn: () => ensureToolProxyCookie(instanceID, toolKind),
    enabled: enabled && instanceID.length > 0 && toolKind.length > 0,
    refetchInterval: REFRESH_INTERVAL_MS,
    refetchOnWindowFocus: false,
    retry: 1,
    staleTime: REFRESH_INTERVAL_MS - 60_000,
  });
}
