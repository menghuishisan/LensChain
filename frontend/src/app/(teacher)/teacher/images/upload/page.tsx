// page.tsx
// 教师端镜像上传与本校镜像页。

import { ExperimentImageUploadPanel } from "@/components/business/ExperimentTemplatePanels";
import { PermissionGate } from "@/components/business/PermissionGate";

/** TeacherImageUploadPage 教师镜像上传页面。 */
export default function TeacherImageUploadPage() {
  return (
    <PermissionGate allowedRoles={["teacher", "school_admin"]}>
      <ExperimentImageUploadPanel />
    </PermissionGate>
  );
}
