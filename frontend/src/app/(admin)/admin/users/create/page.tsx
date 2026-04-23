// page.tsx
// 用户创建页，对应模块01 P-06。

import { PermissionGate } from "@/components/business/PermissionGate";
import { UserFormPanel } from "@/components/business/UserManagementPanels";

/**
 * AdminUserCreatePage 用户创建页。
 */
export default function AdminUserCreatePage() {
  return (
    <PermissionGate allowedRoles={["school_admin", "super_admin"]}>
      <UserFormPanel />
    </PermissionGate>
  );
}
