"use client";

// layout.tsx
// 模块05超级管理员CTF二级布局，统一提供CTF子页面导航。

import { Eye, ListChecks, Server, Swords } from "lucide-react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import type { ReactNode } from "react";

import { cn } from "@/lib/utils";

const CTF_NAV_ITEMS = [
  {
    href: "/super/ctf/competitions",
    label: "竞赛管理",
    icon: Swords,
  },
  {
    href: "/super/ctf/challenge-reviews",
    label: "题目审核",
    icon: ListChecks,
  },
  {
    href: "/super/ctf/resource-quotas",
    label: "资源配额",
    icon: Server,
  },
  {
    href: "/super/ctf/overview",
    label: "全平台概览",
    icon: Eye,
  },
] as const;

/**
 * SuperCtfLayout 超级管理员CTF二级布局。
 */
export default function SuperCtfLayout({ children }: { children: ReactNode }) {
  const pathname = usePathname();

  return (
    <div className="space-y-5">
      <div className="flex flex-wrap gap-2">
        {CTF_NAV_ITEMS.map((item) => {
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
