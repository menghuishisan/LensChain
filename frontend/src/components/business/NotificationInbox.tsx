"use client";

// NotificationInbox.tsx
// 模块07收件箱组件，分开展示系统公告和普通站内信，并提供已读、批量已读、全部已读、删除。

import { BellRing, CheckCheck, Trash2 } from "lucide-react";
import Link from "next/link";
import { useMemo, useState } from "react";

import { Badge } from "@/components/ui/Badge";
import { Button } from "@/components/ui/Button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/Card";
import { EmptyState } from "@/components/ui/EmptyState";
import { ErrorState } from "@/components/ui/ErrorState";
import { FormField } from "@/components/ui/FormField";
import { Input } from "@/components/ui/Input";
import { LoadingState } from "@/components/ui/LoadingState";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/Select";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/Tabs";
import { useAnnouncements } from "@/hooks/useAnnouncements";
import { useNotification, useNotificationMutations, useNotifications, useUnreadCount } from "@/hooks/useNotifications";
import { getNotificationCategoryVariant, NOTIFICATION_CATEGORY_OPTIONS, resolveNotificationSourceHref, stripHtmlToText } from "@/lib/notification";
import { formatDateTime } from "@/lib/format";
import type { ID } from "@/types/api";
import type { NotificationCategory } from "@/types/notification";

/** NotificationInbox 组件属性。 */
export interface NotificationInboxProps {
  messageID?: ID;
}

/** NotificationInbox 收件箱组件。 */
export function NotificationInbox({ messageID }: NotificationInboxProps) {
  const [category, setCategory] = useState<NotificationCategory | "all">("all");
  const [isRead, setIsRead] = useState<"all" | "true" | "false">("all");
  const [keyword, setKeyword] = useState("");
  const inboxQuery = useNotifications({
    page: 1,
    page_size: messageID ? 50 : 20,
    category: category === "all" ? undefined : category,
    is_read: isRead === "all" ? undefined : isRead === "true",
    keyword: keyword || undefined,
  });
  const detailQuery = useNotification(messageID ?? "");
  const announcementsQuery = useAnnouncements({ page: 1, page_size: 5 });
  const detailMutations = useNotificationMutations(messageID);
  const inboxMutations = useNotificationMutations();
  const unreadQuery = useUnreadCount();
  const [selectedIDs, setSelectedIDs] = useState<ID[]>([]);
  const inbox = inboxQuery.data;
  const categoryCounts = useMemo(() => {
    const counts = new Map<string, number>();
    counts.set("all", inbox?.pagination.total ?? 0);
    counts.set("1", unreadQuery.data?.by_category.system ?? 0);
    counts.set("2", unreadQuery.data?.by_category.course ?? 0);
    counts.set("3", unreadQuery.data?.by_category.experiment ?? 0);
    counts.set("4", unreadQuery.data?.by_category.competition ?? 0);
    counts.set("5", unreadQuery.data?.by_category.grade ?? 0);
    return counts;
  }, [inbox?.pagination.total, unreadQuery.data]);

  if (messageID) {
    if (detailQuery.isLoading) {
      return <LoadingState title="正在加载消息详情" description="读取通知内容并自动标记已读。" />;
    }
    if (detailQuery.isError || !detailQuery.data) {
      return <ErrorState title="消息详情加载失败" description={detailQuery.error?.message ?? "消息不存在或已被删除。"} />;
    }
    const message = detailQuery.data;
    const href = resolveNotificationSourceHref(message.source_type, message.source_id);
    return (
      <Card>
        <CardHeader>
          <CardTitle>{message.title}</CardTitle>
          <p className="text-sm text-muted-foreground">{message.category_text} · {formatDateTime(message.created_at)}</p>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="rounded-xl border border-border p-4 text-sm leading-7 text-muted-foreground whitespace-pre-wrap">{stripHtmlToText(message.content)}</div>
          <div className="flex flex-wrap gap-2">
            {href ? <Link className="inline-flex items-center rounded-lg border border-border px-4 py-2 text-sm font-semibold hover:bg-muted" href={href}>前往查看</Link> : null}
            <Button variant="destructive" onClick={() => detailMutations.remove.mutate()} isLoading={detailMutations.remove.isPending}>
              <Trash2 className="h-4 w-4" />
              删除消息
            </Button>
          </div>
        </CardContent>
      </Card>
    );
  }

  if (inboxQuery.isLoading) {
    return <LoadingState title="正在加载消息中心" description="读取收件箱、公告和未读计数。" />;
  }

  if (inboxQuery.isError) {
    return <ErrorState title="消息中心加载失败" description={inboxQuery.error.message} />;
  }

  return (
    <div className="space-y-5">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <h1 className="font-display text-3xl font-semibold">消息中心</h1>
        <div className="flex flex-wrap gap-2">
          <Button variant="outline" onClick={() => inboxMutations.batchRead.mutate(selectedIDs)} disabled={selectedIDs.length === 0} isLoading={inboxMutations.batchRead.isPending}>批量已读</Button>
          <Button onClick={() => inboxMutations.readAll.mutate()} isLoading={inboxMutations.readAll.isPending}>
            <CheckCheck className="h-4 w-4" />
            全部标记已读
          </Button>
        </div>
      </div>

      <Tabs value={String(category)} onValueChange={(value) => setCategory(value === "all" ? "all" : (Number(value) as NotificationCategory))}>
        <TabsList className="flex w-full flex-wrap justify-start">
          <TabsTrigger value="all">全部({categoryCounts.get("all") ?? 0})</TabsTrigger>
          {NOTIFICATION_CATEGORY_OPTIONS.map((item) => (
            <TabsTrigger key={item.value} value={String(item.value)}>{item.label}({categoryCounts.get(String(item.value)) ?? 0})</TabsTrigger>
          ))}
        </TabsList>
      </Tabs>

      <div className="grid gap-3 md:grid-cols-[1fr_180px]">
        <Input value={keyword} onChange={(event) => setKeyword(event.target.value)} placeholder="搜索消息标题" />
        <Select value={isRead} onValueChange={(value) => setIsRead(value as "all" | "true" | "false")}>
          <SelectTrigger><SelectValue /></SelectTrigger>
          <SelectContent>
            <SelectItem value="all">全部状态</SelectItem>
            <SelectItem value="false">仅未读</SelectItem>
            <SelectItem value="true">仅已读</SelectItem>
          </SelectContent>
        </Select>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>系统公告</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          {(announcementsQuery.data?.list ?? []).length === 0 ? <p className="text-sm text-muted-foreground">暂无系统公告。</p> : null}
          {(announcementsQuery.data?.list ?? []).map((item) => (
            <div key={item.id} className="rounded-xl border border-border bg-muted/20 p-4">
              <div className="flex items-center justify-between gap-3">
                <p className="font-semibold">{item.title}</p>
                <div className="flex gap-2">
                  {item.is_pinned ? <Badge variant="secondary">置顶</Badge> : null}
                  <Badge variant="secondary">公告</Badge>
                </div>
              </div>
              <p className="mt-2 text-sm text-muted-foreground">{stripHtmlToText(item.content)}</p>
            </div>
          ))}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>站内信列表</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          {(inbox?.list ?? []).length === 0 ? <EmptyState title="暂无消息" description="当前筛选条件下没有消息。" /> : null}
          {(inbox?.list ?? []).map((item) => (
            <label key={item.id} className="block cursor-pointer rounded-xl border border-border p-4 hover:bg-muted/25">
              <div className="flex items-start gap-3">
                <input className="mt-1" type="checkbox" checked={selectedIDs.includes(item.id)} onChange={(event) => setSelectedIDs((current) => event.target.checked ? [...current, item.id] : current.filter((id) => id !== item.id))} />
                <div className="min-w-0 flex-1">
                  <div className="flex flex-wrap items-center gap-2">
                    {!item.is_read ? <span className="h-2 w-2 rounded-full bg-sky-500" /> : null}
                    <Badge variant={getNotificationCategoryVariant(item.category)}>{item.category_text}</Badge>
                    <Link className={`font-semibold hover:text-primary ${!item.is_read ? "text-foreground" : "text-muted-foreground"}`} href={`/notifications/${item.id}`}>{item.title}</Link>
                    <span className="text-xs text-muted-foreground">{formatDateTime(item.created_at)}</span>
                  </div>
                  <p className="mt-2 line-clamp-2 text-sm text-muted-foreground">{stripHtmlToText(item.content)}</p>
                  {resolveNotificationSourceHref(item.source_type, item.source_id) ? <p className="mt-2 text-xs font-semibold text-primary">支持前往查看</p> : null}
                </div>
              </div>
            </label>
          ))}
          <p className="text-xs text-muted-foreground">当前未读数：{inbox?.unread_count ?? 0}</p>
        </CardContent>
      </Card>
    </div>
  );
}
