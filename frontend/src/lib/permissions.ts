// permissions.ts
// 前端体验层权限工具与角色导航配置，后端 RBAC 仍是最终权限边界。

import type { UserRole } from "@/types/auth";

/**
 * 角色展示文本。
 */
export const ROLE_TEXT: Record<UserRole, string> = {
  student: "学生",
  teacher: "教师",
  school_admin: "学校管理员",
  super_admin: "超级管理员",
};

/**
 * 侧边栏导航入口配置。
 */
export interface NavigationItem {
  id: string;
  label: string;
  href: string;
  icon: string;
  description: string;
  roles: readonly UserRole[];
}

const ROLE_PRIORITY: readonly UserRole[] = ["super_admin", "school_admin", "teacher", "student"];

/**
 * 角色导航清单。
 * 导航只做前端入口聚合；最终权限仍以后端 RBAC 与租户隔离为准。
 */
export const NAVIGATION_BY_ROLE: Record<UserRole, readonly NavigationItem[]> = {
  student: [
    {
      id: "student-courses",
      label: "我的课程",
      href: "/student/courses",
      icon: "BookOpen",
      description: "课程学习、作业与讨论入口",
      roles: ["student"],
    },
    {
      id: "student-experiments",
      label: "我的实验",
      href: "/student/experiment-instances",
      icon: "FlaskConical",
      description: "实验入口、报告与学习记录",
      roles: ["student"],
    },
    {
      id: "student-ctf",
      label: "CTF竞赛",
      href: "/ctf",
      icon: "Trophy",
      description: "竞赛大厅、战队与排行榜",
      roles: ["student"],
    },
    {
      id: "student-grades",
      label: "成绩中心",
      href: "/student/grades",
      icon: "ChartNoAxesColumn",
      description: "我的成绩、GPA与成绩申诉",
      roles: ["student"],
    },
    {
      id: "student-notifications",
      label: "消息中心",
      href: "/notifications",
      icon: "Bell",
      description: "站内信、公告与通知偏好",
      roles: ["student"],
    },
  ],
  teacher: [
    {
      id: "teacher-courses",
      label: "课程教学",
      href: "/teacher/courses",
      icon: "Presentation",
      description: "课程、章节、作业与学生管理",
      roles: ["teacher"],
    },
    {
      id: "teacher-experiments",
      label: "实验安排",
      href: "/teacher/experiment-templates",
      icon: "Network",
      description: "实验内容、分组安排与课堂观察",
      roles: ["teacher"],
    },
    {
      id: "teacher-ctf",
      label: "竞赛内容",
      href: "/teacher/ctf/challenges",
      icon: "ShieldCheck",
      description: "题目准备、校验与内容复用",
      roles: ["teacher"],
    },
    {
      id: "teacher-grades",
      label: "评测成绩",
      href: "/teacher/grades/reviews",
      icon: "ClipboardCheck",
      description: "成绩提交、申诉处理与课程分析",
      roles: ["teacher"],
    },
    {
      id: "teacher-notifications",
      label: "消息中心",
      href: "/notifications",
      icon: "Bell",
      description: "通知、公告与偏好设置",
      roles: ["teacher"],
    },
  ],
  school_admin: [
    {
      id: "admin-users",
      label: "用户管理",
      href: "/admin/users",
      icon: "Users",
      description: "本校师生、导入、状态与密码管理",
      roles: ["school_admin"],
    },
    {
      id: "admin-school",
      label: "学校设置",
      href: "/admin/school/profile",
      icon: "Landmark",
      description: "本校信息、SSO与授权状态",
      roles: ["school_admin"],
    },
    {
      id: "admin-resource",
      label: "教学资源",
      href: "/admin/school/resource-quota",
      icon: "ServerCog",
      description: "资源安排、实验支持与运行查看",
      roles: ["school_admin"],
    },
    {
      id: "admin-grades",
      label: "成绩管理",
      href: "/admin/grades/semesters",
      icon: "ChartSpline",
      description: "学期、审核、预警与全校分析",
      roles: ["school_admin"],
    },
    {
      id: "admin-notifications",
      label: "通知中心",
      href: "/admin/notifications/send",
      icon: "Send",
      description: "发布校内通知并触达相关人员",
      roles: ["school_admin"],
    },
  ],
  super_admin: [
    {
      id: "super-schools",
      label: "学校服务",
      href: "/super/schools",
      icon: "Building2",
      description: "学校入驻、资料与服务状态",
      roles: ["super_admin"],
    },
    {
      id: "super-users",
      label: "用户管理",
      href: "/super/users",
      icon: "Users",
      description: "全平台用户、安全策略与日志",
      roles: ["super_admin"],
    },
    {
      id: "super-experiments",
      label: "实验支持",
      href: "/super/images",
      icon: "Boxes",
      description: "实验内容、运行资源与环境支持",
      roles: ["super_admin"],
    },
    {
      id: "super-ctf",
      label: "竞赛支持",
      href: "/super/ctf/competitions",
      icon: "Swords",
      description: "竞赛运行、内容审核与资源安排",
      roles: ["super_admin"],
    },
    {
      id: "super-grades",
      label: "成绩总览",
      href: "/super/grades/analytics",
      icon: "ChartArea",
      description: "跨校成绩趋势与平台评测概览",
      roles: ["super_admin"],
    },
    {
      id: "super-notifications",
      label: "通知公告",
      href: "/super/notifications/announcements",
      icon: "Megaphone",
      description: "系统公告、消息模板与推送统计",
      roles: ["super_admin"],
    },
    {
      id: "super-system",
      label: "平台运行",
      href: "/super/system/dashboard",
      icon: "Gauge",
      description: "运行状态、安全提醒、备份与数据概览",
      roles: ["super_admin"],
    },
  ],
};

/**
 * 判断用户是否拥有任一允许角色。
 */
export function hasAnyRole(userRoles: readonly UserRole[], allowedRoles: readonly UserRole[]) {
  return userRoles.some((role) => allowedRoles.includes(role));
}

/**
 * 根据平台权限层级推导当前主角色。
 */
export function getPrimaryRole(userRoles: readonly UserRole[]) {
  return ROLE_PRIORITY.find((role) => userRoles.includes(role)) ?? "student";
}

/**
 * 根据用户角色生成当前侧边栏导航。
 */
export function getNavigationForRoles(userRoles: readonly UserRole[]) {
  const primaryRole = getPrimaryRole(userRoles);
  return NAVIGATION_BY_ROLE[primaryRole];
}

/**
 * 判断当前角色是否可见某个导航入口。
 */
export function canAccessNavigation(item: NavigationItem, userRoles: readonly UserRole[]) {
  return hasAnyRole(userRoles, item.roles);
}
