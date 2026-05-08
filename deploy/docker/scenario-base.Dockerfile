# scenario-base.Dockerfile
# 链镜仿真场景算法容器基础镜像。
# 文档对齐：docs/modules/09-部署与运维/03-镜像与容器设计.md §4.4
#   - 体积 ≤ 30MB（alpine:3.19 + ca-certs + tzdata）
#   - 非 root（UID 1001 scenario）
#   - 容器内 gRPC 端口 50100
#
# 用途：
#   - 平台 43 个内置场景的共享运行时镜像（scenario.Dockerfile）通过 FROM 引用
#   - 教师自定义场景独立打包时通过 FROM 引用：
#       FROM registry.lianjing.com/lenschain/scenario-base:v1.0.0
#       COPY --from=builder /out/run /scenario/run

FROM alpine:3.19

LABEL org.opencontainers.image.title="lenschain-scenario-base"
LABEL org.opencontainers.image.description="链镜仿真场景算法容器基础镜像"
LABEL org.opencontainers.image.vendor="LensChain"
LABEL lenschain.io/service="scenario-base"

ENV TZ=UTC
# 与 sim-engine K8sOrchestrator 约定的容器内 gRPC 端口（50100）一致。
ENV SCENARIO_LISTEN_ADDR=":50100"

RUN for i in 1 2 3; do \
      apk add --no-cache ca-certificates tzdata && break; \
      echo ">>> apk add attempt $i failed, waiting 10s..."; sleep 10; \
    done && \
    addgroup -S -g 1001 scenario && \
    adduser -S -u 1001 -G scenario scenario && \
    mkdir -p /scenario && \
    chown -R scenario:scenario /scenario

WORKDIR /scenario
USER scenario
EXPOSE 50100

CMD ["/scenario/run"]
