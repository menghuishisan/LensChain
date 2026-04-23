// page.tsx
// 学校管理员端首页，展示校管可访问的用户、学校、资源、成绩和通知入口。

import { RoleLanding } from "@/components/business/RoleLanding";

/**
 * AdminHomePage 学校管理员端首页。
 */
export default function AdminHomePage() {
  return <RoleLanding role="school_admin" />;
}
