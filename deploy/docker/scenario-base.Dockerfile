# scenario-base.Dockerfile
# 链镜仿真场景算法容器基础镜像（目标体积 ≤ 30MB）
# 供 43 个内置场景和教师自定义场景 FROM 使用
# 场景通过 gRPC SimScenario 接口与 SimEngine Core 通信

FROM alpine:3.19

LABEL org.opencontainers.image.title="lenschain-scenario-base"
LABEL org.opencontainers.image.description="链镜仿真场景算法容器基础镜像"
LABEL org.opencontainers.image.vendor="LensChain"
LABEL lenschain.io/service="scenario-base"

ENV TZ=UTC

RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -S scenario && adduser -S scenario -G scenario

WORKDIR /scenario

USER scenario

EXPOSE 50100

CMD ["/scenario/run"]
