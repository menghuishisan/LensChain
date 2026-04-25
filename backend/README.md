# Backend

`backend/` 是 LensChain 的后端服务。它负责用户认证、课程与教学、实验环境、CTF 竞赛、成绩反馈、通知和系统管理等核心业务能力。

如果你想：

- 启动后端服务
- 连接数据库与缓存
- 初始化数据
- 查看接口和业务实现

那么应该从这个目录开始。

## 目录结构

```text
backend/
├── cmd/            # 程序入口
├── internal/       # 业务实现
├── configs/        # 配置文件
├── migrations/     # 数据库迁移与 demo 数据
├── pkg/            # 公共能力
└── README.md
```

## 后端主要负责什么

- 用户与认证
- 学校与租户管理
- 课程与教学
- 实验环境编排与运行时管理
- CTF 竞赛和题目环境
- 成绩、通知和系统管理

## 本地启动

```bash
cd backend
go run ./cmd/server
```

默认配置文件：

- [configs/config.yaml](./configs/config.yaml)

如果你要指定自定义配置：

```bash
$env:LENSCHAIN_CONFIG="E:\\path\\to\\config.yaml"
go run ./cmd/server
```

## 数据库迁移与 demo 数据

数据库相关文件位于：

- [migrations/](./migrations)

其中包括：

- 结构迁移 SQL
- 一份可重复导入的 demo 数据 SQL

初始化命令：

```bash
./deploy/scripts/bash/init-db.sh
```

```powershell
.\deploy\scripts\powershell\init-db.ps1
```

默认 demo 账号密码统一为：

```text
LensChain2026
```

## 常用命令

```bash
cd backend
go run ./cmd/server
go test ./...
```

## 适合谁阅读

- 想本地运行后端的人
- 后端开发者
- 负责前后端联调的人

如果你只想先了解整个项目，请回到根目录 [README.md](/abs/path/E:/code/LensChain/README.md)。
