# chain-nodes/bitcoind/Dockerfile
# Bitcoin Core 全节点客户端（基于官方镜像薄封装）
# 支持 regtest/testnet/mainnet 模式，教学场景主要使用 regtest
# 构建参数：--build-arg VERSION=26

ARG VERSION=26
FROM bitcoin/bitcoin:${VERSION}

ARG VERSION
LABEL org.opencontainers.image.title="bitcoind"
LABEL org.opencontainers.image.description="Bitcoin Core 全节点客户端"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.source="https://github.com/bitcoin/bitcoin"
LABEL org.opencontainers.image.vendor="LensChain"
LABEL lenschain.io/ecosystem="bitcoin"
LABEL lenschain.io/category="chain-nodes"

ENV TZ=UTC

EXPOSE 8332 8333 18332

HEALTHCHECK --interval=30s --timeout=10s --retries=3 --start-period=30s \
    CMD bitcoin-cli -regtest getblockchaininfo || exit 1

CMD ["-regtest", "-server", "-rpcallowip=0.0.0.0/0", "-rpcbind=0.0.0.0", "-rpcuser=lenschain", "-rpcpassword=lenschain"]
