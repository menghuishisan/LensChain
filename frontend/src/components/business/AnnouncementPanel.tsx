"use client";

// AnnouncementPanel.tsx
// 模块07系统公告组件，区分用户视角公告列表与管理员视角创建/编辑/发布/下架管理。

import Link from "next/link";
import { useEffect, useState } from "react";

import { Badge } from "@/components/ui/Badge";
import { Button, buttonClassName } from "@/components/ui/Button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/Card";
import { EmptyState } from "@/components/ui/EmptyState";
import { FormField } from "@/components/ui/FormField";
import { Input } from "@/components/ui/Input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/Select";
import { Textarea } from "@/components/ui/Textarea";
import { useAnnouncement, useAnnouncementMutations, useAnnouncements } from "@/hooks/useAnnouncements";
import { stripHtmlToText } from "@/lib/notification";
import { formatDateTime } from "@/lib/format";
import type { ID } from "@/types/api";
import type { AnnouncementStatus } from "@/types/notification";

/** AnnouncementPanel 组件属性。 */
export interface AnnouncementPanelProps {
  mode: "user" | "admin";
  announcementID?: ID;
}

/** AnnouncementPanel 系统公告组件。 */
export function AnnouncementPanel({ mode, announcementID }: AnnouncementPanelProps) {
  const [status, setStatus] = useState<AnnouncementStatus | "all">("all");
  const announcementsQuery = useAnnouncements(mode === "admin" ? { page: 1, page_size: 20, status: status === "all" ? undefined : status } : { page: 1, page_size: 20 });
  const detailQuery = useAnnouncement(announcementID ?? "");
  const mutations = useAnnouncementMutations(announcementID);
  const [title, setTitle] = useState("");
  const [content, setContent] = useState("");
  const [scheduledAt, setScheduledAt] = useState("");

  useEffect(() => {
    if (!detailQuery.data) {
      return;
    }
    setTitle(detailQuery.data.title);
    setContent(detailQuery.data.content);
    setScheduledAt(detailQuery.data.scheduled_at ?? "");
  }, [detailQuery.data]);

  if (mode === "user") {
    return (
      <Card>
        <CardHeader>
          <CardTitle>系统公告</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          {(announcementsQuery.data?.list ?? []).length === 0 ? <EmptyState title="暂无公告" description="已发布的系统公告会显示在这里。" /> : null}
          {(announcementsQuery.data?.list ?? []).map((item) => (
            <div key={item.id} className="rounded-xl border border-border p-4">
              <div className="flex items-center justify-between gap-3">
                <p className="font-semibold">{item.title}</p>
                {item.is_pinned ? <Badge variant="secondary">置顶</Badge> : null}
              </div>
              <p className="mt-2 text-sm text-muted-foreground">{stripHtmlToText(item.content)}</p>
            </div>
          ))}
        </CardContent>
      </Card>
    );
  }

  if (announcementID) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>编辑公告</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <FormField label="标题">
            <Input value={title} onChange={(event) => setTitle(event.target.value)} />
          </FormField>
          <FormField label="内容">
            <Textarea value={content} onChange={(event) => setContent(event.target.value)} rows={12} />
          </FormField>
          <FormField label="定时发布">
            <Input type="datetime-local" value={scheduledAt} onChange={(event) => setScheduledAt(event.target.value)} />
          </FormField>
          <div className="flex gap-2">
            <Button onClick={() => mutations.update.mutate({ title, content, scheduled_at: scheduledAt || null })} isLoading={mutations.update.isPending}>保存</Button>
            <Button variant="outline" onClick={() => mutations.publish.mutate()} isLoading={mutations.publish.isPending}>发布</Button>
            <Button variant="outline" onClick={() => mutations.unpublish.mutate()} isLoading={mutations.unpublish.isPending}>下架</Button>
            <Button variant="destructive" onClick={() => mutations.remove.mutate()} isLoading={mutations.remove.isPending}>删除</Button>
          </div>
        </CardContent>
      </Card>
    );
  }

  return (
    <div className="space-y-5">
      <Card>
        <CardHeader>
          <CardTitle>创建公告</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <FormField label="标题">
            <Input value={title} onChange={(event) => setTitle(event.target.value)} />
          </FormField>
          <FormField label="内容">
            <Textarea value={content} onChange={(event) => setContent(event.target.value)} rows={10} />
          </FormField>
          <FormField label="定时发布">
            <Input type="datetime-local" value={scheduledAt} onChange={(event) => setScheduledAt(event.target.value)} />
          </FormField>
          <p className="text-sm text-muted-foreground">如不设置定时发布时间，则公告保存后可由管理员手动发布。</p>
          <Button onClick={() => mutations.create.mutate({ title, content, scheduled_at: scheduledAt || null })} isLoading={mutations.create.isPending}>创建公告</Button>
        </CardContent>
      </Card>

      <Card>
        <CardHeader className="flex-row items-center justify-between">
          <CardTitle>公告列表</CardTitle>
          <Select value={String(status)} onValueChange={(value) => setStatus(value === "all" ? "all" : (Number(value) as AnnouncementStatus))}>
            <SelectTrigger className="w-40"><SelectValue /></SelectTrigger>
            <SelectContent>
              <SelectItem value="all">全部状态</SelectItem>
              <SelectItem value="1">草稿</SelectItem>
              <SelectItem value="2">已发布</SelectItem>
              <SelectItem value="3">已下架</SelectItem>
            </SelectContent>
          </Select>
        </CardHeader>
        <CardContent className="space-y-3">
          {(announcementsQuery.data?.list ?? []).map((item) => (
            <div key={item.id} className="rounded-xl border border-border p-4">
              <div className="flex items-center justify-between gap-3">
                <div>
                  <p className="font-semibold">{item.title}</p>
                  <p className="mt-1 text-sm text-muted-foreground">{formatDateTime(item.published_at ?? item.created_at)}</p>
                  {item.is_pinned ? <p className="mt-1 text-xs text-primary">置顶公告</p> : null}
                </div>
                <div className="flex gap-2">
                  {item.status_text ? <Badge variant="outline">{item.status_text}</Badge> : null}
                  <Link className={buttonClassName({ variant: "outline", size: "sm" })} href={`/super/notifications/announcements/${item.id}/edit`}>编辑</Link>
                </div>
              </div>
            </div>
          ))}
        </CardContent>
      </Card>
    </div>
  );
}
