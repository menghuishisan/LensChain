# frontend-deployment.md
# 前端部署接入规范

当前仓库尚未落地 `frontend/` 源码目录，但根据项目文档与 `CLAUDE.md`，`deploy/` 必须为前端服务保留完整部署接入位。

## 预期来源目录
- `frontend/`
- 预期技术栈：Next.js 14+ / React 18+ / App Router

## Docker 接入位
- 文件：`deploy/docker/frontend.Dockerfile`
- 当前状态：规范占位文件
- 后续要求：
  1. 构建上下文为仓库根目录
  2. 入口目录为 `frontend/`
  3. 支持生产构建与独立运行镜像
  4. 环境变量驱动 API Base URL、认证域、静态资源域

## Compose 接入位
- 当前 compose 未纳入 frontend 服务，因为源码尚未落地
- 源码落地后应补：
  - `frontend` service
  - 依赖 `backend`
  - 暴露默认端口 `3000`

## K8s 接入位
- 已存在：`deploy/k8s/base/platform/frontend-deployment.placeholder.yaml`
- 源码落地后应补充：
  - `frontend-service.yaml`
  - `frontend-configmap.yaml`
  - `frontend-ingress.yaml`
  - 前端特定 HPA/PDB（按实际需要）

## CI 接入位
- deploy 结构检查已要求存在 `deploy/docker/frontend.Dockerfile`
- 源码落地后应在 `deploy/ci/.github/workflows/deploy-build.yml` 中加入 frontend 镜像构建任务
