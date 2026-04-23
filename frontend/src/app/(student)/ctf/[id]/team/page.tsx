// page.tsx
// 学生端 CTF 团队管理页。

import { CtfTeamPanel } from "@/components/business/CtfPanels";
import { PermissionGate } from "@/components/business/PermissionGate";

/** CtfTeamPage CTF团队管理页面。 */
export default function CtfTeamPage({ params }: { params: { id: string } }) {
  return <PermissionGate allowedRoles={["student"]}><CtfTeamPanel competitionID={params.id} /></PermissionGate>;
}
