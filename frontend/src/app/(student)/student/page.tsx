// page.tsx
// 学生端首页，展示学生可访问的课程、实验、竞赛、成绩和消息入口。

import { RoleLanding } from "@/components/business/RoleLanding";

/**
 * StudentHomePage 学生端首页。
 */
export default function StudentHomePage() {
  return <RoleLanding role="student" />;
}
