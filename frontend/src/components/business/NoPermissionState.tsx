// NoPermissionState.tsx
// 通用无权限状态组件，说明前端仅做体验层权限控制，后端 RBAC 是最终边界。

import { ShieldAlert } from "lucide-react";
import type { ReactNode } from "react";

import { Button } from "@/components/ui/Button";
import { ErrorState } from "@/components/ui/ErrorState";

/**
 * NoPermissionState 组件属性。
 */
export interface NoPermissionStateProps {
  title?: string;
  description?: string;
  action?: ReactNode;
}

/**
 * NoPermissionState 无权限提示组件。
 */
export function NoPermissionState({
  title = "无权访问该页面",
  description = "当前账号暂时没有这个页面的访问权限，请联系管理员确认权限范围。",
  action,
}: NoPermissionStateProps) {
  return (
    <ErrorState
      title={title}
      description={description}
      action={
        action ?? (
          <Button type="button" variant="outline" disabled>
            <ShieldAlert className="h-4 w-4" />
            请联系管理员
          </Button>
        )
      }
    />
  );
}
