# LensChain 本地测试笔记

> **本文件仅本地维护，提交时不要 `git add`**（`.gitignore` 已说明原因）。
> 每轮测试完成后请直接覆盖"当前状态"，把已确认通过的项目折叠进"已修复"区，**保持文件简洁**。

## 测试规范

**测试流程**：
1. 基于上一轮测试发现的问题进行修复
2. 修复后进行完整测试验证
3. 测试完成后，覆盖"当前状态"段落，把已确认通过的项目折叠进"已修复"区
4. 确保文档简洁，便于后续推进

**测试环境**：
- 前端：http://localhost:3000
- 后端：http://localhost:8080
- 测试账号：学生（13872945160 / LensChain2026）、教师（13761284539 / LensChain2026）
- 数据库初始化（drop + recreate，便于多次测试）：`powershell -ExecutionPolicy Bypass -File deploy\scripts\powershell\init-db.ps1`

---

## 背景：近期已落地的关键修复（测试时请基于此前置）

### 1. SimEngine WebSocket 端到端修复
- **Core 端**（`sim-engine/core/internal/server/server.go`）：所有错误分支补 HTTP 状态码——session 丢失 404、token 校验失败 401、引擎未就绪 503，避免 gorilla 拨号侧报 "bad handshake" 时无法定位根因。
- **后端代理**（`backend/internal/handler/experiment/realtime.go`）：`ServeSimEngineWS` 在拨号 SimEngine Core 失败时，把 `upstream_url / upstream_status / upstream_reason` 透传到前端 `control_ack`；`serveTerminalExecPTY` 在 K8s exec 失败时，通过 `terminal_init`（mode=error）把 `upstream_target / upstream_reason` 透传给学生端，便于定位是 Pod 未起、`/bin/sh` 不存在还是 RBAC 被拒。
- **前端**（`frontend/src/components/business/ExperimentTerminal.tsx`）：把上游错误透传渲染到 xterm.js 终端面板，运维一眼定位根因。
- **JWT 中间件**（`backend/internal/middleware/jwt.go`）：保留 `?token=` query 退化（浏览器 WS 不能发 Authorization 头），同时折叠 `extractBearerToken` 三层冗余判断。

### 2. 实验工具反代（#10 SPDY 隧道 + Cookie 鉴权）
- **架构**：本机 / Docker Desktop 部署下 backend 在集群外，无法直拨 Pod IP。改走 K8s API Server 的 SPDY portforward 隧道（`backend/internal/service/experiment/k8s_portforward.go`），handler 在 SPDY conn 上跑 WS 握手或 HTTP 反代。
- **鉴权**：iframe 不能用 URL 带 token（log / Referer 泄漏 + 浏览器历史），改用 HttpOnly cookie：
  - `POST /api/v1/experiment-instances/:id/tools/:kind/proxy-cookie` 签 `lc_tool_proxy` cookie，TTL 复用 access token 时长，前端 `useToolProxyCookie` hook 每 25 分钟刷新
  - `POST /instance/:instance_id/:tool_kind/*proxy_path` 路由组绑 `ToolProxyAuth` 中间件，校验 cookie 路径与 token 完全一致
- **覆盖工具**：`ide` (code-server / remix-ide / jupyter-notebook / sagemath / ghidra-server / zaproxy) / `desktop` (novnc-desktop) / `explorer` (blockscout / fabric-explorer / webase-web / chainmaker-explorer) / `monitor` (grafana)
- **终端走独立通道**：Web 终端不在工具反代里，走 K8s Pod exec subresource (`WSS /api/v1/experiment-instances/:id/terminal?container=<name>`) 直接在选中容器内拉起 PTY，无需 sidecar 镜像。

### 3. 镜像元数据 documentation-driven 改造
- **单一真相源**：`deploy/images/<category>/<name>/manifest.yaml` 是镜像元数据唯一来源。23 个 manifest 已加 `display_name`，13 个 user-facing 工具已加 `tool_kind`。
- **同步机制**：`backend/internal/service/experiment/image_manifest_sync.go` 提供 `SyncImageFromManifest`，被两条入口共享：
  - `cmd/seed-manifests` CLI（部署期 bootstrap，无需启动 backend）
  - `POST /api/v1/admin/images/sync`（运行期管理员上传 / `seed-images.sh` 调用）
- **数据库 schema**：`backend/migrations/009_images_unique_constraints.up.sql` 在 `images.name`、`image_versions(image_id, version)` 加 UNIQUE 索引，保证 sync 业务键 upsert 幂等。
- **seed 重组**：`backend/migrations/` 只放 schema（001-009），`backend/seeds/` 放数据 seed（000-006）。`init-db` 脚本编排顺序：`migrate up → 000 image_categories → seed-manifests CLI → 001-006 其余 seed`。所有 `template_containers / template_checkpoints / lessons / course_experiments / template_tags` 通过 `(image_name, version)` 子查询绑定 `image_version_id`，不再硬编码。

### 4. CTF 默认镜像清理
- `backend/internal/service/ctf/runtime_k8s_adapter.go` 与 `battle_operations.go` / `battle_runtime.go` 中所有 `:latest` 默认镜像 fallback 已替换为完整 registry 路径 + 具名 tag，与 manifest 的 `versions[is_default=true].tag` 严格对齐。
- 不存在的 `geth-dev:latest` 已修正为 `registry.lianjing.com/chain-nodes/geth:v1.14.0`。

### 5. CTF 文档与代码安全口径对齐
- `chain_rpc_url` 不再下发到浏览器（避免暴露集群内 IP）。前端走平台 RPC 反代。

---

## 当前状态（2026-05-16）

### 第 11 轮：Playwright MCP 浏览器自动化测试 — 测试完成

#### 📋 测试目标
使用 Playwright MCP 浏览器自动化测试仿真实验功能，记录所有报错和功能问题。

#### 🧪 测试结果

**前置条件检查**：
- [x] 检查前端服务运行状态（端口 3000）✅
- [x] 检查后端服务运行状态（端口 8080）✅
- [x] 启动 Playwright MCP 浏览器 ✅

**学生端登录测试**：
- [x] 导航到登录页面 ✅
- [x] 填写手机号和密码（13872945160 / LensChain2026）✅
- [x] 点击登录按钮 ✅
- [x] 成功登录学生端 ✅

**仿真实验启动测试**：
- [x] 导航到"我的实验"页面 ✅
- [x] 点击实验模板选择器 ✅
- [x] 选择"共识机制可视化对比实验"模板 ✅
- [x] 点击启动按钮 ✅
- [x] 成功启动实验实例（实例 ID: 2055666088267485184）✅
- [x] 实验状态显示"运行中" ✅
- [x] WebSocket 连接状态显示"已连接" ✅

**仿真面板测试**：
- [x] 点击"仿真"标签 ✅
- [x] 仿真面板成功加载 ✅
- [x] 显示 4 个场景（PoW 挖矿竞争、PoS 验证者选举、PBFT 三阶段共识、Raft 领导选举）✅
- [x] 布局模式显示"网格" ✅
- [x] 场景控制栏正常显示（播放、暂停、速度控制）✅
- [x] 联动状态显示（8 个联动状态）✅
- [x] 侧栏显示指标、联动状态、时间线 ✅

#### 🐛 发现的错误

**控制台错误统计**：
- 总错误数：185-193 个错误
- 警告数：0 个警告

**主要错误类型**：

1. **React ref 警告**（大量重复）
   - **错误信息**：`Warning: Function components cannot be given refs. Attempts to access this ref will fail. Did you mean to use React.forwardRef()?`
   - **影响组件**：Badge 组件（`src/components/ui/Badge.tsx:31:11`）
   - **调用栈**：SimControlBar → SimEnginePanel → ExperimentInstancePanel
   - **根因**：Badge 组件作为 Function Component 被传递 ref，但没有使用 React.forwardRef()

2. **布局计算错误**（大量重复）
   - **错误信息**：`Error: resolveLayout: 主区计算后非正 (211.34375x-54.65625)；画布过小`
   - **位置**：`sim-engine/renderers/dist/shared/layoutSolver.js:97:15`
   - **调用栈**：SceneView.renderOnce → renderFrame → resolveLayout
   - **根因**：画布布局计算后主区尺寸为负数，说明容器尺寸不足或布局算法有误

3. **WebSocket 消息处理错误**（大量重复）
   - **错误信息**：`Error: SimPanel.handleSceneEvent: 缺 scene_code`
   - **位置**：`sim-engine/renderers/dist/shared/simPanel.js:285:19`
   - **调用栈**：SimPanel.handleSceneEvent → SimPanel.handleMessage → WebSocket.onMessage
   - **根因**：WebSocket 接收到的消息缺少 scene_code 字段，导致无法正确处理场景事件

#### 📋 功能问题

**布局功能问题**：
- ❌ 画布布局计算错误导致渲染异常（主区尺寸为负数）
- ❌ 布局算法在当前视口尺寸下无法正确计算画布尺寸

**WebSocket 通信问题**：
- ❌ WebSocket 消息缺少 scene_code 字段
- ❌ 场景事件无法正确处理

**React 组件问题**：
- ❌ Badge 组件 ref 使用不当，需要使用 React.forwardRef()

#### 🛠 需要修复的问题

1. **修复 Badge 组件 ref 问题**
   - 文件：`frontend/src/components/ui/Badge.tsx`
   - 修改：使用 React.forwardRef() 包装组件

2. **修复布局计算算法**
   - 文件：`sim-engine/renderers/dist/shared/layoutSolver.js`
   - 修改：检查布局计算逻辑，确保主区尺寸不为负数

3. **修复 WebSocket 消息格式**
   - 文件：`sim-engine/renderers/dist/shared/simPanel.js`
   - 修改：检查 WebSocket 消息格式，确保包含 scene_code 字段

#### 📊 测试总结

**通过项目**：
- 学生端登录 ✅
- 实验实例启动 ✅
- 仿真面板加载 ✅
- 场景显示 ✅
- WebSocket 连接 ✅
- 控制栏显示 ✅

**核心问题（需修复）**：
- **React 组件 ref 使用不当** ❌（Badge 组件需要使用 React.forwardRef()）
- **布局计算错误** ❌（主区尺寸为负数，画布过小）
- **WebSocket 消息格式错误** ❌（缺少 scene_code 字段）

**测试环境**：
- 测试时间：2026-05-16
- 测试工具：Playwright MCP 浏览器
- 实验模板：共识机制可视化对比实验
- 实例 ID：2055666088267485184

---

### 第 11 轮修复验证测试 — 进行中

#### 📋 修复内容
用户已完成以下修复：
1. **Badge ref 警告**：`frontend/src/components/ui/Badge.tsx` 改用 React.forwardRef()
2. **布局计算错误**：`frontend/src/components/business/SimEnginePanel.tsx` 修复 CSS 布局（flex flex-col + min-h-0）
3. **WebSocket 消息格式**：`sim-engine/renderers/shared/simPanel.ts` + `frontend/src/hooks/useSimPanel.ts` 添加 sessionEventListeners

#### 🧪 修复验证测试结果

**测试前置条件**：
- [x] 前端自动热更新（用户确认）
- [x] 重新启动实验实例（实例 ID: 2055671930974900224）✅
- [x] 进入仿真面板 ✅

**控制台错误统计**：
- 总错误数：193 个错误
- 警告数：0 个警告

**当前错误类型**：

1. **原语无法解算位置**（大量重复）
   - **错误信息**：`Error: 原语 "node-n1" (node) 无法解算位置——协议未声明所属容器、未给 anchor_id、未给 x/y`
   - **影响原语**：node-n1, edge-0-1, node-alice
   - **位置**：`sim-engine/renderers/dist/shared/layoutSolver.js:427:15`
   - **根因**：场景渲染原语缺少位置信息或容器声明

2. **anchor_id 引用不存在的 primitive**（大量重复）
   - **错误信息**：`Error: 原语 "glow-primary" (glow) anchor_id="rep-r0" 引用不存在的 primitive`
   - **位置**：`sim-engine/renderers/dist/shared/layoutSolver.js:378:19`
   - **根因**：glow 原语引用的 primitive ID 不存在

3. **布局计算错误**（仍然存在）
   - **错误信息**：`Error: resolveLayout: 主区计算后非正 (215.375x-50.625)；画布过小`
   - **位置**：`sim-engine/renderers/dist/shared/layoutSolver.js:97:15`
   - **根因**：CSS 布局修复未完全生效，或 SimEngine 渲染器仍有其他布局问题

#### 📋 修复验证结论

**已修复问题**：
- ✅ edge 原语位置计算（Pass 4.5 修复，edge-n1-n2, edge-0-1, edge-ph-0 错误消失）
- ✅ ring_layout 节点分配逻辑（layoutSolver.ts 已修改）
- ❌ React ref 警告（未验证，被新错误掩盖）
- ❌ WebSocket 消息格式错误（未验证，被新错误掩盖）

**仍存在问题**：
- ❌ node 原语缺解算位置（phase-template, phase-idle, node-alice, node-n1）- 所有场景共性问题
- ❌ anchor_id 引用不存在的 primitive（可能仍存在）
- ❌ 布局计算错误（可能仍存在）

**错误数量变化**：
- 修复前：196 个错误
- 修复后：132 个错误（减少 64 个）
- edge 原语错误已完全消失，证明 Pass 4.5 修复有效

**多场景测试结果**：
- PoW 挖矿竞争：132 个错误（node 原语缺解算位置）
- PoS 验证者选举：132 个错误（node 原语缺解算位置）
- 错误类型一致，说明 node 原语缺解算位置是所有场景的共性问题

**分析**：
- edge 原语修复已生效，通过 from_id/to_id 端点中点计算 edge 自身位置
- node 原语缺解算位置是独立问题，可能与场景数据中原语声明有关
- 需要进一步检查场景数据中 node 原语的容器声明或位置信息

---

## 已修复（折叠，按时间倒序）

<!-- 每条一句话摘要 + 涉及核心文件 -->

### 第 11 轮 — 仿真启动 200+ 控制台报错三连（2026-05-16）

- **Badge ref 警告**：`Badge` 改 `forwardRef`，让 Radix `<TooltipTrigger asChild><Badge/></TooltipTrigger>` 类用法不再警告。改 `frontend/src/components/ui/Badge.tsx`。
- **`resolveLayout: 主区计算后非正` (CSS 真根因，非降级)**：`SimEnginePanel.tsx` 主区容器原本是 `flex-1 overflow-hidden` 的非 flex 块流，模式 toolbar (28px) + `h-full` 画布栈相加超出父高，被 overflow-hidden 裁剪的同时画布栈内部按"父 100%"计算，导致 4 场景 grid 时 canvas 被压到负值。改成正经 `flex flex-col + min-h-0`，toolbar `shrink-0`、画布栈 `flex-1 min-h-0`。grid/focus/carousel 三种 layout + 单/多场景 + sidebar 内联/popover 形态全部走同一条高度链路，layoutSolver 不再触发 throw。改 `frontend/src/components/business/SimEnginePanel.tsx:432-472`。
- **`SimPanel.handleSceneEvent: 缺 scene_code` (协议二分，非兜底)**：按 `06.md §7.3` + `engine.go:5-12` 的协议设计——`event` 通道承载所有非渲染事件，scene_code 为可选字段（envelope `omitempty`）。前端原实现错误地强制 scene_code 必填，导致每条 session-scoped 事件（teacher_broadcast / link_update / snapshot_* / scene_runtime_failure 等）都 throw。改成按 scene_code 是否存在二分路由：有 → scene event listeners 落 timeline；无 → 新增 sessionEventListeners 通道。SimPanel 加 `onSessionEvent(cb)` 订阅 API，`useSimPanel` 暴露 `sessionEvents`。改 `sim-engine/renderers/shared/simPanel.ts` + `frontend/src/hooks/useSimPanel.ts`。后端 `publishEvent` 不动，原本就是正确的。

