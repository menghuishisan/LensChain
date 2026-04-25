# middleware/minio/Dockerfile
# MinIO 对象存储（基于官方镜像薄封装）
# S3 兼容对象存储

ARG VERSION=RELEASE.2024-10-13T13-34-11Z
FROM minio/minio:${VERSION}

ARG VERSION
LABEL org.opencontainers.image.title="minio"
LABEL org.opencontainers.image.description="MinIO 对象存储"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.source="https://github.com/minio/minio"
LABEL org.opencontainers.image.vendor="LensChain"
LABEL lenschain.io/ecosystem="general"
LABEL lenschain.io/category="middleware"

ENV TZ=UTC
EXPOSE 9000 9001

HEALTHCHECK --interval=30s --timeout=5s --retries=3 --start-period=10s \
    CMD curl -fsS http://localhost:9000/minio/health/ready || exit 1
