// page.tsx
// 帖子详情页，对应模块03 P-31。

import { DiscussionThread } from "@/components/business/DiscussionThread";
import { PermissionGate } from "@/components/business/PermissionGate";

/**
 * DiscussionDetailPage 帖子详情页。
 */
export default function DiscussionDetailPage({ params }: { params: { id: string } }) {
  return <PermissionGate allowedRoles={["student", "teacher"]}><DiscussionThread discussionID={params.id} /></PermissionGate>;
}
