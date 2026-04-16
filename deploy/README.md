# deploy/ — 部署与运维目录

> 本目录是链镜平台的部署与运行时交付目录，负责平台服务镜像、实验环境与 CTF 竞赛环境镜像、本地编排、Kubernetes 部署与 CI/CD 配置。

## 目录职责

```text
deploy/
├── docker/                 # 平台服务 Dockerfile
├── images/                 # 模块04/05 运行时镜像定义
├── docker-compose/         # 本地/单机编排
├── k8s/                    # Kubernetes 基础清单与环境差异化配置
└── ci/                     # CI/CD 工作流
```

### `docker/`
只放平台自身服务镜像：
- backend
- sim-engine
- scenario 基础镜像
- frontend（当前代码未落地，仅预留规范接入位）

### `images/`
只放业务运行环境镜像：
- 模块04 实验环境镜像
- 模块05 CTF 竞赛环境镜像

严禁把平台服务 Dockerfile 放进 `images/`，也严禁把链节点/靶机/工具镜像放进 `docker/`。

### `docker-compose/`
用于本地开发、单机联调、演示环境编排。

### `k8s/`
用于长期平台服务与共享基础设施的 Kubernetes 清单，以及 dev/staging/prod 差异化 overlay。

### `ci/`
用于镜像构建、部署配置校验与交付流水线。

## 设计约束

- 文档驱动：目录设计必须同时满足模块04《实验环境》和模块05《CTF竞赛》文档要求。
- 职责单一：平台服务部署与实验/竞赛运行环境严格分离。
- 可扩展：即使 `frontend/` 当前不存在，`deploy/` 中也必须为其保留规范接入位。
- 共享编排：模块04 与模块05 共享环境编排服务层，镜像体系与 K8s 运行时模板需可复用。

## 当前实现范围

- 平台服务 Dockerfile：backend、sim-engine、scenario、frontend 占位规范
- 本地 compose：backend + sim-engine + PostgreSQL + Redis + MinIO + NATS
- K8s：平台基础清单、运行时模板、环境 overlay
- 镜像体系：experiment / ctf 两套运行时目录与首批镜像定义
- CI：镜像构建与配置校验工作流
