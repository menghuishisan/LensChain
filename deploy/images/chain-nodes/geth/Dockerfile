# chain-nodes/geth/Dockerfile
# Go-Ethereum 全功能客户端（基于官方镜像薄封装）
# 支持 dev/mainnet/testnet 模式，教学场景主要使用 dev 模式
# 构建参数：--build-arg VERSION=1.14.0

ARG VERSION=1.14.0
FROM ethereum/client-go:v${VERSION}

ARG VERSION
LABEL org.opencontainers.image.title="geth"
LABEL org.opencontainers.image.description="Go-Ethereum 全功能客户端"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.source="https://github.com/ethereum/go-ethereum"
LABEL org.opencontainers.image.vendor="LensChain"
LABEL lenschain.io/ecosystem="ethereum"
LABEL lenschain.io/category="chain-nodes"

ENV TZ=UTC

EXPOSE 8545 8546 30303 30303/udp

HEALTHCHECK --interval=30s --timeout=10s --retries=3 --start-period=20s \
    CMD wget -qO- http://localhost:8545 \
        --post-data='{"jsonrpc":"2.0","id":1,"method":"eth_blockNumber","params":[]}' \
        --header='Content-Type: application/json' || exit 1

CMD ["--dev", "--http", "--http.addr", "0.0.0.0", "--http.api", "eth,net,web3,personal,debug", "--http.corsdomain", "*", "--http.vhosts", "*", "--ws", "--ws.addr", "0.0.0.0", "--ws.api", "eth,net,web3"]
