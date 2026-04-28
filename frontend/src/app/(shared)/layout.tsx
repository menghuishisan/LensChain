// layout.tsx
// 已登录共享路由布局，承载个人中心与消息中心等跨角色页面入口。

import type { ReactNode } from "react";

import { AuthenticatedLayout } from "@/components/business/AuthenticatedLayout";
import { PermissionGate } from "@/components/business/PermissionGate";

export const dynamic = "force-dynamic";

/**
 * SharedLayout 已登录共享布局。
 */
export default function SharedLayout({ children }: { children: ReactNode }) {
  return (
    <AuthenticatedLayout defaultRole="student">
      <PermissionGate allowedRoles={["student", "teacher", "school_admin", "super_admin"]}>{children}</PermissionGate>
    </AuthenticatedLayout>
  );
}
