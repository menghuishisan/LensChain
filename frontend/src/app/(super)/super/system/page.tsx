// page.tsx
// 模块08系统管理入口页，统一重定向到运维仪表盘。

import { redirect } from "next/navigation";

/**
 * SuperSystemIndexPage 系统管理入口页。
 */
export default function SuperSystemIndexPage() {
  redirect("/super/system/dashboard");
}
