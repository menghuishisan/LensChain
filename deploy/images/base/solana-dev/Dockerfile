# base/solana-dev/Dockerfile
# Solana 开发环境镜像（完整自建）
# 预装：Rust + Anchor + Solana CLI

FROM rust:1.77-slim-bookworm

ARG DEBIAN_MIRROR=mirrors.ustc.edu.cn
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
    apt-get install -y --no-install-recommends \
        ca-certificates curl git make gcc pkg-config libssl-dev && \
    rm -rf /var/lib/apt/lists/*

RUN useradd -m -s /bin/bash -u 1001 developer

USER developer
WORKDIR /home/developer

CMD ["sleep", "infinity"]
