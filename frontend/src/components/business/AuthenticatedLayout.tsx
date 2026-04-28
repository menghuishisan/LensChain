"use client";

// AuthenticatedLayout.tsx
// 已登录主布局壳，组合顶部栏、侧边栏和角色导航。

import type { ReactNode } from "react";
import { useState } from "react";
import { usePathname } from "next/navigation";

import { Sidebar } from "@/components/business/Sidebar";
import { TopBar } from "@/components/business/TopBar";
import { useAuth } from "@/hooks/useAuth";
import { shouldShowBackButton } from "@/lib/app-navigation";
import { useLayoutStore } from "@/stores/layoutStore";
import type { UserRole } from "@/types/auth";

/**
 * AuthenticatedLayout 组件属性。
 */
export interface AuthenticatedLayoutProps {
  children: ReactNode;
  defaultRole: UserRole;
}

/**
 * AuthenticatedLayout 已登录主布局壳组件。
 */
export function AuthenticatedLayout({ children, defaultRole }: AuthenticatedLayoutProps) {
  const [isSidebarOpen, setIsSidebarOpen] = useState(false);
  const pathname = usePathname();
  const { user, primaryRole, navigation } = useAuth(defaultRole);
  const isSidebarCollapsed = useLayoutStore((state) => state.isSidebarCollapsed);
  const toggleSidebarCollapsed = useLayoutStore((state) => state.toggleSidebarCollapsed);
  const showBackButton = shouldShowBackButton(pathname);

  return (
    <div className={isSidebarCollapsed ? "min-h-screen lg:grid lg:grid-cols-[5rem_1fr]" : "min-h-screen lg:grid lg:grid-cols-[18rem_1fr]"}>
      <Sidebar
        navigation={navigation}
        primaryRole={primaryRole}
        isOpen={isSidebarOpen}
        isCollapsed={isSidebarCollapsed}
        onClose={() => setIsSidebarOpen(false)}
      />
      <div className="min-w-0">
        <TopBar
          user={user}
          primaryRole={primaryRole}
          onMenuClick={() => setIsSidebarOpen(true)}
          onSidebarToggle={toggleSidebarCollapsed}
          isSidebarCollapsed={isSidebarCollapsed}
          showBackButton={showBackButton}
        />
        <main className="px-4 py-6 lg:px-8">{children}</main>
      </div>
    </div>
  );
}
