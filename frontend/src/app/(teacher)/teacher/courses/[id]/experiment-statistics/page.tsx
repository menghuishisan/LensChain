// page.tsx
// 教师端课程实验统计页。

import { TeacherExperimentStatisticsPanel } from "@/components/business/ExperimentInstanceListPanels";
import { PermissionGate } from "@/components/business/PermissionGate";

/** TeacherCourseExperimentStatisticsPage 教师课程实验统计页面。 */
export default function TeacherCourseExperimentStatisticsPage({ params }: { params: { id: string } }) {
  return (
    <PermissionGate allowedRoles={["teacher"]}>
      <TeacherExperimentStatisticsPanel courseID={params.id} />
    </PermissionGate>
  );
}
