// layout.tsx
// 学生端路由布局，挂载已登录主布局壳。

import type { ReactNode } from "react";

import { AuthenticatedLayout } from "@/components/business/AuthenticatedLayout";
import { PermissionGate } from "@/components/business/PermissionGate";

export const dynamic = "force-dynamic";

/**
 * StudentLayout 学生端布局。
 */
export default function StudentLayout({ children }: { children: ReactNode }) {
  return (
    <AuthenticatedLayout defaultRole="student">
      <PermissionGate allowedRoles={["student"]}>{children}</PermissionGate>
    </AuthenticatedLayout>
  );
}
