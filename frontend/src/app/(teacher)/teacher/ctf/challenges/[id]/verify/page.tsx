// page.tsx
// 教师端 CTF 题目预验证页。

import { ChallengeVerificationPanel } from "@/components/business/ChallengeVerificationPanel";
import { PermissionGate } from "@/components/business/PermissionGate";

/** TeacherCtfChallengeVerifyPage 教师CTF题目预验证页面。 */
export default function TeacherCtfChallengeVerifyPage({ params }: { params: { id: string } }) {
  return <PermissionGate allowedRoles={["teacher"]}><ChallengeVerificationPanel challengeID={params.id} /></PermissionGate>;
}
