// page.tsx
// 课程设置页，对应模块03 P-12。

import { CourseSettingsPanel } from "@/components/business/CourseSettingsPanel";
import { PermissionGate } from "@/components/business/PermissionGate";

/**
 * TeacherCourseSettingsPage 课程设置页。
 */
export default function TeacherCourseSettingsPage({ params }: { params: { id: string } }) {
  return <PermissionGate allowedRoles={["teacher", "school_admin", "super_admin"]}><CourseSettingsPanel courseID={params.id} /></PermissionGate>;
}
