// app-navigation.ts
// 统一管理页面级返回入口与导航外壳的展示规则。

const ROOT_PATHS = new Set([
  "/teacher/courses",
  "/teacher/experiment-templates",
  "/teacher/experiment-groups",
  "/teacher/shared-experiment-templates",
  "/teacher/ctf/challenges",
  "/teacher/grades/reviews",
  "/teacher/grades/appeals",
  "/student/courses",
  "/student/experiment-instances",
  "/student/grades",
  "/student/grades/gpa",
  "/student/grades/transcripts",
  "/student/grades/appeals",
  "/student/schedule",
  "/admin/users",
  "/admin/notifications/send",
  "/admin/grades/semesters",
  "/admin/grades/reviews",
  "/admin/grades/analytics",
  "/admin/grades/warnings",
  "/admin/grades/warning-configs",
  "/admin/grades/level-configs",
  "/admin/school/profile",
  "/admin/school/license",
  "/admin/school/sso-config",
  "/admin/school/resource-quota",
  "/admin/school/images",
  "/admin/school/experiment-monitor",
  "/super/users",
  "/super/schools",
  "/super/images",
  "/super/resource-quotas",
  "/super/resource-monitor",
  "/super/experiment-instances",
  "/super/image-pull-status",
  "/super/sim-scenarios",
  "/super/k8s-cluster",
  "/super/security",
  "/super/ctf/competitions",
  "/super/ctf/challenge-reviews",
  "/super/ctf/overview",
  "/super/ctf/resource-quotas",
  "/super/system/dashboard",
  "/super/system/statistics",
  "/super/system/configs",
  "/super/system/configs/change-logs",
  "/super/system/audit",
  "/super/system/alert-rules",
  "/super/system/alert-events",
  "/super/system/backups",
  "/super/grades/analytics",
  "/super/notifications/statistics",
  "/super/notifications/templates",
  "/super/notifications/announcements",
  "/profile",
  "/profile/password",
  "/notifications",
  "/notifications/preferences",
  "/admin/logs/login",
  "/admin/logs/operation",
  "/super/logs/login",
  "/super/logs/operation",
  "/super/school-applications",
  "/shared-courses",
  "/ctf",
]);

const DETAIL_SEGMENTS = new Set([
  "create",
  "edit",
  "preview",
  "import",
  "review",
  "verify",
  "grade",
  "assist",
  "history",
  "result",
  "report",
  "launch",
  "settings",
  "statistics",
  "monitor",
  "students",
  "content",
  "assignments",
  "grades",
  "announcements",
  "discussions",
  "evaluations",
  "team",
  "leaderboard",
  "jeopardy",
  "attack",
  "defense",
]);

/**
 * stripQueryAndHash 去除地址中的查询参数与 hash。
 */
export function stripQueryAndHash(pathname: string) {
  return pathname.split("#")[0]?.split("?")[0] ?? pathname;
}

/**
 * getPathSegments 返回路径片段，忽略首尾空段。
 */
export function getPathSegments(pathname: string) {
  return stripQueryAndHash(pathname)
    .split("/")
    .filter((segment) => segment.length > 0);
}

/**
 * shouldShowBackButton 判断当前页面是否需要展示统一返回上一页入口。
 */
export function shouldShowBackButton(pathname: string) {
  const normalizedPath = stripQueryAndHash(pathname);
  if (ROOT_PATHS.has(normalizedPath)) {
    return false;
  }

  const segments = getPathSegments(normalizedPath);
  if (segments.length === 0) {
    return false;
  }

  if (segments.some((segment) => DETAIL_SEGMENTS.has(segment))) {
    return true;
  }

  return segments.length >= 3;
}

