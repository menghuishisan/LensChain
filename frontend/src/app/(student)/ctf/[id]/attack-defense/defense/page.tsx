// page.tsx
// 学生端 CTF 攻防赛防守页。

import { AttackDefenseRoundPanel } from "@/components/business/AttackDefenseRoundPanel";
import { PermissionGate } from "@/components/business/PermissionGate";

/** CtfDefensePage CTF攻防赛防守页面。 */
export default function CtfDefensePage({ params }: { params: { id: string } }) {
  return <PermissionGate allowedRoles={["student"]}><AttackDefenseRoundPanel competitionID={params.id} mode="defense" /></PermissionGate>;
}
