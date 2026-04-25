# tools/code-server/Dockerfile
# VS Code Web 在线编辑器（基于官方镜像薄封装）
# 提供浏览器内 VS Code 开发环境，支持所有链生态

ARG VERSION=4.89.1
FROM codercom/code-server:${VERSION}

ARG VERSION
LABEL org.opencontainers.image.title="code-server"
LABEL org.opencontainers.image.description="VS Code Web 在线编辑器"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.source="https://github.com/coder/code-server"
LABEL org.opencontainers.image.vendor="LensChain"
LABEL lenschain.io/ecosystem="general"
LABEL lenschain.io/category="tools"

ENV TZ=UTC

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --retries=3 --start-period=15s \
    CMD curl -fsS http://localhost:8080/healthz || exit 1
