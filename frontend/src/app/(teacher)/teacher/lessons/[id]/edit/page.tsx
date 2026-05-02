// page.tsx
// 课时编辑页，对应模块03 P-05。

import { LessonContentEditor } from "@/components/business/LessonContentEditor";
import { PermissionGate } from "@/components/business/PermissionGate";

/**
 * TeacherLessonEditPage 课时编辑页。
 */
export default function TeacherLessonEditPage({ params }: { params: { id: string } }) {
  return <PermissionGate allowedRoles={["teacher"]}><LessonContentEditor lessonID={params.id} /></PermissionGate>;
}
