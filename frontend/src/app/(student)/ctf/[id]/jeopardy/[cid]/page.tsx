// page.tsx
// 学生端 CTF 题目详情和解题页。

import { CtfChallengePanel } from "@/components/business/CtfChallengePanel";
import { PermissionGate } from "@/components/business/PermissionGate";

/** CtfChallengeSolvePage CTF题目详情和解题页面。 */
export default function CtfChallengeSolvePage({ params }: { params: { id: string; cid: string } }) {
  return <PermissionGate allowedRoles={["student"]}><CtfChallengePanel competitionID={params.id} challengeID={params.cid} /></PermissionGate>;
}
