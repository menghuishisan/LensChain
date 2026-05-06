# collector-agent.Dockerfile
# 链镜混合实验数据采集 sidecar 镜像
# 按链生态通过 build-arg ADAPTER 编译不同适配器（ethereum / fabric / chainmaker / fisco）
# 对应 config.yaml 的 collector_image_template: registry.lianjing.com/lenschain/collector-agent-%s:v1.0.0
# 构建上下文：sim-engine/core/
# 体积上限 ≤ 50MB

# ============================
# 构建阶段
# ============================
FROM golang:1.25-alpine AS builder

ARG ADAPTER=ethereum
ARG VERSION=dev
ARG COMMIT_SHA=unknown

WORKDIR /src

ENV GOPROXY=https://goproxy.cn,direct

COPY core/go.mod core/go.sum ./
COPY proto/gen/go /proto/gen/go
RUN go mod download

COPY core/ .

RUN CGO_ENABLED=0 GOOS=linux go build \
    -tags ${ADAPTER} \
    -ldflags="-s -w -X main.Version=${VERSION} -X main.Adapter=${ADAPTER}" \
    -o /out/collector \
    ./cmd/collector-agent

# ============================
# 运行阶段
# ============================
FROM alpine:3.19

ARG ADAPTER=ethereum
ARG VERSION=dev

LABEL org.opencontainers.image.title="lenschain-collector-agent-${ADAPTER}"
LABEL org.opencontainers.image.description="链镜混合实验 ${ADAPTER} 链数据采集 sidecar"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.vendor="LensChain"
LABEL lenschain.io/service="collector-agent"
LABEL lenschain.io/adapter="${ADAPTER}"

ENV TZ=UTC

RUN for i in 1 2 3; do \
      apk add --no-cache ca-certificates tzdata && break; \
      echo ">>> apk add attempt $i failed, waiting 10s..."; sleep 10; \
    done && \
    addgroup -S collector && adduser -S collector -G collector

COPY --from=builder --chown=collector:collector /out/collector /usr/local/bin/collector

USER collector

ENTRYPOINT ["/usr/local/bin/collector"]
