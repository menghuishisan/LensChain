"use client";

// PermissionGate.tsx
// 前端体验层权限门禁，按角色隐藏不可用入口，不替代后端权限校验。

import type { ReactNode } from "react";

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
 */
export function PermissionGate({ allowedRoles, children }: PermissionGateProps) {
  const { user, roles } = useAuth();

  if (user === null || !hasAnyRole(roles, allowedRoles)) {
    return <NoPermissionState />;
  }

  return children;
}
