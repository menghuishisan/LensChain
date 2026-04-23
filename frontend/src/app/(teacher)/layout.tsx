// layout.tsx
// 教师端路由布局，挂载已登录主布局壳。

import type { ReactNode } from "react";

import { AuthenticatedLayout } from "@/components/business/AuthenticatedLayout";
import { PermissionGate } from "@/components/business/PermissionGate";

/**
 * TeacherLayout 教师端布局。
 */
export default function TeacherLayout({ children }: { children: ReactNode }) {
  return (
    <AuthenticatedLayout defaultRole="teacher">
      <PermissionGate allowedRoles={["teacher"]}>{children}</PermissionGate>
    </AuthenticatedLayout>
  );
}
