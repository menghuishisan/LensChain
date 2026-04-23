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
    label: "运维仪表盘",
    description: "健康状态、资源使用和最近告警",
    icon: Gauge,
  },
  {
    href: "/super/system/audit",
    label: "统一审计",
    description: "跨模块日志聚合查询与导出",
    icon: ScrollText,
  },
  {
    href: "/super/system/configs",
    label: "全局配置",
    description: "平台默认值、安全基线和变更记录",
    icon: Settings2,
  },
  {
    href: "/super/system/alert-rules",
    label: "告警规则",
    description: "阈值、事件和服务状态规则",
    icon: BellRing,
  },
  {
    href: "/super/system/alert-events",
    label: "告警事件",
    description: "事件详情、处理和忽略记录",
    icon: FileClock,
  },
  {
    href: "/super/system/statistics",
    label: "平台统计",
    description: "趋势图与学校活跃排行",
    icon: BarChart3,
  },
  {
    href: "/super/system/backups",
    label: "数据备份",
    description: "备份配置、触发、下载与历史",
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
              <p className="text-xs font-semibold uppercase tracking-[0.24em] text-primary">System Management</p>
              <h1 className="mt-2 font-display text-3xl font-semibold tracking-tight text-foreground">系统管理与监控</h1>
              <p className="mt-2 max-w-3xl text-sm leading-7 text-muted-foreground">
                统一审计、全局配置、告警规则、告警事件、运维仪表盘、平台统计与数据备份入口全部收口在同一导航层。
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
