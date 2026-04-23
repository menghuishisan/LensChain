// page.tsx
// 创建课程页，对应模块03 P-02。

import { CourseEditorForm } from "@/components/business/CourseEditorForm";
import { PermissionGate } from "@/components/business/PermissionGate";

/**
 * TeacherCourseCreatePage 创建课程页。
 */
export default function TeacherCourseCreatePage() {
  return <PermissionGate allowedRoles={["teacher", "school_admin", "super_admin"]}><CourseEditorForm /></PermissionGate>;
}
