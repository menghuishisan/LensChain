// RoleLanding.tsx
// 角色首页工作台，展示当前角色可进入的模块导航和按文档组织的业务入口。

import Link from "next/link";
import React from "react";

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/Card";
import { getNavigationForRoles, ROLE_TEXT } from "@/lib/permissions";
import type { UserRole } from "@/types/auth";

/**
 * RoleLanding 组件属性。
 */
export interface RoleLandingProps {
  role: UserRole;
}

/**
 * RoleLanding 角色首页工作台组件。
 */
export function RoleLanding({ role }: RoleLandingProps) {
  const navigation = getNavigationForRoles([role]);
  const roleText = ROLE_TEXT[role];

  return (
    <div className="space-y-6">
      <section className="rounded-[2rem] bg-[radial-gradient(circle_at_top_left,hsl(var(--primary)/0.18),transparent_28rem),linear-gradient(135deg,hsl(var(--card)),hsl(40_76%_95%))] p-6 shadow-panel">
        <p className="text-sm font-semibold text-primary">{roleText}</p>
        <h2 className="mt-3 font-display text-4xl font-semibold tracking-tight">{roleText}工作台</h2>
        <p className="mt-3 max-w-2xl text-sm leading-6 text-muted-foreground">
          根据当前身份进入课程、实验、竞赛、成绩和通知等常用入口，快速开始今天的工作或学习。
        </p>
      </section>

      <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
        {navigation.map((item) => (
          <Link key={item.id} href={item.href}>
            <Card className="h-full transition hover:-translate-y-1 hover:border-primary/40 hover:shadow-glow">
              <CardHeader>
                <CardTitle>{item.label}</CardTitle>
                <CardDescription>{item.description}</CardDescription>
              </CardHeader>
              <CardContent>
                <p className="text-xs font-semibold uppercase tracking-[0.22em] text-primary">{item.href}</p>
              </CardContent>
            </Card>
          </Link>
        ))}
      </div>

      <Card>
        <CardHeader>
          <CardTitle>使用提示</CardTitle>
          <CardDescription>请选择上方入口进入对应页面，系统会根据你的身份展示可用内容。</CardDescription>
        </CardHeader>
        <CardContent>
          <p className="text-sm text-muted-foreground">
            当前账号可见 {navigation.length} 个常用入口。若缺少预期内容，请确认账号角色或联系管理员协助处理。
          </p>
        </CardContent>
      </Card>
    </div>
  );
}
