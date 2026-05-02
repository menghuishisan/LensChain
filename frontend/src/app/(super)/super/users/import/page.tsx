// page.tsx
// 用户导入页，对应模块01 P-07。

import { PermissionGate } from "@/components/business/PermissionGate";
import { UserImportPanel } from "@/components/business/UserImportPanels";

/**
 * AdminUserImportPage 用户导入页。
 */
export default function AdminUserImportPage() {
  return (
    <PermissionGate allowedRoles={["school_admin"]}>
      <UserImportPanel />
    </PermissionGate>
  );
}
