# base/rust-dev/Dockerfile
# Rust 区块链开发环境镜像（完整自建）
# 预装：Rust + WASM 工具链 + cargo-contract（ink! 智能合约 CLI）+ binaryen（WASM 优化）
# Polkadot / 通用 WASM 合约开发

FROM rust:1.77-slim-bookworm

ARG DEBIAN_MIRROR=mirrors.tuna.tsinghua.edu.cn
ARG VERSION=v1.0.0
LABEL org.opencontainers.image.title="rust-dev"
LABEL org.opencontainers.image.description="Rust 区块链开发环境镜像"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.vendor="LensChain"
LABEL lenschain.io/ecosystem="polkadot"
LABEL lenschain.io/category="base"

ENV TZ=UTC DEBIAN_FRONTEND=noninteractive

RUN sed -i "s|deb.debian.org|${DEBIAN_MIRROR}|g" /etc/apt/sources.list.d/debian.sources && \
    apt-get -o Acquire::Retries=5 update && \
    for i in 1 2 3; do \
        apt-get -o Acquire::Retries=5 install -y --no-install-recommends --fix-missing \
            ca-certificates curl git make gcc pkg-config libssl-dev binaryen && break; \
        echo ">>> apt-get install attempt $i failed, waiting 10s..."; \
        sleep 10; apt-get clean; apt-get -o Acquire::Retries=5 update; \
    done && \
    rm -rf /var/lib/apt/lists/*

RUN rustup target add wasm32-unknown-unknown && \
    rustup component add rust-src

# 安装 cargo-contract：ink! 智能合约框架的 CLI 工具
# 用于创建、构建、测试和部署 ink! 合约，依赖 binaryen 做 WASM 优化
RUN cargo install cargo-contract --locked

RUN useradd -m -s /bin/bash -u 1001 developer

USER developer
WORKDIR /home/developer

CMD ["sleep", "infinity"]
