# tools/webase-front/Dockerfile
# WeBase 前端节点管理（基于官方镜像薄封装）
# FISCO BCOS 节点前端管理工具

ARG VERSION=v1.5.5
FROM webasepro/webase-front:${VERSION}

ARG VERSION
LABEL org.opencontainers.image.title="webase-front"
LABEL org.opencontainers.image.description="WeBase 前端节点管理"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.source="https://github.com/WeBankBlockchain/WeBASE-Front"
LABEL org.opencontainers.image.vendor="LensChain"
LABEL lenschain.io/ecosystem="fisco"
LABEL lenschain.io/category="tools"

ENV TZ=UTC

EXPOSE 5002

HEALTHCHECK --interval=30s --timeout=5s --retries=3 --start-period=30s \
    CMD curl -fsS http://localhost:5002/ || exit 1
