// page.tsx
// 超级管理员端首页，展示平台级学校、用户、系统、成绩和通知入口。

import { RoleLanding } from "@/components/business/RoleLanding";

/**
 * SuperHomePage 超级管理员端首页。
 */
export default function SuperHomePage() {
  return <RoleLanding role="super_admin" />;
}
