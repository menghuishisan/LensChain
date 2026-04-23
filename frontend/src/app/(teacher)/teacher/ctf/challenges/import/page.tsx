// page.tsx
// 教师端 CTF 漏洞转化工具页。

import { PermissionGate } from "@/components/business/PermissionGate";
import { VulnerabilityConvertPanel } from "@/components/business/VulnerabilityConvertPanel";

/** TeacherCtfChallengeImportPage 教师CTF漏洞转化页面。 */
export default function TeacherCtfChallengeImportPage() {
  return <PermissionGate allowedRoles={["teacher"]}><VulnerabilityConvertPanel /></PermissionGate>;
}
