// page.tsx
// 教师端 CTF 参数化模板库页。

import { PermissionGate } from "@/components/business/PermissionGate";
import { VulnerabilityConvertPanel } from "@/components/business/VulnerabilityConvertPanel";

/** TeacherCtfTemplatesPage 教师CTF模板库页面。 */
export default function TeacherCtfTemplatesPage() {
  return <PermissionGate allowedRoles={["teacher"]}><VulnerabilityConvertPanel /></PermissionGate>;
}
