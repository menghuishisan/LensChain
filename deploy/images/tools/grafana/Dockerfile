# tools/grafana/Dockerfile
# Grafana 可视化监控仪表盘（基于官方镜像薄封装）
# 实验环境内的指标可视化

ARG VERSION=10.4.0
FROM grafana/grafana:${VERSION}

ARG VERSION
LABEL org.opencontainers.image.title="grafana"
LABEL org.opencontainers.image.description="Grafana 可视化监控仪表盘"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.source="https://github.com/grafana/grafana"
LABEL org.opencontainers.image.vendor="LensChain"
LABEL lenschain.io/ecosystem="general"
LABEL lenschain.io/category="tools"

ENV TZ=UTC

EXPOSE 3000

HEALTHCHECK --interval=30s --timeout=5s --retries=3 --start-period=20s \
    CMD curl -fsS http://localhost:3000/api/health || exit 1
