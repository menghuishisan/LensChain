// page.tsx
// 模块08运维仪表盘页面。

import { SystemHealthDashboard } from "@/components/business/SystemHealthDashboard";
import { PermissionGate } from "@/components/business/PermissionGate";

/**
 * SuperSystemDashboardPage 运维仪表盘页面。
 */
export default function SuperSystemDashboardPage() {
  return (
    <PermissionGate allowedRoles={["super_admin"]}>
      <SystemHealthDashboard />
    </PermissionGate>
  );
}
