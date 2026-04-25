# base/python-dev/Dockerfile
# Python 区块链开发环境镜像（完整自建）
# 预装：Python 3.11 + web3.py + Brownie + eth-ape
# EVM / 通用区块链 Python 开发

FROM python:3.11-slim-bookworm

ARG VERSION=v1.0.0
LABEL org.opencontainers.image.title="python-dev"
LABEL org.opencontainers.image.description="Python 区块链开发环境镜像"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.vendor="LensChain"
LABEL lenschain.io/ecosystem="ethereum"
LABEL lenschain.io/category="base"

ENV TZ=UTC DEBIAN_FRONTEND=noninteractive

RUN apt-get update && \
    apt-get install -y --no-install-recommends \
        ca-certificates curl git gcc && \
    pip install --no-cache-dir \
        web3 eth-brownie eth-ape vyper && \
    rm -rf /var/lib/apt/lists/*

RUN useradd -m -s /bin/bash -u 1001 developer

USER developer
WORKDIR /home/developer

CMD ["bash"]
