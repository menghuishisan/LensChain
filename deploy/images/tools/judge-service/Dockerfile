# tools/judge-service/Dockerfile
# CTF 攻防赛裁判结算服务（平台自研）
# 负责代理执行攻击、验证结果、Token 结算
# 多阶段构建：golang:1.22-alpine 构建 → alpine:3.19 运行

FROM golang:1.22-alpine AS builder

ARG VERSION=dev
ARG COMMIT_SHA=unknown

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w -X main.Version=${VERSION}" \
    -o /out/judge-service \
    ./cmd/judge

FROM alpine:3.19

ARG VERSION=dev
LABEL org.opencontainers.image.title="judge-service"
LABEL org.opencontainers.image.description="CTF 攻防赛裁判结算服务"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.vendor="LensChain"
LABEL lenschain.io/category="tools"
LABEL lenschain.io/scope="ctf"

ENV TZ=UTC

RUN apk add --no-cache ca-certificates tzdata curl && \
    addgroup -S -g 1001 judge && \
    adduser -S -u 1001 -G judge judge

COPY --from=builder --chown=judge:judge /out/judge-service /usr/local/bin/judge-service

USER judge

EXPOSE 8090

HEALTHCHECK --interval=30s --timeout=5s --retries=3 --start-period=15s \
    CMD curl -fsS http://localhost:8090/healthz || exit 1

ENTRYPOINT ["/usr/local/bin/judge-service"]
