# LensChain 前端开发规范

> **适用范围：** 本文件适用于 `frontend/` 目录及其所有子目录。
> **继承关系：** 本文件细化根目录 `AGENTS.md` 的前端规范；如与根目录规范或用户直接指令冲突，以根目录规范和用户直接指令为准。
> **核心原则：** 文档驱动、职责单一、调用链单向、契约统一、组件可复用、体验完整、注释清晰。

---

## 一、文档驱动开发

开发任何前端功能前，必须先阅读对应模块文档：

- `docs/modules/{模块}/01-功能需求说明.md`
- `docs/modules/{模块}/03-API接口设计.md`
- `docs/modules/{模块}/04-前端页面设计.md`
- `docs/modules/{模块}/05-验收标准.md`

前端页面、字段、按钮、状态、权限入口、交互流程、空状态、错误提示和验收方式都必须以文档为准。

如发现文档、后端实现和实际需求冲突，先更新文档，再改前端代码。不得先在前端做临时兼容、别名字段、双路径调用或隐藏逻辑。

---

## 二、技术栈约束

前端使用以下技术栈：

- Next.js 14+ App Router
- React 18+
- TypeScript strict mode
- Tailwind CSS 3.4+
- shadcn/ui 风格基础组件
- Lucide React 图标
- Zustand 4+
- TanStack Query 5+

不得引入第二套路由系统、第二套 HTTP 客户端、第二套服务端状态库或与现有技术栈重复的 UI 框架。新增依赖必须有明确必要性，并优先复用项目已有能力。

---

## 三、标准目录结构

`frontend/` 必须保持以下职责分层：

```text
frontend/
├── src/
│   ├── app/                     # Next.js App Router 页面与布局
│   ├── components/
│   │   ├── ui/                  # 基础 UI 组件，不含业务逻辑
│   │   └── business/            # 业务组件，按功能域扁平组织
│   ├── hooks/                   # 自定义 Hooks，按功能域分文件
│   ├── lib/                     # 基础设施与纯工具函数
│   ├── services/                # API 调用层
│   ├── stores/                  # Zustand 客户端状态
│   └── types/                   # TypeScript 类型定义
├── public/                      # 静态资源
├── package.json
├── tsconfig.json
├── tailwind.config.ts
└── next.config.js
```

禁止把后端代码、SQL 迁移、部署配置、运维脚本、业务文档生成逻辑放入 `frontend/`。

---

## 四、调用链规则

前端调用链必须保持严格单向：

```text
app 页面 -> components/business -> hooks -> services -> lib/api-client -> 后端 API
                         ↓
                   components/ui
```

禁止倒置依赖：

- 页面不得直接调用 `services/`、`fetch`、`axios` 或后端 URL。
- 业务组件不得直接调用 `services/`、`fetch`、`axios` 或 `lib/api-client`。
- UI 基础组件不得引用业务类型、hooks、stores 或 services。
- hooks 不得渲染 JSX。
- stores 不得直接调用 API。
- services 不得写 UI 逻辑、权限判断或复杂业务流程。
- lib 工具不得引用 React 组件或业务组件。

---

## 五、目录职责

### 5.1 `src/app/`

职责：

- 定义 App Router 页面、布局、路由分组和页面级边界。
- 组合业务组件和 hooks。
- 处理页面级 `loading.tsx`、`error.tsx`、`not-found.tsx`。
- 按角色组织路由入口。

禁止：

- 直接调用 API。
- 写可复用业务组件。
- 在 `page.tsx` 中堆叠大型表格、复杂表单或跨页面复用逻辑。

### 5.2 `src/components/ui/`

职责：

- 存放基础展示组件、shadcn/ui 风格组件、通用交互外壳。
- 关注样式、可访问性、基础状态和组合插槽。

禁止：

- 引用业务类型、业务 hooks、stores、services。
- 写权限判断、接口请求、业务字段映射。
- 按模块创建业务化组件。

### 5.3 `src/components/business/`

职责：

- 存放业务组件，例如课程卡片、实验状态面板、竞赛榜单、告警徽标、成绩概览卡。
- 通过 props 接收数据，或通过 hooks 获取当前功能域数据。
- 组合 `components/ui/`。

要求：

- 扁平放置，不按模块建子目录。
- 文件名体现功能域，例如 `CourseCard.tsx`、`ExperimentStatusPanel.tsx`、`CtfLeaderboardTable.tsx`。
- 单文件建议 500-800 行以内。
- 超过 800 行必须评估拆分为子组件、hooks 或工具函数。

禁止：

- 直接调用 API。
- 重复实现基础按钮、弹窗、表格、表单控件。
- 把页面路由结构硬编码进通用业务组件，除非该组件职责就是导航入口。

### 5.4 `src/hooks/`

职责：

- 封装 TanStack Query 查询、mutation、副作用和 UI 状态组合。
- 调用 `services/`。
- 管理 loading、error、empty、权限派生等页面可复用状态。

要求：

- 按功能域命名，例如 `useCourses.ts`、`useExperimentInstances.ts`、`useCtfCompetitions.ts`。
- Query key 必须稳定、集中、可复用。
- mutation 成功后必须按影响范围更新或失效缓存。

禁止：

- 渲染 JSX。
- 绕过 services 调接口。
- 把大段业务 UI 状态塞进全局 store。

### 5.5 `src/services/`

职责：

- 按模块或功能域封装 API。
- 定义请求参数类型、响应类型和服务函数。
- 只调用 `src/lib/api-client.ts`。

要求：

- 文件按模块或功能域拆分，例如 `auth.ts`、`course.ts`、`courseAssignment.ts`。
- API 路径、HTTP 方法、请求体、响应结构必须与 `docs/modules/{模块}/03-API接口设计.md` 一致。
- 后端雪花 ID 在前端一律使用 `string`，不得使用 `number`。
- 不得为同一个后端接口保留两套 service 封装。

禁止：

- 直接使用 `fetch`、`axios` 或第二套 HTTP 客户端。
- 写组件逻辑、样式逻辑、页面跳转。
- 为兼容旧字段保留永久别名或双字段解析。

### 5.6 `src/lib/`

职责：

- `api-client.ts`：统一 HTTP 客户端、baseURL、Token 注入、响应解包、错误归一、401 处理。
- `format.ts`：日期、文件大小、数字、分数等格式化。
- `utils.ts`：纯工具函数。
- 权限、枚举映射、常量等基础能力可按职责拆分。

禁止：

- 引用 React 组件。
- 引用业务组件。
- 放具体模块业务流程。
- 在工具函数里发起 API 请求。

### 5.7 `src/stores/`

职责：

- 存放 Zustand 客户端全局状态。
- 适合认证态、当前用户、主题偏好、侧边栏折叠态、临时 UI 偏好等。

要求：

- 服务端权威数据优先用 TanStack Query。
- store 不长期保存列表、详情、分页、统计等服务端状态。

禁止：

- store 直接调用 API。
- 用 Zustand 替代 TanStack Query 管理服务端状态。

### 5.8 `src/types/`

职责：

- 存放跨 services、hooks、components 复用的 TypeScript 类型。
- 按模块命名，例如 `auth.ts`、`course.ts`、`experiment.ts`。

禁止：

- 放运行时代码。
- 使用与 API 文档不一致的字段。
- 随意使用 `any`。

---

## 六、注释规范

注释必须使用中文，除非注释对象是第三方协议、英文错误码、标准缩写或外部 API 原文。

### 6.1 必须写注释的场景

- 所有导出的函数、组件、hook、类型、常量必须有中文注释。
- 业务组件文件顶部必须说明组件职责。
- 自定义 hook 必须说明封装的数据来源、Query key 语义和副作用。
- service 函数必须说明对应的后端接口路径和用途。
- 复杂权限判断、跨模块数据组合、表单联动、缓存失效策略必须写注释。
- 难以一眼看懂的 Tailwind 动态 class 组合必须写注释。
- 对接 WebSocket、文件上传、下载、导入导出、长轮询、富文本渲染、图表聚合时必须说明数据流和异常处理。
- 使用 `any`、类型断言、第三方库绕开类型推断时必须写中文注释说明原因。

### 6.2 不应写注释的场景

不要写解释显而易见代码的废话注释，例如：

```ts
// 设置 loading 为 true
setLoading(true)
```

应该把注释放在业务意图、边界条件和不变量上。

### 6.3 注释格式

导出组件：

```tsx
// CourseCard 展示课程摘要、学习进度和进入课程的主操作。
export function CourseCard(props: CourseCardProps) {
  // ...
}
```

导出 hook：

```ts
// useCourseList 读取课程列表，并统一维护筛选条件对应的 Query key。
export function useCourseList(params: CourseListParams) {
  // ...
}
```

service 函数：

```ts
// listCourses 对应 GET /api/v1/courses，用于课程列表页分页查询。
export function listCourses(params: CourseListParams) {
  return apiClient.get<CourseListResp>('/courses', { params })
}
```

复杂逻辑块：

```ts
// 后端权限仍是最终边界；这里仅用于隐藏不可用入口，避免用户误操作。
const canManageCourse = user.role === 'teacher' && course.teacher_id === user.id
```

### 6.4 禁止事项

- 禁止英文模板注释充数。
- 禁止保留过期注释。
- 禁止注释与代码行为不一致。
- 禁止用注释掩盖错误实现。
- 禁止写 `TODO`、`FIXME` 后不解决；确需保留必须说明阻塞原因、负责人和后续处理路径。

---

## 七、路由与角色页面

路由按角色和功能组织：

- `(auth)`：登录、SSO 回调、首次改密等认证页面。
- `(student)`：学生课程、实验、CTF、成绩、通知、个人中心相关页面。
- `(teacher)`：教师课程、作业、实验、竞赛管理、成绩审核。
- `(admin)`：学校管理员用户、学校、成绩、通知、租户设置。
- `(super)`：超级管理员学校、系统、监控、平台统计。

页面文件遵循 Next.js 约定：

- `page.tsx`
- `layout.tsx`
- `loading.tsx`
- `error.tsx`
- `not-found.tsx`

页面权限入口必须与后端 RBAC 和模块前端页面设计一致。前端只做体验层控制，真正权限以后端为准。

---

## 八、API 契约规范

所有 API 必须走 `src/lib/api-client.ts`。

后端统一响应格式：

```json
{
  "code": 200,
  "message": "success",
  "data": {},
  "timestamp": "2026-04-09T10:00:00Z"
}
```

要求：

- `api-client` 负责响应解包和错误归一。
- 分页参数统一为 `page`、`page_size`。
- 分页响应字段必须与 API 文档一致。
- 时间字段按 ISO 8601 / RFC3339 处理，展示时本地化格式化。
- 雪花 ID 一律使用 `string`。
- 多租户数据范围以后端返回为准，前端不得自行跨学校拼接数据。
- API 字段变化必须先同步文档和 types/services，不允许在组件里临时兼容。

禁止：

- 在组件里拼接后端 URL。
- 在页面里解析后端统一响应壳。
- 对同一后端字段长期保留旧字段名、新字段名两套逻辑。
- 把后端错误静默吞掉。

---

## 九、状态管理

服务端状态使用 TanStack Query：

- 列表
- 详情
- 分页
- 搜索结果
- 统计数据
- 当前服务端配置
- 后端权限结果

客户端状态使用 Zustand：

- 当前登录用户基础状态
- 主题偏好
- 导航折叠态
- 本地临时筛选草稿
- 非后端权威源的 UI 状态

表单状态优先由组件或表单库管理。可分享筛选条件应放到 URL query，例如页码、关键词、状态筛选。

---

## 十、UI、样式与体验

必须使用 Tailwind CSS 原子类和主题变量，不新增普通 CSS 文件，除非 Tailwind 无法覆盖且有明确理由。

基础组件使用 shadcn/ui 风格并放入 `components/ui/`。业务组件放入 `components/business/`。

图标统一使用 `lucide-react`。禁止用 Emoji 作为图标。

每个页面必须覆盖：

- 加载态
- 空状态
- 错误态
- 无权限态
- 操作成功反馈
- 关键危险操作确认

界面设计要求：

- 不做默认模板感强的“AI 生成式平庸页面”。
- 有明确视觉方向和层次。
- 不默认使用紫色主题。
- 不默认使用纯白平铺页面。
- 不默认使用 `Inter`、`Roboto`、`Arial`、系统字体作为唯一字体方案，除非现有设计系统已经明确使用。
- 背景、排版、卡片、数据区应有清晰的信息层级。
- 动效应服务于状态变化和信息过渡，不做无意义动画。
- 移动端、平板、桌面都要可用。
- 暗色模式类名应预留，不写死不可适配的浅色值。

如果项目已有确定设计系统，则优先保持一致，不为“新颖”破坏统一性。

---

## 十一、TypeScript 规范

要求：

- 启用严格模式。
- 组件使用具名导出，例如 `export function CourseCard() {}`。
- Props 类型命名为 `组件名Props`。
- 变量使用小驼峰。
- 布尔值使用 `is`、`has`、`should` 前缀。
- 常量使用全大写下划线。
- 状态、角色、枚举使用集中类型或常量，不散落魔法字符串。
- 与后端枚举对应的展示文本集中映射，不在多个页面重复写。

禁止：

- 随意使用 `any`。
- 用类型断言掩盖接口字段错误。
- 在 UI 组件里定义后端响应类型。
- 在多个文件重复定义同一 API 类型。

只有在无法准确表达第三方库类型时才允许 `any`，并必须写中文注释说明原因。

---

## 十二、React 与 Next.js 规范

要求：

- 默认使用函数组件。
- 能用 Server Component 的页面优先保持服务端组件；需要浏览器状态、事件、浏览器 API 时才使用 `"use client"`。
- Client Component 的边界要尽量小。
- 不默认使用 `useMemo`、`useCallback`。只有确有性能或引用稳定性需求时才使用。
- 不使用当前 React 版本不支持的实验 API。
- Suspense、loading、error 边界按页面复杂度合理设置。
- 表格、图表、长列表要考虑分页、虚拟化或后端分页。

禁止：

- 整个页面无理由标记 `"use client"`。
- 在 Server Component 中访问浏览器 API。
- 在组件 render 中触发副作用或请求。
- 用本地状态复制服务端 Query 数据。

---

## 十三、权限与安全

要求：

- 前端权限只负责展示控制，后端仍是最终权限边界。
- Token 注入、刷新、401 跳转登录统一在 `api-client` 或认证 hook 中实现。
- 用户输入做基础校验，但不替代后端校验。
- 富文本或用户提交内容必须经过安全处理。
- 上传、导入、下载功能必须处理文件类型、大小、进度、失败重试和错误提示。
- 敏感配置值不得明文展示，除非后端文档明确允许。

禁止：

- 在前端保存明文密码。
- 在 localStorage 保存长期敏感密钥。
- 直接插入未清洗 HTML。
- 绕过后端权限规则显示跨租户数据。

---

## 十四、模块接口归属提示

前端实现时必须按后端最新模块职责调用接口：

- 模块01 `/profile` 只返回个人基础资料，不返回学习概览。
- 个人中心学习概览调用模块06：`GET /api/v1/grades/my/learning-overview`。
- 模块07 负责通知、系统公告、模板和偏好。
- 模块08 负责系统审计、配置、告警、仪表盘、统计、备份。
- CTF 题目、竞赛、队伍、环境、排行榜都走模块05 service 文件，不在前端复制多套调用。

如果页面需要组合多个模块数据，组合逻辑放在 hooks 或页面层，API 调用仍通过各模块 services，不得在一个 service 文件里直接混写多个模块接口，除非该后端接口本身就是聚合接口。

---

## 十五、文件拆分与复用

建议范围：

- 组件文件：500-800 行以内。
- hook 文件：保持单一功能域。
- service 文件：按模块；过大时按功能域拆分。
- types 文件：按模块；共享基础类型可放公共文件。

必须复用：

- `lib/api-client`
- 统一格式化函数
- 统一枚举文本映射
- 统一权限判断工具
- `components/ui` 基础组件
- 已存在的业务组件和 hooks

禁止重复实现：

- HTTP 客户端
- 分页参数处理
- 时间格式化
- 文件大小格式化
- Token 处理
- 枚举文本映射
- Toast / Modal / Table / Form 基础组件

---

## 十六、测试与验证

修改前端代码后至少运行：

```bash
npm run lint
npm run build
```

如果项目配置了测试命令，还必须运行相关测试。

涉及接口契约变化时，必须对照：

- `docs/modules/{模块}/03-API接口设计.md`
- `src/services/{模块}.ts`
- `src/types/{模块}.ts`

涉及页面交互时，必须对照：

- `docs/modules/{模块}/04-前端页面设计.md`
- `docs/modules/{模块}/05-验收标准.md`

不允许在未运行验证命令的情况下声称 lint、build、类型检查或测试通过。

---

## 十七、禁止事项清单

- 禁止组件或页面直接调用 `fetch`、`axios` 或后端 URL。
- 禁止页面直接调用 `services/`。
- 禁止 `components/ui/` 引入业务逻辑。
- 禁止 `stores/` 直接调用 API。
- 禁止 Emoji 图标。
- 禁止随意使用 `any`。
- 禁止把服务端状态复制进 Zustand。
- 禁止私自新增文档未定义页面、按钮、字段、状态或接口。
- 禁止长期保留同一 API 的两套 service 封装。
- 禁止为了兼容临时后端返回引入永久分叉类型。
- 禁止单文件无限膨胀。
- 禁止把部署脚本、后端模型、SQL 迁移放进前端目录。
- 禁止绕过 `api-client` 处理认证、错误和响应解包。
- 禁止前端自行制造跨租户数据视图。

---

## 十八、完成判定

前端任务完成前必须确认：

- 已阅读对应模块文档。
- 页面实现与前端页面设计一致。
- API 调用与 API 文档一致。
- TypeScript 类型中的 ID 使用字符串。
- 页面、业务组件、hooks、services、api-client 调用链没有倒置。
- 加载态、空状态、错误态、无权限态齐全。
- 没有重复 service、重复 enum 映射或重复基础组件。
- 没有旧字段兼容分支或临时别名逻辑。
- 导出组件、hook、service 函数和复杂逻辑都有必要中文注释。
- 已运行必要的 lint、build 或测试验证。

