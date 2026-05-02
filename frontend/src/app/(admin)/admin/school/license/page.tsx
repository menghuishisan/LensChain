// page.tsx
// 校管授权状态页，对应模块02 P-10。

import { PermissionGate } from "@/components/business/PermissionGate";
import { SchoolLicensePanel } from "@/components/business/SchoolProfilePanels";

/**
 * SchoolLicensePage 授权状态页。
 */
export default function SchoolLicensePage() {
  return (
    <PermissionGate allowedRoles={["school_admin"]}>
      <SchoolLicensePanel />
    </PermissionGate>
  );
}
