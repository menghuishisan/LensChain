// page.tsx
// 学生我的课程表页，对应模块03 P-27。

import { SchedulePanel } from "@/components/business/CoursePanels";
import { PermissionGate } from "@/components/business/PermissionGate";

/**
 * StudentSchedulePage 我的课程表页。
 */
export default function StudentSchedulePage() {
  return <PermissionGate allowedRoles={["student", "teacher", "school_admin", "super_admin"]}><SchedulePanel /></PermissionGate>;
}
