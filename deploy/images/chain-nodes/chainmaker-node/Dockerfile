# chain-nodes/chainmaker-node/Dockerfile
# 长安链 ChainMaker 节点（基于官方镜像薄封装）
# 国产联盟链平台，支持多种共识算法

ARG VERSION=v2.3.0
FROM chainmakerofficial/chainmaker:${VERSION}

ARG VERSION
LABEL org.opencontainers.image.title="chainmaker-node"
LABEL org.opencontainers.image.description="长安链 ChainMaker 节点"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.source="https://git.chainmaker.org.cn/chainmaker/chainmaker-go"
LABEL org.opencontainers.image.vendor="LensChain"
LABEL lenschain.io/ecosystem="chainmaker"
LABEL lenschain.io/category="chain-nodes"

ENV TZ=UTC

EXPOSE 11301 12301

HEALTHCHECK --interval=30s --timeout=10s --retries=3 --start-period=30s \
    CMD curl -fsS http://localhost:12301/health || exit 1
