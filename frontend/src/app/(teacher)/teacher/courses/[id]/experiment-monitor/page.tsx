// page.tsx
// 教师端课程实验监控页。

import { TeacherExperimentMonitorPanel } from "@/components/business/ExperimentInstanceListPanels";
import { PermissionGate } from "@/components/business/PermissionGate";

/** TeacherCourseExperimentMonitorPage 教师课程实验监控页面。 */
export default function TeacherCourseExperimentMonitorPage({ params }: { params: { id: string } }) {
  return (
    <PermissionGate allowedRoles={["teacher"]}>
      <TeacherExperimentMonitorPanel courseID={params.id} />
    </PermissionGate>
  );
}
