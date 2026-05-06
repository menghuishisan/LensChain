# backend.Dockerfile
# 链镜平台后端 Go API 服务镜像
# 多阶段构建：golang:1.25-alpine 构建 → alpine:3.19 运行
# 监听端口：8080

# ============================
# 构建阶段
# ============================
FROM golang:1.25-alpine AS builder

ARG VERSION=dev
ARG COMMIT_SHA=unknown

WORKDIR /src

ENV GOPROXY=https://goproxy.cn,direct

# 依赖层（利用 Docker cache）
COPY backend/go.mod backend/go.sum ./
COPY sim-engine/proto/gen/go /sim-engine/proto/gen/go
RUN go mod download

# 源码层
COPY backend/ .

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w -X main.Version=${VERSION} -X main.Commit=${COMMIT_SHA}" \
    -o /out/lenschain-backend \
    ./cmd/server

# ============================
# 运行阶段
# ============================
FROM alpine:3.19

ARG VERSION=dev
ARG COMMIT_SHA=unknown

LABEL org.opencontainers.image.title="lenschain-backend"
LABEL org.opencontainers.image.description="链镜平台后端 Go API 服务"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.revision="${COMMIT_SHA}"
LABEL org.opencontainers.image.vendor="LensChain"
LABEL lenschain.io/service="backend"

ENV TZ=UTC

RUN for i in 1 2 3; do \
      apk add --no-cache ca-certificates tzdata curl && break; \
      echo ">>> apk add attempt $i failed, waiting 10s..."; sleep 10; \
    done && \
    addgroup -S -g 1001 lenschain && \
    adduser -S -u 1001 -G lenschain lenschain && \
    mkdir -p /app/configs /app/logs && \
    chown -R lenschain:lenschain /app

WORKDIR /app

COPY --from=builder --chown=lenschain:lenschain /out/lenschain-backend /app/lenschain-backend

USER lenschain

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --retries=3 --start-period=20s \
    CMD curl -fsS http://localhost:8080/healthz || exit 1

ENV LENSCHAIN_CONFIG=/app/configs/config.yaml

ENTRYPOINT ["/app/lenschain-backend"]
