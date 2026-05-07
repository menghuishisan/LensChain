// page.tsx
// 学生课时学习页，对应模块03 P-23。

import { PermissionGate } from "@/components/business/PermissionGate";
import { StudentLessonPanel } from "@/components/business/CourseContentPanels";

/**
 * StudentLessonPage 课时学习页。
 */
export default function StudentLessonPage({ params }: { params: { id: string } }) {
  return <PermissionGate allowedRoles={["student"]}><StudentLessonPanel lessonID={params.id} /></PermissionGate>;
}
