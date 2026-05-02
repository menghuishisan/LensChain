// layout.tsx
// 学校管理员端路由布局，挂载已登录主布局壳。

import type { ReactNode } from "react";

import { AuthenticatedLayout } from "@/components/business/AuthenticatedLayout";
import { PermissionGate } from "@/components/business/PermissionGate";

export const dynamic = "force-dynamic";

/**
 * AdminLayout 学校管理员端布局。
 */
export default function AdminLayout({ children }: { children: ReactNode }) {
  return (
    <AuthenticatedLayout defaultRole="school_admin">
      <PermissionGate allowedRoles={["school_admin"]}>{children}</PermissionGate>
    </AuthenticatedLayout>
  );
}
