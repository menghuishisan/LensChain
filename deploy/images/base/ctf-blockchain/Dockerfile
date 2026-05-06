# base/ctf-blockchain/Dockerfile
# CTF 区块链方向基础镜像（多阶段构建）
# builder 阶段获取 Foundry 二进制，runtime 阶段预装完整工具链
# 预装：Foundry + Hardhat + ethers + web3.py + slither + mythril

# ============================
# 构建阶段：获取 Foundry 二进制
# ============================
FROM alpine:3.19 AS foundry-builder

RUN apk add --no-cache curl tar && \
    curl -fsSL https://github.com/foundry-rs/foundry/releases/download/nightly-5b7e4cb3c882b28f3c32ba580de27ce7381f415a/foundry_nightly_linux_amd64.tar.gz \
    -o /tmp/foundry.tar.gz && \
    tar -xzf /tmp/foundry.tar.gz -C /usr/local/bin && \
    rm /tmp/foundry.tar.gz

# ============================
# 运行阶段：CTF 开发环境
# ============================
FROM node:20-bookworm-slim

ARG DEBIAN_MIRROR=mirrors.tuna.tsinghua.edu.cn
ARG VERSION=v1.0.0
LABEL org.opencontainers.image.title="ctf-blockchain"
LABEL org.opencontainers.image.description="CTF 区块链方向基础镜像"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.vendor="LensChain"
LABEL lenschain.io/ecosystem="ethereum"
LABEL lenschain.io/category="base"
LABEL lenschain.io/scope="ctf"

ENV TZ=UTC DEBIAN_FRONTEND=noninteractive \
    PIP_INDEX_URL=https://mirrors.aliyun.com/pypi/simple/ \
    PIP_TRUSTED_HOST=mirrors.aliyun.com

RUN sed -i "s|deb.debian.org|${DEBIAN_MIRROR}|g" /etc/apt/sources.list.d/debian.sources && \
    apt-get -o Acquire::Retries=5 update && \
    for i in 1 2 3; do \
        apt-get -o Acquire::Retries=5 install -y --no-install-recommends --fix-missing \
            ca-certificates curl git python3 python3-pip python3-dev make g++ && break; \
        echo ">>> apt-get install attempt $i failed, waiting 10s..."; \
        sleep 10; apt-get clean; apt-get -o Acquire::Retries=5 update; \
    done && \
    npm install -g hardhat solc ethers && \
    pip3 install --no-cache-dir --break-system-packages \
        web3 slither-analyzer && \
    pip3 install --no-cache-dir --break-system-packages \
        mythril && \
    rm -rf /var/lib/apt/lists/*

COPY --from=foundry-builder /usr/local/bin/forge /usr/local/bin/forge
COPY --from=foundry-builder /usr/local/bin/cast /usr/local/bin/cast
COPY --from=foundry-builder /usr/local/bin/anvil /usr/local/bin/anvil
COPY --from=foundry-builder /usr/local/bin/chisel /usr/local/bin/chisel

RUN useradd -m -s /bin/bash -u 1001 ctfplayer

USER ctfplayer
WORKDIR /home/ctfplayer

CMD ["sleep", "infinity"]
