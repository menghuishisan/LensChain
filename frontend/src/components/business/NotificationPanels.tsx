"use client";

// NotificationPanels.tsx
// 模块07页面级通知面板，组合收件箱、偏好、公告、模板、统计和定向通知。

import { BarChart3, Send } from "lucide-react";
import { useState } from "react";

import { AnnouncementPanel } from "@/components/business/AnnouncementPanel";
import { NotificationInbox } from "@/components/business/NotificationInbox";
import { NotificationPreferenceForm } from "@/components/business/NotificationPreferenceForm";
import { NotificationTemplateEditor } from "@/components/business/NotificationTemplateEditor";
import { PermissionGate } from "@/components/business/PermissionGate";
import { Button } from "@/components/ui/Button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/Card";
import { FormField } from "@/components/ui/FormField";
import { Input } from "@/components/ui/Input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/Select";
import { Textarea } from "@/components/ui/Textarea";
import { useDirectNotificationMutation, useNotificationStatistics } from "@/hooks/useNotificationTemplates";
import { NOTIFICATION_CATEGORY_OPTIONS } from "@/lib/notification";
import type { ID } from "@/types/api";
import type { NotificationCategory } from "@/types/notification";

/** NotificationInboxPagePanel 消息中心页面面板。 */
export function NotificationInboxPagePanel({ messageID }: { messageID?: ID }) {
  return <NotificationInbox messageID={messageID} />;
}

/** NotificationPreferencePagePanel 通知偏好页面面板。 */
export function NotificationPreferencePagePanel() {
  return <NotificationPreferenceForm />;
}

/** AdminAnnouncementPagePanel 管理端公告页面面板。 */
export function AdminAnnouncementPagePanel({ announcementID }: { announcementID?: ID }) {
  return (
    <PermissionGate allowedRoles={["super_admin"]}>
      <AnnouncementPanel mode="admin" announcementID={announcementID} />
    </PermissionGate>
  );
}

/** AdminTemplatePagePanel 管理端模板页面面板。 */
export function AdminTemplatePagePanel({ templateID }: { templateID?: ID }) {
  return (
    <PermissionGate allowedRoles={["super_admin"]}>
      <NotificationTemplateEditor templateID={templateID} />
    </PermissionGate>
  );
}

/** DirectNotificationPanel 定向通知发送页面。 */
export function DirectNotificationPanel() {
  const mutation = useDirectNotificationMutation();
  const [targetType, setTargetType] = useState<"all_school" | "course" | "user" | "users">("course");
  const [category, setCategory] = useState<NotificationCategory>(2);
  const [targetID, setTargetID] = useState("");
  const [title, setTitle] = useState("");
  const [content, setContent] = useState("");

  return (
    <PermissionGate allowedRoles={["school_admin", "teacher"]}>
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Send className="h-5 w-5 text-primary" />
            发送通知
          </CardTitle>
        </CardHeader>
        <CardContent className="grid gap-4">
          <div className="grid gap-4 md:grid-cols-3">
            <FormField label="发送对象">
              <Select value={targetType} onValueChange={(value) => setTargetType(value as "all_school" | "course" | "user" | "users")}>
                <SelectTrigger><SelectValue /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="all_school">全校用户</SelectItem>
                  <SelectItem value="course">指定课程学生</SelectItem>
                  <SelectItem value="user">指定用户</SelectItem>
                  <SelectItem value="users">多个用户</SelectItem>
                </SelectContent>
              </Select>
            </FormField>
            <FormField label="目标ID">
              <Input value={targetID} onChange={(event) => setTargetID(event.target.value)} placeholder={targetType === "all_school" ? "可填当前学校ID" : "课程/用户ID"} />
            </FormField>
            <FormField label="分类">
              <Select value={String(category)} onValueChange={(value) => setCategory(Number(value) as NotificationCategory)}>
                <SelectTrigger><SelectValue /></SelectTrigger>
                <SelectContent>
                  {NOTIFICATION_CATEGORY_OPTIONS.map((item) => <SelectItem key={item.value} value={String(item.value)}>{item.label}</SelectItem>)}
                </SelectContent>
              </Select>
            </FormField>
          </div>
          <FormField label="标题">
            <Input value={title} onChange={(event) => setTitle(event.target.value)} />
          </FormField>
          <FormField label="内容">
            <Textarea value={content} onChange={(event) => setContent(event.target.value)} rows={8} />
          </FormField>
          <Button disabled={!title || !content || !targetID} onClick={() => mutation.mutate({ title, content, target_type: targetType, target_id: targetID, category })} isLoading={mutation.isPending}>
            发送通知
          </Button>
        </CardContent>
      </Card>
    </PermissionGate>
  );
}

/** NotificationStatisticsPanel 消息统计页面。 */
export function NotificationStatisticsPanel() {
  const statisticsQuery = useNotificationStatistics();
  const data = statisticsQuery.data;
  return (
    <PermissionGate allowedRoles={["super_admin"]}>
      <div className="space-y-5">
        <div className="grid gap-3 md:grid-cols-3">
          <MetricCard title="发送总数" value={String(data?.total_sent ?? 0)} />
          <MetricCard title="已读总数" value={String(data?.total_read ?? 0)} />
          <MetricCard title="已读率" value={`${Math.round((data?.read_rate ?? 0) * 100)}%`} />
        </div>
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <BarChart3 className="h-5 w-5 text-primary" />
              分类统计
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            {(data?.by_category ?? []).map((item) => (
              <div key={item.category} className="rounded-xl border border-border p-4">
                <div className="flex items-center justify-between">
                  <p className="font-semibold">{item.category}</p>
                  <span className="text-sm text-muted-foreground">{Math.round(item.read_rate * 100)}%</span>
                </div>
                <p className="mt-1 text-sm text-muted-foreground">发送 {item.sent}，已读 {item.read}</p>
              </div>
            ))}
          </CardContent>
        </Card>
      </div>
    </PermissionGate>
  );
}

function MetricCard({ title, value }: { title: string; value: string }) {
  return (
    <Card>
      <CardContent className="p-4">
        <p className="text-sm text-muted-foreground">{title}</p>
        <p className="mt-1 font-display text-2xl font-semibold">{value}</p>
      </CardContent>
    </Card>
  );
}
