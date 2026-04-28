// layout.tsx
// 认证路由布局，承载登录、SSO和强制改密等未登录页面。

import type { ReactNode } from "react";

export const dynamic = "force-dynamic";

/**
 * AuthLayout 认证路由布局。
 */
export default function AuthLayout({ children }: { children: ReactNode }) {
  return children;
}
