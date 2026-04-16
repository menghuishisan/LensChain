# backend.Dockerfile
# 链镜平台后端服务镜像构建文件
# 负责构建 backend/cmd/server 并提供生产运行镜像

FROM golang:1.25-bookworm AS builder
WORKDIR /workspace

COPY backend/go.mod backend/go.sum ./backend/
COPY sim-engine/proto/gen/go/go.mod ./sim-engine/proto/gen/go/
WORKDIR /workspace/backend
RUN go mod download

WORKDIR /workspace
COPY backend ./backend
COPY sim-engine/proto ./sim-engine/proto

WORKDIR /workspace/backend
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/backend-server ./cmd/server

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=builder /out/backend-server /app/backend-server
COPY --from=builder /workspace/backend/configs /app/configs

ENV LENSCHAIN_CONFIG=/app/configs/config.yaml
EXPOSE 3000

ENTRYPOINT ["/app/backend-server"]
