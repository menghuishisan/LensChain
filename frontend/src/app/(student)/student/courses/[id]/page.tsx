// page.tsx
// 学生课程学习主页，对应模块03 P-22。

import { StudentCourseHomePanel } from "@/components/business/CourseContentPanels";
import { PermissionGate } from "@/components/business/PermissionGate";

/**
 * StudentCourseHomePage 课程学习主页。
 */
export default function StudentCourseHomePage({ params }: { params: { id: string } }) {
  return <PermissionGate allowedRoles={["student"]}><StudentCourseHomePanel courseID={params.id} /></PermissionGate>;
}
