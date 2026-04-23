// page.tsx
// 教师端共享实验模板库页。

import { PermissionGate } from "@/components/business/PermissionGate";
import { SharedExperimentTemplatePanel } from "@/components/business/SharedExperimentTemplatePanel";

/** TeacherSharedExperimentTemplatesPage 教师共享实验模板库页面。 */
export default function TeacherSharedExperimentTemplatesPage() {
  return (
    <PermissionGate allowedRoles={["teacher"]}>
      <SharedExperimentTemplatePanel />
    </PermissionGate>
  );
}
