# scenario.Dockerfile
# SimEngine 场景算法容器通用镜像构建文件
# 通过 build arg 指定场景目录，构建单个场景算法容器

FROM golang:1.24-bookworm AS builder
ARG SCENARIO_PATH=consensus/pbft-consensus
WORKDIR /workspace

COPY sim-engine/scenarios/go.mod sim-engine/scenarios/go.sum ./sim-engine/scenarios/
COPY sim-engine/sdk/go/go.mod ./sim-engine/sdk/go/
COPY sim-engine/proto/gen/go/go.mod ./sim-engine/proto/gen/go/
WORKDIR /workspace/sim-engine/scenarios
RUN go mod download

WORKDIR /workspace
COPY sim-engine ./sim-engine

WORKDIR /workspace/sim-engine/scenarios
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/scenario-server ./$(printf "%s" "$SCENARIO_PATH")

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=builder /out/scenario-server /app/scenario-server

ENV SCENARIO_LISTEN_ADDR=:8080
EXPOSE 8080
ENTRYPOINT ["/app/scenario-server"]
