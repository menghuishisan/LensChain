// page.tsx
// 作业批改页，对应模块03 P-08。

import { PermissionGate } from "@/components/business/PermissionGate";
import { SubmissionReviewPanel } from "@/components/business/SubmissionReviewPanel";

/**
 * TeacherSubmissionGradePage 作业批改页。
 */
export default function TeacherSubmissionGradePage({ params }: { params: { id: string } }) {
  return <PermissionGate allowedRoles={["teacher", "school_admin", "super_admin"]}><SubmissionReviewPanel submissionID={params.id} /></PermissionGate>;
}
