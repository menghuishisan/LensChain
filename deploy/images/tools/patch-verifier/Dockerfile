# tools/patch-verifier/Dockerfile
# CTF 攻防赛补丁验证服务（平台自研）
# 负责验证队伍提交的补丁是否修复漏洞且不破坏功能
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
    -o /out/patch-verifier \
    ./cmd/verifier

FROM alpine:3.19

ARG VERSION=dev
LABEL org.opencontainers.image.title="patch-verifier"
LABEL org.opencontainers.image.description="CTF 攻防赛补丁验证服务"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.vendor="LensChain"
LABEL lenschain.io/category="tools"
LABEL lenschain.io/scope="ctf"

ENV TZ=UTC

RUN apk add --no-cache ca-certificates tzdata curl && \
    addgroup -S -g 1001 verifier && \
    adduser -S -u 1001 -G verifier verifier

COPY --from=builder --chown=verifier:verifier /out/patch-verifier /usr/local/bin/patch-verifier

USER verifier

EXPOSE 8091

HEALTHCHECK --interval=30s --timeout=5s --retries=3 --start-period=15s \
    CMD curl -fsS http://localhost:8091/healthz || exit 1

ENTRYPOINT ["/usr/local/bin/patch-verifier"]
