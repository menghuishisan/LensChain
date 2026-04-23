// page.tsx
// 用户导入预览与确认页，对应模块01 P-08。

import { PermissionGate } from "@/components/business/PermissionGate";
import { UserImportPreviewPanel } from "@/components/business/UserImportPanels";

/**
 * AdminUserImportPreviewPage 导入预览与确认页。
 */
export default function AdminUserImportPreviewPage() {
  return (
    <PermissionGate allowedRoles={["school_admin"]}>
      <UserImportPreviewPanel />
    </PermissionGate>
  );
}
