# base/substrate-dev/Dockerfile
# Substrate 开发环境镜像（完整自建）
# 预装：Rust + WASM 工具链 + FRAME 框架 + substrate-node-template + Node.js + @polkadot/api

FROM rust:1.77-slim-bookworm

ARG DEBIAN_MIRROR=mirrors.tuna.tsinghua.edu.cn
ARG VERSION=v1.0.0
LABEL org.opencontainers.image.title="substrate-dev"
LABEL org.opencontainers.image.description="Substrate 开发环境镜像"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.vendor="LensChain"
LABEL lenschain.io/ecosystem="polkadot"
LABEL lenschain.io/category="base"

ENV TZ=UTC DEBIAN_FRONTEND=noninteractive

RUN sed -i "s|deb.debian.org|${DEBIAN_MIRROR}|g" /etc/apt/sources.list.d/debian.sources && \
    apt-get -o Acquire::Retries=5 update && \
    for i in 1 2 3; do \
        apt-get -o Acquire::Retries=5 install -y --no-install-recommends --fix-missing \
            ca-certificates curl git make gcc pkg-config libssl-dev clang cmake nodejs npm && break; \
        echo ">>> apt-get install attempt $i failed, waiting 10s..."; \
        sleep 10; apt-get clean; apt-get -o Acquire::Retries=5 update; \
    done && \
    rm -rf /var/lib/apt/lists/*

RUN rustup target add wasm32-unknown-unknown && \
    rustup component add rust-src

# 预置 substrate-node-template 作为开发模板
RUN git clone --depth=1 https://github.com/niceDeve/substrate-node-template.git /opt/substrate-node-template

# 安装 @polkadot/api 和 @polkadot/keyring — Polkadot.js SDK
RUN npm install -g @polkadot/api @polkadot/keyring

RUN useradd -m -s /bin/bash -u 1001 developer

USER developer
WORKDIR /home/developer

CMD ["sleep", "infinity"]
