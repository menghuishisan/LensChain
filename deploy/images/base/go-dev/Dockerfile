# base/go-dev/Dockerfile
# Go 区块链开发环境镜像（完整自建）
# 预装：Go 1.22 + Fabric SDK Go + ChainMaker SDK Go（预拉取至 GOPATH 缓存）
# Fabric / ChainMaker 生态链码和应用开发

FROM golang:1.23-bookworm

ARG DEBIAN_MIRROR=mirrors.tuna.tsinghua.edu.cn
ARG VERSION=v1.0.0
LABEL org.opencontainers.image.title="go-dev"
LABEL org.opencontainers.image.description="Go 区块链开发环境镜像"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.vendor="LensChain"
LABEL lenschain.io/ecosystem="fabric"
LABEL lenschain.io/category="base"

ENV TZ=UTC DEBIAN_FRONTEND=noninteractive \
    GOPROXY=https://goproxy.cn,direct \
    GONOSUMCHECK=* \
    GONOSUMDB=*

RUN sed -i "s|deb.debian.org|${DEBIAN_MIRROR}|g" /etc/apt/sources.list.d/debian.sources && \
    apt-get -o Acquire::Retries=5 update && \
    for i in 1 2 3; do \
        apt-get -o Acquire::Retries=5 install -y --no-install-recommends --fix-missing \
            ca-certificates curl git make gcc && break; \
        echo ">>> apt-get install attempt $i failed, waiting 10s..."; \
        sleep 10; apt-get clean; apt-get -o Acquire::Retries=5 update; \
    done && \
    rm -rf /var/lib/apt/lists/*

# 预拉取 Fabric SDK Go 和 ChainMaker SDK Go 依赖到全局 GOPATH 缓存
# 使用临时模块上下文执行 go get，用户创建项目时无需重新下载
RUN mkdir -p /tmp/preload && cd /tmp/preload && \
    go mod init preload && \
    go get github.com/hyperledger/fabric-sdk-go@latest && \
    go get chainmaker.org/chainmaker/sdk-go/v2@latest && \
    cd / && rm -rf /tmp/preload

RUN useradd -m -s /bin/bash -u 1001 developer && \
    mkdir -p /home/developer/go && \
    chown -R developer:developer /home/developer

USER developer
ENV GOPATH=/home/developer/go
WORKDIR /home/developer

CMD ["sleep", "infinity"]
