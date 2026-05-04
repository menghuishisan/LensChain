// page.tsx
// 超级管理员 CTF 竞赛创建页。

import { CtfCompetitionEditorPanel } from "@/components/business/CtfPanels";
import { PermissionGate } from "@/components/business/PermissionGate";

/** SuperCtfCompetitionCreatePage 超级管理员CTF竞赛创建页面。 */
export default function SuperCtfCompetitionCreatePage() {
  return <PermissionGate allowedRoles={["super_admin"]}><CtfCompetitionEditorPanel /></PermissionGate>;
}
