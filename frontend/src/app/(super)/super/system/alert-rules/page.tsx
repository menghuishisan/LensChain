// page.tsx
// 模块08告警规则页面。

import { AlertRuleEditor } from "@/components/business/AlertRuleEditor";
import { PermissionGate } from "@/components/business/PermissionGate";

/**
 * SuperSystemAlertRulesPage 告警规则页面。
 */
export default function SuperSystemAlertRulesPage() {
  return <PermissionGate allowedRoles={["super_admin"]}><AlertRuleEditor /></PermissionGate>;
}
