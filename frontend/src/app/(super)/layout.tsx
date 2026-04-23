// layout.tsx
// 超级管理员端路由布局，挂载已登录主布局壳。

import type { ReactNode } from "react";

import { AuthenticatedLayout } from "@/components/business/AuthenticatedLayout";
import { PermissionGate } from "@/components/business/PermissionGate";

/**
 * SuperLayout 超级管理员端布局。
 */
export default function SuperLayout({ children }: { children: ReactNode }) {
  return (
    <AuthenticatedLayout defaultRole="super_admin">
      <PermissionGate allowedRoles={["super_admin"]}>{children}</PermissionGate>
    </AuthenticatedLayout>
  );
}
