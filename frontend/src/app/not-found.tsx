// not-found.tsx
// 全局 404 页面。

import Link from "next/link";

import { buttonClassName } from "@/components/ui/Button";
import { EmptyState } from "@/components/ui/EmptyState";

/**
 * NotFound 全局未找到页面。
 */
export default function NotFound() {
  return (
    <EmptyState
      className="m-6 min-h-[70vh]"
      title="页面尚未实现"
      description="该路由将在对应模块开发阶段按文档接入。"
      action={
        <Link className={buttonClassName({ variant: "primary" })} href="/login">
          返回登录页
        </Link>
      }
    />
  );
}
