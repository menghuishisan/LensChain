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
      title="暂时找不到这个页面"
      description="你访问的内容可能已移动、下线，或当前账号暂时无法查看。"
      action={
        <Link className={buttonClassName({ variant: "primary" })} href="/login">
          返回登录页
        </Link>
      }
    />
  );
}
