// page.tsx
// 超级管理员 CTF 资源配额管理页。

import { CtfResourceQuotaPanel } from "@/components/business/CtfPanels";
import { PermissionGate } from "@/components/business/PermissionGate";

/** SuperCtfResourceQuotasPage 超级管理员CTF资源配额页面。 */
export default function SuperCtfResourceQuotasPage() {
  return <PermissionGate allowedRoles={["super_admin"]}><CtfResourceQuotaPanel /></PermissionGate>;
}
