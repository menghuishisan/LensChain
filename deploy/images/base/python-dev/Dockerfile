# base/python-dev/Dockerfile
# Python 区块链开发环境镜像（完整自建）
# 预装：Python 3.11 + web3.py + Vyper + eth-brownie + eth-ape
# EVM / 通用区块链 Python 开发
# 注意：eth-brownie 和 eth-ape 存在依赖冲突风险，分步安装避免 pip 回溯

FROM python:3.11-slim-bookworm

ARG DEBIAN_MIRROR=mirrors.tuna.tsinghua.edu.cn
ARG VERSION=v1.0.0
LABEL org.opencontainers.image.title="python-dev"
LABEL org.opencontainers.image.description="Python 区块链开发环境镜像"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.vendor="LensChain"
LABEL lenschain.io/ecosystem="ethereum"
LABEL lenschain.io/category="base"

ENV TZ=UTC DEBIAN_FRONTEND=noninteractive \
    PIP_INDEX_URL=https://mirrors.aliyun.com/pypi/simple/ \
    PIP_TRUSTED_HOST=mirrors.aliyun.com

RUN sed -i "s|deb.debian.org|${DEBIAN_MIRROR}|g" /etc/apt/sources.list.d/debian.sources && \
    apt-get -o Acquire::Retries=5 update && \
    for i in 1 2 3; do \
        apt-get -o Acquire::Retries=5 install -y --no-install-recommends --fix-missing \
            ca-certificates curl git gcc g++ python3-dev && break; \
        echo ">>> apt-get install attempt $i failed, waiting 10s..."; \
        sleep 10; apt-get clean; apt-get -o Acquire::Retries=5 update; \
    done && \
    rm -rf /var/lib/apt/lists/*

# 分步安装 Python 包，避免 eth-brownie 与 eth-ape 之间的 pip 依赖冲突
RUN pip install --no-cache-dir web3 vyper && \
    pip install --no-cache-dir eth-brownie && \
    pip install --no-cache-dir eth-ape

RUN useradd -m -s /bin/bash -u 1001 developer

USER developer
WORKDIR /home/developer

CMD ["sleep", "infinity"]
