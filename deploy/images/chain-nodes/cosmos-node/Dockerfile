# chain-nodes/cosmos-node/Dockerfile
# Cosmos SDK 节点（基于官方 simapp 镜像薄封装）
# Cosmos 生态应用链开发框架

ARG VERSION=v0.50.0
FROM interchainio/simapp:${VERSION}

ARG VERSION
LABEL org.opencontainers.image.title="cosmos-node"
LABEL org.opencontainers.image.description="Cosmos SDK 节点"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.source="https://github.com/cosmos/cosmos-sdk"
LABEL org.opencontainers.image.vendor="LensChain"
LABEL lenschain.io/ecosystem="cosmos"
LABEL lenschain.io/category="chain-nodes"

ENV TZ=UTC

EXPOSE 26657 1317 9090

HEALTHCHECK --interval=30s --timeout=10s --retries=3 --start-period=30s \
    CMD curl -fsS http://localhost:26657/health || exit 1
