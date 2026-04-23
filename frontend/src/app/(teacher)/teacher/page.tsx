// page.tsx
// 教师端首页，展示教师可访问的课程、实验、CTF、成绩和消息入口。

import { RoleLanding } from "@/components/business/RoleLanding";

/**
 * TeacherHomePage 教师端首页。
 */
export default function TeacherHomePage() {
  return <RoleLanding role="teacher" />;
}
