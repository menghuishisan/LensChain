# tools/blockscout/Dockerfile
# Blockscout EVM 区块链浏览器（基于官方镜像薄封装）
# 支持所有 EVM 兼容链的区块、交易、合约浏览

ARG VERSION=6.3.0
FROM blockscout/blockscout:${VERSION}

ARG VERSION
LABEL org.opencontainers.image.title="blockscout"
LABEL org.opencontainers.image.description="Blockscout EVM 区块链浏览器"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.source="https://github.com/blockscout/blockscout"
LABEL org.opencontainers.image.vendor="LensChain"
LABEL lenschain.io/ecosystem="ethereum"
LABEL lenschain.io/category="tools"

ENV TZ=UTC

EXPOSE 4000

HEALTHCHECK --interval=30s --timeout=10s --retries=3 --start-period=60s \
    CMD curl -fsS http://localhost:4000/api/v2/stats || exit 1
