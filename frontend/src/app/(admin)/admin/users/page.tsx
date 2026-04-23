// page.tsx
// 用户管理列表页，对应模块01 P-04。

import { PermissionGate } from "@/components/business/PermissionGate";
import { UserListPanel } from "@/components/business/UserManagementPanels";

/**
 * AdminUsersPage 用户管理列表页。
 */
export default function AdminUsersPage() {
  return (
    <PermissionGate allowedRoles={["school_admin", "super_admin"]}>
      <UserListPanel />
    </PermissionGate>
  );
}
