# chain-nodes/besu/Dockerfile
# Hyperledger Besu 以太坊客户端（基于官方镜像薄封装）
# 支持 EVM 兼容链，教学场景使用 dev 模式
# 构建参数：--build-arg VERSION=24.3.0

ARG VERSION=24.3.0
FROM hyperledger/besu:${VERSION}

ARG VERSION
LABEL org.opencontainers.image.title="besu"
LABEL org.opencontainers.image.description="Hyperledger Besu 以太坊客户端"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.source="https://github.com/hyperledger/besu"
LABEL org.opencontainers.image.vendor="LensChain"
LABEL lenschain.io/ecosystem="ethereum"
LABEL lenschain.io/category="chain-nodes"

ENV TZ=UTC

EXPOSE 8545 8546 30303

HEALTHCHECK --interval=30s --timeout=10s --retries=3 --start-period=30s \
    CMD curl -fsS http://localhost:8545 -X POST -H 'Content-Type: application/json' \
        -d '{"jsonrpc":"2.0","id":1,"method":"eth_blockNumber","params":[]}' || exit 1

CMD ["--network=dev", "--rpc-http-enabled", "--rpc-http-host=0.0.0.0", "--rpc-ws-enabled", "--rpc-ws-host=0.0.0.0", "--host-allowlist=*", "--rpc-http-cors-origins=*"]
