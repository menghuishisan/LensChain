// page.tsx
// 超级管理员 CTF 全平台概览页。

import { CtfAdminOverviewPanel } from "@/components/business/CtfPanels";
import { PermissionGate } from "@/components/business/PermissionGate";

/** SuperCtfOverviewPage 超级管理员CTF全平台概览页面。 */
export default function SuperCtfOverviewPage() {
  return <PermissionGate allowedRoles={["super_admin"]}><CtfAdminOverviewPanel /></PermissionGate>;
}
