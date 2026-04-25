# base/circom-dev/Dockerfile
# Circom 零知识证明开发环境镜像（完整自建）
# 预装：Circom + SnarkJS + Node.js

FROM node:20-bullseye-slim

ARG VERSION=v1.0.0
LABEL org.opencontainers.image.title="circom-dev"
LABEL org.opencontainers.image.description="Circom 零知识证明开发环境镜像"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.vendor="LensChain"
LABEL lenschain.io/ecosystem="general"
LABEL lenschain.io/category="base"

ENV TZ=UTC DEBIAN_FRONTEND=noninteractive

RUN apt-get update && \
    apt-get install -y --no-install-recommends \
        ca-certificates curl git make gcc g++ && \
    npm install -g snarkjs circomlib && \
    rm -rf /var/lib/apt/lists/*

RUN useradd -m -s /bin/bash -u 1001 developer

USER developer
WORKDIR /home/developer

CMD ["bash"]
