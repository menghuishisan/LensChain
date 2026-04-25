# Frontend User Language Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将前端所有用户可见的研发语义文案替换为面向教师、学生、学校管理员和超级管理员的产品语义文案，同时保持路由、接口和内部命名不变。

**Architecture:** 先修改前端规范文件，建立统一的用户语义约束；再从统一入口配置和全局布局收口导航、页头与系统页文案；最后批量清理高频业务组件中的标题、说明、空状态与提示语。所有修改仅限用户可见文本层，不改变页面结构和调用链。

**Tech Stack:** Next.js App Router、React 18、TypeScript、Tailwind CSS、shadcn/ui 风格组件

---

### Task 1: 更新前端表达规范

**Files:**
- Modify: `frontend/AGENTS.md`

- [ ] **Step 1: 在前端规范中补充用户语义要求**

补充以下约束：

- 前端所有用户可见文案必须采用面向最终用户的产品语义，禁止直接暴露模块编号、内部设计术语、对象建模方式和实现细节。
- 路由、接口、类型、内部变量命名允许保留研发语义，但不得直接出现在用户界面。
- 页面标题、导航、按钮、空状态、错误提示、帮助说明必须以用户目标为中心组织文案。

- [ ] **Step 2: 检查约束与现有规范不冲突**

确认新增内容仅作用于用户可见表达，不要求修改服务层、类型层和路由层命名。

### Task 2: 收口统一入口文案

**Files:**
- Modify: `frontend/src/lib/permissions.ts`
- Modify: `frontend/src/components/business/Sidebar.tsx`
- Modify: `frontend/src/components/business/TopBar.tsx`

- [ ] **Step 1: 调整角色和导航文案**

在 `permissions.ts` 中将高风险研发语义替换为更自然的产品表达，重点修改：

- 超级管理员导航中的“学校租户”“实验资源”“竞赛治理”“系统运维”
- 学生、教师、学校管理员导航中的“实验环境”“实验教学”“通知发送”等表达

- [ ] **Step 2: 调整全局页头文案**

在 `TopBar.tsx` 与 `Sidebar.tsx` 中去除“Console”“后续模块接入”等研发语气，改成稳定的产品文案。

### Task 3: 收口系统与平台级页面文案

**Files:**
- Modify: `frontend/src/app/(super)/super/system/layout.tsx`
- Modify: `frontend/src/components/business/SystemHealthDashboard.tsx`
- Modify: `frontend/src/components/business/SystemConfigPanel.tsx`
- Modify: `frontend/src/components/business/SystemStatisticsPanel.tsx`
- Modify: `frontend/src/components/business/AlertEventPanel.tsx`

- [ ] **Step 1: 修改系统页布局标题与栏目说明**

将系统页的标题、副标题、导航说明改成平台运行和保障语境，避免直接暴露“统一审计、配置分组、模块统计”等设计语感。

- [ ] **Step 2: 修改系统面板内高风险说明**

重点清理：

- 直接出现表名、字段名、分组 key 的文案
- “配置项”“配置分组”“平台统计聚合模块 01 至模块 05”等说明
- 以研发视角描述的数据读取过程

### Task 4: 批量修改高频业务组件文案

**Files:**
- Modify: `frontend/src/components/business/UserManagementPanels.tsx`
- Modify: `frontend/src/components/business/SchoolProfilePanels.tsx`
- Modify: `frontend/src/components/business/SchoolApplicationAdminPanels.tsx`
- Modify: `frontend/src/components/business/NotificationPanels.tsx`
- Modify: `frontend/src/components/business/NotificationTemplateEditor.tsx`
- Modify: `frontend/src/components/business/ExperimentTemplatePanels.tsx`
- Modify: `frontend/src/components/business/ExperimentInstancePanel.tsx`
- Modify: `frontend/src/components/business/CtfPanels.tsx`
- Modify: `frontend/src/components/business/GradePanels.tsx`

- [ ] **Step 1: 修改高风险标题与说明**

对以上高频组件中的标题、副标题、按钮说明和段落说明做统一替换，原则是：

- 不出现模块编号
- 不暴露内部建模对象
- 不用后台治理语气描述用户任务

- [ ] **Step 2: 修改空状态、错误态与成功反馈**

将状态文案从“读取某表/某日志/某接口”改为用户可理解的“暂时无法加载”“当前没有内容”“保存成功”等表达。

### Task 5: 验证与收尾

**Files:**
- Modify: `frontend/AGENTS.md`
- Modify: `frontend/src/...` 受影响前端文件

- [ ] **Step 1: 运行代码检查**

Run: `npm run lint`
Expected: lint 通过；如失败，修复本次改动引入的问题。

- [ ] **Step 2: 运行生产构建**

Run: `npm run build`
Expected: build 通过；如失败，修复本次改动引入的问题。

- [ ] **Step 3: 自查改动范围**

确认本次仅修改用户可见文案与前端规范，没有改变路由、接口、类型定义和内部调用链。
