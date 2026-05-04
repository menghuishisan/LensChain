// page.tsx
// 超级管理员用户详情页。

import { PermissionGate } from "@/components/business/PermissionGate";
import { UserDetailPanel } from "@/components/business/UserManagementPanels";

/** SuperUserDetailPage 超级管理员用户详情页。 */
export default function SuperUserDetailPage({ params }: { params: { id: string } }) {
  return (
    <PermissionGate allowedRoles={["super_admin"]}>
      <UserDetailPanel userID={params.id} basePath="/super/users" />
    </PermissionGate>
  );
}