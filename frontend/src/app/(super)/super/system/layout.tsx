"use client";

// layout.tsx
// 模块08系统管理二级布局，统一提供超级管理员权限门禁与系统页面导航。

import { BarChart3, BellRing, FileClock, Gauge, HardDriveDownload, ScrollText, Settings2 } from "lucide-react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import type { ReactNode } from "react";

import { PermissionGate } from "@/components/business/PermissionGate";
import { cn } from "@/lib/utils";

const SYSTEM_NAV_ITEMS = [
  {
    href: "/super/system/dashboard",
    label: "运行总览",
    description: "查看平台状态、资源占用和最新提醒",
    icon: Gauge,
  },
  {
    href: "/super/system/audit",
    label: "操作记录",
    description: "查看平台关键操作与处理记录",
    icon: ScrollText,
  },
  {
    href: "/super/system/configs",
    label: "平台设置",
    description: "维护平台规则、默认设置与调整记录",
    icon: Settings2,
  },
  {
    href: "/super/system/alert-rules",
    label: "提醒规则",
    description: "设置状态、事件和服务提醒条件",
    icon: BellRing,
  },
  {
    href: "/super/system/alert-events",
    label: "运行提醒",
    description: "查看提醒详情和后续处理情况",
    icon: FileClock,
  },
  {
    href: "/super/system/statistics",
    label: "数据概览",
    description: "查看平台趋势和学校使用情况",
    icon: BarChart3,
  },
  {
    href: "/super/system/backups",
    label: "数据保障",
    description: "管理备份策略、执行记录与下载",
    icon: HardDriveDownload,
  },
] as const;

function isActivePath(pathname: string, href: string) {
  return pathname === href || pathname.startsWith(`${href}/`);
}

/**
 * SuperSystemLayout 模块08系统管理布局。
 */
export default function SuperSystemLayout({ children }: { children: ReactNode }) {
  const pathname = usePathname();

  return (
    <PermissionGate allowedRoles={["super_admin"]}>
      <div className="mx-auto flex max-w-[1600px] flex-col gap-6">
        <div className="rounded-[2rem] border border-border/70 bg-card/80 p-4 shadow-panel backdrop-blur lg:p-5">
          <div className="flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
            <div>
              <p className="text-xs font-semibold uppercase tracking-[0.24em] text-primary">Platform Operations</p>
              <h1 className="mt-2 font-display text-3xl font-semibold tracking-tight text-foreground">平台运行中心</h1>
              <p className="mt-2 max-w-3xl text-sm leading-7 text-muted-foreground">
                在同一入口查看平台状态、设置、提醒、数据概览与备份保障，便于持续支持学校侧运行。
              </p>
            </div>
          </div>

          <div className="mt-5 grid gap-3 lg:grid-cols-3 2xl:grid-cols-7">
            {SYSTEM_NAV_ITEMS.map((item) => {
              const Icon = item.icon;
              const active = isActivePath(pathname, item.href);

              return (
                <Link
                  key={item.href}
                  href={item.href}
                  className={cn(
                    "rounded-[1.35rem] border px-4 py-4 transition",
                    active
                      ? "border-primary/25 bg-primary/8 shadow-[0_18px_45px_-30px_rgba(13,148,136,0.45)]"
                      : "border-border/70 bg-muted/20 hover:border-primary/20 hover:bg-muted/35",
                  )}
                >
                  <div className="flex items-start gap-3">
                    <div className={cn("rounded-2xl p-3", active ? "bg-primary/12 text-primary" : "bg-card text-muted-foreground")}>
                      <Icon className="h-5 w-5" />
                    </div>
                    <div className="min-w-0">
                      <p className="font-semibold text-foreground">{item.label}</p>
                      <p className="mt-1 text-sm leading-6 text-muted-foreground">{item.description}</p>
                    </div>
                  </div>
                </Link>
              );
            })}
          </div>
        </div>

        <div>{children}</div>
      </div>
    </PermissionGate>
  );
}
