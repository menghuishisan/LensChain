# tools/prometheus/Dockerfile
# Prometheus 指标采集服务（基于官方镜像薄封装）
# 实验环境内的指标采集和存储

ARG VERSION=v2.51.0
FROM prom/prometheus:${VERSION}

ARG VERSION
LABEL org.opencontainers.image.title="prometheus"
LABEL org.opencontainers.image.description="Prometheus 指标采集服务"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.source="https://github.com/prometheus/prometheus"
LABEL org.opencontainers.image.vendor="LensChain"
LABEL lenschain.io/ecosystem="general"
LABEL lenschain.io/category="tools"

ENV TZ=UTC

EXPOSE 9090

HEALTHCHECK --interval=30s --timeout=5s --retries=3 --start-period=15s \
    CMD wget -qO- http://localhost:9090/-/healthy || exit 1
