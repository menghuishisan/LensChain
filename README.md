# LensChain

LensChain（链镜）是一个面向高校区块链教学场景的开源平台。它把课程教学、实验实践、可视化仿真、CTF 竞赛和成绩反馈整合在同一个系统里，帮助教师组织教学，也让学生能够在统一入口中完成学习、实验和演练。

如果你第一次打开这个仓库，可以把它理解成一套完整的区块链教学基础设施：

- 教师可以管理课程、课时、实验和竞赛
- 学生可以进入实验环境、完成仿真和提交结果
- 平台可以提供容器化实验、共享链基础设施和可视化仿真能力
- 学校或实验室可以按自己的环境部署和扩展

## 核心能力

- 课程教学：课程、章节、课时、作业与学习过程管理
- 实验环境：容器化实验、共享基础设施、分组协作、教师监控
- 可视化仿真：多领域区块链仿真场景、状态联动和交互演示
- CTF 竞赛：题目管理、报名、解题赛、攻防赛和排行榜
- 评测反馈：自动评分、手动评分、成绩审核和通知回传
- 多租户支持：以学校为单位进行数据和资源隔离

## 适合谁

- 想了解区块链教学平台长什么样的使用者
- 想本地运行一套教学平台的开发者
- 想扩展实验、仿真或竞赛能力的贡献者
- 想私有化部署平台的学校、实验室和教学团队

## 仓库结构

```text
LensChain/
├── backend/        # 平台后端服务
├── frontend/       # 平台 Web 前端
├── sim-engine/     # 可视化仿真引擎
├── deploy/         # 部署、镜像、Kubernetes、初始化脚本
├── docs/           # 项目设计文档
└── README.md
```

如果你要继续看具体子系统，可以直接进入：

- [backend/README.md](./backend/README.md)
- [frontend/README.md](./frontend/README.md)
- [sim-engine/README.md](./sim-engine/README.md)
- [deploy/README.md](./deploy/README.md)

## 快速开始

如果你想尽快把项目跑起来，推荐按下面顺序进行。

### 1. 启动本地依赖

```bash
docker compose -f deploy/docker-compose/docker-compose.dev.yml up -d
```

### 2. 初始化数据库和 demo 数据

```bash
./deploy/scripts/init-db.sh
```

### 3. 启动后端

```bash
cd backend
go run ./cmd/server
```

### 4. 启动前端

```bash
cd frontend
npm install
npm run dev
```

初始化完成后，数据库中会导入一套演示用数据，包括学校、教师、学生、课程、实验模板和镜像记录，适合本地联调和功能体验。

默认 demo 账号密码统一为：

```text
LensChain2026
```

## 文档入口

如果你想继续了解更完整的设计和模块边界，可以查看：

- [docs/00-项目总览与文档规范.md](./docs/00-项目总览与文档规范.md)
- [docs/项目功能总览.md](./docs/项目功能总览.md)
- [docs/API接口总览.md](./docs/API接口总览.md)
- [docs/数据库表总览.md](./docs/数据库表总览.md)

## 当前状态

LensChain 仍在持续开发中。仓库已经包含核心模块、初始化脚本和 demo 数据，但不同模块的实现细节仍会继续完善。

## Contributing

欢迎通过 Issue 和 Pull Request 参与改进，包括：

- 修复问题
- 改进 README 和文档
- 补充实验模板和仿真场景
- 完善部署与联调体验

如果你要做较大改动，建议先讨论范围和目标。

## License

当前仓库尚未发布正式 License。在将代码用于生产或对外分发前，请先确认项目后续的授权方式。
