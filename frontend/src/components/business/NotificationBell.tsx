"use client";

// NotificationBell.tsx
// 顶部通知入口，展示未读数、最近消息预览和 WebSocket 实时状态。

import { Bell, ChevronRight } from "lucide-react";
import Link from "next/link";
import { useState } from "react";

import { Badge } from "@/components/ui/Badge";
import { Button } from "@/components/ui/Button";
import { useNotificationRealtime } from "@/hooks/useNotificationRealtime";
import { useNotificationMutations, useNotifications, useUnreadCount } from "@/hooks/useNotifications";
import { formatDateTime } from "@/lib/format";

/**
 * NotificationBell 顶部通知铃铛组件。
 */
export function NotificationBell() {
  const [isOpen, setIsOpen] = useState(false);
  const unreadQuery = useUnreadCount();
  const previewQuery = useNotifications({ page: 1, page_size: 5 });
  const mutations = useNotificationMutations();
  const realtime = useNotificationRealtime(true);
  const unreadTotal = unreadQuery.data?.total ?? previewQuery.data?.unread_count ?? 0;
  const unreadText = unreadTotal > 99 ? "99+" : String(unreadTotal);

  return (
    <div className="relative">
      <Button variant="ghost" size="icon" onClick={() => setIsOpen((current) => !current)} aria-label="打开通知入口">
        <Bell className="h-5 w-5" />
        {unreadTotal > 0 ? <span className="absolute right-0 top-0 rounded-full bg-destructive px-1.5 text-[10px] font-semibold text-destructive-foreground">{unreadText}</span> : null}
      </Button>
      {isOpen ? (
        <div className="absolute right-0 top-12 z-40 w-80 rounded-xl border border-border bg-card p-4 text-card-foreground shadow-panel">
          <div className="mb-3 flex items-center justify-between">
            <div>
              <p className="font-semibold">通知</p>
              <p className="text-xs text-muted-foreground">实时连接：{realtime.status}</p>
            </div>
            <Badge variant={unreadTotal > 0 ? "destructive" : "outline"}>{unreadText}</Badge>
          </div>
          <div className="max-h-80 space-y-2 overflow-y-auto">
            {(previewQuery.data?.list ?? []).length === 0 ? <div className="rounded-lg bg-muted/70 p-4 text-sm text-muted-foreground">暂无通知。</div> : null}
            {(previewQuery.data?.list ?? []).map((item) => (
              <Link key={item.id} href={`/notifications/${item.id}`} className="block rounded-lg border border-border p-3 hover:bg-muted" onClick={() => setIsOpen(false)}>
                <div className="flex items-center gap-2">
                  {!item.is_read ? <span className="h-2 w-2 rounded-full bg-sky-500" /> : null}
                  <p className="line-clamp-1 text-sm font-semibold">{item.title}</p>
                </div>
                <p className="mt-1 line-clamp-1 text-xs text-muted-foreground">{item.category_text} · {formatDateTime(item.created_at)}</p>
              </Link>
            ))}
          </div>
          <Button className="mt-3 w-full" variant="outline" size="sm" onClick={() => mutations.readAll.mutate()} isLoading={mutations.readAll.isPending}>
            全部标记已读
          </Button>
          <Link
            href="/notifications"
            className="mt-3 flex items-center justify-between rounded-lg px-2 py-2 text-sm font-semibold text-primary hover:bg-primary/8"
            onClick={() => setIsOpen(false)}
          >
            查看消息中心
            <ChevronRight className="h-4 w-4" />
          </Link>
        </div>
      ) : null}
    </div>
  );
}
