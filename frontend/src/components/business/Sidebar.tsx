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
  onClose: () => void;
}

function isActivePath(pathname: string, href: string) {
  return pathname === href || pathname.startsWith(`${href}/`);
}

/**
 * Sidebar 已登录主布局侧边栏组件。
 */
export function Sidebar({ navigation, isOpen, onClose }: SidebarProps) {
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
          "fixed inset-y-0 left-0 z-40 flex w-72 flex-col border-r border-white/12 bg-slate-950 text-white transition-transform lg:static lg:translate-x-0",
          isOpen ? "translate-x-0" : "-translate-x-full",
        )}
      >
        <div className="flex h-16 items-center justify-between border-b border-white/10 px-5">
          <div>
            <p className="font-display text-xl font-semibold">链镜</p>
            <p className="text-xs text-white/50">区块链教学平台</p>
          </div>
          <Button variant="ghost" size="icon" className="text-white hover:bg-white/10 lg:hidden" onClick={onClose}>
            <X className="h-5 w-5" />
            <span className="sr-only">关闭导航</span>
          </Button>
        </div>
        <nav className="flex-1 space-y-2 overflow-y-auto px-3 py-5">
          {navigation.map((item) => {
            const Icon = ICONS[item.icon] ?? BookOpen;
            const isActive = isActivePath(pathname, item.href);

            return (
              <Link
                key={item.id}
                href={item.href}
                className={cn(
                  "group flex gap-3 rounded-xl px-3 py-3 transition",
                  isActive ? "bg-white text-slate-950 shadow-glow" : "text-white/72 hover:bg-white/10 hover:text-white",
                )}
                onClick={onClose}
              >
                <Icon className={cn("mt-0.5 h-5 w-5 shrink-0", isActive ? "text-primary" : "text-white/60 group-hover:text-white")} />
                <span>
                  <span className="block text-sm font-semibold">{item.label}</span>
                  <span className={cn("mt-1 block text-xs leading-5", isActive ? "text-slate-600" : "text-white/42")}>{item.description}</span>
                </span>
              </Link>
            );
          })}
        </nav>
      </aside>
    </>
  );
}
