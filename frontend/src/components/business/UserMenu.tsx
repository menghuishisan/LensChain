"use client";

// UserMenu.tsx
// 顶部用户菜单壳，展示当前用户、角色和个人中心入口。

import { ChevronDown, LockKeyhole, UserRound } from "lucide-react";
import Link from "next/link";
import { useState } from "react";

import { Badge } from "@/components/ui/Badge";
import { Button } from "@/components/ui/Button";
import { useLogoutMutation } from "@/hooks/useAuth";
import { ROLE_TEXT } from "@/lib/permissions";
import type { AuthUser, UserRole } from "@/types/auth";

/**
 * UserMenu 组件属性。
 */
export interface UserMenuProps {
  user: AuthUser | null;
  primaryRole: UserRole;
}

/**
 * UserMenu 顶部用户菜单组件。
 */
export function UserMenu({ user, primaryRole }: UserMenuProps) {
  const [isOpen, setIsOpen] = useState(false);
  const logoutMutation = useLogoutMutation();
  const userName = user?.name ?? "访客预览";
  const schoolName = user?.school_name ?? "链镜平台";

  return (
    <div className="relative">
      <Button variant="outline" className="h-10 rounded-full border-border/80 bg-card/80 px-2 pr-3" onClick={() => setIsOpen((current) => !current)}>
        <span className="flex h-7 w-7 items-center justify-center rounded-full bg-primary/12 text-primary">
          <UserRound className="h-4 w-4" />
        </span>
        <span className="hidden max-w-24 truncate text-sm md:inline">{userName}</span>
        <ChevronDown className="h-4 w-4 text-muted-foreground" />
      </Button>
      {isOpen ? (
        <div className="absolute right-0 top-12 z-40 w-72 rounded-xl border border-border bg-card p-3 text-card-foreground shadow-panel">
          <div className="rounded-lg bg-muted/60 p-3">
            <p className="font-semibold">{userName}</p>
            <p className="mt-1 text-xs text-muted-foreground">{schoolName}</p>
            <Badge className="mt-3" variant="secondary">
              {ROLE_TEXT[primaryRole]}
            </Badge>
          </div>
          <div className="mt-2 grid gap-1">
            <Link className="rounded-lg px-3 py-2 text-sm hover:bg-muted" href="/profile" onClick={() => setIsOpen(false)}>
              个人中心
            </Link>
            <Link className="rounded-lg px-3 py-2 text-sm hover:bg-muted" href="/profile/password" onClick={() => setIsOpen(false)}>
              修改密码
            </Link>
            <button
              type="button"
              className="flex items-center gap-2 rounded-lg px-3 py-2 text-left text-sm text-destructive hover:bg-destructive/8"
              onClick={() => logoutMutation.mutate()}
            >
              <LockKeyhole className="h-4 w-4" />
              退出登录
            </button>
          </div>
        </div>
      ) : null}
    </div>
  );
}
