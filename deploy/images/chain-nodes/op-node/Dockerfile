# chain-nodes/op-node/Dockerfile
# Optimism OP Node L2 节点（基于官方镜像薄封装）
# 以太坊 Layer 2 扩展方案节点
# 构建参数：--build-arg VERSION=v1.7.0

ARG VERSION=v1.7.0
FROM us-docker.pkg.dev/oplabs-tools-artifacts/images/op-node:${VERSION}

ARG VERSION
LABEL org.opencontainers.image.title="op-node"
LABEL org.opencontainers.image.description="Optimism OP Node L2 节点"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.source="https://github.com/ethereum-optimism/optimism"
LABEL org.opencontainers.image.vendor="LensChain"
LABEL lenschain.io/ecosystem="ethereum"
LABEL lenschain.io/category="chain-nodes"

ENV TZ=UTC

EXPOSE 8545 9003

HEALTHCHECK --interval=30s --timeout=10s --retries=3 --start-period=30s \
    CMD wget -qO- http://localhost:8545/healthz || exit 1
