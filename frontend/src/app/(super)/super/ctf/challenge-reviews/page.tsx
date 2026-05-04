// page.tsx
// 超级管理员 CTF 题目审核列表页。

import { CtfChallengeReviewPanel } from "@/components/business/CtfPanels";
import { PermissionGate } from "@/components/business/PermissionGate";

/** SuperCtfChallengeReviewsPage 超级管理员CTF题目审核列表页面。 */
export default function SuperCtfChallengeReviewsPage() {
  return <PermissionGate allowedRoles={["super_admin"]}><CtfChallengeReviewPanel /></PermissionGate>;
}
