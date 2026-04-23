// page.tsx
// 超级管理员消息模板管理别名页。

import { AdminTemplatePagePanel } from "@/components/business/NotificationPanels";

/** SuperNotificationTemplatesPage 超级管理员模板页面。 */
export default function SuperNotificationTemplatesPage({ searchParams }: { searchParams: { id?: string } }) {
  return <AdminTemplatePagePanel templateID={searchParams.id} />;
}
