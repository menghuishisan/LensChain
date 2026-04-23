// page.tsx
// 教师端 CTF 题目管理页。

import { CtfChallengeManagementPanel } from "@/components/business/CtfPanels";
import { PermissionGate } from "@/components/business/PermissionGate";

/** TeacherCtfChallengesPage 教师CTF题目管理页面。 */
export default function TeacherCtfChallengesPage() {
  return <PermissionGate allowedRoles={["teacher"]}><CtfChallengeManagementPanel /></PermissionGate>;
}
