"use client";

// TopBar.tsx
// 已登录主布局顶部栏，提供返回入口、菜单切换、通知入口和用户菜单。

import { Menu, PanelLeftClose, PanelLeftOpen } from "lucide-react";

import { BackButton } from "@/components/business/BackButton";
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
  onSidebarToggle: () => void;
  isSidebarCollapsed: boolean;
  showBackButton: boolean;
}

/**
 * TopBar 已登录主布局顶部栏组件。
 */
export function TopBar({ user, primaryRole, onMenuClick, onSidebarToggle, isSidebarCollapsed, showBackButton }: TopBarProps) {
  return (
    <header className="sticky top-0 z-20 flex h-16 items-center gap-3 border-b border-border/80 bg-background/82 px-4 backdrop-blur-xl lg:px-6">
      <Button variant="ghost" size="icon" className="lg:hidden" onClick={onMenuClick}>
        <Menu className="h-5 w-5" />
        <span className="sr-only">打开导航</span>
      </Button>
      <Button variant="ghost" size="icon" className="hidden lg:inline-flex" onClick={onSidebarToggle}>
        {isSidebarCollapsed ? <PanelLeftOpen className="h-5 w-5" /> : <PanelLeftClose className="h-5 w-5" />}
        <span className="sr-only">{isSidebarCollapsed ? "展开导航" : "收起导航"}</span>
      </Button>
      {showBackButton ? <BackButton fallbackHref="/" className="hidden sm:inline-flex" /> : null}
      <div className="min-w-0 flex-1">
        <p className="text-xs font-semibold uppercase tracking-[0.24em] text-primary">{ROLE_TEXT[primaryRole]}</p>
        <h1 className="truncate font-display text-xl font-semibold">链镜平台</h1>
      </div>
      <NotificationBell />
      <UserMenu user={user} primaryRole={primaryRole} />
    </header>
  );
}
