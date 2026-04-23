// loading.tsx
// 全局页面加载状态。

import { LoadingState } from "@/components/ui/LoadingState";

/**
 * Loading 全局加载页。
 */
export default function Loading() {
  return <LoadingState className="m-6 min-h-[70vh]" />;
}
