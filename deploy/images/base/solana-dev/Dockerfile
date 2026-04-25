# base/solana-dev/Dockerfile
# Solana 开发环境镜像（完整自建）
# 预装：Rust + Anchor + Solana CLI

FROM rust:1.77-slim-bookworm

ARG VERSION=v1.0.0
LABEL org.opencontainers.image.title="solana-dev"
LABEL org.opencontainers.image.description="Solana 开发环境镜像"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.vendor="LensChain"
LABEL lenschain.io/ecosystem="solana"
LABEL lenschain.io/category="base"

ENV TZ=UTC DEBIAN_FRONTEND=noninteractive

RUN apt-get update && \
    apt-get install -y --no-install-recommends \
        ca-certificates curl git make gcc pkg-config libssl-dev && \
    rm -rf /var/lib/apt/lists/*

RUN useradd -m -s /bin/bash -u 1001 developer

USER developer
WORKDIR /home/developer

CMD ["bash"]
