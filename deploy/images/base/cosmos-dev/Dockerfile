# base/cosmos-dev/Dockerfile
# Cosmos 开发环境镜像（完整自建）
# 预装：Go 1.22 + Cosmos SDK + Ignite CLI（Cosmos 生态脚手架工具）

FROM golang:1.23-bookworm

ARG DEBIAN_MIRROR=mirrors.tuna.tsinghua.edu.cn
ARG VERSION=v1.0.0
LABEL org.opencontainers.image.title="cosmos-dev"
LABEL org.opencontainers.image.description="Cosmos 开发环境镜像"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.vendor="LensChain"
LABEL lenschain.io/ecosystem="cosmos"
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

# 安装 Ignite CLI — Cosmos 生态脚手架工具，用于创建链、模块、消息等
RUN curl -fsSL https://get.ignite.com/cli! | bash && \
    mv /usr/local/bin/ignite /usr/local/bin/ignite 2>/dev/null || true

RUN useradd -m -s /bin/bash -u 1001 developer

USER developer
WORKDIR /home/developer

CMD ["sleep", "infinity"]
