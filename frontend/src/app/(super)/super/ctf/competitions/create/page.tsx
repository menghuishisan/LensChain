// page.tsx
// 超级管理员 CTF 竞赛创建页。

import { CtfCompetitionEditorPanel } from "@/components/business/CtfPanels";
import { PermissionGate } from "@/components/business/PermissionGate";

/** AdminCtfCompetitionCreatePage 超级管理员CTF竞赛创建页面。 */
export default function AdminCtfCompetitionCreatePage() {
  return <PermissionGate allowedRoles={["super_admin", "school_admin"]}><CtfCompetitionEditorPanel /></PermissionGate>;
}
