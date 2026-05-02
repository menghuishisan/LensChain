// page.tsx
// 校管 SSO 配置页，对应模块02 P-09。

import { PermissionGate } from "@/components/business/PermissionGate";
import { SchoolSSOConfigForm } from "@/components/business/SchoolSSOConfigForm";

/**
 * SchoolSsoConfigPage SSO配置页。
 */
export default function SchoolSsoConfigPage() {
  return (
    <PermissionGate allowedRoles={["school_admin"]}>
      <SchoolSSOConfigForm />
    </PermissionGate>
  );
}
