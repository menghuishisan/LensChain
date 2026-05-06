# chain-nodes/aptos-node/Dockerfile
# Aptos 验证节点（基于官方镜像薄封装）
# Move 语言生态公链

ARG VERSION=mainnet
FROM aptoslabs/validator:${VERSION}

ARG VERSION
LABEL org.opencontainers.image.title="aptos-node"
LABEL org.opencontainers.image.description="Aptos 验证节点"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.source="https://github.com/aptos-labs/aptos-core"
LABEL org.opencontainers.image.vendor="LensChain"
LABEL lenschain.io/ecosystem="move"
LABEL lenschain.io/category="chain-nodes"

ENV TZ=UTC

EXPOSE 8080 9101

HEALTHCHECK --interval=30s --timeout=10s --retries=3 --start-period=60s \
    CMD curl -fsS http://localhost:8080/v1 || exit 1
