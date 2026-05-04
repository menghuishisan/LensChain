// page.tsx
// 超级管理员用户编辑页。

import { PermissionGate } from "@/components/business/PermissionGate";
import { UserFormPanel } from "@/components/business/UserManagementPanels";

/** SuperUserEditPage 超级管理员用户编辑页。 */
export default function SuperUserEditPage({ params }: { params: { id: string } }) {
  return (
    <PermissionGate allowedRoles={["super_admin"]}>
      <UserFormPanel userID={params.id} basePath="/super/users" canCreateSuperAdmin />
    </PermissionGate>
  );
}