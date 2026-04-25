# middleware/ipfs/Dockerfile
# IPFS 分布式存储节点（基于官方 Kubo 镜像薄封装）
# 去中心化文件存储

ARG VERSION=v0.27.0
FROM ipfs/kubo:${VERSION}

ARG VERSION
LABEL org.opencontainers.image.title="ipfs"
LABEL org.opencontainers.image.description="IPFS 分布式存储节点"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.source="https://github.com/ipfs/kubo"
LABEL org.opencontainers.image.vendor="LensChain"
LABEL lenschain.io/ecosystem="general"
LABEL lenschain.io/category="middleware"

ENV TZ=UTC
EXPOSE 5001 8080

HEALTHCHECK --interval=30s --timeout=5s --retries=3 --start-period=20s \
    CMD ipfs id || exit 1
