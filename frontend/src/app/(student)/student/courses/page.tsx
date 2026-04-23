// page.tsx
// 学生我的课程列表页，对应模块03 P-20。

import { StudentCourseListPanel } from "@/components/business/CoursePanels";
import { PermissionGate } from "@/components/business/PermissionGate";

/**
 * StudentCoursesPage 我的课程列表页。
 */
export default function StudentCoursesPage() {
  return <PermissionGate allowedRoles={["student"]}><StudentCourseListPanel /></PermissionGate>;
}
