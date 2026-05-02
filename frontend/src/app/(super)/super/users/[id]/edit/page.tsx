// page.tsx
// 用户编辑页，对应模块01 P-06 编辑场景。

import { PermissionGate } from "@/components/business/PermissionGate";
import { UserFormPanel } from "@/components/business/UserManagementPanels";

/**
 * AdminUserEditPage 用户编辑页。
 */
export default function AdminUserEditPage({ params }: { params: { id: string } }) {
  return (
    <PermissionGate allowedRoles={["super_admin"]}>
      <UserFormPanel userID={params.id} />
    </PermissionGate>
  );
}
