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
- **后端代理**（`backend/internal/handler/experiment/realtime.go`）：`ServeSimEngineWS` 与 `serveTerminalPTYProxy` 在拨号上游失败时，把 `upstream_url / upstream_status / upstream_reason` 透传到前端 `control_ack` / `terminal_init`。
- **前端**（`frontend/src/components/business/ExperimentTerminal.tsx`）：把上游错误三件套渲染到 xterm，运维一眼定位是容器没起、端口未暴露还是路径不对。
- **JWT 中间件**（`backend/internal/middleware/jwt.go`）：保留 `?token=` query 退化（浏览器 WS 不能发 Authorization 头），同时折叠 `extractBearerToken` 三层冗余判断。

### 2. 实验工具反代（#10 SPDY 隧道 + Cookie 鉴权）
- **架构**：本机 / Docker Desktop 部署下 backend 在集群外，无法直拨 Pod IP。改走 K8s API Server 的 SPDY portforward 隧道（`backend/internal/service/experiment/k8s_portforward.go`），handler 在 SPDY conn 上跑 WS 握手或 HTTP 反代。
- **鉴权**：iframe 不能用 URL 带 token（log / Referer 泄漏 + 浏览器历史），改用 HttpOnly cookie：
  - `POST /api/v1/experiment-instances/:id/tools/:kind/proxy-cookie` 签 `lc_tool_proxy` cookie，TTL 复用 access token 时长，前端 `useToolProxyCookie` hook 每 25 分钟刷新
  - `POST /instance/:instance_id/:tool_kind/*proxy_path` 路由组绑 `ToolProxyAuth` 中间件，校验 cookie 路径与 token 完全一致
- **覆盖工具**：`terminal` (xterm-server) / `ide` (code-server / remix-ide / jupyter-notebook / sagemath / ghidra-server / zaproxy) / `desktop` (novnc-desktop) / `explorer` (blockscout / fabric-explorer / webase-web / chainmaker-explorer) / `monitor` (grafana)

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

## 当前状态（2026-05-11）

### 第 9 轮：后端重启后测试

#### 🐛 本轮问题
1. **仿真会话创建超时**：后端重启后，启动新实验实例时出现"创建仿真会话失败: rpc error: code = DeadlineExceeded desc = context deadline exceeded"错误。
2. **场景容器日志为空**：场景容器虽然启动成功（4个Pod都是Running状态），但日志为空，说明场景运行时没有启动。
3. **场景容器 gRPC 服务器未启动**：与第8轮相同的根因，场景容器的 gRPC 服务器没有正确启动。

#### � 根因（已通过 Playwright 浏览器测试 + Kubernetes 日志验证）
1. **后端重启无效**：
   - 后端重启后，问题仍然存在
   - 场景容器镜像可能没有更新，或者场景运行时代码本身有问题

2. **场景容器启动但无日志**：
   - 4个场景容器全部 Running：scn-710189820d84-pbft-consensus, scn-710189820d84-pos-validator, scn-710189820d84-pow-mining, scn-710189820d84-raft-election
   - 场景容器日志为空（kubectl logs 无输出）
   - 说明场景运行时进程没有启动或立即崩溃

3. **仿真会话创建超时**：
   - 错误码: 50010
   - 错误消息: "仿真会话创建失败: rpc error: code = DeadlineExceeded desc = context deadline exceeded"
   - SimEngine Core 在尝试连接场景容器的 gRPC 服务器时超时

4. **可能原因**：
   - 场景容器镜像（scenarios/runtime）可能使用了错误的版本
   - 场景运行时的启动脚本可能有问题
   - 场景运行时的环境变量配置可能不正确
   - 场景运行时的 gRPC 服务器实现代码可能有问题

#### 🧪 测试结果
**学生端测试**：
- [x] 学生登录（13872945160 / LensChain2026）— ✅ 成功
- [x] 导航到"我的实验"页面 — ✅ 成功
- [x] 选择仿真实验模板（共识机制可视化对比实验）— ✅ 成功
- [x] 启动仿真实验 — ❌ 失败（新实例 ID: 2053746113386647552）
- [x] 场景容器启动 — ✅ 成功（4个Pod都是Running状态）
- [ ] 仿真会话创建 — ❌ 失败（超时错误）
- [ ] interaction-schema 接口 — ❌ 无法测试（仿真会话创建失败）
- [ ] 步数计数器 — ❌ 无法测试（仿真会话创建失败）
- [ ] 场景容器 gRPC 响应 — ❌ 失败（日志为空，gRPC 服务器未启动）

#### 📋 测试详情
- **后端重启无效**：后端重启后，问题仍然存在，场景容器日志为空
- **场景容器启动成功**：kubectl 显示 4 个场景容器都是 Running 状态
- **仿真会话创建超时**：错误码 50010，gRPC 超时错误
- **阻塞原因**：场景容器的 gRPC 服务器没有启动，导致 SimEngine Core 无法创建仿真会话

#### 🛠 需要的操作
**当前状态**：后端重启无效，场景容器日志为空，仿真会话创建超时。

建议检查：
1. 确认场景容器镜像是否正确（scenarios/runtime 镜像版本）
2. 检查场景运行时的启动脚本和入口点
3. 检查场景运行时的环境变量配置
4. 检查场景运行时的 gRPC 服务器实现代码
5. 手动运行场景容器镜像，查看是否有启动错误

---

### 第 7 轮（已处理）：场景容器gRPC通信问题

#### 🐛 本轮问题
1. **页面渲染失败**：启动仿真实验后，页面显示"页面渲染失败"，错误信息"Cannot read properties of undefined (reading 'success)"。
2. **场景容器gRPC无响应**：场景容器虽然启动成功（4个Pod都是Running状态），但场景容器的gRPC服务器没有正确响应SimEngine Core的调用。
3. **protobuf unmarshaling错误**：SimEngine Core日志显示protobuf unmarshaling错误，说明场景容器返回的gRPC响应格式不正确。

#### 🔍 根因（已通过 Playwright 浏览器测试 + Docker 日志验证）
1. **场景容器启动成功**：
   - `kubectl get pods -n lenschain` 显示4个场景容器都是Running状态：scn-862c360c5fae-pow-mining, scn-862c360c5fae-pos-validator, scn-862c360c5fae-pbft-consensus, scn-862c360c5fae-raft-election
   - 场景容器日志显示：`2026/05/11 07:24:15 scenario runtime starting: scene_code=pow-mining`，但无后续日志

2. **gRPC通信失败**：
   - SimEngine Core日志显示protobuf unmarshaling错误，错误堆栈指向`GetInteractionSchema`方法
   - 错误发生在SimEngine Core尝试调用场景容器的gRPC服务时
   - 场景容器日志没有gRPC服务器启动的日志，说明gRPC服务器可能没有正确初始化

3. **前端渲染失败**：
   - 浏览器控制台显示15个错误
   - 页面显示"页面渲染失败"，错误信息"Cannot read properties of undefined (reading 'success)"
   - 错误发生在SimEnginePanel.tsx:144:18

4. **可能原因**：
   - 场景容器镜像本身可能有问题
   - 场景容器的gRPC服务器实现与SimEngine Core的期望不匹配
   - 场景容器的gRPC服务器没有正确启动或监听错误的端口

#### 🧪 测试结果
**学生端测试**：
- [x] 学生登录（13872945160 / LensChain2026）— ✅ 成功
- [x] 导航到"我的实验"页面 — ✅ 成功（点击导航按钮）
- [x] 选择仿真实验模板（共识机制可视化对比实验）— ✅ 成功
- [x] 启动仿真实验 — ✅ 成功（实例状态：运行中，SimEngine Core 会话创建成功）
- [x] 场景容器启动 — ✅ 成功（4个Pod都是Running状态）
- [x] WebSocket 连接 — ✅ 成功（状态：已连接）
- [ ] 场景容器gRPC响应 — ❌ 失败（protobuf unmarshaling错误）
- [ ] InteractionForm 交互表单 — ❌ 失败（页面渲染失败）
- [ ] 仿真画布加载 — ❌ 失败（页面渲染失败）
- [ ] 步数计数器 — ❌ 失败（页面渲染失败）
- [ ] 视图切换 — ❌ 失败（页面渲染失败）

#### 📋 测试详情
- **场景容器启动**：kubectl显示4个场景容器都是Running状态，说明SPDY portforward修复有效，场景容器能够成功启动
- **gRPC通信问题**：场景容器日志只有"scenario runtime starting"，没有后续gRPC服务器启动日志，说明场景容器的gRPC服务器没有正确初始化
- **SimEngine Core配置**：`in_cluster: false`配置正确，符合本地开发环境要求
- **阻塞原因**：场景容器的gRPC服务器没有正确响应SimEngine Core的调用，导致前端无法渲染仿真面板

#### 🛠 需要的操作
**当前状态**：场景容器启动成功，但gRPC服务器无响应，导致页面渲染失败。

建议检查：
1. 检查场景容器镜像的构建过程，确认gRPC服务器代码是否正确包含
2. 检查场景容器的gRPC服务器实现，确认是否与SimEngine Core的期望匹配
3. 检查场景容器的gRPC服务器启动日志，确认gRPC服务器是否正确初始化
4. 检查场景容器的gRPC服务器监听端口，确认是否与SimEngine Core期望的端口一致
5. 检查场景容器的环境变量配置，确认SCENE_CODE等必要参数是否正确注入

---

## 已修复（折叠，按时间倒序）

<!-- 每条一句话摘要 + 涉及核心文件 -->

- **2026-05-11 第 7 轮：sim-engine protobuf 自身不一致 panic 修复**
  - **现象**：sim-engine 进程在反射 `ActionDef` 描述符时 `panic: invalid Go type string for field lenschain.sim_scenario.v1.ActionDef.category`；导致 `GetInteractionSchema` 整条链路 unmarshal 失败，前端报 `Cannot read properties of undefined (reading 'success')`。
  - **根因**：`sim-engine/proto/gen/go/lenschain/sim_scenario/v1/sim_scenario.pb.go` 内部不一致——Go struct 字段 `Category/Trigger` 类型是 `string`（行 1560-1561），但同文件内嵌的 raw FileDescriptor 二进制 blob 中这两个字段仍声明为 enum `ActionCategory/ActionTrigger`（旧版本残留，`\x0e2).lenschain...ActionCategoryR`）。`protoimpl.MessageInfo.initOnce` 会 cross-check Go reflect 类型 vs descriptor 类型，一不一致就 panic。原因是有人改 `.proto` 后没跑 `buf generate`，可能只手工改了 Go struct。
  - **修复**：`buf generate` 重新生成 `proto/gen/go/`，descriptor 与 Go struct 同步（行 2045 现为 `\x01(\tR\bcategory`，`\t`=TYPE_STRING）。重建 `sim-engine-core` 与 `scenarios/runtime` 两个镜像（都依赖 `proto/gen/go/`），重启 sim-engine 容器，清理残留场景 Pod/Service。
- **2026-05-11 第 6 轮：仿真实验 UI 布局 + 场景容器连接根因修复**
  - **场景 gRPC 拨号 connection refused（核心根因）**：sim-engine 在本地 docker-compose 桥接网络与 docker-desktop K8s ClusterIP 段不连通，原代码强行直拨 ClusterIP 注定失败。改为按 `in_cluster` 标志双路径：生产（in_cluster=true）保留 ClusterIP 直拨；开发（in_cluster=false）走 K8s API server 的 SPDY portforward 隧道（与 backend tool proxy 同一技术栈）。新增 `sim-engine/core/internal/scene/k8s_portforward.go`；`k8s_orchestrator.go` 加 `restConfig` 字段 + `dialAndWaitReady` 双路径分支；同步修订 `docs/modules/04-实验环境/06.1-场景算法容器编排设计.md` §一 §8.3 §8.4，纠正"docker 桥接网络与 ClusterIP 段连通"的错误论断。
  - **派生问题自愈**：interaction-schema 500、仿真步数不推进、WebSocket 收不到 render 帧——三者根因都是场景 gRPC 不通，根因修复后自愈。
  - **grid 模式 4 场景垂直堆叠**：`SimSceneGrid.tsx` 用 `lg:grid-cols-2` 依赖 viewport 1024px 断点，但场景画布容器宽度被外层导航 + 侧栏挤压不足 1024，断点不触发回退到单列。改为按 sceneCount 显式指定 `grid-cols-{N}`（1/2/3/4 场景对应 1/2/3/2 列），与 docs/06.2 §3.1 矩阵一致。
  - 副作用澄清：用户报告"点击网格按钮后布局模式文字仍是对照模式"——这不是 bug。`对照模式` 是 SimMode（业务模式 Badge），`grid/focus/carousel` 是 SimLayoutMode（布局），二者独立。docs/06.2 §1.1 §2.1 §三明确区分。
- 2026-05-09 第 4 轮：仿真画布与检查点修复（画布尺寸同步、检查点 DSL 解析、终端 stale closure、K8s EnableServiceLinks、快照恢复、blockscout postgres 依赖）

---
