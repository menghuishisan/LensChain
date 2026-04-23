// page.tsx
// 学生端 CTF 解题赛主页。

import { CtfJeopardyPanel } from "@/components/business/CtfPanels";
import { PermissionGate } from "@/components/business/PermissionGate";

/** CtfJeopardyPage CTF解题赛主页。 */
export default function CtfJeopardyPage({ params }: { params: { id: string } }) {
  return <PermissionGate allowedRoles={["student"]}><CtfJeopardyPanel competitionID={params.id} /></PermissionGate>;
}
