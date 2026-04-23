// page.tsx
// 课程讨论区，对应模块03 P-30。

import { DiscussionListPanel } from "@/components/business/CoursePanels";
import { PermissionGate } from "@/components/business/PermissionGate";

/**
 * CourseDiscussionsPage 课程讨论区。
 */
export default function CourseDiscussionsPage({ params }: { params: { id: string } }) {
  return <PermissionGate allowedRoles={["student", "teacher", "school_admin", "super_admin"]}><DiscussionListPanel courseID={params.id} /></PermissionGate>;
}
