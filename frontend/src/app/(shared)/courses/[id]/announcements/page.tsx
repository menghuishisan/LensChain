"use client";

// page.tsx
// 课程公告页，对应模块03 P-32。

import { AnnouncementPanel } from "@/components/business/CourseInteractionPanels";
import { PermissionGate } from "@/components/business/PermissionGate";
import { useAuth } from "@/hooks/useAuth";

/**
 * CourseAnnouncementsPage 课程公告页。
 */
export default function CourseAnnouncementsPage({ params }: { params: { id: string } }) {
  const { roles } = useAuth();
  const role = roles.includes("teacher") ? "teacher" : "student";
  return <PermissionGate allowedRoles={["student", "teacher"]}><AnnouncementPanel courseID={params.id} role={role} /></PermissionGate>;
}
