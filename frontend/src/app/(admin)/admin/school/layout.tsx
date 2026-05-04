"use client";

// layout.tsx
// 模块02学校管理二级布局，统一提供学校管理员校级子页面导航。

import { Building2, KeyRound, Shield, Boxes, Activity, Server } from "lucide-react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import type { ReactNode } from "react";

import { cn } from "@/lib/utils";

const SCHOOL_NAV_ITEMS = [
  {
    href: "/admin/school/profile",
    label: "学校信息",
    icon: Building2,
  },
  {
    href: "/admin/school/sso-config",
    label: "SSO配置",
    icon: KeyRound,
  },
  {
    href: "/admin/school/license",
    label: "授权状态",
    icon: Shield,
  },
  {
    href: "/admin/school/resource-quota",
    label: "资源配额",
    icon: Server,
  },
  {
    href: "/admin/school/images",
    label: "镜像管理",
    icon: Boxes,
  },
  {
    href: "/admin/school/experiment-monitor",
    label: "实验监控",
    icon: Activity,
  },
] as const;

/**
 * AdminSchoolLayout 学校管理二级布局。
 */
export default function AdminSchoolLayout({ children }: { children: ReactNode }) {
  const pathname = usePathname();

  return (
    <div className="space-y-5">
      <div className="flex flex-wrap gap-2">
        {SCHOOL_NAV_ITEMS.map((item) => {
          const Icon = item.icon;
          const active = pathname === item.href || pathname.startsWith(`${item.href}/`);
          return (
            <Link
              key={item.href}
              href={item.href}
              className={cn(
                "inline-flex items-center gap-2 rounded-lg border px-4 py-2 text-sm font-semibold transition",
                active
                  ? "border-primary/25 bg-primary/8 text-primary"
                  : "border-border bg-background text-muted-foreground hover:border-primary/20 hover:bg-muted/35",
              )}
            >
              <Icon className="h-4 w-4" />
              {item.label}
            </Link>
          );
        })}
      </div>
      <div>{children}</div>
    </div>
  );
}
