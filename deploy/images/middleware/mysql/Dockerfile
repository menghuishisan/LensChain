# middleware/mysql/Dockerfile
# MySQL 关系型数据库（基于官方镜像薄封装）
# 实验环境应用数据存储

ARG VERSION=8.0
FROM mysql:${VERSION}

ARG VERSION
LABEL org.opencontainers.image.title="mysql"
LABEL org.opencontainers.image.description="MySQL 关系型数据库"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.source="https://github.com/mysql/mysql-server"
LABEL org.opencontainers.image.vendor="LensChain"
LABEL lenschain.io/ecosystem="general"
LABEL lenschain.io/category="middleware"

ENV TZ=UTC
EXPOSE 3306

HEALTHCHECK --interval=30s --timeout=5s --retries=3 --start-period=30s \
    CMD mysqladmin ping -h localhost || exit 1
