# middleware/postgres/Dockerfile
# PostgreSQL 关系型数据库（基于官方镜像薄封装）
# 实验环境应用数据存储

ARG VERSION=15
FROM postgres:${VERSION}

ARG VERSION
LABEL org.opencontainers.image.title="postgres"
LABEL org.opencontainers.image.description="PostgreSQL 关系型数据库"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.source="https://github.com/postgres/postgres"
LABEL org.opencontainers.image.vendor="LensChain"
LABEL lenschain.io/ecosystem="general"
LABEL lenschain.io/category="middleware"

ENV TZ=UTC
EXPOSE 5432

HEALTHCHECK --interval=30s --timeout=5s --retries=3 --start-period=15s \
    CMD pg_isready -U postgres || exit 1
