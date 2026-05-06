# sim-engine-core.Dockerfile
# 链镜 SimEngine Core 仿真引擎微服务镜像
# 多阶段构建：golang:1.25-alpine 构建 → alpine:3.19 运行
# 监听端口：50051 (gRPC) + 50052 (WebSocket)

# ============================
# 构建阶段
# ============================
FROM golang:1.25-alpine AS builder

ARG VERSION=dev
ARG COMMIT_SHA=unknown

WORKDIR /src

ENV GOPROXY=https://goproxy.cn,direct

COPY core/go.mod core/go.sum ./
COPY proto/gen/go /proto/gen/go
RUN go mod download

COPY core/ .

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w -X main.Version=${VERSION} -X main.Commit=${COMMIT_SHA}" \
    -o /out/sim-engine-core \
    ./cmd/server

# ============================
# 运行阶段
# ============================
FROM alpine:3.19

ARG VERSION=dev
ARG COMMIT_SHA=unknown

LABEL org.opencontainers.image.title="lenschain-sim-engine-core"
LABEL org.opencontainers.image.description="链镜 SimEngine 仿真引擎 Core 微服务"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.revision="${COMMIT_SHA}"
LABEL org.opencontainers.image.vendor="LensChain"
LABEL lenschain.io/service="sim-engine-core"

ENV TZ=UTC

RUN for i in 1 2 3; do \
      apk add --no-cache ca-certificates tzdata curl && break; \
      echo ">>> apk add attempt $i failed, waiting 10s..."; sleep 10; \
    done && \
    addgroup -S -g 1001 lenschain && \
    adduser -S -u 1001 -G lenschain lenschain && \
    mkdir -p /app && \
    chown -R lenschain:lenschain /app

WORKDIR /app

COPY --from=builder --chown=lenschain:lenschain /out/sim-engine-core /app/sim-engine-core

USER lenschain

EXPOSE 50051 50052

HEALTHCHECK --interval=30s --timeout=5s --retries=3 --start-period=15s \
    CMD wget -qO- http://localhost:50052/healthz || exit 1

ENTRYPOINT ["/app/sim-engine-core"]
