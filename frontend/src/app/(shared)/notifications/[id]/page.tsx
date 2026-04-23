// page.tsx
// 通用消息详情页。

import { NotificationInboxPagePanel } from "@/components/business/NotificationPanels";

/** NotificationDetailPage 消息详情页面。 */
export default function NotificationDetailPage({ params }: { params: { id: string } }) {
  return <NotificationInboxPagePanel messageID={params.id} />;
}
