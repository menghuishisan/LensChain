// page.tsx
// 超级管理员创建用户页。

import { PermissionGate } from "@/components/business/PermissionGate";
import { UserFormPanel } from "@/components/business/UserManagementPanels";

/** SuperUserCreatePage 超级管理员创建用户页。 */
export default function SuperUserCreatePage() {
  return (
    <PermissionGate allowedRoles={["super_admin"]}>
      <UserFormPanel basePath="/super/users" canCreateSuperAdmin />
    </PermissionGate>
  );
}
