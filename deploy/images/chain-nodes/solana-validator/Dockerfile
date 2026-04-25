# chain-nodes/solana-validator/Dockerfile
# Solana 验证节点（基于官方镜像薄封装）
# 高性能公链，支持 PoH + PoS 共识

ARG VERSION=v1.18.15
FROM solanalabs/solana:${VERSION}

ARG VERSION
LABEL org.opencontainers.image.title="solana-validator"
LABEL org.opencontainers.image.description="Solana 验证节点"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.source="https://github.com/solana-labs/solana"
LABEL org.opencontainers.image.vendor="LensChain"
LABEL lenschain.io/ecosystem="solana"
LABEL lenschain.io/category="chain-nodes"

ENV TZ=UTC

EXPOSE 8899 8900 8001

HEALTHCHECK --interval=30s --timeout=10s --retries=3 --start-period=60s \
    CMD curl -fsS http://localhost:8899 -X POST -H 'Content-Type: application/json' \
        -d '{"jsonrpc":"2.0","id":1,"method":"getHealth"}' || exit 1
