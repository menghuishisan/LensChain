# 学校管理员端与超级管理员端路由重构方案

> 目标：严格分离学校管理员端和超级管理员端的路由，避免权限混乱

---

## 一、问题分析

### 1.1 当前问题
- 矩阵文档中，学校管理员端和超级管理员端共享部分路由（如 `/admin/users`）
- `(admin)` layout 允许 `["school_admin", "super_admin"]`，导致权限边界不清
- 实际代码中存在大量路由混乱：
  - 超级管理员的页面在 `(admin)/admin/` 下
  - 超级管理员的页面在 `(super)/super/` 下
  - 两者混杂，无法区分

### 1.2 设计原则
1. **职责分离**：学校管理员管理"本校"，超级管理员管理"全平台"
2. **路由独立**：每个端有独立的路由前缀，不共享
3. **权限清晰**：layout 层面就限制角色，页面无需重复检查

---

## 二、路由规范

### 2.1 学校管理员端（19个页面）
**路由前缀**：`/admin/*`  
**Layout 权限**：`["school_admin"]`  
**默认落点**：`/admin/users`

| 功能模块 | 页面 | 路由 |
|---------|------|------|
| 用户管理 | 用户列表 | `/admin/users` |
| 用户管理 | 用户详情 | `/admin/users/:id` |
| 用户管理 | 创建用户 | `/admin/users/create` |
| 用户管理 | 编辑用户 | `/admin/users/:id/edit` |
| 用户管理 | 用户导入 | `/admin/users/import` |
| 用户管理 | 导入预览 | `/admin/users/import/preview` |
| 日志管理 | 登录日志 | `/admin/logs/login` |
| 日志管理 | 操作日志 | `/admin/logs/operation` |
| 学校管理 | 本校信息 | `/admin/school/profile` |
| 学校管理 | SSO配置 | `/admin/school/sso-config` |
| 学校管理 | 授权状态 | `/admin/school/license` |
| 资源管理 | 资源配额 | `/admin/school/resource-quota` |
| 资源管理 | 镜像管理 | `/admin/school/images` |
| 资源管理 | 实验监控 | `/admin/school/experiment-monitor` |
| 成绩管理 | 学期管理 | `/admin/grades/semesters` |
| 成绩管理 | 等级配置 | `/admin/grades/level-configs` |
| 成绩管理 | 成绩审核 | `/admin/grades/reviews` |
| 成绩管理 | 审核详情 | `/admin/grades/reviews/:id` |
| 成绩管理 | 学业预警 | `/admin/grades/warnings` |
| 成绩管理 | 预警配置 | `/admin/grades/warning-configs` |
| 成绩管理 | 成绩分析 | `/admin/grades/analytics` |
| 通知管理 | 发送通知 | `/admin/notifications/send` |

### 2.2 超级管理员端（35个页面）
**路由前缀**：`/super/*`  
**Layout 权限**：`["super_admin"]`  
**默认落点**：`/super/schools`

| 功能模块 | 页面 | 路由 |
|---------|------|------|
| 用户管理 | 用户列表 | `/super/users` |
| 用户管理 | 用户详情 | `/super/users/:id` |
| 用户管理 | 创建用户 | `/super/users/create` |
| 用户管理 | 编辑用户 | `/super/users/:id/edit` |
| 用户管理 | 安全策略 | `/super/security` |
| 日志管理 | 登录日志 | `/super/logs/login` |
| 日志管理 | 操作日志 | `/super/logs/operation` |
| 学校管理 | 入驻申请 | `/super/school-applications` |
| 学校管理 | 申请详情 | `/super/school-applications/:id` |
| 学校管理 | 学校列表 | `/super/schools` |
| 学校管理 | 学校详情 | `/super/schools/:id` |
| 学校管理 | 创建学校 | `/super/schools/create` |
| 镜像管理 | 镜像仓库 | `/super/images` |
| 镜像管理 | 镜像详情 | `/super/images/:id` |
| 镜像管理 | 镜像审核 | `/super/images/:id/review` |
| 镜像管理 | 仿真场景 | `/super/sim-scenarios` |
| 资源管理 | 资源监控 | `/super/resource-monitor` |
| 资源管理 | 资源配额 | `/super/resource-quotas` |
| 资源管理 | 实验实例 | `/super/experiment-instances` |
| 资源管理 | K8s集群 | `/super/k8s-cluster` |
| 资源管理 | 镜像预拉取 | `/super/image-pull-status` |
| CTF管理 | 竞赛列表 | `/super/ctf/competitions` |
| CTF管理 | 创建竞赛 | `/super/ctf/competitions/create` |
| CTF管理 | 竞赛监控 | `/super/ctf/competitions/:id/monitor` |
| CTF管理 | 题目审核 | `/super/ctf/challenge-reviews` |
| CTF管理 | 审核详情 | `/super/ctf/challenge-reviews/:id` |
| CTF管理 | 资源配额 | `/super/ctf/resource-quotas` |
| CTF管理 | 竞赛概览 | `/super/ctf/overview` |
| 成绩管理 | 成绩总览 | `/super/grades/analytics` |
| 通知管理 | 系统公告 | `/super/notifications/announcements` |
| 通知管理 | 公告编辑 | `/super/notifications/announcements/:id/edit` |
| 通知管理 | 消息模板 | `/super/notifications/templates` |
| 通知管理 | 消息统计 | `/super/notifications/statistics` |
| 系统管理 | 运维仪表盘 | `/super/system/dashboard` |
| 系统管理 | 审计中心 | `/super/system/audit` |
| 系统管理 | 全局配置 | `/super/system/configs` |
| 系统管理 | 配置变更 | `/super/system/configs/change-logs` |
| 系统管理 | 告警规则 | `/super/system/alert-rules` |
| 系统管理 | 告警事件 | `/super/system/alert-events` |
| 系统管理 | 使用统计 | `/super/system/statistics` |
| 系统管理 | 数据备份 | `/super/system/backups` |

---

## 三、重构步骤

### 3.1 学校管理员端整改
1. ✅ 修改 `(admin)` layout：只允许 `["school_admin"]`
2. 检查 `(admin)/admin/` 下的所有页面，确保只有学校管理员的19个页面
3. 删除 `(admin)/admin/notifications`（消息中心，应该用公共页）
4. 保留 `(admin)/admin/notifications/send`（发送通知）
5. 将 `/school/` 路由移到 `(admin)/admin/school/` 下统一管理

### 3.2 超级管理员端整改
1. 修改 `(super)` layout：只允许 `["super_admin"]`
2. 将所有超级管理员页面从 `(admin)/admin/` 迁移到 `(super)/super/`
3. 删除 `(super)/super/notifications`（消息中心，应该用公共页）
4. 保留超级管理员的通知管理功能（公告、模板、统计）

### 3.3 侧边栏导航更新
1. 学校管理员侧边栏：只显示本校管理功能
2. 超级管理员侧边栏：只显示平台管理功能
3. 两者完全独立，无交集

---

## 四、验收标准

### 4.1 路由验收
- [ ] `(admin)` 下只有19个学校管理员页面
- [ ] `(super)` 下只有35个超级管理员页面
- [ ] 无路由重复（除公共页外）

### 4.2 权限验收
- [ ] `(admin)` layout 只允许 `["school_admin"]`
- [ ] `(super)` layout 只允许 `["super_admin"]`
- [ ] 学校管理员无法访问 `/super/*`
- [ ] 超级管理员无法访问 `/admin/*`

### 4.3 功能验收
- [ ] 学校管理员登录后默认落点 `/admin/users`
- [ ] 超级管理员登录后默认落点 `/super/schools`
- [ ] 侧边栏导航正确显示
- [ ] 前端构建通过

---

*文档版本：v1.0*
*创建日期：2026-04-30*
