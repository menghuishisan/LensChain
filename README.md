# LensChain

链镜（LensChain）是一个面向高校场景的区块链教学与实践平台，聚焦 **课程教学、实验实践、CTF 竞赛** 一体化建设，支持多学校、多角色、多模块扩展。

## 项目简介

当前仓库以 **Go 后端** 与 **项目文档体系** 为主，采用文档驱动开发方式推进。

已包含：
- 用户与认证模块基础实现
- 学校与租户管理模块基础实现
- 完整的项目规划、模块文档、API 与数据库规范
- PostgreSQL / Redis / MinIO / NATS 等基础设施接入骨架

## 技术栈

### 后端
- Go 1.24
- Gin
- GORM
- PostgreSQL
- Redis
- MinIO
- NATS
- Zap
- Viper

### 规划中的整体技术栈
- 前端：React / Next.js
- 容器编排：Docker / Kubernetes
- 对象存储：MinIO / S3 Compatible

## 目录结构

```text
.
├── backend/               # Go 后端服务
│   ├── cmd/               # 服务入口与命令
│   ├── configs/           # 配置文件
│   ├── internal/          # 业务实现
│   ├── migrations/        # 数据库迁移脚本
│   ├── go.mod
│   └── go.sum
├── docs/                  # 项目文档
│   ├── modules/           # 模块设计文档
│   ├── standards/         # 开发规范
│   ├── 00-项目总览与文档规范.md
│   ├── API接口总览.md
│   ├── 数据库表总览.md
│   └── 项目功能总览.md
└── CLAUDE.md
```

## 已实现/当前状态

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

以下模块在代码中预留了位置，后续逐步实现：
- 课程与教学
- 实验环境
- CTF 竞赛
- 评测与成绩
- 通知与消息
- 系统管理与监控

## 本地运行

### 1. 准备依赖
请先准备：
- Go 1.24+
- PostgreSQL
- Redis
- MinIO
- NATS

### 2. 修改配置
默认配置文件：`backend/configs/config.yaml`

也可以通过环境变量覆盖，前缀为：
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

服务默认监听：
- `0.0.0.0:3000`

## 数据库迁移

迁移命令入口位于：
- `backend/cmd/migrate`

如需运行，请根据该命令实现与本地数据库配置执行对应迁移。

## 文档入口

建议优先阅读以下文档：
- `docs/00-项目总览与文档规范.md`
- `docs/项目功能总览.md`
- `docs/API接口总览.md`
- `docs/数据库表总览.md`
- `docs/standards/API规范.md`
- `docs/standards/数据库规范.md`

## 开发说明

本项目采用 **文档驱动开发（Documentation-Driven Development）**：
1. 先确认模块文档
2. 再落地数据库与接口设计
3. 最后进入开发与联调

## 安全提醒

当前仓库中的 `backend/configs/config.yaml` 主要用于本地开发配置，包含默认开发配置，例如：
- 数据库连接参数
- JWT secret 示例值
- MinIO 默认账号

如果仓库用于公开协作，建议后续改为：
- 提交 `backend/configs/config.example.yaml`
- 将真实配置放入本地环境变量或私有配置文件

## License

暂未添加 License。
