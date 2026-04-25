# chain-nodes/substrate-node/Dockerfile
# Substrate/Polkadot 节点（基于官方镜像薄封装）
# Polkadot 生态底层框架节点

ARG VERSION=v1.7.0
FROM parity/polkadot:${VERSION}

ARG VERSION
LABEL org.opencontainers.image.title="substrate-node"
LABEL org.opencontainers.image.description="Substrate/Polkadot 节点"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.source="https://github.com/paritytech/polkadot-sdk"
LABEL org.opencontainers.image.vendor="LensChain"
LABEL lenschain.io/ecosystem="polkadot"
LABEL lenschain.io/category="chain-nodes"

ENV TZ=UTC

EXPOSE 9944 9933 30333

HEALTHCHECK --interval=30s --timeout=10s --retries=3 --start-period=30s \
    CMD curl -fsS http://localhost:9933/health || exit 1
