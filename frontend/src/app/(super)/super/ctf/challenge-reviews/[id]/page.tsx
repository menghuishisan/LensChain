// page.tsx
// 超级管理员 CTF 题目审核详情页。

import { CtfChallengeReviewDetailPanel } from "@/components/business/CtfPanels";
import { PermissionGate } from "@/components/business/PermissionGate";

/** SuperCtfChallengeReviewDetailPage 超级管理员CTF题目审核详情页面。 */
export default function SuperCtfChallengeReviewDetailPage({ params }: { params: { id: string } }) {
  return <PermissionGate allowedRoles={["super_admin"]}><CtfChallengeReviewDetailPanel challengeID={params.id} /></PermissionGate>;
}
