# base/rust-dev/Dockerfile
# Rust 区块链开发环境镜像（完整自建）
# 预装：Rust + WASM 工具链 + ink! 智能合约框架
# Polkadot / 通用 WASM 合约开发

FROM rust:1.77-slim-bookworm

ARG VERSION=v1.0.0
LABEL org.opencontainers.image.title="rust-dev"
LABEL org.opencontainers.image.description="Rust 区块链开发环境镜像"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.vendor="LensChain"
LABEL lenschain.io/ecosystem="polkadot"
LABEL lenschain.io/category="base"

ENV TZ=UTC DEBIAN_FRONTEND=noninteractive

RUN apt-get update && \
    apt-get install -y --no-install-recommends \
        ca-certificates curl git make gcc pkg-config libssl-dev && \
    rm -rf /var/lib/apt/lists/*

RUN rustup target add wasm32-unknown-unknown && \
    rustup component add rust-src

RUN useradd -m -s /bin/bash -u 1001 developer

USER developer
WORKDIR /home/developer

CMD ["bash"]
