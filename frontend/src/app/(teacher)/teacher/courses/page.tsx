// page.tsx
// 教师课程列表页，对应模块03 P-01。

import { PermissionGate } from "@/components/business/PermissionGate";
import { TeacherCourseListPanel } from "@/components/business/CoursePanels";

/**
 * TeacherCoursesPage 教师课程列表页。
 */
export default function TeacherCoursesPage() {
  return <PermissionGate allowedRoles={["teacher"]}><TeacherCourseListPanel /></PermissionGate>;
}
