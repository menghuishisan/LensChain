"use client";

// TopBar.tsx
// 已登录主布局顶部栏，提供移动端菜单、通知入口和用户菜单。

import { Menu, Search } from "lucide-react";

import { NotificationBell } from "@/components/business/NotificationBell";
import { UserMenu } from "@/components/business/UserMenu";
import { Button } from "@/components/ui/Button";
import { ROLE_TEXT } from "@/lib/permissions";
import type { AuthUser, UserRole } from "@/types/auth";

/**
 * TopBar 组件属性。
 */
export interface TopBarProps {
  user: AuthUser | null;
  primaryRole: UserRole;
  onMenuClick: () => void;
}

/**
 * TopBar 已登录主布局顶部栏组件。
 */
export function TopBar({ user, primaryRole, onMenuClick }: TopBarProps) {
  return (
    <header className="sticky top-0 z-20 flex h-16 items-center gap-3 border-b border-border/80 bg-background/82 px-4 backdrop-blur-xl lg:px-6">
      <Button variant="ghost" size="icon" className="lg:hidden" onClick={onMenuClick}>
        <Menu className="h-5 w-5" />
        <span className="sr-only">打开导航</span>
      </Button>
      <div className="min-w-0 flex-1">
        <p className="text-xs font-semibold uppercase tracking-[0.24em] text-primary">{ROLE_TEXT[primaryRole]}</p>
        <h1 className="truncate font-display text-xl font-semibold">链镜工作台</h1>
      </div>
      <div className="hidden min-w-64 items-center gap-2 rounded-full border border-border bg-card/70 px-3 py-2 text-sm text-muted-foreground md:flex">
        <Search className="h-4 w-4" />
        <span>全局搜索将在后续模块接入</span>
      </div>
      <NotificationBell />
      <UserMenu user={user} primaryRole={primaryRole} />
    </header>
  );
}
