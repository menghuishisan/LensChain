# chain-nodes/hardhat-node/Dockerfile
# Hardhat Network 本地开发节点（自建镜像）
# 基于 Node.js 20，预装 Hardhat，启动内置 EVM 节点
# 无官方独立镜像，需完整自建

FROM node:20-alpine

ARG VERSION=v1.0.0
LABEL org.opencontainers.image.title="hardhat-node"
LABEL org.opencontainers.image.description="Hardhat Network 本地开发节点"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.source="https://github.com/NomicFoundation/hardhat"
LABEL org.opencontainers.image.vendor="LensChain"
LABEL lenschain.io/ecosystem="ethereum"
LABEL lenschain.io/category="chain-nodes"

ENV TZ=UTC

RUN npm install -g hardhat@2.22.0 && \
    addgroup -S lenschain && adduser -S lenschain -G lenschain && \
    mkdir -p /home/lenschain/project && \
    chown -R lenschain:lenschain /home/lenschain

USER lenschain
WORKDIR /home/lenschain/project

RUN npx hardhat init --force 2>/dev/null || true

EXPOSE 8545

HEALTHCHECK --interval=30s --timeout=5s --retries=3 --start-period=15s \
    CMD wget -qO- http://localhost:8545 \
        --post-data='{"jsonrpc":"2.0","id":1,"method":"eth_blockNumber","params":[]}' \
        --header='Content-Type: application/json' || exit 1

CMD ["npx", "hardhat", "node", "--hostname", "0.0.0.0"]
