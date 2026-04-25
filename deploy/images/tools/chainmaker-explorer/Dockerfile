# tools/chainmaker-explorer/Dockerfile
# 长安链浏览器（基于官方镜像薄封装）
# ChainMaker 区块链浏览器，查看区块、交易、合约信息

ARG VERSION=v2.3.0
FROM chainmakerofficial/chainmaker-explorer:${VERSION}

ARG VERSION
LABEL org.opencontainers.image.title="chainmaker-explorer"
LABEL org.opencontainers.image.description="长安链浏览器"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.source="https://git.chainmaker.org.cn/chainmaker/chainmaker-explorer"
LABEL org.opencontainers.image.vendor="LensChain"
LABEL lenschain.io/ecosystem="chainmaker"
LABEL lenschain.io/category="tools"

ENV TZ=UTC

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --retries=3 --start-period=30s \
    CMD curl -fsS http://localhost:8080/ || exit 1
