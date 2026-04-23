// page.tsx
// 教师端创建实验模板页。

import { ExperimentTemplateEditorPanel } from "@/components/business/ExperimentTemplatePanels";
import { PermissionGate } from "@/components/business/PermissionGate";

/** TeacherCreateExperimentTemplatePage 教师创建实验模板页面。 */
export default function TeacherCreateExperimentTemplatePage() {
  return (
    <PermissionGate allowedRoles={["teacher"]}>
      <ExperimentTemplateEditorPanel />
    </PermissionGate>
  );
}
