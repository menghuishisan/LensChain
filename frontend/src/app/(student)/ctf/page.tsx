// page.tsx
// 学生端 CTF 竞赛大厅页。

import { CtfHallPanel } from "@/components/business/CtfPanels";
import { PermissionGate } from "@/components/business/PermissionGate";

/** CtfHallPage CTF竞赛大厅页面。 */
export default function CtfHallPage() {
  return <PermissionGate allowedRoles={["student"]}><CtfHallPanel /></PermissionGate>;
}
