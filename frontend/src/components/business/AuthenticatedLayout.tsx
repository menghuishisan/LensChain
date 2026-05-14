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

  // 标准 app-shell 布局（参考 VS Code web / Notion 等 IDE 风格应用）：
  //   - 外层 grid：h-screen + overflow-hidden 锁死视口高度，避免内容驱动整页滚动；
  //   - 主列：flex-col，TopBar 占自然高，main 用 flex-1 + overflow-y-auto 接管纵向滚动；
  //   - 任何"工作区"页面（仿真 / 终端 / IDE）传 h-full 即可拿到 main 的确定高度，
  //     无需再用 calc(100vh-...) 这类脆弱写法；
  //   - 任何普通内容页（列表、详情）保持自然 block 流，溢出由 main 滚动条接管，行为与之前一致。
  return (
    <div className={isSidebarCollapsed ? "h-screen overflow-hidden lg:grid lg:grid-cols-[5rem_1fr]" : "h-screen overflow-hidden lg:grid lg:grid-cols-[18rem_1fr]"}>
      <Sidebar
        navigation={navigation}
        primaryRole={primaryRole}
        isOpen={isSidebarOpen}
        isCollapsed={isSidebarCollapsed}
        onClose={() => setIsSidebarOpen(false)}
      />
      <div className="min-w-0 flex h-full flex-col overflow-hidden">
        <TopBar
          user={user}
          primaryRole={primaryRole}
          onMenuClick={() => setIsSidebarOpen(true)}
          onSidebarToggle={toggleSidebarCollapsed}
          isSidebarCollapsed={isSidebarCollapsed}
          showBackButton={showBackButton}
        />
        <main className="flex-1 min-h-0 overflow-y-auto px-4 py-6 lg:px-8">{children}</main>
      </div>
    </div>
  );
}
