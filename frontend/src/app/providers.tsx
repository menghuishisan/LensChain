"use client";

// providers.tsx
// App Router 客户端 Provider 汇总，统一挂载 TanStack Query 与 Toast。

import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ReactNode } from "react";
import { useState } from "react";

import { ToastProvider } from "@/components/ui/Toast";

/**
 * 应用级客户端 Provider。
 */
export function AppProviders({ children }: { children: ReactNode }) {
  const [queryClient] = useState(
    () =>
      new QueryClient({
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
