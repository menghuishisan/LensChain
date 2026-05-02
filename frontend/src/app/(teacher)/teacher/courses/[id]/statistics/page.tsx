// page.tsx
// 课程统计页，对应模块03 P-11。

import { CourseStatisticsPanel } from "@/components/business/CourseStatisticsPanel";
import { PermissionGate } from "@/components/business/PermissionGate";

/**
 * TeacherCourseStatisticsPage 课程统计页。
 */
export default function TeacherCourseStatisticsPage({ params }: { params: { id: string } }) {
  return <PermissionGate allowedRoles={["teacher"]}><CourseStatisticsPanel courseID={params.id} /></PermissionGate>;
}
