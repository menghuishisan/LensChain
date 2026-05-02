# 学生端页面入口规划

> 基于 `docs/前端四端页面矩阵与默认落点规范.md` 第五节
> 目的：明确每个页面的入口路径，确保所有页面可达

---

## 一、侧边栏一级入口（5个）

| 导航项 | 路由 | 说明 |
|--------|------|------|
| 我的课程 | `/student/courses` | 课程学习、作业与讨论入口 |
| 我的实验 | `/student/experiment-instances` | 实验入口、报告与学习记录 |
| CTF竞赛 | `/ctf` | 竞赛大厅、战队与排行榜 |
| 成绩中心 | `/student/grades` | 我的成绩、GPA与成绩申诉 |
| 消息中心 | `/notifications` | 站内信、公告与通知偏好（公共页） |

---

## 二、页面入口矩阵（共30个页面）

### 2.1 课程与教学（11个页面）

| 页面 | 路由 | 入口路径 |
|------|------|----------|
| 我的课程列表 | `/student/courses` | **侧边栏直达** |
| 加入课程 | `/student/courses/join` | 课程列表页 → "加入课程"按钮 |
| 课程学习主页 | `/student/courses/:id` | 课程列表 → 点击课程卡片 |
| 课时学习页 | `/student/lessons/:id` | 课程主页 → 章节列表 → 点击课时 |
| 作业列表 | `/student/courses/:id/assignments` | 课程主页 → "作业"Tab |
| 作业作答页 | `/student/assignments/:id` | 作业列表 → 点击作业 |
| 我的成绩（单课程） | `/student/courses/:id/grades` | 课程主页 → "成绩"Tab |
| 我的课程表 | `/student/schedule` | 课程列表页 → "课程表"按钮 |
| 课程讨论区 | `/courses/:id/discussions` | 课程主页 → "讨论"Tab（公共页） |
| 帖子详情 | `/discussions/:id` | 讨论区 → 点击帖子（公共页） |
| 课程公告 | `/courses/:id/announcements` | 课程主页 → "公告"Tab（公共页） |

### 2.2 实验环境（7个页面）

| 页面 | 路由 | 入口路径 |
|------|------|----------|
| 实验环境列表 | `/student/experiment-instances` | **侧边栏直达** |
| 实验操作主页 | `/student/experiment-instances/:id` | 实验列表 → 点击实验 |
| 实验启动/排队 | `/student/experiments/:template_id/launch` | 课程主页 → 实验列表 → "启动实验" |
| 多人实验分组 | `/student/experiment-groups/:id` | 实验列表 → 点击分组实验 |
| 实验结果查看 | `/student/experiment-instances/:id/result` | 实验主页 → "结果"Tab |
| 实验报告提交 | `/student/experiment-instances/:id/report` | 实验主页 → "报告"Tab |
| 操作历史查看 | `/student/experiment-instances/:id/history` | 实验主页 → "历史"Tab |

### 2.3 CTF竞赛（9个页面）

| 页面 | 路由 | 入口路径 |
|------|------|----------|
| 竞赛大厅 | `/ctf` | **侧边栏直达** |
| 竞赛详情/报名 | `/ctf/:id` | 竞赛大厅 → 点击竞赛卡片 |
| 团队管理 | `/ctf/:id/team` | 竞赛详情 → "团队"Tab |
| 解题赛主页 | `/ctf/:id/jeopardy` | 竞赛详情 → "解题赛"Tab |
| 题目详情/解题 | `/ctf/:id/jeopardy/:cid` | 解题赛主页 → 点击题目 |
| 攻防赛主页 | `/ctf/:id/attack-defense` | 竞赛详情 → "攻防赛"Tab |
| 攻防赛攻击页 | `/ctf/:id/attack-defense/attack` | 攻防赛主页 → "攻击"Tab |
| 攻防赛防守页 | `/ctf/:id/attack-defense/defense` | 攻防赛主页 → "防守"Tab |
| 排行榜 | `/ctf/:id/leaderboard` | 竞赛详情 → "排行榜"Tab |
| 竞赛结果 | `/ctf/:id/results` | 竞赛详情 → "结果"Tab（竞赛结束后） |

### 2.4 评测与成绩（4个页面）

| 页面 | 路由 | 入口路径 |
|------|------|----------|
| 我的成绩 | `/student/grades` | **侧边栏直达** |
| GPA总览 | `/student/grades/gpa` | 成绩中心 → "GPA"Tab |
| 成绩申诉 | `/student/grades/appeals` | 成绩中心 → "申诉"Tab |
| 成绩单下载 | `/student/grades/transcripts` | 成绩中心 → "成绩单"Tab |

### 2.5 通知与消息（3个页面，公共页）

| 页面 | 路由 | 入口路径 |
|------|------|----------|
| 消息中心 | `/notifications` | **侧边栏直达**（公共页） |
| 消息详情 | `/notifications/:id` | 消息中心 → 点击消息 |
| 通知偏好设置 | `/notifications/preferences` | 消息中心 → "偏好设置"按钮 |

---

## 三、验收标准

### 3.1 侧边栏验收
- [ ] 侧边栏只有5个一级入口
- [ ] 每个入口的 `href` 与矩阵文档一致
- [ ] 每个入口的 `roles` 只包含 `["student"]`（消息中心除外）

### 3.2 路由验收
- [ ] 所有30个页面的路由文件存在
- [ ] 每个页面的 `PermissionGate` 配置正确
- [ ] 学生端 layout 只允许 `student` 角色

### 3.3 入口可达性验收
- [ ] 从侧边栏可以到达5个一级页面
- [ ] 从一级页面可以通过点击/Tab切换到达所有二级页面
- [ ] 没有孤立页面（无入口可达）

### 3.4 重复路由验收
- [ ] 不存在 `/student/notifications`（应统一用 `/notifications`）
- [ ] 不存在其他与公共页重复的路由

---

## 四、当前问题

### 已发现问题
1. ❌ 重复路由：`/student/notifications` 存在，应删除

### 待验证问题
1. ⚠️ CTF 9个页面是否都已实现
2. ⚠️ 实验环境7个页面的 Tab 切换是否正确
3. ⚠️ 成绩中心4个页面的 Tab 切换是否正确

---

*文档版本：v1.0*
*创建日期：2026-04-29*
