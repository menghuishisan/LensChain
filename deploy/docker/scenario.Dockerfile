# scenario.Dockerfile
# 链镜平台 43 个内置仿真场景的共享运行时镜像。
#
# 设计：所有内置场景共享同一个二进制（catalog.NewRegistry 一次性注册全部 43 场景），
# 启动时通过 SCENE_CODE 环境变量选择运行哪个场景定义；
# K8s SceneManager (K8sOrchestrator) 在创建场景 Pod 时从 sim_scenarios.code 注入。
#
# 文档对齐：docs/modules/04-实验环境/06.1-场景编排实施方案.md
#
# 教师自定义场景**不**使用本文件，而是基于 scenario-base.Dockerfile 构建独立镜像。
#
# 构建（在仓库根目录执行）：
#   docker build \
#     -t registry.lianjing.com/scenarios/runtime:v1.0.0 \
#     -f deploy/docker/scenario.Dockerfile \
#     sim-engine

# ============================
# 构建阶段
# ============================
FROM golang:1.25-alpine AS builder

ARG VERSION=dev
ARG COMMIT_SHA=unknown

WORKDIR /src
ENV GOPROXY=https://goproxy.cn,direct

# scenarios 模块通过 replace 引用 framework、sdk/go 与 proto/gen/go：
#   replace github.com/lenschain/sim-engine/framework => ../framework
#   replace github.com/lenschain/sim-engine/sdk/go => ../sdk/go
#   replace github.com/lenschain/sim-engine/proto/gen/go => ../proto/gen/go
# 相对路径 ../ 从 /src/scenarios/ 出发指向同级目录，
# 因此目录布局必须与宿主源码保持一致。
COPY scenarios/go.mod scenarios/go.sum ./scenarios/
COPY framework ./framework
COPY sdk/go ./sdk/go
COPY proto/gen/go ./proto/gen/go

WORKDIR /src/scenarios
RUN go mod download

COPY scenarios/ .

RUN CGO_ENABLED=0 GOOS=linux go build \
      -ldflags="-s -w -X main.Version=${VERSION} -X main.Commit=${COMMIT_SHA}" \
      -o /out/scenario \
      ./cmd/scenario

# ============================
# 运行阶段
# ============================
FROM registry.lianjing.com/lenschain/scenario-base:v1.0.0

ARG VERSION=dev
ARG COMMIT_SHA=unknown

LABEL org.opencontainers.image.title="lenschain-scenarios-runtime"
LABEL org.opencontainers.image.description="链镜内置仿真场景共享运行时镜像（43 场景）"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.revision="${COMMIT_SHA}"
LABEL lenschain.io/service="scenarios-runtime"

COPY --from=builder --chown=scenario:scenario /out/scenario /scenario/run

# scenario-base 已声明 ENTRYPOINT/CMD/USER/EXPOSE，无需重复
