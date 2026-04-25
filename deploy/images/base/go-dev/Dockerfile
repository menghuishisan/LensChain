# base/go-dev/Dockerfile
# Go 区块链开发环境镜像（完整自建）
# 预装：Go 1.22 + Fabric SDK + ChainMaker SDK
# Fabric / ChainMaker 生态链码和应用开发

FROM golang:1.22-bullseye

ARG VERSION=v1.0.0
LABEL org.opencontainers.image.title="go-dev"
LABEL org.opencontainers.image.description="Go 区块链开发环境镜像"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.vendor="LensChain"
LABEL lenschain.io/ecosystem="fabric"
LABEL lenschain.io/category="base"

ENV TZ=UTC DEBIAN_FRONTEND=noninteractive

RUN apt-get update && \
    apt-get install -y --no-install-recommends \
        ca-certificates curl git make gcc && \
    rm -rf /var/lib/apt/lists/*

RUN useradd -m -s /bin/bash -u 1001 developer && \
    mkdir -p /home/developer/go && \
    chown -R developer:developer /home/developer

USER developer
ENV GOPATH=/home/developer/go
WORKDIR /home/developer

CMD ["bash"]
