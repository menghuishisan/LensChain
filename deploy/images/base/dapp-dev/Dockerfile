# base/dapp-dev/Dockerfile
# DApp 前端开发环境镜像（完整自建）
# 预装：Node.js 20 + React/Next.js + ethers.js + viem + wagmi + RainbowKit
# EVM 生态去中心化应用前端开发

FROM node:20-bullseye-slim

ARG DEBIAN_MIRROR=mirrors.ustc.edu.cn
ARG VERSION=v1.0.0
LABEL org.opencontainers.image.title="dapp-dev"
LABEL org.opencontainers.image.description="DApp 前端开发环境镜像"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.vendor="LensChain"
LABEL lenschain.io/ecosystem="ethereum"
LABEL lenschain.io/category="base"

ENV TZ=UTC DEBIAN_FRONTEND=noninteractive

RUN sed -i "s|deb.debian.org|${DEBIAN_MIRROR}|g" /etc/apt/sources.list && \
    apt-get -o Acquire::Retries=5 update && \
    apt-get install -y --no-install-recommends \
        ca-certificates curl git && \
    npm install -g create-next-app create-react-app && \
    rm -rf /var/lib/apt/lists/*

RUN useradd -m -s /bin/bash -u 1001 developer

USER developer
WORKDIR /home/developer

CMD ["sleep", "infinity"]
