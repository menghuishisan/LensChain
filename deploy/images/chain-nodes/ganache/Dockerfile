# chain-nodes/ganache/Dockerfile
# Ganache 以太坊模拟器（基于官方镜像薄封装）
# 轻量级 EVM 模拟器，适合快速开发和测试
# 构建参数：--build-arg VERSION=v7.9.2

ARG VERSION=v7.9.2
FROM trufflesuite/ganache:${VERSION}

ARG VERSION
LABEL org.opencontainers.image.title="ganache"
LABEL org.opencontainers.image.description="Ganache 以太坊模拟器"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.source="https://github.com/trufflesuite/ganache"
LABEL org.opencontainers.image.vendor="LensChain"
LABEL lenschain.io/ecosystem="ethereum"
LABEL lenschain.io/category="chain-nodes"

ENV TZ=UTC

EXPOSE 8545

HEALTHCHECK --interval=30s --timeout=5s --retries=3 --start-period=10s \
    CMD wget -qO- http://localhost:8545 \
        --post-data='{"jsonrpc":"2.0","id":1,"method":"eth_blockNumber","params":[]}' \
        --header='Content-Type: application/json' || exit 1

CMD ["--host", "0.0.0.0", "--accounts", "10", "--deterministic"]
