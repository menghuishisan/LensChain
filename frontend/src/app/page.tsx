// page.tsx
// 根路径入口，未登录时默认跳转到登录页。

import { redirect } from "next/navigation";

/**
 * HomePage 根路径页面。
 */
export default function HomePage() {
  redirect("/login");
}
