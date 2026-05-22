"use client";

// providers.tsx
// App Router 客户端 Provider 汇总，统一挂载 TanStack Query 与 Toast。

import { MutationCache, QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ReactNode } from "react";
import { useState } from "react";

import { ToastProvider, emitToast } from "@/components/ui/Toast";
import { ApiClientError } from "@/lib/api-client";

/**
 * 应用级客户端 Provider。
 */
export function AppProviders({ children }: { children: ReactNode }) {
  const [queryClient] = useState(
    () =>
      new QueryClient({
        // MutationCache 全局 onError：把所有 mutation 失败统一弹 Toast。
        //
        // 之前 mutation 配置只有 onSuccess（如 lifecycle.resume.mutate），抛错被 TanStack
        // Query 静默吞掉，学生点"恢复"撞到并发上限 / 模板异常时前端无任何反馈，体感
        // 为按钮卡死。把错误反馈下沉到全局：业务侧仍可用 mutation 的 onError 做局部
        // 替代逻辑（return false 不阻断全局 Toast，但可在自己 onError 里 setError 显示
        // 行内错误），缺省路径下错误一定会以 Toast 形式触达用户。
        mutationCache: new MutationCache({
          onError: (error) => {
            const message = error instanceof ApiClientError
              ? error.message
              : error instanceof Error
                ? error.message
                : "操作失败";
            emitToast({ title: message, variant: "destructive" });
          },
        }),
        defaultOptions: {
          queries: {
            // 默认 1 分钟内数据视为新鲜，避免列表页频繁重复请求；
            // 列表/表格类数据通过 refetchOnMount 手动控制，详情类用 invalidateQueries 强制刷新。
            staleTime: 60 * 1000,
            // 缓存保留 5 分钟，路由切换返回时直接复用而不闪烁加载。
            gcTime: 5 * 60 * 1000,
            refetchOnWindowFocus: false,
            refetchOnReconnect: true,
            retry: 1,
            retryDelay: (attemptIndex) => Math.min(1000 * 2 ** attemptIndex, 5000),
          },
          mutations: {
            retry: 0,
          },
        },
      }),
  );

  return (
    <QueryClientProvider client={queryClient}>
      <ToastProvider>{children}</ToastProvider>
    </QueryClientProvider>
  );
}
