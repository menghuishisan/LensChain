# tools/webase-web/Dockerfile
# WeBase Web 管理平台（基于官方镜像薄封装）
# FISCO BCOS 综合管理平台 Web 界面

ARG VERSION=v3.1.1
FROM webasepro/webase-web:${VERSION}

ARG VERSION
LABEL org.opencontainers.image.title="webase-web"
LABEL org.opencontainers.image.description="WeBase Web 管理平台"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.source="https://github.com/WeBankBlockchain/WeBASE-Web"
LABEL org.opencontainers.image.vendor="LensChain"
LABEL lenschain.io/ecosystem="fisco"
LABEL lenschain.io/category="tools"

ENV TZ=UTC

EXPOSE 5000

HEALTHCHECK --interval=30s --timeout=5s --retries=3 --start-period=20s \
    CMD curl -fsS http://localhost:5000/ || exit 1
