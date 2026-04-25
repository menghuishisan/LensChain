# Deploy

`deploy/` 目录负责 LensChain 的部署、镜像构建、Kubernetes 清单、CI/CD 流水线和初始化脚本。

如果你想知道：

- 本地开发依赖怎么启动
- 数据库怎么初始化
- demo 数据怎么导入
- 镜像怎么同步和完整拉取
- Kubernetes 怎么部署
- Secret / registry 要怎么配置

都应该从这个目录开始。

建议先阅读：

- [config.example.env](/abs/path/E:/code/LensChain/deploy/config.example.env)

这份文件统一列出了部署、初始化、镜像拉取和 Secret 相关的常见配置项，适合作为 dev / staging / prod 的配置模板。

## 目录结构

```text
deploy/
├── docker/           # 平台服务镜像与运维运行时镜像 Dockerfile
├── images/           # 实验环境镜像定义与 manifest
├── docker-compose/   # 本地开发依赖
├── k8s/              # Kubernetes base 与 overlays
├── ci/               # CI/CD workflow
├── scripts/          # 初始化、备份、预拉取、种子化脚本
└── README.md
```

## 各目录是做什么的

- `docker/`
  平台后端、前端、SimEngine 以及运维镜像的 Dockerfile

- `images/`
  实验环境镜像和 `manifest.yaml`

- `docker-compose/`
  本地开发依赖，例如 PostgreSQL、Redis、MinIO、NATS

- `k8s/`
  Kubernetes 资源，按 `base/` 和 `overlays/` 组织

- `ci/`
  GitHub Actions 流水线

- `scripts/`
  初始化数据库、同步镜像清单、预拉取镜像等脚本

## 本地开发怎么起依赖

```bash
docker compose -f deploy/docker-compose/docker-compose.dev.yml up -d
```

可选地启动 SimEngine：

```bash
docker compose -f deploy/docker-compose/docker-compose.dev.yml --profile full up -d
```

## 数据库怎么初始化

初始化入口是：

- [deploy/scripts/init-db.sh](/abs/path/E:/code/LensChain/deploy/scripts/init-db.sh)

它负责：

1. 检查 PostgreSQL 连接
2. 创建数据库
3. 执行 `backend/migrations/` 下的结构迁移
4. 执行 demo 种子数据

说明：

- 结构迁移由 `backend/cmd/migrate/main.go up` 执行
- demo 数据来自 [backend/migrations/010_seed_demo_data.up.sql](/abs/path/E:/code/LensChain/backend/migrations/010_seed_demo_data.up.sql)
- 也就是说，初始化脚本会**先建表，再灌种子数据**

示例：

```bash
./deploy/scripts/init-db.sh
```

如果你本机端口不是默认值，可以覆盖：

```bash
DB_HOST=localhost DB_PORT=5442 DB_USER=lenschain DB_PASSWORD=lenschain DB_NAME=lenschain ./deploy/scripts/init-db.sh
```

## Demo 数据

演示 / 联调用的数据库种子数据放在：

- [backend/migrations/010_seed_demo_data.up.sql](/abs/path/E:/code/LensChain/backend/migrations/010_seed_demo_data.up.sql)

这份数据会初始化：

- 学校
- 教师 / 学生 / 学校管理员账号
- 镜像分类 / 镜像 / 镜像版本
- 课程、章节、课时
- 单人实验模板
- 共享基础设施实验模板
- 课程与实验关联
- 选课数据

它的目标是让前后端、实验环境和部署联调后直接有可用内容，而不是空库。

默认 demo 账号统一密码：

```text
LensChain2026
```

## 镜像清单怎么同步

镜像清单同步脚本：

- [deploy/scripts/seed-images.sh](/abs/path/E:/code/LensChain/deploy/scripts/seed-images.sh)

它会扫描 `deploy/images/**/manifest.yaml`，再通过后端 API 同步到数据库中的 `images` 和 `image_versions` 表。

使用前需要：

- 后端服务已启动
- 提供有权限的 `ADMIN_TOKEN`

示例：

```bash
BACKEND_URL=http://localhost:8080/api/v1 ADMIN_TOKEN='<token>' ./deploy/scripts/seed-images.sh
```

## 镜像怎么完整拉取 / 预拉取

### 1. 平台服务镜像

平台服务镜像由：

- `deploy/docker/`
- `deploy/ci/.github/workflows/*.yml`

管理和构建。

### 2. 实验环境镜像

实验环境镜像定义在：

- `deploy/images/**/manifest.yaml`

运行时依赖：

- `image_versions.registry_url`
- 集群中的 `registry-pull-secret`

### 3. 预拉取

预拉取相关清单位于：

- `deploy/k8s/base/daemonset/image-prepuller.yaml`
- `deploy/k8s/base/daemonset/image-prepuller-configmap.yaml`

手动触发方式：

```bash
./deploy/scripts/preload-images.sh
```

## 统一配置示例

下面这份示例描述的是部署时最常需要准备的配置。不要再分别维护多套说明，直接以此为准。

统一示例文件见：

- [config.example.env](/abs/path/E:/code/LensChain/deploy/config.example.env)

### 1. backend-secret

需要的键：

- `database-password`
- `jwt-access-secret`
- `jwt-refresh-secret`

生成示例：

```bash
kubectl -n lenschain create secret generic backend-secret \
  --from-literal=database-password='<db-password>' \
  --from-literal=jwt-access-secret='<access-secret>' \
  --from-literal=jwt-refresh-secret='<refresh-secret>' \
  --dry-run=client -o yaml > backend-secret.yaml
```

### 2. postgres-secret

需要的键：

- `password`

```bash
kubectl -n lenschain create secret generic postgres-secret \
  --from-literal=password='<postgres-password>' \
  --dry-run=client -o yaml > postgres-secret.yaml
```

### 3. minio-secret

需要的键：

- `root-user`
- `root-password`

```bash
kubectl -n lenschain create secret generic minio-secret \
  --from-literal=root-user='minioadmin' \
  --from-literal=root-password='<minio-password>' \
  --dry-run=client -o yaml > minio-secret.yaml
```

### 4. registry-pull-secret

类型必须是 `kubernetes.io/dockerconfigjson`。

```bash
kubectl -n lenschain create secret docker-registry registry-pull-secret \
  --docker-server=registry.lianjing.com \
  --docker-username='<registry-user>' \
  --docker-password='<registry-password>' \
  --dry-run=client -o yaml > registry-pull-secret.yaml
```

说明：

- 平台静态 deployment 需要它
- 模块04/05 运行时动态创建的 namespace 也会复用它
- 如果没有这个 Secret，运行时 pod 可能会 `ImagePullBackOff`

## Kubernetes 怎么部署

### base

`deploy/k8s/base/` 提供环境中性的资源定义。

### overlays

- `deploy/k8s/overlays/dev/`
- `deploy/k8s/overlays/staging/`
- `deploy/k8s/overlays/prod/`

示例：

```bash
kubectl apply -k deploy/k8s/overlays/staging/
```

## rollout / smoke / rollback

### rollout

```bash
kubectl -n lenschain rollout status deployment/backend --timeout=10m
kubectl -n lenschain rollout status deployment/frontend --timeout=10m
kubectl -n lenschain rollout status deployment/sim-engine --timeout=10m
```

### smoke

后端探针端点：

- `/healthz`
- `/readyz`
- `/startupz`

示例：

```bash
kubectl run smoke-test --rm -i --restart=Never --image=curlimages/curl:8.5.0 -- \
  curl -fsS "http://backend.lenschain.svc.cluster.local:8080/healthz"
```

### rollback

```bash
kubectl -n lenschain rollout undo deployment/backend
```

## 适合谁阅读

- 部署者
- 运维人员
- 需要本地搭开发依赖的开发者
- 需要做镜像、K8s、CI 联调的工程师

如果你只想了解项目本身，请先看根目录 [README.md](/abs/path/E:/code/LensChain/README.md)。
