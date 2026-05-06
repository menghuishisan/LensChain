# base/circom-dev/Dockerfile
# Circom 零知识证明开发环境镜像（多阶段构建）
# builder 阶段从源码编译 Circom CLI（Rust），runtime 阶段拷贝二进制
# 预装：Circom（编译器） + SnarkJS + circomlib + Node.js

# ============================
# 构建阶段：编译 Circom 编译器（Rust）
# ============================
FROM rust:1.82-slim-bookworm AS builder

RUN cargo install --git https://github.com/iden3/circom.git --locked && \
    strip /usr/local/cargo/bin/circom

# ============================
# 运行阶段：Circom 开发环境
# ============================
FROM node:20-bookworm-slim

ARG DEBIAN_MIRROR=mirrors.tuna.tsinghua.edu.cn
ARG VERSION=v1.0.0
LABEL org.opencontainers.image.title="circom-dev"
LABEL org.opencontainers.image.description="Circom 零知识证明开发环境镜像"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.vendor="LensChain"
LABEL lenschain.io/ecosystem="general"
LABEL lenschain.io/category="base"

ENV TZ=UTC DEBIAN_FRONTEND=noninteractive

RUN sed -i "s|deb.debian.org|${DEBIAN_MIRROR}|g" /etc/apt/sources.list.d/debian.sources && \
    apt-get -o Acquire::Retries=5 update && \
    for i in 1 2 3; do \
        apt-get -o Acquire::Retries=5 install -y --no-install-recommends --fix-missing \
            ca-certificates curl git make gcc g++ && break; \
        echo ">>> apt-get install attempt $i failed, waiting 10s..."; \
        sleep 10; apt-get clean; apt-get -o Acquire::Retries=5 update; \
    done && \
    npm install -g snarkjs circomlib && \
    rm -rf /var/lib/apt/lists/*

# 从构建阶段拷贝 circom 编译器二进制
COPY --from=builder /usr/local/cargo/bin/circom /usr/local/bin/circom

RUN useradd -m -s /bin/bash -u 1001 developer

USER developer
WORKDIR /home/developer

CMD ["sleep", "infinity"]
