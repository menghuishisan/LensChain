// page.tsx
// 用户详情页，对应模块01 P-05。

import { PermissionGate } from "@/components/business/PermissionGate";
import { UserDetailPanel } from "@/components/business/UserManagementPanels";

/**
 * AdminUserDetailPage 用户详情页。
 */
export default function AdminUserDetailPage({ params }: { params: { id: string } }) {
  return (
    <PermissionGate allowedRoles={["super_admin"]}>
      <UserDetailPanel userID={params.id} />
    </PermissionGate>
  );
}
