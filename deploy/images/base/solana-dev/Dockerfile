# base/solana-dev/Dockerfile
# Solana 开发环境镜像（完整自建）
# 预装：Rust + Solana CLI + Anchor（AVM）+ Node.js + @solana/web3.js

FROM rust:1.77-slim-bookworm

ARG DEBIAN_MIRROR=mirrors.tuna.tsinghua.edu.cn
ARG VERSION=v1.0.0
LABEL org.opencontainers.image.title="solana-dev"
LABEL org.opencontainers.image.description="Solana 开发环境镜像"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.vendor="LensChain"
LABEL lenschain.io/ecosystem="solana"
LABEL lenschain.io/category="base"

ENV TZ=UTC DEBIAN_FRONTEND=noninteractive

RUN sed -i "s|deb.debian.org|${DEBIAN_MIRROR}|g" /etc/apt/sources.list.d/debian.sources && \
    apt-get -o Acquire::Retries=5 update && \
    for i in 1 2 3; do \
        apt-get -o Acquire::Retries=5 install -y --no-install-recommends --fix-missing \
            ca-certificates curl git make gcc pkg-config libssl-dev nodejs npm && break; \
        echo ">>> apt-get install attempt $i failed, waiting 10s..."; \
        sleep 10; apt-get clean; apt-get -o Acquire::Retries=5 update; \
    done && \
    rm -rf /var/lib/apt/lists/*

# 安装 Solana CLI（官方安装脚本，安装到 ~/.local/share/solana/install/）
ENV PATH="/root/.local/share/solana/install/active_release/bin:${PATH}"
RUN sh -c "$(curl -fsSL https://release.anza.xyz/v1.18.15/install)"

# 安装 Anchor Version Manager（AVM）— Solana 智能合约框架
RUN cargo install --git https://github.com/coral-xyz/anchor avm --locked

# 安装 @solana/web3.js — Solana JavaScript SDK
RUN npm install -g @solana/web3.js

RUN useradd -m -s /bin/bash -u 1001 developer

USER developer
WORKDIR /home/developer

CMD ["sleep", "infinity"]
