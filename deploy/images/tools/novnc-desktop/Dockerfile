# tools/novnc-desktop/Dockerfile
# noVNC 图形化桌面（基于官方镜像薄封装）
# 提供浏览器内图形化桌面环境，用于 GUI 工具操作

ARG VERSION=1.4.0
FROM theasp/novnc:${VERSION}

ARG VERSION
LABEL org.opencontainers.image.title="novnc-desktop"
LABEL org.opencontainers.image.description="noVNC 图形化桌面"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.source="https://github.com/novnc/noVNC"
LABEL org.opencontainers.image.vendor="LensChain"
LABEL lenschain.io/ecosystem="general"
LABEL lenschain.io/category="tools"

ENV TZ=UTC

EXPOSE 6080

HEALTHCHECK --interval=30s --timeout=5s --retries=3 --start-period=20s \
    CMD curl -fsS http://localhost:6080/ || exit 1
