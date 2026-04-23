"use client";

// error.tsx
// 全局错误边界页面。

import { Button } from "@/components/ui/Button";
import { ErrorState } from "@/components/ui/ErrorState";

/**
 * GlobalError 全局错误边界。
 */
export default function GlobalError({ error, reset }: { error: Error & { digest?: string }; reset: () => void }) {
  return (
    <ErrorState
      className="m-6 min-h-[70vh]"
      title="页面渲染失败"
      description={error.message}
      action={
        <Button type="button" onClick={reset}>
          重新加载
        </Button>
      }
    />
  );
}
