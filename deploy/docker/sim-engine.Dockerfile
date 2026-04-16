# sim-engine.Dockerfile
# SimEngine Core 平台服务镜像构建文件
# 负责构建 sim-engine/core/cmd/server 并提供 HTTP/gRPC 运行时

FROM golang:1.24-bookworm AS builder
WORKDIR /workspace

COPY sim-engine/core/go.mod sim-engine/core/go.sum ./sim-engine/core/
COPY sim-engine/proto/gen/go/go.mod ./sim-engine/proto/gen/go/
WORKDIR /workspace/sim-engine/core
RUN go mod download

WORKDIR /workspace
COPY sim-engine ./sim-engine

WORKDIR /workspace/sim-engine/core
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/sim-engine-core ./cmd/server

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=builder /out/sim-engine-core /app/sim-engine-core

EXPOSE 8090 9090
ENTRYPOINT ["/app/sim-engine-core"]
