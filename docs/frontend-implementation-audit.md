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
| FE-AUDIT-019 | 模块03 | 课程管理主页、课程设置页、共享课程库仍为简化版实现，缺少贴近设计稿的导航、共享/课表表达与筛选体验 | 不满足 P-03/P-12/P-33 | 已修复 |
| FE-AUDIT-020 | 模块03 | 作业管理页与学生作业列表页仍为简化卡片，缺少发布状态、提交概况、剩余提交次数与最近提交结果等用户语义 | 不满足 P-06/P-24 | 已修复 |
| FE-AUDIT-021 | 模块03 | 成绩管理页、课程统计页和帖子详情页仍为简化实现，缺少权重校验提示、统计分布表达与帖子元信息/回复语义 | 不满足 P-10/P-11/P-31 | 已修复 |
| FE-AUDIT-008 | 模块04 | 实验模板编辑页不是 6 步可视化编排器 | P-21 与 AC-47 至 AC-63 大量不满足 | 已修复 |
| FE-AUDIT-009 | 模块04 | 远程协助页/手动评分页复用实例面板，未按 P-25/P-26 专门布局实现 | 教师监控、只读终端、评分流程体验不完整 | 已修复 |
| FE-AUDIT-010 | 模块04 | SimEngine 面板未完整实现交互 schema、联动组视觉标识、三种时间控制模式 | 不满足 AC-29 至 AC-45 | 已修复 |
| FE-AUDIT-011 | 模块05 | 攻防赛攻击/防守页使用手填 ID，缺少目标队伍/漏洞选择与左右合约对比 | 不满足 P-25/P-26/P-27 | 已修复 |
| FE-AUDIT-012 | 模块05 | 题目编辑页合约、断言、附件、链配置为简化实现 | 不满足 P-11 完整创建/编辑体验 | 已修复 |
| FE-AUDIT-013 | 模块05 | CTF 竞赛创建页配置项未完全覆盖攻防 Token、回合、题目发布校验、编辑限制 | 不满足 P-02 与 AC-01 至 AC-06 | 已修复 |
| FE-AUDIT-014 | 模块07 | 通知"前往查看"跳转需逐事件核对目标路由 | 跨模块联动验收可能失败 | 待逐页验收 |
| FE-AUDIT-015 | 模块08 | 配置更新后"前端标题立即更新"需联调确认 | AC-03 涉及全局元信息刷新 | 待逐页验收 |
| FE-AUDIT-022 | 模块04 | 学生实验主操作页 P-41 中 `desktopUrl` 在 `ExperimentInstancePanel.tsx:100` 硬编码为空字符串，导致设计稿要求的"桌面"Tab 永远不渲染；模块04 API/数据库仅定义统一入口 `access_url`，没有 `desktop_url` 字段，无法在前端单边修复 | 不满足 P-41 桌面工具入口；纯仿真以外的实验类型缺失图形桌面通路 | 文档决策 |
| FE-AUDIT-023 | 模块04 | `WebIDEPanel` 的 `ideUrl` 直接复用 `instance.access_url`（`ExperimentInstancePanel.tsx:99`），但同一 `access_url` 不可能同时承担 IDE 与桌面两个独立服务（不同容器、不同端口） | 真实环境/混合实验在切换 IDE 与终端/桌面 Tab 时会指向同一 URL，与"工具面板（终端/IDE/桌面）"设计不一致 | 文档决策 |
| FE-AUDIT-024 | 模块04 | 学生实验链路中 P-42 启动/排队页、P-43 分组页、P-44 结果页、P-45 报告页、P-46 历史页仍标记"待逐页验收"，未做端到端验证 | 即使 P-41 已实现也不能视为整链完整 | 待逐页验收 |
| FE-AUDIT-025 | 模块05 | 学生 CTF 路由集合（`/ctf`、`/ctf/:id`、`/ctf/:id/team`、`/ctf/:id/jeopardy`、`/ctf/:id/jeopardy/:cid`、`/ctf/:id/attack-defense`、`/ctf/:id/attack-defense/attack`、`/ctf/:id/attack-defense/defense`、`/ctf/:id/leaderboard`、`/ctf/:id/results`）页级结构齐全且均挂接 `CtfHallPanel`/`AttackDefenseRoundPanel`/`CtfChallengePanel` 等真实业务面板，但报名与组队边界、排行榜实时刷新、公告联动、题目环境与提交链路、攻防赛交互与结果链路仍未做端到端验收 | 不能凭页面齐全推断业务完整 | 待逐页验收 |
| FE-AUDIT-026 | 全局 | 删除四端工作台首页 `(student)/student/page.tsx`、`(teacher)/teacher/page.tsx`、`(admin)/admin/page.tsx`、`(super)/super/page.tsx` 与 `RoleLanding.tsx` 后，`/student`、`/teacher`、`/admin`、`/super` 四个根路径变为 404；登录默认落点已改为各端首个业务页（`/student/courses`、`/teacher/courses`、`/admin/users`、`/admin/schools`） | 任何外部链接、书签或代码中残留的四个根路径都将 404；需在后续巡检中确认没有跨模块代码再生成这些路径 | 已修复 |

## 全量页面矩阵

### 模块01 用户与认证

| 页面 | 路由 | 当前判断 | 后续动作 |
|------|------|----------|----------|
| P-01 登录页 | `/login` | 已修复 | 已补手机号密码校验、首次登录跳转、SSO 入口和登录失败提示 |
| P-02 SSO学校选择页 | `/sso` | 已修复 | 已补 SSO 学校搜索、空状态和跳转入口 |
| P-02A SSO回调处理页 | `/auth/sso/callback` | 已修复 | 已补加载态、失败态和账号未开通提示 |
| P-03 强制修改密码页 | `/change-password` | 已修复 | 已补 temp_token 失效处理、密码强度规则和确认密码提示 |
| P-04 用户管理列表页 | `/admin/users` | 已修复 | 已补脱敏手机号、筛选、批量操作和状态操作入口 |
| P-05 用户详情页 | `/admin/users/:id` | 已修复 | 已补详情卡片、重置密码、状态操作、解锁和删除入口 |
| P-06 用户创建/编辑页 | `/admin/users/create` | 已修复 | 已补角色/学校约束、初始密码和字段编辑限制 |
| P-06A 用户编辑页 | `/admin/users/:id/edit` | 已修复 | 已按后端契约保留手机号和角色只读，支持学籍字段修改 |
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
| P-01 入驻申请页 | `/apply` | 已修复 | 已补字段校验、重新申请上下文提示、成功编号反馈和查询入口 |
| P-02 申请状态查询页 | `/apply/query` | 已修复 | 已补短信验证码获取限制、状态提示和重新申请入口 |
| P-03 入驻申请管理列表页 | `/admin/school-applications` | 已修复 | 已补状态标签统计、搜索筛选和审核入口表达 |
| P-04 入驻申请审核详情页 | `/admin/school-applications/:id` | 已修复 | 已补审核记录、授权有效期输入、拒绝原因和审核操作区 |
| P-05 学校管理列表页 | `/admin/schools` | 已修复 | 已补联系人脱敏、有效期设置、冻结/解冻/注销/恢复操作 |
| P-06 学校详情/编辑页 | `/admin/schools/:id` | 已修复 | 已补学校详情编辑表单和联系人/授权信息表达 |
| P-07 创建学校页 | `/admin/schools/create` | 已修复 | 已补创建成功后首个校管账号与短信通知的用户提示 |
| P-08 本校信息管理页 | `/school/profile` | 已修复 | 已补学校名称/编码只读展示和校管可编辑字段表达 |
| P-09 SSO配置页 | `/school/sso-config` | 已修复 | 已补 CAS/OAuth2 切换、测试连接、启停逻辑与测试状态提示 |
| P-10 授权状态页 | `/school/license` | 已修复 | 已补到期提醒、缓冲期/冻结提示和授权周期展示 |

### 模块03 课程与教学

| 页面 | 路由 | 当前判断 | 后续动作 |
|------|------|----------|----------|
| P-01 教师课程列表页 | `/teacher/courses` | 待逐页验收 | 核对筛选、状态、创建入口 |
| P-02 创建/编辑课程页 | `/teacher/courses/create` | 待逐页验收 | 核对学分、学期、封面、时间 |
| P-03 课程管理主页 | `/teacher/courses/:id` | 已修复 | 已补返回课程列表入口、课程导航、概览指标和最近更新摘要 |
| P-04 内容管理页 | `/teacher/courses/:id/content` | 部分修复 | 已补课时标题/内容类型/预计分钟创建表单；章节编辑、实验关联展示细节仍待完善 |
| P-05 课时编辑页 | `/teacher/lessons/:id/edit` | 部分修复 | 已补按内容类型字段显隐；附件与保存链路可用，页面布局仍可继续打磨 |
| P-06 作业管理页 | `/teacher/courses/:id/assignments` | 已修复 | 已补发布/未发布状态、题目数量、提交概况、删除限制与操作入口 |
| P-07 创建/编辑作业页 | `/teacher/assignments/:id/edit` | 部分修复 | 已补多次提交、迟交扣分、按题型显示选项/答案/参考答案/判题配置 |
| P-08 作业批改页 | `/teacher/submissions/:id/grade` | 待逐页验收 | 核对逐题评分、总评、锁定状态 |
| P-09 学生管理页 | `/teacher/courses/:id/students` | 待逐页验收 | 核对单个/批量添加、移除、进度 |
| P-10 成绩管理页 | `/teacher/courses/:id/grades` | 已修复 | 已补权重配置提示、总和校验、成绩汇总与导出入口 |
| P-11 课程统计页 | `/teacher/courses/:id/statistics` | 已修复 | 已补课程概览、学习进度分布、作业分数分布与导出入口 |
| P-12 课程设置页 | `/teacher/courses/:id/settings` | 已修复 | 已补共享/生命周期状态卡、状态限制文案和人类可读课程表展示 |
| P-20 我的课程列表页 | `/student/courses` | 待逐页验收 | 核对课程状态和学习进度 |
| P-21 加入课程页 | `/student/courses/join` | 待逐页验收 | 核对邀请码格式和错误态 |
| P-22 课程学习主页 | `/student/courses/:id` | 部分修复 | 已补内容/作业/讨论/公告/成绩入口和课时状态展示；页面布局仍需继续贴合设计稿 |
| P-23 课时学习页 | `/student/lessons/:id` | 部分修复 | 已补 30 秒节流上报、离开页面补报、实验启动跳转、上一/下一课时导航；完整播放器布局仍待完善 |
| P-24 作业列表页 | `/student/courses/:id/assignments` | 已修复 | 已补截止状态、剩余提交次数、最近一次提交状态与最近得分展示 |
| P-25 作业作答页 | `/student/assignments/:id` | 部分修复 | 已补 60 秒自动保存、localStorage 草稿和离开确认；题型完整渲染仍待完善 |
| P-26 我的成绩页 | `/student/courses/:id/grades` | 部分修复 | 已补各作业成绩、加权总分和调整标识；细节样式与权重说明仍待完善 |
| P-27 我的课程表页 | `/student/schedule` | 部分修复 | 已补周视图分组与课程跳转；视觉布局仍可继续贴合设计稿 |
| P-30 课程讨论区 | `/courses/:id/discussions` | 部分修复 | 已补置顶分区与分页；详情交互与权限仍待逐页验收 |
| P-31 帖子详情页 | `/discussions/:id` | 已修复 | 已补帖子元信息、置顶/点赞状态、回复层次与发布回复入口 |
| P-32 课程公告页 | `/courses/:id/announcements` | 部分修复 | 已补置顶分区与分页；教师/学生视角差异仍待逐页验收 |
| P-33 共享课程库 | `/shared-courses` | 已修复 | 已补标题/类型/难度/主题筛选、空状态和分页入口 |

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
| P-21 实验模板创建/编辑页 | `/teacher/experiment-templates/create` | 已修复 | 已补 6 步向导、条件步骤显隐、镜像编排/仿真场景/检查点与资源发布收口 |
| P-22 实验模板详情页 | `/teacher/experiment-templates/:id` | 已修复 | 与创建页共用同一套 6 步模板编排器能力 |
| P-23 多人实验分组管理页 | `/teacher/experiment-groups` | 待逐页验收 | 核对手动/自选/随机分组 |
| P-24 学生实验监控面板 | `/teacher/courses/:id/experiment-monitor` | 待逐页验收 | 核对实时状态、协助、评分、回收 |
| P-25 远程协助页 | `/teacher/experiment-instances/:id/assist` | 已修复 | 已补只读终端、指导消息、快照/操作历史联动与教师协助模式 |
| P-26 实验手动评分页 | `/teacher/experiment-instances/:id/grade` | 已修复 | 已补评分项视图、实验报告与评分参考、人工评分与总评提交流程 |
| P-27 实验统计页 | `/teacher/courses/:id/experiment-statistics` | 待逐页验收 | 核对模板维度统计 |
| P-28 自定义镜像上传页 | `/teacher/images/upload` | 待逐页验收 | 核对配置模板、文档、审核状态 |
| P-29 自定义仿真场景上传页 | `/teacher/sim-scenarios/upload` | 待逐页验收 | 核对场景包、交互 schema |
| P-30 共享实验库页 | `/teacher/shared-experiment-templates` | 待逐页验收 | 核对浏览、克隆 |
| P-40 实验环境列表页 | `/student/experiment-instances` | 待逐页验收 | 核对状态、继续/结果/历史 |
| P-41 实验操作主页 | `/student/experiment-instances/:id` | 已修复 | 已补学生工作台模式、多标签面板、报告/快照/SimEngine 信息区与实例操作收口 |
| P-42 实验启动/排队页 | `/student/experiments/:template_id/launch` | 待逐页验收 | 核对排队位置、资源不足 |
| P-43 多人实验分组页 | `/student/experiment-groups/:id` | 待逐页验收 | 核对组内通信、终端只读 |
| P-44 实验结果查看页 | `/student/experiment-instances/:id/result` | 待逐页验收 | 核对自动/手动评分、报告 |
| P-45 实验报告提交页 | `/student/experiment-instances/:id/report` | 待逐页验收 | 核对 Markdown/PDF/Word、50MB 限制 |
| P-46 操作历史查看页 | `/student/experiment-instances/:id/history` | 待逐页验收 | 核对终端命令、生命周期、检查点 |

### 模块05 CTF竞赛

| 页面 | 路由 | 当前判断 | 后续动作 |
|------|------|----------|----------|
| P-01 竞赛管理列表页 | `/admin/ctf/competitions` | 待逐页验收 | 核对筛选、发布、编辑、监控 |
| P-02 竞赛创建/编辑页 | `/admin/ctf/competitions/create` | 已修复 | 已补攻防赛 Token/回合配置、题目配置提示与发布确认视图 |
| P-03 竞赛监控面板 | `/admin/ctf/competitions/:id/monitor` | 待逐页验收 | 核对资源、提交、公告、强制终止 |
| P-04 题目审核列表页 | `/admin/ctf/challenge-reviews` | 待逐页验收 | 核对待审题列表 |
| P-05 题目审核详情页 | `/admin/ctf/challenge-reviews/:id` | 待逐页验收 | 核对合约、断言、预验证和审核拒绝原因 |
| P-06 CTF资源配额管理页 | `/admin/ctf/resource-quotas` | 待逐页验收 | 核对竞赛资源配额 |
| P-07 全平台竞赛概览页 | `/admin/ctf/overview` | 待逐页验收 | 核对运行竞赛和告警 |
| P-10 题目管理列表页 | `/teacher/ctf/challenges` | 待逐页验收 | 核对状态、验证、提交审核 |
| P-11 题目创建/编辑页 | `/teacher/ctf/challenges/create` | 已修复 | 已补题目分类/难度/Flag 类型、合约与断言配置及验证审核状态提示 |
| P-12 漏洞转化工具页 | `/teacher/ctf/challenges/import` | 待逐页验收 | 核对 SWC、模板、外部源 A/B/C 分级 |
| P-13 题目预验证页 | `/teacher/ctf/challenges/:id/verify` | 待逐页验收 | 核对六步验证、断言结果、失败原因 |
| P-14 模板库浏览页 | `/teacher/ctf/templates` | 待逐页验收 | 核对参数弹窗、模板生成 |
| P-20 竞赛大厅页 | `/ctf` | 待逐页验收 | 核对报名中/进行中/已结束 |
| P-21 竞赛详情/报名页 | `/ctf/:id` | 待逐页验收 | 核对个人赛/组队赛报名 |
| P-22 团队管理页 | `/ctf/:id/team` | 待逐页验收 | 核对队长/队员权限、锁定状态 |
| P-23 解题赛主页 | `/ctf/:id/jeopardy` | 待逐页验收 | 核对题目列表、环境、排行榜 |
| P-24 题目详情/解题页 | `/ctf/:id/jeopardy/:cid` | 待逐页验收 | 核对环境启动、Flag/链上提交、限流 |
| P-25 攻防赛主页 | `/ctf/:id/attack-defense` | 已修复 | 已补回合状态、目标队伍/漏洞选择、战场总览与实时动态区域 |
| P-26 攻防赛攻击页 | `/ctf/:id/attack-defense/attack` | 已修复 | 已改为目标队伍与漏洞选择式提交流程，不再手填 ID |
| P-27 攻防赛防守页 | `/ctf/:id/attack-defense/defense` | 已修复 | 已补原始合约只读占位与补丁编辑区左右对比布局 |
| P-28 排行榜页 | `/ctf/:id/leaderboard` | 待逐页验收 | 核对冻结、历史快照、赛制差异 |
| P-29 竞赛结果页 | `/ctf/:id/results` | 待逐页验收 | 核对最终排名、题目统计 |

### 模块06 评测与成绩

| 页面 | 路由 | 当前判断 | 后续动作 |
|------|------|----------|----------|
| P-01 学期管理 | `/admin/grades/semesters` | 已修复 | 已补创建学期、当前学期设置提示与删除限制说明 |
| P-02 等级映射配置 | `/admin/grades/level-configs` | 已修复 | 已补区间编辑、区间预览和保存前校验提示 |
| P-03 成绩审核列表 | `/admin/grades/reviews` | 已修复 | 已补审核列表与状态展示入口 |
| P-04 成绩审核详情 | `/admin/grades/reviews/:id` | 已修复 | 已补学生数/平均分/已调整人数、成绩明细和审核操作区 |
| P-05 学业预警管理 | `/admin/grades/warnings` | 已修复 | 已补预警详情、课程明细和处理备注区域 |
| P-06 预警配置 | `/admin/grades/warning-configs` | 已修复 | 已补 GPA/挂科阈值与开关保存能力 |
| P-07 全校成绩分析 | `/admin/grades/analytics` | 已修复 | 已补全校概览、课程排行和 GPA 分布图形表达 |
| P-08 平台成绩总览 | `/super/grades/analytics` | 已修复 | 已补跨学校成绩对比与平台概览数据展示 |
| P-10 成绩审核提交 | `/teacher/grades/reviews` | 已修复 | 已补课程审核提交入口和课程完成状态提示 |
| P-11 申诉处理列表 | `/teacher/grades/appeals` | 已修复 | 已补申诉详情、原成绩、新成绩和处理操作区 |
| P-12 课程成绩分析 | `/teacher/grades/analytics/:courseId` | 已修复 | 已补课程成绩指标和分布图表达 |
| P-20 我的成绩 | `/student/grades` | 已修复 | 已补学习概览、学期成绩表和跨页操作入口 |
| P-21 GPA总览 | `/student/grades/gpa` | 已修复 | 已补累计 GPA、累计学分和趋势视图 |
| P-22 成绩申诉 | `/student/grades/appeals` | 已修复 | 已补申诉提交入口、申诉记录和状态列表 |
| P-23 成绩单下载 | `/student/grades/transcripts` | 已修复 | 已补成绩单生成、列表和下载入口 |

### 模块07 通知与消息

| 页面 | 路由 | 当前判断 | 后续动作 |
|------|------|----------|----------|
| P-01 系统公告管理 | `/admin/notifications/announcements` | 已修复 | 已补创建公告、公告列表、置顶标识和状态筛选表达 |
| P-02 公告编辑 | `/admin/notifications/announcements/:id/edit` | 已修复 | 已补编辑、发布、下架与定时发布字段表达 |
| P-03 消息模板管理 | `/admin/notifications/templates` | 已修复 | 已补模板列表、变量提示、保存和安全预览表达 |
| P-04 消息统计 | `/admin/notifications/statistics` | 已修复 | 已补分类统计、已读率和每日趋势展示 |
| P-05 发送通知 | `/admin/notifications/send` | 已修复 | 已补发送对象语义提示和分类发送表单表达 |
| P-10 消息中心 | `/notifications` | 已修复 | 已补分类Tab、系统公告置顶区、批量已读和来源跳转提示 |
| P-11 消息详情 | `/notifications/:id` | 已修复 | 已补详情阅读、自动已读后的前往查看入口 |
| P-12 通知偏好设置 | `/notifications/preferences` | 已修复 | 已补强制分类说明与偏好保存流程 |

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
| 2026-04-24 | `npm test -- src/components/business/CoursePanels.test.tsx` | 通过 | 模块03第一批页面簇：P-03/P-12/P-33，3 个测试 |
| 2026-04-24 | `npm test -- src/components/business/AssignmentListPanel.test.tsx` | 通过 | 模块03第二批页面簇：P-06/P-24，2 个测试 |
| 2026-04-24 | `npm test -- src/components/business/CourseInsightsPanels.test.tsx` | 通过 | 模块03第三批页面簇：P-10/P-11，2 个测试 |
| 2026-04-24 | `npm test -- src/components/business/DiscussionThread.test.tsx` | 通过 | 模块03第三批页面簇：P-31，1 个测试 |
| 2026-04-24 | `npm run lint` | 通过 | 前端 lint 无警告无错误 |
| 2026-04-24 | `npm run build` | 通过 | Next.js 14 生产构建通过，102 个 App Router 页面生成成功 |
| 2026-04-24 | `npm run lint` | 通过 | 模块04整包收口后再次验证通过 |
| 2026-04-24 | `npm run build` | 通过 | 模块04整包收口后再次完成 Next.js 14 生产构建 |
| 2026-04-24 | `npm run lint` | 通过 | 模块05整包收口后再次验证通过 |
| 2026-04-24 | `npm run build` | 通过 | 模块05整包收口后再次完成 Next.js 14 生产构建 |
| 2026-04-24 | `npm run lint` | 通过 | 模块01整包收口后再次验证通过 |
| 2026-04-24 | `npm run build` | 通过 | 模块01整包收口后再次完成 Next.js 14 生产构建 |
| 2026-04-24 | `npm run lint` | 通过 | 模块02整包收口后再次验证通过 |
| 2026-04-24 | `npm run build` | 通过 | 模块02整包收口后再次完成 Next.js 14 生产构建 |
| 2026-04-24 | `npm run lint` | 通过 | 模块06整包收口后再次验证通过 |
| 2026-04-24 | `npm run build` | 通过 | 模块06整包收口后再次完成 Next.js 14 生产构建 |
| 2026-04-25 | `npm run lint` | 通过 | 模块07整包收口后再次验证通过 |
| 2026-04-25 | `npm run build` | 通过 | 模块07整包收口后再次完成 Next.js 14 生产构建 |
| 2026-04-29 | `npm run lint` | 通过 | 角色边界整改后整包验证；修复 `ExperimentInstancePanel` 中两个 `react-hooks/rules-of-hooks` 错误（pre-existing），剩余仅 hooks 依赖项警告 |
| 2026-04-29 | `npm run build` | 通过 | 角色边界整改后整包构建；同步修复 sim-engine/renderers 由 `.js` 扩展引用 `.ts` 源导致的 webpack 模块解析失败（pre-existing），方法是在 `next.config.js` 配置 `resolve.extensionAlias`，未改动 renderers 源码 |
| 2026-04-29 | FE-AUDIT-022/023 修复 | 通过 | 删除 `access_url` 字段，新增 `tools[]` 数组（DB、API、后端 DTO、前端类型、前端组件全链路修改）；`ExperimentInstancePanel` 改为从 `tools[]` 动态渲染工具 Tab（terminal/ide/desktop/explorer/monitor），每个 Tab 的 iframe 直接使用后端签发的 `proxy_url`，不做前端拼接 |

## 角色边界整改补记（2026-04-29）

### 四端工作台首页删除

源码位置：`frontend/src/hooks/useAuth.ts`、`frontend/src/lib/app-navigation.ts`、`frontend/src/components/business/ProfilePanel.tsx`。

- `getAuthHomePath` 从落点 `/student`、`/teacher`、`/admin`、`/super` 改为各端首个业务页 `/student/courses`、`/teacher/courses`、`/admin/users`、`/admin/schools`，AuthPanels 登录成功路径同步改变（`useAuth.ts:159`）。
- 删除 `(student)/student/page.tsx`、`(teacher)/teacher/page.tsx`、`(admin)/admin/page.tsx`、`(super)/super/page.tsx`、`components/business/RoleLanding.tsx`，并从 `app-navigation.ts` 的 `ROOT_PATHS` 中清理 `/student`、`/teacher`、`/admin`、`/super` 四个空根路径。
- `ProfilePanel` 在非学生角色下不再渲染"学习概览"卡片，并把右侧网格降级为单列布局，避免出现空缺位置。

### 学生实验主操作页 P-41 完整性核查

源码位置：`frontend/src/components/business/ExperimentInstancePanel.tsx`、`frontend/src/components/business/SimEnginePanel.tsx`、`frontend/src/app/(student)/student/experiment-instances/[id]/page.tsx`。

- 学生实验主操作页已存在，入口为 `/student/experiment-instances/:id`，由 `ExperimentInstancePanel` 承接，使用 `mode="student"` 渲染学生工作台。
- 当前页面已具备终端（`ExperimentTerminal`）、Web IDE（`WebIDEPanel`）、检查点（`CheckpointPanel`）、快照（`SnapshotPanel`）、报告区域、生命周期操作（暂停/恢复/重启/提交/销毁）和可视化 SimEngine 面板（`SimEnginePanel`）。
- 该页面不是空壳页面，但仍不能直接判定为"完整验收通过"。
- 已确认风险：
  - `desktopUrl` 当前为空字符串硬编码（`ExperimentInstancePanel.tsx:100`），VNC/桌面分支虽然写了渲染条件（`experimentType !== 1 && desktopUrl`），但永远不会展示，未真实接通后端桌面访问 URL。模块04 API/数据库目前只定义 `access_url` 单一入口，没有 `desktop_url` 字段，前端无法独立修复；详见 FE-AUDIT-022。
  - `ideUrl` 直接使用 `instance.access_url`，但 `access_url` 与桌面 URL 在真实环境/混合实验中应该是不同容器的不同端口，复用同一字段会让 IDE 与桌面 Tab 内容指向同一来源；详见 FE-AUDIT-023。
  - SimEngine 面板依赖 `instance.sim_session_id` 和模板 `sim_scenes`，是否完整可用仍需联调验证（`SimEnginePanel.tsx:42` 通过 `useSimPanel` 拉取场景）。
  - 学生实验完整链路中的启动页（P-42）、分组页（P-43）、结果页（P-44）、报告页（P-45）、历史页（P-46）仍需继续逐页验收，详见 FE-AUDIT-024。

### 学生比赛页面完整性核查

源码位置：`frontend/src/app/(student)/ctf/`、`frontend/src/components/business/CtfPanels.tsx`、`frontend/src/components/business/CtfChallengePanel.tsx`、`frontend/src/components/business/AttackDefenseRoundPanel.tsx`、`frontend/src/components/business/CtfLeaderboard.tsx`。

- 学生 CTF 页面集合已齐：`/ctf`（大厅）、`/ctf/:id`（详情/报名）、`/ctf/:id/team`（团队管理）、`/ctf/:id/jeopardy`（解题赛主页）、`/ctf/:id/jeopardy/:cid`（题目详情/解题）、`/ctf/:id/attack-defense`（攻防赛主页）、`/ctf/:id/attack-defense/attack`（攻击）、`/ctf/:id/attack-defense/defense`（防守）、`/ctf/:id/leaderboard`（排行榜）、`/ctf/:id/results`（结果页）。
- 这些页面不是空壳或导航占位，每个 `page.tsx` 都通过 `PermissionGate` 限定学生角色，并挂接到 `CtfHallPanel`、`CtfCompetitionDetailPanel`、`CtfTeamPanel`、`CtfJeopardyPanel`、`CtfChallengePanel`、`AttackDefenseRoundPanel`、`CtfLeaderboardPagePanel`、`CtfResultsPanel` 等真实业务面板。
- 当前可以确认"页面已齐"，但仍不能直接判定为"完整验收通过"，仍需继续逐页深验报名与组队边界、排行榜实时更新、公告联动、题目环境与提交链路、攻防赛交互与结果链路，详见 FE-AUDIT-025。

### 验证结果

- `rg "RoleLanding" frontend/src` 无结果，无残留引用。
- `rg "[\"'](?:/student|/teacher|/admin|/super)[\"']" frontend/src` 仅命中 `lib/permissions.ts` 中的次级路径（`/student/courses` 等），四个根路径均已清理。
- `npx vitest run src/hooks/useAuth.test.ts` 通过（验证后已删除临时测试）。
- `npx vitest run src/components/business/ProfilePanel.test.tsx` 通过（验证后已删除临时测试）。


