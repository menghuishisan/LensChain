"use client";

// BackButton.tsx
// 通用返回上一页按钮，优先使用浏览器历史，缺失时回退到首页。

import { ArrowLeft } from "lucide-react";
import { useRouter } from "next/navigation";

import { Button } from "@/components/ui/Button";

/**
 * BackButton 组件属性。
 */
export interface BackButtonProps {
  fallbackHref?: string;
  label?: string;
  className?: string;
}

/**
 * BackButton 返回上一页按钮组件。
 */
export function BackButton({ fallbackHref = "/", label = "返回上一页", className }: BackButtonProps) {
  const router = useRouter();

  const handleBack = () => {
    if (typeof window !== "undefined" && window.history.length > 1) {
      router.back();
      return;
    }

    router.push(fallbackHref);
  };

  return (
    <Button type="button" variant="outline" size="sm" className={className} onClick={handleBack}>
      <ArrowLeft className="h-4 w-4" />
      {label}
    </Button>
  );
}

