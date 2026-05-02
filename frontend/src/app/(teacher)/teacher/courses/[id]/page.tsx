// page.tsx
// 课程管理主页，对应模块03 P-03。

import { CourseOverviewPanel } from "@/components/business/CourseOverviewPanel";
import { PermissionGate } from "@/components/business/PermissionGate";

/**
 * TeacherCourseDetailPage 教师课程管理主页。
 */
export default function TeacherCourseDetailPage({ params }: { params: { id: string } }) {
  return <PermissionGate allowedRoles={["teacher"]}><CourseOverviewPanel courseID={params.id} /></PermissionGate>;
}
