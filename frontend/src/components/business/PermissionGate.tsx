"use client";

// PermissionGate.tsx
// 前端体验层权限门禁，按角色隐藏不可用入口，不替代后端权限校验。

import type { ReactNode } from "react";
import { useEffect, useState } from "react";

import { LoadingState } from "@/components/ui/LoadingState";
import { NoPermissionState } from "@/components/business/NoPermissionState";
import { hasAnyRole } from "@/lib/permissions";
import { useAuth } from "@/hooks/useAuth";
import type { UserRole } from "@/types/auth";

/**
 * PermissionGate 组件属性。
 */
export interface PermissionGateProps {
  allowedRoles: readonly UserRole[];
  children: ReactNode;
}

/**
 * PermissionGate 角色权限门禁组件。
 * 等待 Zustand persist 水合完成后再判断权限，避免首帧误判为无权限。
 */
export function PermissionGate({ allowedRoles, children }: PermissionGateProps) {
  const { user, roles } = useAuth();
  const [hydrated, setHydrated] = useState(false);

  useEffect(() => {
    setHydrated(true);
  }, []);

  if (!hydrated) {
    return <LoadingState variant="spinner" title="正在验证权限" description="" />;
  }

  if (user === null || !hasAnyRole(roles, allowedRoles)) {
    return <NoPermissionState />;
  }

  return children;
}
