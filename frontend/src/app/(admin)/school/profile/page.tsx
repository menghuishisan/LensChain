// page.tsx
// 校管本校信息管理页，对应模块02 P-08。

import { PermissionGate } from "@/components/business/PermissionGate";
import { SchoolProfilePanel } from "@/components/business/SchoolProfilePanels";

/**
 * SchoolProfilePage 本校信息管理页。
 */
export default function SchoolProfilePage() {
  return (
    <PermissionGate allowedRoles={["school_admin"]}>
      <SchoolProfilePanel />
    </PermissionGate>
  );
}
