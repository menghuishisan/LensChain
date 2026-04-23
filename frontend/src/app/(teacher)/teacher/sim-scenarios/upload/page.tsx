// page.tsx
// 教师端仿真场景上传页。

import { PermissionGate } from "@/components/business/PermissionGate";
import { SimScenarioLibraryPanel } from "@/components/business/ExperimentTemplatePanels";

/** TeacherSimScenarioUploadPage 教师仿真场景上传页面。 */
export default function TeacherSimScenarioUploadPage() {
  return (
    <PermissionGate allowedRoles={["teacher"]}>
      <SimScenarioLibraryPanel />
    </PermissionGate>
  );
}
