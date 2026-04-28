// page.tsx
// 教师端 CTF 参数化模板库页。

import { PermissionGate } from "@/components/business/PermissionGate";
import { CtfTemplateLibraryPanel } from "@/components/business/CtfTemplateLibraryPanel";

/** TeacherCtfTemplatesPage 教师CTF模板库页面。 */
export default function TeacherCtfTemplatesPage() {
  return <PermissionGate allowedRoles={["teacher"]}><CtfTemplateLibraryPanel /></PermissionGate>;
}
