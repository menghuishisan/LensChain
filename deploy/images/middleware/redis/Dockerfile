# middleware/redis/Dockerfile
# Redis 内存缓存数据库（基于官方镜像薄封装）
# 实验环境缓存和消息队列

ARG VERSION=7-alpine
FROM redis:${VERSION}

ARG VERSION
LABEL org.opencontainers.image.title="redis"
LABEL org.opencontainers.image.description="Redis 内存缓存数据库"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.source="https://github.com/redis/redis"
LABEL org.opencontainers.image.vendor="LensChain"
LABEL lenschain.io/ecosystem="general"
LABEL lenschain.io/category="middleware"

ENV TZ=UTC
EXPOSE 6379

HEALTHCHECK --interval=30s --timeout=5s --retries=3 \
    CMD redis-cli ping || exit 1
