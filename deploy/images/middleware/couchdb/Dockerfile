# middleware/couchdb/Dockerfile
# CouchDB 文档数据库（基于官方镜像薄封装）
# Fabric 状态数据库，替代 LevelDB 提供富查询能力

ARG VERSION=3.3
FROM couchdb:${VERSION}

ARG VERSION
LABEL org.opencontainers.image.title="couchdb"
LABEL org.opencontainers.image.description="CouchDB 文档数据库"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.source="https://github.com/apache/couchdb"
LABEL org.opencontainers.image.vendor="LensChain"
LABEL lenschain.io/ecosystem="fabric"
LABEL lenschain.io/category="middleware"

ENV TZ=UTC

EXPOSE 5984

HEALTHCHECK --interval=30s --timeout=5s --retries=3 --start-period=15s \
    CMD curl -fsS http://localhost:5984/_up || exit 1
