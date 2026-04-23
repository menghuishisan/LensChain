// page.tsx
// 教师端实验模板详情编辑页。

import { ExperimentTemplateEditorPanel } from "@/components/business/ExperimentTemplatePanels";
import { PermissionGate } from "@/components/business/PermissionGate";

/** TeacherExperimentTemplateDetailPage 教师实验模板详情页面。 */
export default function TeacherExperimentTemplateDetailPage({ params }: { params: { id: string } }) {
  return (
    <PermissionGate allowedRoles={["teacher"]}>
      <ExperimentTemplateEditorPanel templateID={params.id} />
    </PermissionGate>
  );
}
