// page.tsx
// 管理端消息模板管理页。

import { AdminTemplatePagePanel } from "@/components/business/NotificationPanels";

/** AdminNotificationTemplatesPage 消息模板管理页面。 */
export default function AdminNotificationTemplatesPage({ searchParams }: { searchParams: { id?: string } }) {
  return <AdminTemplatePagePanel templateID={searchParams.id} />;
}
