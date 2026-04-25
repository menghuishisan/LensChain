# tools/fabric-explorer/Dockerfile
# Hyperledger Explorer Fabric 区块链浏览器（基于官方镜像薄封装）
# 查看 Fabric 网络的区块、交易、链码信息

ARG VERSION=1.1.8
FROM hyperledger/explorer:${VERSION}

ARG VERSION
LABEL org.opencontainers.image.title="fabric-explorer"
LABEL org.opencontainers.image.description="Hyperledger Explorer Fabric 区块链浏览器"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.source="https://github.com/hyperledger-labs/blockchain-explorer"
LABEL org.opencontainers.image.vendor="LensChain"
LABEL lenschain.io/ecosystem="fabric"
LABEL lenschain.io/category="tools"

ENV TZ=UTC

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --retries=3 --start-period=30s \
    CMD curl -fsS http://localhost:8080/ || exit 1
