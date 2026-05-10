// page.tsx
// 教师端仿真场景上传页（06.2 §十一）。

import { PermissionGate } from "@/components/business/PermissionGate";
import { SimScenarioUploadWizard } from "@/components/business/SimScenarioUploadWizard";

/** TeacherSimScenarioUploadPage 教师仿真场景上传页面。 */
export default function TeacherSimScenarioUploadPage() {
  return (
    <PermissionGate allowedRoles={["teacher"]}>
      <div className="space-y-5">
        <h1 className="font-display text-3xl font-semibold">上传自定义仿真场景</h1>
        <SimScenarioUploadWizard />
      </div>
    </PermissionGate>
  );
}
