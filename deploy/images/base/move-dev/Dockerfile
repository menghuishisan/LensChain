# base/move-dev/Dockerfile
# Move 语言开发环境镜像（完整自建）
# 预装：Rust 1.77 + Move 编译器 + Aptos CLI（cargo install 编译安装）

FROM rust:1.77-slim-bookworm

ARG DEBIAN_MIRROR=mirrors.tuna.tsinghua.edu.cn
ARG VERSION=v1.0.0
LABEL org.opencontainers.image.title="move-dev"
LABEL org.opencontainers.image.description="Move 语言开发环境镜像"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.vendor="LensChain"
LABEL lenschain.io/ecosystem="move"
LABEL lenschain.io/category="base"

ENV TZ=UTC DEBIAN_FRONTEND=noninteractive

RUN sed -i "s|deb.debian.org|${DEBIAN_MIRROR}|g" /etc/apt/sources.list.d/debian.sources && \
    apt-get -o Acquire::Retries=5 update && \
    for i in 1 2 3; do \
        apt-get -o Acquire::Retries=5 install -y --no-install-recommends --fix-missing \
            ca-certificates curl git make gcc pkg-config libssl-dev python3 && break; \
        echo ">>> apt-get install attempt $i failed, waiting 10s..."; \
        sleep 10; apt-get clean; apt-get -o Acquire::Retries=5 update; \
    done && \
    rm -rf /var/lib/apt/lists/*

# 安装 Aptos CLI — Aptos / Move 生态命令行工具，用于编译、测试、部署 Move 合约
# 基于 Rust 源码编译安装，耗时较长但版本可控
RUN cargo install aptos --locked

RUN useradd -m -s /bin/bash -u 1001 developer

USER developer
WORKDIR /home/developer

CMD ["sleep", "infinity"]
