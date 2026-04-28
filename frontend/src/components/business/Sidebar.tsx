"use client";

// Sidebar.tsx
// 已登录主布局侧边栏，根据当前主角色展示文档定义的导航入口。

import {
  Bell,
  BookOpen,
  Boxes,
  Building2,
  ChartArea,
  ChartNoAxesColumn,
  ChartSpline,
  ClipboardCheck,
  FlaskConical,
  Gauge,
  Landmark,
  Network,
  Presentation,
  Send,
  ServerCog,
  ShieldCheck,
  Swords,
  Trophy,
  Users,
  X,
  type LucideIcon,
} from "lucide-react";
import Link from "next/link";
import { usePathname } from "next/navigation";

import { Button } from "@/components/ui/Button";
import { cn } from "@/lib/utils";
import type { NavigationItem } from "@/lib/permissions";
import type { UserRole } from "@/types/auth";

const ICONS: Record<string, LucideIcon> = {
  Bell,
  BookOpen,
  Boxes,
  Building2,
  ChartArea,
  ChartNoAxesColumn,
  ChartSpline,
  ClipboardCheck,
  FlaskConical,
  Gauge,
  Landmark,
  Network,
  Presentation,
  Send,
  ServerCog,
  ShieldCheck,
  Swords,
  Trophy,
  Users,
};

/**
 * Sidebar 组件属性。
 */
export interface SidebarProps {
  navigation: readonly NavigationItem[];
  primaryRole: UserRole;
  isOpen: boolean;
  isCollapsed: boolean;
  onClose: () => void;
}

function isActivePath(pathname: string, href: string) {
  return pathname === href || pathname.startsWith(`${href}/`);
}

/**
 * Sidebar 已登录主布局侧边栏组件。
 */
export function Sidebar({ navigation, isOpen, isCollapsed, onClose }: SidebarProps) {
  const pathname = usePathname();

  return (
    <>
      <div
        className={cn(
          "fixed inset-0 z-30 bg-slate-950/45 backdrop-blur-sm transition lg:hidden",
          isOpen ? "opacity-100" : "pointer-events-none opacity-0",
        )}
        onClick={onClose}
      />
      <aside
        className={cn(
          "fixed inset-y-0 left-0 z-40 flex w-72 flex-col border-r border-border bg-card text-card-foreground transition-[width,transform] lg:static lg:translate-x-0",
          isCollapsed ? "lg:w-20" : "lg:w-72",
          isOpen ? "translate-x-0" : "-translate-x-full",
        )}
      >
        <div className={cn("flex h-16 items-center justify-between border-b border-border", isCollapsed ? "px-4" : "px-5")}>
          <div>
            <p className="font-display text-xl font-semibold text-primary">{isCollapsed ? "链" : "链镜"}</p>
            {!isCollapsed ? <p className="text-xs text-muted-foreground">区块链教学平台</p> : null}
          </div>
          <Button variant="ghost" size="icon" className="lg:hidden" onClick={onClose}>
            <X className="h-5 w-5" />
            <span className="sr-only">关闭导航</span>
          </Button>
        </div>
        <nav className="flex-1 space-y-1.5 overflow-y-auto px-3 py-5">
          {navigation.map((item) => {
            const Icon = ICONS[item.icon] ?? BookOpen;
            const isActive = isActivePath(pathname, item.href);

            return (
              <Link
                key={item.id}
                href={item.href}
                title={isCollapsed ? item.label : undefined}
                className={cn(
                  "group flex rounded-xl px-3 py-2.5 transition",
                  isCollapsed ? "items-center justify-center" : "gap-3",
                  isActive
                    ? "bg-primary/10 text-primary"
                    : "text-foreground/72 hover:bg-muted hover:text-foreground",
                )}
                onClick={onClose}
              >
                <Icon
                  className={cn(
                    "mt-0.5 h-5 w-5 shrink-0",
                    isActive ? "text-primary" : "text-muted-foreground group-hover:text-foreground",
                  )}
                />
                {!isCollapsed ? (
                  <span className="min-w-0">
                    <span className="block text-sm font-semibold">{item.label}</span>
                    <span
                      className={cn(
                        "mt-0.5 block text-xs leading-5",
                        isActive ? "text-primary/72" : "text-muted-foreground",
                      )}
                    >
                      {item.description}
                    </span>
                  </span>
                ) : null}
              </Link>
            );
          })}
        </nav>
      </aside>
    </>
  );
}
