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

## 当前状态（2026-05-09）

### 第 4 轮：普通 / 混合实验回归 + 仿真画布与检查点修复

#### 🐛 本轮问题（第 3 轮的延续）
1. **仿真画布无动画**：纯仿真实验场景 tick 在跳但 Canvas 不绘制（用户看到的"img 占位"）。
2. **检查点验证永远未通过**：单项验证 / 验证全部都显示"未通过"，得分恒 0。
3. **终端命令无输出**：以太坊本地开发实验里输 `ls` / `pwd` 无回显也无结果，只剩 `~ $` 提示符。
4. **K8s 部署失败 50011**：EVM 全栈 DApp 的 geth、EVM 混合的 remix-ide 启动后立刻崩溃。
5. **仿真暂停后恢复报错**："环境异常：恢复仿真快照失败: snapshot not found"。
6. **EVM 全栈 blockscout 容器 Completed**：blockscout 在 K8s 中状态变 Completed（exit 0）。

#### 🔍 根因（已通过 Playwright 浏览器抓帧 + kubectl 日志 + 后端日志验证）
1. **画布无动画**（两个独立 bug 叠加）：
   - `SimEnginePanel.tsx` 的 `<canvas>` 没把容器实际像素同步到 `canvas.width/height`，drawingbuffer 默认 300×150，渲染器画在 y=180~396 的图全部超出画布被裁掉。
   - `useSimPanel.ts` 用 `useRef` 持 SimPanel：React effect 提交顺序"子先父后"，子组件 `SceneCanvas` 第一次 `attachScene` 时 `panelRef.current` 还是 null，调用变成 no-op，于是场景永远没绑定 SceneView，即使后续 panel 创建好也再不会重试。
2. **检查点恒不过**：
   - 后端 `instance_service_checkpoint.go::evaluateSimAssertion` 只做平铺 KV 严格相等，不解析文档规定的 `{scene_code, conditions:[{path,operator,value}], require_all}` DSL（见 `docs/modules/04-实验环境/02-数据库设计.md §2.6`）。
   - seed `005_seed_sim_scenarios.sql` 写的全是非法字段（`condition: "tick >= 10"` / `action_code` / `all_scenes_completed`），既不被代码识别，也不符合文档。
   - **二次发现**：评估器跑通后报 "json: cannot unmarshal string"，因为 `simcore.SceneStateSnapshot.RenderStateJSON` 是 `[]byte`，`encoding/json` 默认把 []byte 当 base64 字符串编码，下游解析需多一层 base64 decode。
3. **终端无输出**：`XTermTerminal.tsx` 的 `terminal.onData(onData)` 注册在 `useEffect(..., [readOnly])` 里只跑一次，捕获的 `handleTerminalData` 闭包里 `ready=false`，`if (!ready) return` 永远命中，键击全被丢弃。Stale closure 经典翻车。
4. **K8s 50011**：`backend/internal/service/experiment/k8s_client.go::DeployPod` 没设 `EnableServiceLinks: false`，K8s 默认把 namespace 内每个 Service 注入 `<NAME>_PORT=tcp://...`、`<NAME>_HOST=...` 等环境变量。geth CLI（urfave/cli）把 `--port` 绑到 `$GETH_PORT`，于是把 `tcp://10.x.x.x:8545` 当 int 解析直接 panic exit 1。remix-ide / blockscout 同理（kubectl logs 里 `could not parse "tcp://..." as int value from environment variable "GETH_PORT"`）。
5. **暂停后恢复报错**：`sim-engine/core/internal/app/engine.go::RestoreSnapshot` 先查内存 `runtime.snapshots[snapshotID]` 决定 snapshot 是否存在，但暂停链路把整个 SimEngine 会话销毁，新会话的内存映射必然为空。MinIO `e.store` 已经持久化了快照，却被这道前置检查阻断了跨会话恢复路径。
6. **blockscout Completed**：模板 8006 (EVM 全栈) seed 没配 postgres 容器（manifest `required_dependencies: [geth, postgres]`），blockscout Phoenix 启动时连不到 DB 立即退出。env 里也缺 `DATABASE_URL` / `SECRET_KEY_BASE`。

#### 🛠 修复
| # | 涉及文件 | 关键改动 |
|---|---------|---------|
| 1 | `frontend/src/components/business/SimEnginePanel.tsx` | `SceneCanvas` 加 `ResizeObserver` 把容器 px×devicePixelRatio 同步到 canvas.width/height + svg viewBox，每次变化调 `redrawScene` |
| 1 | `frontend/src/hooks/useSimPanel.ts` | panel 改用 `useState`，所有 callback 依赖 panel；解决子组件 effect 先于父组件创建 panel 的竞态 |
| 1 | `sim-engine/renderers/shared/simPanel.ts` | 新增 `redrawScene(sceneCode)`：尺寸变化后用 stateCache 当前状态强制重绘一帧 |
| 2 | `backend/internal/pkg/assertion/jsonpath.go`、`operator.go` | 新增通用 JSONPath 子集 (`$.a.b[0][*].length`) + 算子集 (`eq/ne/gt/gte/lt/lte/contains`)，与 CTF 模块 `challenge_assertions.operator` 共用 |
| 2 | `backend/internal/service/experiment/instance_service_checkpoint.go` | 重写 `evaluateSimAssertion` 走文档 DSL，结果按 `{passed, conditions[]}` 输出可读理由；删旧扁平比较实现 |
| 2 | `backend/seeds/005_seed_sim_scenarios.sql` | 21 条 SimEngine 断言全部按文档 DSL 重写；模板 8005/8006 的脚本检查点改用 `script_content + script_language + target_container` 列写真正可跑的 curl 脚本 |
| 2 | `sim-engine/core/internal/simcore/state_manager.go` | `SceneStateSnapshot.{StateJSON,RenderStateJSON,SharedStateJSON}` 由 `[]byte` 改 `json.RawMessage`，`BuildSceneSummary` 内联 struct 同步；消除 base64 编码污染 |
| 3 | `frontend/src/components/business/XTermTerminal.tsx` | `onData/onResize` 用 `useRef` 转发，xterm 注册的是稳定的 ref 解引用，组件 re-render 后新 callback 立即生效，无需重建终端 |
| 4 | `backend/internal/service/experiment/k8s_client.go` | `DeployPod` 的 PodSpec 加 `EnableServiceLinks: boolPtr(false)`，K8s 服务发现走 DNS，env 注入污染彻底关闭 |
| 4 | `sim-engine/core/internal/scene/k8s_orchestrator.go` | 同上，对场景算法容器 PodSpec 加同字段（防御性同步） |
| 5 | `sim-engine/core/internal/app/engine.go` | `RestoreSnapshot` 删除 `runtime.snapshots[id]` 内存检查，由 `e.store.Load(id)` 做唯一权威源；恢复路径不依赖会话生命周期。同步把无人读的 `runtime.snapshots` 字段及其写入语句一并清除（不留死代码） |
| 6 | `backend/seeds/004_seed_images_experiments.sql` | 模板 8006 加 postgres 容器（id 9033，startup_order=1）；blockscout 补 `DATABASE_URL` / `SECRET_KEY_BASE` / `ETHEREUM_JSONRPC_VARIANT` env 与 `depends_on: [geth, postgres]` |

#### ✅ 验证编译
- [x] `cd backend && go build ./...`
- [x] `cd sim-engine/core && go build ./...`
- [x] `cd sim-engine/renderers && npm run build`
- [x] `cd frontend && npx tsc --noEmit`

#### 🧪 验收步骤
> 必须先 `init-db.ps1` drop+recreate，再重启 backend 与 sim-engine Core，前端 HMR。

1. 学生端登录（13872945160 / LensChain2026） — ✅
2. **#3 终端**：以太坊本地开发与部署实践 → 终端 tab → `ls\r`、`pwd\r`
   - ✅ Playwright 抓帧确认 `~ $ ls` 回显、`pwd` → `/home/lenschain` 正确输出
3. **#4 K8s 部署**：EVM 全栈 DApp 启动 → `kubectl get pods -n exp-<id>`
   - ✅ geth Running，HTTP-RPC :8545 / WS :8546 启动正常（kubectl logs 无 GETH_PORT 解析报错）
4. **#1 仿真画布**：共识机制可视化对比 → 仿真 tab → 播放
   - ✅ Playwright 抓 4 个 canvas 全部 953×384 drawingbuffer，全画面有像素绘制（pixel count = w×h）
5. **#2 检查点**（待 sim-engine Core 重启后再跑）：进入仿真实验 → 推 tick 到对应阈值 → 检查点 tab → 验证全部
   - 期望：tick 达标的场景检查点显示"通过"且 `check_output` JSON 含 `{"passed":true,"conditions":[...]}`
6. **#5 暂停-恢复**（待 sim-engine Core 重启后跑）：纯仿真实验运行中 → 顶部"暂停"→ 等几秒 →"恢复"
   - 期望：实例恢复运行，无"snapshot not found"，sim 会话状态从 MinIO 快照恢复
7. **#6 blockscout**（待 init-db 已重跑 + EVM 全栈 重新启动）：
   - 期望：postgres + geth 同时 Running；blockscout 不再 Completed，能在 60s 内进入 Running，:4000 健康检查通过

#### 📋 测试详情
- **#1**：用户视觉确认"仿真未达预期"，本轮先确保 Canvas 真的在画；具体动画细腻度（PoW 算力柱图等）属于场景容器渲染数据丰富度，不在本轮代码 bug 范围。
- **#3**：浏览器拦截 WS 帧（前端注入 `WebSocket` patch）确认按键已发出 → 上游 PTY 已回显，环节链路全通。
- **#4**：geth pod logs 已经看到 `HTTP server started endpoint=[::]:8545`、`WebSocket enabled url=ws://[::]:8546`，env 污染问题彻底消除。
- **#5/#6**：根因已锁定，代码 / seed 修复已落地，编译通过，**等用户重启服务后回归**。

---

## 已修复（折叠，按时间倒序）

<!-- 每条一句话摘要 + 涉及核心文件 -->

- (示例) 2026-05-09 实验工具反代基础设施 + manifest sync 架构落地：`cmd/seed-manifests` / `image_manifest_sync.go` / `seeds/` 目录结构 / `init-db` 脚本编排

---

## 待办（非紧急）

- 仿真"动画质感"提升：当前 Canvas 已正确绘制，但用户期望的"算力柱状图竞速、Nonce 搜索动画、64 轮压缩函数逐轮高亮、椭圆曲线点乘动画"等需要在 `sim-engine/renderers/<domain>/` 下增强各领域渲染器，并要求场景算法容器在 `RenderEnvelope.data` 里输出对应粒度的中间状态。这是渲染器 + 场景容器协同的能力扩展，不是 bug。
- 仿真场景的"独有交互动作"：reactive 场景（SHA-256 / ECDSA / Merkle）的 `修改输入` / `重新签名` / `篡改叶子` 等按钮当前没在 UI 上渲染；接口契约 (`interaction_schema`) 已经在 `sim_scenarios.interaction_schema` 列里有数据，需要查 SimEngine WS 是否把 schema 推到前端、`SimEnginePanel` 的 `InteractionForm` 为何拿不到 actions 列表。

