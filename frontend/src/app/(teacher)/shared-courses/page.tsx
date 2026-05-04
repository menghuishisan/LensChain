// page.tsx
// 共享课程库页，对应模块03 P-33。

import { PermissionGate } from "@/components/business/PermissionGate";
import { SharedCoursesPanel } from "@/components/business/SharedCoursesPanel";

/**
 * SharedCoursesCanonicalPage 共享课程库页。
 */
export default function SharedCoursesCanonicalPage() {
  return <PermissionGate allowedRoles={["teacher"]}><SharedCoursesPanel /></PermissionGate>;
}
