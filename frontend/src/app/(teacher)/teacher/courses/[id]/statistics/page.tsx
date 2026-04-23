// page.tsx
// 课程统计页，对应模块03 P-11。

import { CourseStatisticsPanel } from "@/components/business/CoursePanels";
import { PermissionGate } from "@/components/business/PermissionGate";

/**
 * TeacherCourseStatisticsPage 课程统计页。
 */
export default function TeacherCourseStatisticsPage({ params }: { params: { id: string } }) {
  return <PermissionGate allowedRoles={["teacher", "school_admin", "super_admin"]}><CourseStatisticsPanel courseID={params.id} /></PermissionGate>;
}
