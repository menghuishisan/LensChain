// page.tsx
// 模块08告警事件页面。

import { AlertEventPanel } from "@/components/business/AlertEventPanel";
import { PermissionGate } from "@/components/business/PermissionGate";

/**
 * SuperSystemAlertEventsPage 告警事件页面。
 */
export default function SuperSystemAlertEventsPage() {
  return <PermissionGate allowedRoles={["super_admin"]}><AlertEventPanel /></PermissionGate>;
}
