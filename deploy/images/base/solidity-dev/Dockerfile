# base/solidity-dev/Dockerfile
# Solidity 开发环境镜像（多阶段构建）
# 预装：Node.js 20 + solc + Hardhat + Truffle + Foundry + OpenZeppelin
# EVM 生态智能合约开发基础环境
# builder 阶段编译 Foundry 工具链，runtime 阶段拷贝产物

# ============================
# 构建阶段：编译 Foundry
# ============================
FROM rust:1.77-slim-bookworm AS foundry-builder

ARG DEBIAN_MIRROR=mirrors.ustc.edu.cn

RUN sed -i "s|deb.debian.org|${DEBIAN_MIRROR}|g" /etc/apt/sources.list.d/debian.sources && \
    apt-get -o Acquire::Retries=5 update && \
    apt-get install -y --no-install-recommends ca-certificates curl git && \
    rm -rf /var/lib/apt/lists/*

RUN curl -fsSL https://github.com/foundry-rs/foundry/releases/download/nightly-5b7e4cb3c882b28f3c32ba580de27ce7381f415a/foundry_nightly_linux_amd64.tar.gz \
    -o /tmp/foundry.tar.gz && \
    tar -xzf /tmp/foundry.tar.gz -C /usr/local/bin && \
    rm /tmp/foundry.tar.gz && \
    chmod +x /usr/local/bin/forge /usr/local/bin/cast /usr/local/bin/anvil /usr/local/bin/chisel

# ============================
# 运行阶段：开发环境
# ============================
FROM node:20-bullseye-slim

ARG DEBIAN_MIRROR=mirrors.ustc.edu.cn

ARG VERSION=v1.0.0
LABEL org.opencontainers.image.title="solidity-dev"
LABEL org.opencontainers.image.description="Solidity 开发环境镜像"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.vendor="LensChain"
LABEL lenschain.io/ecosystem="ethereum"
LABEL lenschain.io/category="base"

ENV TZ=UTC \
    DEBIAN_FRONTEND=noninteractive

RUN sed -i "s|deb.debian.org|${DEBIAN_MIRROR}|g" /etc/apt/sources.list && \
    apt-get -o Acquire::Retries=5 update && \
    apt-get install -y --no-install-recommends \
        ca-certificates curl git python3 make g++ && \
    rm -rf /var/lib/apt/lists/*

RUN npm install -g solc@0.8.25 hardhat truffle @openzeppelin/contracts

COPY --from=foundry-builder /usr/local/bin/forge /usr/local/bin/forge
COPY --from=foundry-builder /usr/local/bin/cast /usr/local/bin/cast
COPY --from=foundry-builder /usr/local/bin/anvil /usr/local/bin/anvil
COPY --from=foundry-builder /usr/local/bin/chisel /usr/local/bin/chisel

RUN useradd -m -s /bin/bash -u 1001 developer && \
    chown -R developer:developer /home/developer

USER developer
WORKDIR /home/developer

CMD ["sleep", "infinity"]
