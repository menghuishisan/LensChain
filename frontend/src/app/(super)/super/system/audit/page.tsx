// page.tsx
// 模块08统一审计页面。

import { AuditLogTable } from "@/components/business/AuditLogTable";
import { PermissionGate } from "@/components/business/PermissionGate";

/**
 * SuperSystemAuditPage 统一审计页面。
 */
export default function SuperSystemAuditPage() {
  return <PermissionGate allowedRoles={["super_admin"]}><AuditLogTable /></PermissionGate>;
}
