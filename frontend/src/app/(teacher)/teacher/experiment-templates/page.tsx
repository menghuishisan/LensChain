// page.tsx
// 教师端实验模板列表页。

import { ExperimentTemplateListPanel } from "@/components/business/ExperimentTemplatePanels";
import { PermissionGate } from "@/components/business/PermissionGate";

/** TeacherExperimentTemplatesPage 教师实验模板列表页面。 */
export default function TeacherExperimentTemplatesPage() {
  return (
    <PermissionGate allowedRoles={["teacher"]}>
      <ExperimentTemplateListPanel />
    </PermissionGate>
  );
}
