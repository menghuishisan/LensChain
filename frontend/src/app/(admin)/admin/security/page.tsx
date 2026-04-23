// page.tsx
// 安全策略配置页，对应模块01 P-11。

import { PermissionGate } from "@/components/business/PermissionGate";
import { SecurityPolicyPanel } from "@/components/business/SecurityPolicyPanel";

/**
 * AdminSecurityPage 安全策略配置页。
 */
export default function AdminSecurityPage() {
  return (
    <PermissionGate allowedRoles={["super_admin"]}>
      <SecurityPolicyPanel />
    </PermissionGate>
  );
}
