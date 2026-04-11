# LensChain

**链镜** 是一个覆盖高校场景的区块链综合教学平台，集 **教学、实验实践、CTF 竞赛** 三位一体，支持多链生态、多层次学生、多学校混合部署。

## 项目简介

当前仓库以 **Go 后端服务** 与 **项目文档体系** 为主，采用文档驱动开发方式推进。

平台围绕高校区块链教学场景，提供从学校入驻、用户认证、课程教学到实验实践和竞赛训练的统一能力。当前后端已包含以下基础业务能力：

- 用户与认证
- 学校与租户管理
- 课程与教学

后续模块将在现有规范和接口体系上继续扩展。

## 技术栈

### 后端

- Go
- Gin
- GORM
- PostgreSQL
- Redis
- MinIO
- NATS
- Zap
- Viper
- robfig/cron

### 规划中的整体技术栈

- 前端：React / Next.js
- 容器化：Docker
- 编排部署：Kubernetes
- 对象存储：MinIO / S3 Compatible

## 目录结构

```text
.
├── backend/               # Go 后端服务
│   ├── cmd/               # 服务入口与命令
│   │   ├── migrate/       # 数据库迁移命令入口
│   │   └── server/        # HTTP 服务入口与模块初始化
│   ├── configs/           # 本地开发配置
│   ├── internal/          # 后端内部实现
│   │   ├── handler/       # HTTP 处理层
│   │   ├── middleware/    # 中间件
│   │   ├── model/         # DTO、实体、枚举
│   │   ├── pkg/           # 基础设施封装
│   │   ├── repository/    # 数据访问层
│   │   ├── router/        # 路由注册
│   │   └── service/       # 业务逻辑层
│   ├── migrations/        # 数据库迁移脚本
│   ├── go.mod
│   └── go.sum
├── docs/                  # 项目文档
│   ├── modules/           # 模块设计文档
│   ├── standards/         # API、数据库等规范
│   ├── 00-项目总览与文档规范.md
│   ├── API接口总览.md
│   ├── 数据库表总览.md
│   └── 项目功能总览.md
├── CLAUDE.md              # 项目开发规范与目录职责说明
└── README.md
```

## 当前状态

当前后端入口位于：

- `backend/cmd/server/main.go`

目前代码中已接入或初始化的基础组件：

- 配置加载
- 日志系统
- 雪花 ID
- PostgreSQL
- Redis
- MinIO
- NATS
- WebSocket 管理器
- 定时任务调度器

当前已挂载的业务模块：

- Auth（用户与认证）
- School（学校与租户管理）
- Course（课程与教学）

以下模块在代码中预留了位置，后续逐步实现：

- Experiment（实验环境）
- CTF（CTF 竞赛）
- Grade（评测与成绩）
- Notification（通知与消息）
- System（系统管理与监控）

## 本地运行

### 1. 准备依赖

请先准备：

- Go
- PostgreSQL
- Redis
- MinIO
- NATS

### 2. 修改配置

默认配置文件：

- `backend/configs/config.yaml`

也可以通过环境变量覆盖配置，前缀为：

- `LENSCHAIN_`

例如：

- `LENSCHAIN_DATABASE_HOST`

> 建议不要在生产环境中直接使用仓库中的默认密钥与默认密码。

### 3. 启动服务

在项目根目录执行：

```bash
cd backend
go run ./cmd/server
```

服务监听地址以配置文件为准，默认本地开发配置通常为：

- `0.0.0.0:3000`

## 数据库迁移

迁移脚本位于：

- `backend/migrations`

迁移命令入口位于：

- `backend/cmd/migrate`

请根据本地数据库配置执行对应迁移。

## 运行检查

在后端目录执行：

```bash
go test ./...
```

如需指定本地 Go 构建缓存目录，可以使用：

```bash
mkdir -p .gocache
GOCACHE="$(pwd)/.gocache" go test ./...
```

`backend/.gocache/` 属于本地缓存目录，已加入 `.gitignore`，不会提交到仓库。

## 文档入口

建议优先阅读以下文档：

- `docs/00-项目总览与文档规范.md`
- `docs/项目功能总览.md`
- `docs/API接口总览.md`
- `docs/数据库表总览.md`
- `docs/standards/API规范.md`
- `docs/standards/数据库规范.md`
- `docs/modules/01-用户与认证/`
- `docs/modules/02-学校与租户管理/`
- `docs/modules/03-课程与教学/`
- `CLAUDE.md`

## 开发说明

本项目采用 **文档驱动开发（Documentation-Driven Development）**：

1. 先确认模块功能需求、API 设计、数据库设计和验收标准
2. 再落地数据库迁移、实体、DTO、repository、service 和 handler
3. 后端实现遵循 handler / service / repository 分层职责
4. 跨模块调用通过接口与 adapter 解耦
5. 数据库字段、迁移脚本、实体结构和接口 DTO 需要保持一致

## 安全提醒

当前仓库中的 `backend/configs/config.yaml` 主要用于本地开发配置，可能包含默认开发配置，例如：

- 数据库连接参数
- JWT secret 示例值
- MinIO 默认账号

如果仓库用于公开协作，建议后续改为：

- 提交 `backend/configs/config.example.yaml`
- 将真实配置放入本地环境变量或私有配置文件
- 在生产环境中启用 HTTPS、独立密钥和更严格的访问控制策略

## License

暂未添加 License。
