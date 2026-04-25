# deploy/ — 部署与运维开发规范与要求

> 本文件是 `deploy/` 目录下所有代码与配置的开发规范入口。  
> **设计文档是唯一真相源**，`deploy/` 的实现必须严格对应 `docs/modules/09-部署与运维/` 下的 5 份标准文档。  
> 本文件只回答 **“怎么写”**，不重复回答 **“为什么这么设计”**。

---

## 一、开发前必须阅读的文档

修改 `deploy/` 下任何内容前，必须先阅读以下文档：

| 场景 | 必读文档 |
|------|----------|
| 所有 `deploy/` 改动 | `docs/modules/09-部署与运维/01-功能需求说明.md` |
| 改端口 / 网络 / RBAC / Secret / PV / Registry | `docs/modules/09-部署与运维/02-基础设施设计.md` |
| 改 Dockerfile / 镜像标签 / manifest.yaml / CI 构建镜像 | `docs/modules/09-部署与运维/03-镜像与容器设计.md` |
| 改 K8s YAML / compose / GitHub Actions / 部署脚本 | `docs/modules/09-部署与运维/04-部署架构设计.md` |
| 改验收口径 / Smoke Test / Rollback / Backup | `docs/modules/09-部署与运维/05-验收标准.md` |

**规则：**
- 设计文档未写明的内容，**不要自行发挥**
- 如发现设计文档与实际需求冲突，**先改文档，再改 `deploy/` 实现**
- 不允许“代码先行、文档后补”

---

## 二、目录职责边界

严格遵守 `CLAUDE.md` 和模块09 文档：

| 目录 | 只允许放什么 | 禁止放什么 |
|------|-------------|-----------|
| `deploy/docker/` | 平台自身服务镜像（backend / frontend / sim-engine-core / scenario-base / collector-agent-*）Dockerfile | 实验环境镜像 Dockerfile |
| `deploy/images/` | 实验环境镜像（链节点 / 中间件 / 工具 / 基础开发环境）的 Dockerfile 和 `manifest.yaml` | 平台服务 Dockerfile |
| `deploy/docker-compose/` | 本地开发 compose 文件 | 生产部署文件 |
| `deploy/k8s/base/` | 环境中性的基础资源 | 具体环境值（域名、正式证书、生产镜像 tag） |
| `deploy/k8s/overlays/` | 各环境差异化补丁 | 重新定义一整套 base 资源 |
| `deploy/ci/` | CI/CD 流水线定义 | 应用代码 |
| `deploy/scripts/` | 运维脚本（初始化、备份、预拉取、种子化） | 业务逻辑代码 |

**禁止混放。**

---

## 三、Dockerfile 编写规范

### 3.1 通用要求

所有 Dockerfile 必须满足：

1. **文件头中文注释**：说明镜像用途、基础镜像、关键步骤
2. **显式基础镜像版本**：禁止 `latest`
3. **OCI LABEL 完整**
4. **统一时区**：`TZ=UTC`
5. **平台自研镜像必须非 root 运行**
6. **对外服务镜像必须有 `HEALTHCHECK`**
7. **显式 `EXPOSE` 端口**
8. **有 `.dockerignore`**

### 3.2 平台服务镜像要求

| 镜像 | 要求 |
|------|------|
| backend | 多阶段构建；runtime ≤ 100MB；用户非 root；暴露 8080 |
| frontend | Next.js standalone；runtime ≤ 250MB；暴露 3000 |
| sim-engine-core | 多阶段构建；暴露 50051 / 50052 |
| scenario-base | 极小镜像（≤ 30MB）；仅提供场景运行基础环境 |
| collector-agent-* | 四个镜像按生态拆分；通过构建 tag 编译适配器 |

### 3.3 实验环境镜像要求

- **官方镜像**：使用薄封装（`FROM 官方:固定版本` + LABEL + EXPOSE + HEALTHCHECK）
- **基础开发镜像 / 平台自研工具镜像**：完整自建，多阶段构建优先
- **CTF 漏洞镜像**：允许保留漏洞，但必须在 `manifest.yaml` 中标记 `security_scan.allow_critical=true`

### 3.4 `.dockerignore` 最低要求

每个镜像目录必须包含 `.dockerignore`，至少排除：

```
.git
.env
.env.local
node_modules
.next
dist
build
coverage
.vscode
.idea
*.pem
*.key
```

---

## 四、manifest.yaml 规范

每个 `deploy/images/**/` 目录下必须存在 `manifest.yaml`。

### 4.1 必填字段

- `name`
- `category`
- `ecosystem`
- `description`
- `official_source`
- `registry_project`
- `versions[]`
- `default_ports`
- `typical_companions`
- `documentation_url`
- `prepull_phase`
- `security_scan`

### 4.2 维护规则

- 改 Dockerfile 时，**同步检查 `manifest.yaml` 是否仍匹配**
- 改端口 / 默认环境变量 / 典型搭配时，**必须先改 `manifest.yaml`**
- `manifest.yaml` 是 `seed-images.sh` 的输入，字段不完整会导致 images 表错误

### 4.3 版本规则

- `versions[].tag` 与 Registry 实际标签必须一致
- 平台服务镜像使用 `vX.Y.Z`
- 实验环境镜像优先跟随上游版本（如 `1.14` / `2.5`）
- 不允许只保留 `latest`

---

## 五、K8s YAML 编写规范

### 5.1 base / overlay 划分规则

| 内容 | 放哪里 |
|------|--------|
| Deployment / Service / StatefulSet 的基础定义 | `base/` |
| 正式域名、证书、Secret、HPA 副本数 | `overlays/prod/` |
| 自签证书、单副本 | `overlays/staging/` |
| 最小资源、本地域名、关闭 TLS | `overlays/dev/` |

### 5.2 必备字段

所有 Deployment / StatefulSet / DaemonSet 必须包含：

- `app.kubernetes.io/name`
- `app.kubernetes.io/component`
- `app.kubernetes.io/part-of`
- `app.kubernetes.io/managed-by`
- `resources.requests`
- `resources.limits`
- `livenessProbe`
- `readinessProbe`
- 必要时 `startupProbe`
- `securityContext`
- `imagePullSecrets`

### 5.3 不允许的写法

- base 中写死生产域名
- base 中写明文 Secret
- 直接使用 `latest` 镜像 tag
- Deployment / StatefulSet 不写 resources
- 缺 probe 就上线
- 用 NodePort 暴露实验容器

### 5.4 Namespace 与 Network Policy

- 平台服务统一部署在 `lenschain` Namespace
- 实验实例 / CTF 队伍 Namespace 由业务代码动态创建，本目录只提供模板与 RBAC
- 所有实验 / CTF Namespace 默认 `deny-all`
- 仅按模块09 `02-基础设施设计.md §9` 明确的白名单放行

---

## 六、docker-compose 编写规范

`deploy/docker-compose/` 只维护本地开发环境。

### 6.1 范围

- 必含：PostgreSQL / Redis / MinIO / NATS
- 可选 profile：SimEngine Core
- 不默认启动：backend / frontend

### 6.2 约束

- 所有服务都要 `healthcheck`
- 数据目录全部挂到 `deploy/docker-compose/data/`
- 不允许生产专用配置（TLS、HA、HPA）混入 dev compose
- 端口必须与 `docs/modules/09-部署与运维/02-基础设施设计.md §2` 对齐

---

## 七、CI/CD 编写规范

### 7.1 路径触发

必须按路径触发流水线，不允许所有改动都跑全量流水线：

| 路径 | 流水线 |
|------|--------|
| `backend/**` | `backend.yml` |
| `frontend/**` | `frontend.yml` |
| `sim-engine/**` | `sim-engine.yml` |
| `deploy/images/**` | `images.yml` |
| `deploy/k8s/**` / `deploy/scripts/**` | `deploy-staging.yml` / `deploy-prod.yml` |

### 7.2 必做检查

所有镜像构建前必须完成：

- hadolint
- 类型检查 / lint / 单元测试（按项目类型）
- manifest schema 校验
- Trivy 漏洞扫描
- 生产镜像 Cosign 签名

### 7.3 Secret 管理

- GitHub Actions 中**禁止明文密钥**
- 全部通过 `secrets.*` 或外部 Secret Manager 注入
- PR 中如出现明文 token / password / key，直接拒绝

---

## 八、脚本编写规范

### 8.1 脚本要求

- shell 使用 `bash`
- 文件头说明用途
- `set -euo pipefail`
- 日志输出清晰（`==>` 前缀）
- 所有可配置项支持环境变量覆盖
- 幂等：重复执行不应破坏系统状态

### 8.2 脚本边界

| 脚本 | 职责 | 不负责 |
|------|------|--------|
| `init-db.sh` | 建库、迁移、种子数据 | 业务表结构定义 |
| `init-minio.sh` | 创建 Bucket、权限策略 | 文件上传逻辑 |
| `seed-images.sh` | 扫描 manifest，同步 images 表 | 业务接口实现 |
| `preload-images.sh` | 触发 DaemonSet 预拉取 | 实验实例启动 |
| `backup.sh` | 创建备份 Job | 业务层备份管理流程 |

---

## 九、与设计文档的对应关系

修改任何文件前，先确认它对应哪份设计文档：

| 路径 | 设计文档 |
|------|---------|
| `deploy/docker/**` | `03-镜像与容器设计.md` |
| `deploy/images/**` | `03-镜像与容器设计.md` |
| `deploy/k8s/base/**` | `02-基础设施设计.md` + `04-部署架构设计.md` |
| `deploy/k8s/overlays/**` | `04-部署架构设计.md` |
| `deploy/docker-compose/**` | `04-部署架构设计.md` |
| `deploy/ci/**` | `04-部署架构设计.md` |
| `deploy/scripts/**` | `04-部署架构设计.md` + `05-验收标准.md` |

如果代码改动找不到对应设计文档,先补文档再改代码。

---

## 十、提交前自查清单

每次提交 `deploy/` 相关改动前，必须自查：

### 10.1 Dockerfile / 镜像

- [ ] Dockerfile 有中文头注释
- [ ] 基础镜像版本固定
- [ ] 有 `.dockerignore`
- [ ] `manifest.yaml` 已同步更新
- [ ] 镜像标签与 Registry 命名规范一致
- [ ] 平台自研镜像非 root
- [ ] 健康检查已配置

### 10.2 K8s / compose

- [ ] `kustomize build` 可通过
- [ ] 资源 limits / requests 已声明
- [ ] 探针配置完整
- [ ] 未使用 `latest`
- [ ] base 不含生产环境值
- [ ] 端口与 `02-基础设施设计.md` 一致

### 10.3 CI / 脚本

- [ ] 路径触发正确
- [ ] Secret 未明文写入
- [ ] 幂等性已验证
- [ ] 验收标准对应项能被验证

### 10.4 文档同步

- [ ] 如果改了设计决策，已同步更新 `docs/modules/09-部署与运维/`
- [ ] 如果只是实现细节变化，已更新 `deploy/README.md`（如需要）
- [ ] 文档与代码无矛盾

---

## 十一、红线规则

1. **不允许跳过文档直接改 `deploy/`。**
2. **不允许把平台镜像和实验镜像混放。**
3. **不允许用 `latest` 当生产镜像版本。**
4. **不允许在 Git 中提交明文密钥 / 证书 / 凭据。**
5. **不允许让实验容器直接暴露 NodePort / LoadBalancer。**
6. **不允许绕过 Network Policy 让队伍之间直接互访。**
7. **不允许让未签名的生产镜像进入 prod。**
8. **不允许只改代码不改文档。**

---

## 十二、开发顺序建议

建议按以下顺序开发 `deploy/`：

1. `deploy/docker/` — 5 个平台镜像
2. `deploy/images/` — Phase 1 全部镜像 + manifest
3. `deploy/docker-compose/docker-compose.dev.yml`
4. `deploy/k8s/base/`
5. `deploy/k8s/overlays/dev`
6. `deploy/ci/` 流水线
7. `deploy/k8s/overlays/staging`
8. `deploy/scripts/`
9. `deploy/k8s/overlays/prod`

每完成一层，都要对照 `05-验收标准.md` 做验证，再进入下一层。
