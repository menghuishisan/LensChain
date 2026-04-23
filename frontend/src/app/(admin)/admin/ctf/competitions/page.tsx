// page.tsx
// 超级管理员 CTF 竞赛管理页。

import { CtfAdminCompetitionListPanel } from "@/components/business/CtfPanels";
import { PermissionGate } from "@/components/business/PermissionGate";

/** AdminCtfCompetitionsPage 超级管理员CTF竞赛管理页面。 */
export default function AdminCtfCompetitionsPage() {
  return <PermissionGate allowedRoles={["super_admin"]}><CtfAdminCompetitionListPanel /></PermissionGate>;
}
