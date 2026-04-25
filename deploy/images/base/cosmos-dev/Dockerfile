# base/cosmos-dev/Dockerfile
# Cosmos 开发环境镜像（完整自建）
# 预装：Go + Cosmos SDK + Ignite CLI

FROM golang:1.22-bullseye

ARG VERSION=v1.0.0
LABEL org.opencontainers.image.title="cosmos-dev"
LABEL org.opencontainers.image.description="Cosmos 开发环境镜像"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.vendor="LensChain"
LABEL lenschain.io/ecosystem="cosmos"
LABEL lenschain.io/category="base"

ENV TZ=UTC DEBIAN_FRONTEND=noninteractive

RUN apt-get update && \
    apt-get install -y --no-install-recommends \
        ca-certificates curl git make gcc && \
    rm -rf /var/lib/apt/lists/*

RUN useradd -m -s /bin/bash -u 1001 developer

USER developer
WORKDIR /home/developer

CMD ["bash"]
