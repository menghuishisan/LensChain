# LensChain 前端实现审计与补齐清单

> 本文档是前端后续补齐开发的任务源。所有页面、字段、按钮、交互、权限、状态和接口以 `docs/modules/*` 为准；未逐条验收的页面不得标记为完成。

## 审计口径

- 文档来源：每个模块的 `01-功能需求说明.md`、`03-API接口设计.md`、`04-前端页面设计.md`、`05-验收标准.md`。
- 页面覆盖：不仅检查 `page.tsx` 是否存在，还检查页面是否按设计实现字段、按钮、交互、权限、状态和接口调用。
- 完成标准：页面实现与设计一致，service 契约与 API 文档一致，加载/空/错误/无权限状态齐全，相关 `npm run lint`、`npm run test`、`npm run build` 通过。
- 状态说明：`已修复` 表示本轮已改并通过验证；`需完善` 表示静态审计已确认与文档不一致；`待逐页验收` 表示路由存在但还未完成逐项深度验收；`文档决策` 表示必须先更新文档/API 契约。

## 总体结论

| 范围 | 当前状态 | 说明 |
|------|----------|------|
| App Router 页面入口 | 路由存在 | 文档页面清单中的 132 个路由均能映射到 `frontend/src/app` |
| 功能完整度 | 未完成 | 模块03/04/05 多个复杂页面为简化实现，不满足完整验收 |
| 设计一致性 | 部分不一致 | 实验编排器、CTF 攻防页、课程课时页、镜像审核详情等存在明确差距 |
| 遗留占位 | 已清理第一批 | 删除未引用 `ShellPlaceholder`，移除角色首页阶段性开发文案 |

## 已确认缺口与处理状态

| ID | 模块 | 问题 | 影响 | 状态 |
|----|------|------|------|------|
| FE-AUDIT-001 | 全局 | 角色首页显示阶段性开发文案 | 正式页面暴露未完成状态，误导验收 | 已修复 |
| FE-AUDIT-002 | 全局 | `ShellPlaceholder` 未使用且为历史占位组件 | 保留会误导后续审计 | 已删除 |
| FE-AUDIT-003 | 模块04 | `/admin/images/:id/review` 未使用 `:id`，渲染全量镜像列表 | 审核详情页语义错误，不能按单镜像审核 | 已修复 |
| FE-AUDIT-004 | 模块01 | 个人中心设计要求“更换头像”，但模块01 API 只定义 `avatar_url`，未定义头像上传接口 | 不能擅自复用模块04实验文件上传；需先补文档/API | 文档决策 |
| FE-AUDIT-005 | 模块03 | 课时学习页未实现视频播放器、进度上报、上一/下一课时、实验启动真实入口 | 不满足 P-23 与视频进度验收 | 部分修复 |
| FE-AUDIT-006 | 模块03 | 作业作答页缺少 60 秒自动保存和离开确认 | 不满足 P-25 与全局交互规范 | 已修复 |
| FE-AUDIT-007 | 模块03 | 内容管理页缺少拖拽排序、内容类型选择、完整课时编辑入口 | 不满足 P-04/P-05 | 部分修复 |
| FE-AUDIT-016 | 模块03 | 学生课程学习主页缺少文档定义的学习入口和课时状态展示 | 不满足 P-22 | 已修复 |
| FE-AUDIT-017 | 模块03 | 学生成绩页仅显示最终分，不展示各作业成绩和加权总分 | 不满足 P-26 | 已修复 |
| FE-AUDIT-018 | 模块03 | 讨论区/公告页未将置顶内容收口到顶部 | 不满足 P-30/P-32 | 已修复 |
| FE-AUDIT-008 | 模块04 | 实验模板编辑页不是 6 步可视化编排器 | P-21 与 AC-47 至 AC-63 大量不满足 | 需完善 |
| FE-AUDIT-009 | 模块04 | 远程协助页/手动评分页复用实例面板，未按 P-25/P-26 专门布局实现 | 教师监控、只读终端、评分流程体验不完整 | 需完善 |
| FE-AUDIT-010 | 模块04 | SimEngine 面板未完整实现交互 schema、联动组视觉标识、三种时间控制模式 | 不满足 AC-29 至 AC-45 | 需完善 |
| FE-AUDIT-011 | 模块05 | 攻防赛攻击/防守页使用手填 ID，缺少目标队伍/漏洞选择与左右合约对比 | 不满足 P-25/P-26/P-27 | 需完善 |
| FE-AUDIT-012 | 模块05 | 题目编辑页合约、断言、附件、链配置为简化实现 | 不满足 P-11 完整创建/编辑体验 | 需完善 |
| FE-AUDIT-013 | 模块05 | CTF 竞赛创建页配置项未完全覆盖攻防 Token、回合、题目发布校验、编辑限制 | 不满足 P-02 与 AC-01 至 AC-06 | 需完善 |
| FE-AUDIT-014 | 模块07 | 通知“前往查看”跳转需逐事件核对目标路由 | 跨模块联动验收可能失败 | 待逐页验收 |
| FE-AUDIT-015 | 模块08 | 配置更新后“前端标题立即更新”需联调确认 | AC-03 涉及全局元信息刷新 | 待逐页验收 |

## 全量页面矩阵

### 模块01 用户与认证

| 页面 | 路由 | 当前判断 | 后续动作 |
|------|------|----------|----------|
| P-01 登录页 | `/login` | 待逐页验收 | 核对登录失败次数、首次登录、SSO 入口、错误提示 |
| P-02 SSO学校选择页 | `/sso` | 待逐页验收 | 核对 SSO 学校列表、搜索、302 跳转 |
| P-02A SSO回调处理页 | `/auth/sso/callback` | 待逐页验收 | 核对成功/失败/未导入错误态 |
| P-03 强制修改密码页 | `/change-password` | 待逐页验收 | 核对 temp_token、密码强度、确认密码 |
| P-04 用户管理列表页 | `/admin/users` | 待逐页验收 | 核对筛选、状态变更、批量操作、权限边界 |
| P-05 用户详情页 | `/admin/users/:id` | 待逐页验收 | 核对详情字段、重置密码、状态操作 |
| P-06 用户创建/编辑页 | `/admin/users/create` | 待逐页验收 | 核对角色、学校、初始密码、校管创建限制 |
| P-06A 用户编辑页 | `/admin/users/:id/edit` | 待逐页验收 | 核对不可编辑字段和学籍变更 |
| P-07 用户导入页 | `/admin/users/import` | 待逐页验收 | 核对模板下载、文件类型、上传预览 |
| P-08 导入预览与确认页 | `/admin/users/import/preview` | 待逐页验收 | 核对冲突处理、失败明细下载 |
| P-09 个人中心页 | `/profile` | 文档决策 | 头像上传接口缺失，需先补文档或确认 avatar_url 口径 |
| P-10 修改密码页 | `/profile/password` | 待逐页验收 | 核对旧密码、新密码规则、成功后处理 |
| P-11 安全策略配置页 | `/admin/security` | 待逐页验收 | 核对最大失败次数、锁定时长、联动登录 |
| P-12 登录日志页 | `/admin/logs/login` | 待逐页验收 | 核对校管/超管数据范围和筛选 |
| P-13 操作日志页 | `/admin/logs/operation` | 待逐页验收 | 核对操作类型、目标对象、详情展示 |

### 模块02 学校与租户管理

| 页面 | 路由 | 当前判断 | 后续动作 |
|------|------|----------|----------|
| P-01 入驻申请页 | `/apply` | 待逐页验收 | 核对字段校验、重复申请、成功编号 |
| P-02 申请状态查询页 | `/apply/query` | 待逐页验收 | 核对短信验证码、重新申请预填 |
| P-03 入驻申请管理列表页 | `/admin/school-applications` | 待逐页验收 | 核对状态筛选、搜索、待审高亮 |
| P-04 入驻申请审核详情页 | `/admin/school-applications/:id` | 待逐页验收 | 核对通过、拒绝、授权有效期、短信结果 |
| P-05 学校管理列表页 | `/admin/schools` | 待逐页验收 | 核对冻结/解冻/注销/恢复 |
| P-06 学校详情/编辑页 | `/admin/schools/:id` | 待逐页验收 | 核对学校信息、联系人、授权期 |
| P-07 创建学校页 | `/admin/schools/create` | 待逐页验收 | 核对自动创建首个校管账号 |
| P-08 本校信息管理页 | `/school/profile` | 待逐页验收 | 核对校管可编辑字段 |
| P-09 SSO配置页 | `/school/sso-config` | 待逐页验收 | 核对 CAS/OAuth2、测试连接、启用禁用 |
| P-10 授权状态页 | `/school/license` | 待逐页验收 | 核对到期、缓冲期、冻结提示 |

### 模块03 课程与教学

| 页面 | 路由 | 当前判断 | 后续动作 |
|------|------|----------|----------|
| P-01 教师课程列表页 | `/teacher/courses` | 待逐页验收 | 核对筛选、状态、创建入口 |
| P-02 创建/编辑课程页 | `/teacher/courses/create` | 待逐页验收 | 核对学分、学期、封面、时间 |
| P-03 课程管理主页 | `/teacher/courses/:id` | 待逐页验收 | 核对概览卡、最近动态、快捷入口 |
| P-04 内容管理页 | `/teacher/courses/:id/content` | 部分修复 | 已补课时标题/内容类型/预计分钟创建表单；章节编辑、实验关联展示细节仍待完善 |
| P-05 课时编辑页 | `/teacher/lessons/:id/edit` | 部分修复 | 已补按内容类型字段显隐；附件与保存链路可用，页面布局仍可继续打磨 |
| P-06 作业管理页 | `/teacher/courses/:id/assignments` | 待逐页验收 | 核对发布、删除、状态限制 |
| P-07 创建/编辑作业页 | `/teacher/assignments/:id/edit` | 部分修复 | 已补多次提交、迟交扣分、按题型显示选项/答案/参考答案/判题配置 |
| P-08 作业批改页 | `/teacher/submissions/:id/grade` | 待逐页验收 | 核对逐题评分、总评、锁定状态 |
| P-09 学生管理页 | `/teacher/courses/:id/students` | 待逐页验收 | 核对单个/批量添加、移除、进度 |
| P-10 成绩管理页 | `/teacher/courses/:id/grades` | 待逐页验收 | 核对权重配置、手动调整、导出 |
| P-11 课程统计页 | `/teacher/courses/:id/statistics` | 待逐页验收 | 核对概览、作业统计、导出报告 |
| P-12 课程设置页 | `/teacher/courses/:id/settings` | 待逐页验收 | 核对生命周期、共享、邀请码、课程表 |
| P-20 我的课程列表页 | `/student/courses` | 待逐页验收 | 核对课程状态和学习进度 |
| P-21 加入课程页 | `/student/courses/join` | 待逐页验收 | 核对邀请码格式和错误态 |
| P-22 课程学习主页 | `/student/courses/:id` | 部分修复 | 已补内容/作业/讨论/公告/成绩入口和课时状态展示；页面布局仍需继续贴合设计稿 |
| P-23 课时学习页 | `/student/lessons/:id` | 部分修复 | 已补 30 秒节流上报、离开页面补报、实验启动跳转、上一/下一课时导航；完整播放器布局仍待完善 |
| P-24 作业列表页 | `/student/courses/:id/assignments` | 待逐页验收 | 核对可提交/已截止/成绩状态 |
| P-25 作业作答页 | `/student/assignments/:id` | 部分修复 | 已补 60 秒自动保存、localStorage 草稿和离开确认；题型完整渲染仍待完善 |
| P-26 我的成绩页 | `/student/courses/:id/grades` | 部分修复 | 已补各作业成绩、加权总分和调整标识；细节样式与权重说明仍待完善 |
| P-27 我的课程表页 | `/student/schedule` | 部分修复 | 已补周视图分组与课程跳转；视觉布局仍可继续贴合设计稿 |
| P-30 课程讨论区 | `/courses/:id/discussions` | 部分修复 | 已补置顶分区与分页；详情交互与权限仍待逐页验收 |
| P-31 帖子详情页 | `/discussions/:id` | 待逐页验收 | 核对回复、删除、点赞 |
| P-32 课程公告页 | `/courses/:id/announcements` | 部分修复 | 已补置顶分区与分页；教师/学生视角差异仍待逐页验收 |
| P-33 共享课程库 | `/shared-courses` | 待逐页验收 | 核对克隆、筛选、详情 |

### 模块04 实验环境

| 页面 | 路由 | 当前判断 | 后续动作 |
|------|------|----------|----------|
| P-01 镜像仓库管理页 | `/admin/images` | 待逐页验收 | 核对分类、状态、审核入口 |
| P-02 镜像详情/编辑页 | `/admin/images/:id` | 待逐页验收 | 核对版本、配置模板、文档侧栏 |
| P-03 镜像审核页 | `/admin/images/:id/review` | 已修复 | 已按 `:id` 加载单镜像审核面板，仍需联调审核结果 |
| P-04 仿真场景库管理页 | `/admin/sim-scenarios` | 待逐页验收 | 核对审核、上传、预拉取 |
| P-05 全局资源监控页 | `/admin/resource-monitor` | 待逐页验收 | 核对集群、学校资源、异常实例 |
| P-06 学校资源配额管理页 | `/admin/resource-quotas` | 待逐页验收 | 核对学校/课程级配额 |
| P-07 全平台实验实例管理页 | `/admin/experiment-instances` | 待逐页验收 | 核对强制回收、筛选、详情 |
| P-08 K8s集群状态页 | `/admin/k8s-cluster` | 待逐页验收 | 核对节点、命名空间、Pod 状态 |
| P-09 镜像预拉取状态页 | `/admin/image-pull-status` | 待逐页验收 | 核对节点维度、手动触发 |
| P-10 本校资源配额查看页 | `/school/resource-quota` | 待逐页验收 | 核对本校限额与课程分配 |
| P-11 本校镜像管理页 | `/school/images` | 待逐页验收 | 核对本校镜像只读范围 |
| P-12 本校实验监控页 | `/school/experiment-monitor` | 待逐页验收 | 核对本校运行实例和异常 |
| P-20 实验模板列表页 | `/teacher/experiment-templates` | 待逐页验收 | 核对筛选、状态、共享、克隆 |
| P-21 实验模板创建/编辑页 | `/teacher/experiment-templates/create` | 需完善 | 6 步可视化编排器、条件步骤、5 层验证 |
| P-22 实验模板详情页 | `/teacher/experiment-templates/:id` | 需完善 | 与 P-21 同编辑器能力 |
| P-23 多人实验分组管理页 | `/teacher/experiment-groups` | 待逐页验收 | 核对手动/自选/随机分组 |
| P-24 学生实验监控面板 | `/teacher/courses/:id/experiment-monitor` | 待逐页验收 | 核对实时状态、协助、评分、回收 |
| P-25 远程协助页 | `/teacher/experiment-instances/:id/assist` | 需完善 | 专门只读终端、检查点、操作历史、指导消息 |
| P-26 实验手动评分页 | `/teacher/experiment-instances/:id/grade` | 需完善 | 专门评分页、报告、评分项、总评 |
| P-27 实验统计页 | `/teacher/courses/:id/experiment-statistics` | 待逐页验收 | 核对模板维度统计 |
| P-28 自定义镜像上传页 | `/teacher/images/upload` | 待逐页验收 | 核对配置模板、文档、审核状态 |
| P-29 自定义仿真场景上传页 | `/teacher/sim-scenarios/upload` | 待逐页验收 | 核对场景包、交互 schema |
| P-30 共享实验库页 | `/teacher/shared-experiment-templates` | 待逐页验收 | 核对浏览、克隆 |
| P-40 实验环境列表页 | `/student/experiment-instances` | 待逐页验收 | 核对状态、继续/结果/历史 |
| P-41 实验操作主页 | `/student/experiment-instances/:id` | 需完善 | 多面板、终端/IDE/SimEngine、心跳和超时 |
| P-42 实验启动/排队页 | `/student/experiments/:template_id/launch` | 待逐页验收 | 核对排队位置、资源不足 |
| P-43 多人实验分组页 | `/student/experiment-groups/:id` | 待逐页验收 | 核对组内通信、终端只读 |
| P-44 实验结果查看页 | `/student/experiment-instances/:id/result` | 待逐页验收 | 核对自动/手动评分、报告 |
| P-45 实验报告提交页 | `/student/experiment-instances/:id/report` | 待逐页验收 | 核对 Markdown/PDF/Word、50MB 限制 |
| P-46 操作历史查看页 | `/student/experiment-instances/:id/history` | 待逐页验收 | 核对终端命令、生命周期、检查点 |

### 模块05 CTF竞赛

| 页面 | 路由 | 当前判断 | 后续动作 |
|------|------|----------|----------|
| P-01 竞赛管理列表页 | `/admin/ctf/competitions` | 待逐页验收 | 核对筛选、发布、编辑、监控 |
| P-02 竞赛创建/编辑页 | `/admin/ctf/competitions/create` | 需完善 | 完整赛制配置、题目配置、确认发布 |
| P-03 竞赛监控面板 | `/admin/ctf/competitions/:id/monitor` | 待逐页验收 | 核对资源、提交、公告、强制终止 |
| P-04 题目审核列表页 | `/admin/ctf/challenge-reviews` | 待逐页验收 | 核对待审题列表 |
| P-05 题目审核详情页 | `/admin/ctf/challenge-reviews/:id` | 待逐页验收 | 核对合约、断言、预验证和审核拒绝原因 |
| P-06 CTF资源配额管理页 | `/admin/ctf/resource-quotas` | 待逐页验收 | 核对竞赛资源配额 |
| P-07 全平台竞赛概览页 | `/admin/ctf/overview` | 待逐页验收 | 核对运行竞赛和告警 |
| P-10 题目管理列表页 | `/teacher/ctf/challenges` | 待逐页验收 | 核对状态、验证、提交审核 |
| P-11 题目创建/编辑页 | `/teacher/ctf/challenges/create` | 需完善 | 合约/断言/附件/链配置/提交审核 |
| P-12 漏洞转化工具页 | `/teacher/ctf/challenges/import` | 待逐页验收 | 核对 SWC、模板、外部源 A/B/C 分级 |
| P-13 题目预验证页 | `/teacher/ctf/challenges/:id/verify` | 待逐页验收 | 核对六步验证、断言结果、失败原因 |
| P-14 模板库浏览页 | `/teacher/ctf/templates` | 待逐页验收 | 核对参数弹窗、模板生成 |
| P-20 竞赛大厅页 | `/ctf` | 待逐页验收 | 核对报名中/进行中/已结束 |
| P-21 竞赛详情/报名页 | `/ctf/:id` | 待逐页验收 | 核对个人赛/组队赛报名 |
| P-22 团队管理页 | `/ctf/:id/team` | 待逐页验收 | 核对队长/队员权限、锁定状态 |
| P-23 解题赛主页 | `/ctf/:id/jeopardy` | 待逐页验收 | 核对题目列表、环境、排行榜 |
| P-24 题目详情/解题页 | `/ctf/:id/jeopardy/:cid` | 待逐页验收 | 核对环境启动、Flag/链上提交、限流 |
| P-25 攻防赛主页 | `/ctf/:id/attack-defense` | 需完善 | 回合状态、战场总览、Token 流水 |
| P-26 攻防赛攻击页 | `/ctf/:id/attack-defense/attack` | 需完善 | 目标队伍和漏洞选择，不再手填 ID |
| P-27 攻防赛防守页 | `/ctf/:id/attack-defense/defense` | 需完善 | 原始合约只读 + 补丁合约编辑 |
| P-28 排行榜页 | `/ctf/:id/leaderboard` | 待逐页验收 | 核对冻结、历史快照、赛制差异 |
| P-29 竞赛结果页 | `/ctf/:id/results` | 待逐页验收 | 核对最终排名、题目统计 |

### 模块06 评测与成绩

| 页面 | 路由 | 当前判断 | 后续动作 |
|------|------|----------|----------|
| P-01 学期管理 | `/admin/grades/semesters` | 待逐页验收 | 核对当前学期、删除限制 |
| P-02 等级映射配置 | `/admin/grades/level-configs` | 待逐页验收 | 核对区间合法性和默认重置 |
| P-03 成绩审核列表 | `/admin/grades/reviews` | 待逐页验收 | 核对审核/驳回/解锁 |
| P-04 成绩审核详情 | `/admin/grades/reviews/:id` | 待逐页验收 | 核对学生明细、调整原因 |
| P-05 学业预警管理 | `/admin/grades/warnings` | 待逐页验收 | 核对处理记录 |
| P-06 预警配置 | `/admin/grades/warning-configs` | 待逐页验收 | 核对 GPA/挂科阈值 |
| P-07 全校成绩分析 | `/admin/grades/analytics` | 待逐页验收 | 核对趋势、分布、导出 |
| P-08 平台成绩总览 | `/super/grades/analytics` | 待逐页验收 | 核对跨学校统计 |
| P-10 成绩审核提交 | `/teacher/grades/reviews` | 待逐页验收 | 核对课程学分/学期前置校验 |
| P-11 申诉处理列表 | `/teacher/grades/appeals` | 待逐页验收 | 核对同意/驳回 |
| P-12 课程成绩分析 | `/teacher/grades/analytics/:courseId` | 待逐页验收 | 核对课程维度 |
| P-20 我的成绩 | `/student/grades` | 待逐页验收 | 核对学期成绩 |
| P-21 GPA总览 | `/student/grades/gpa` | 待逐页验收 | 核对 GPA 趋势 |
| P-22 成绩申诉 | `/student/grades/appeals` | 待逐页验收 | 核对可申诉范围 |
| P-23 成绩单下载 | `/student/grades/transcripts` | 待逐页验收 | 核对生成和下载 |

### 模块07 通知与消息

| 页面 | 路由 | 当前判断 | 后续动作 |
|------|------|----------|----------|
| P-01 系统公告管理 | `/admin/notifications/announcements` | 待逐页验收 | 核对创建、发布、下架、定时发布 |
| P-02 公告编辑 | `/admin/notifications/announcements/:id/edit` | 待逐页验收 | 核对定时发布和状态限制 |
| P-03 消息模板管理 | `/admin/notifications/templates` | 待逐页验收 | 核对变量、预览、启用状态 |
| P-04 消息统计 | `/admin/notifications/statistics` | 待逐页验收 | 核对发送量、已读率 |
| P-05 发送通知 | `/admin/notifications/send` | 待逐页验收 | 核对学校/课程/用户目标和权限 |
| P-10 消息中心 | `/notifications` | 待逐页验收 | 核对系统公告置顶、筛选、批量已读 |
| P-11 消息详情 | `/notifications/:id` | 待逐页验收 | 核对自动已读和“前往查看” |
| P-12 通知偏好设置 | `/notifications/preferences` | 待逐页验收 | 核对分类偏好立即生效 |

### 模块08 系统管理与监控

| 页面 | 路由 | 当前判断 | 后续动作 |
|------|------|----------|----------|
| P-01 运维仪表盘 | `/super/system/dashboard` | 待逐页验收 | 核对 30 秒刷新、健康状态、最近告警 |
| P-02 统一审计中心 | `/super/system/audit` | 待逐页验收 | 核对三类日志聚合、导出限制 |
| P-03 全局配置管理 | `/super/system/configs` | 待逐页验收 | 核对敏感脱敏、配置更新、标题刷新 |
| P-04 配置变更记录 | `/super/system/configs/change-logs` | 待逐页验收 | 核对变更前后值和筛选 |
| P-05 告警规则管理 | `/super/system/alert-rules` | 待逐页验收 | 核对阈值/事件/服务规则 |
| P-06 告警事件列表 | `/super/system/alert-events` | 待逐页验收 | 核对处理/忽略和 60 秒刷新 |
| P-07 平台使用统计 | `/super/system/statistics` | 待逐页验收 | 核对趋势周期、学校排行 |
| P-08 数据备份管理 | `/super/system/backups` | 待逐页验收 | 核对手动备份、自动策略、下载 |

## 修复顺序

1. 修复全局占位和路由语义错误：FE-AUDIT-001、FE-AUDIT-002、FE-AUDIT-003。
2. 模块03：先修课时学习、作业自动保存、内容管理，再修讨论/公告/成绩统计。
3. 模块04：重构 P-21/P-22 编排器，再修 P-25/P-26 和 P-41 SimEngine 操作页。
4. 模块05：先修竞赛创建和题目编辑，再修攻防赛 P-25/P-26/P-27。
5. 模块01/02/06/07/08：按页面矩阵逐页验收补齐。

## 验证记录

| 时间 | 命令 | 结果 | 说明 |
|------|------|------|------|
| 2026-04-23 | `npm run lint` | 通过 | 第一批修复前后均通过 |
| 2026-04-23 | `npm run test` | 通过 | 第一批修复后 21 个测试文件、51 个测试 |
| 2026-04-23 | `npm run build` | 通过 | 第一批修复后 102 个 App Router 页面 |
| 2026-04-23 | `npm run test -- src/hooks/useAssignmentAutosave.test.tsx` | 通过 | 模块03作业自动保存 hook，3 个测试 |
| 2026-04-23 | `npm run test -- src/hooks/useLessonVideoProgress.test.ts` | 通过 | 模块03视频课时进度工具，3 个测试 |
| 2026-04-23 | `npm run test -- src/lib/course-navigation.test.ts` | 通过 | 模块03课时前后导航工具，2 个测试 |
| 2026-04-23 | `npm run test -- src/components/business/CourseContentManagerPanel.test.tsx` | 通过 | 模块03内容管理页课时创建表单，1 个测试 |
| 2026-04-23 | `npm run test -- src/components/business/StudentCourseHomePanel.test.tsx` | 通过 | 模块03学生课程主页入口与课时状态，1 个测试 |
| 2026-04-23 | `npm run test -- src/components/business/AssignmentEditor.test.tsx` | 通过 | 模块03作业编辑器回填、迟交配置与题型配置，3 个测试 |
| 2026-04-23 | `npm run test -- src/components/business/LessonContentEditor.test.tsx` | 通过 | 模块03课时编辑器内容类型字段显隐，1 个测试 |
| 2026-04-23 | `npm run test -- src/components/business/DiscussionAnnouncementPanels.test.tsx` | 通过 | 模块03讨论区与公告置顶分区，2 个测试 |
| 2026-04-23 | `npm run test -- src/components/business/GradePanelStudent.test.tsx` | 通过 | 模块03学生成绩页各作业成绩与加权总分，1 个测试 |
| 2026-04-23 | `npm run test -- src/lib/schedule-grid.test.ts` | 通过 | 模块03课程表周视图分组，1 个测试 |
