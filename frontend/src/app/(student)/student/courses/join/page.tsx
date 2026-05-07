// page.tsx
// 学生加入课程页，对应模块03 P-21。

import { JoinCoursePanel } from "@/components/business/CourseListPanels";
import { PermissionGate } from "@/components/business/PermissionGate";

/**
 * StudentCourseJoinPage 加入课程页。
 */
export default function StudentCourseJoinPage() {
  return <PermissionGate allowedRoles={["student"]}><JoinCoursePanel /></PermissionGate>;
}
