// page.tsx
// 教师端 CTF 题目创建页。

import { CtfChallengeEditorPanel } from "@/components/business/CtfPanels";
import { PermissionGate } from "@/components/business/PermissionGate";

/** TeacherCtfChallengeCreatePage 教师CTF题目创建页面。 */
export default function TeacherCtfChallengeCreatePage() {
  return <PermissionGate allowedRoles={["teacher"]}><CtfChallengeEditorPanel /></PermissionGate>;
}
