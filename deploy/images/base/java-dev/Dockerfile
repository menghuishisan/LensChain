# base/java-dev/Dockerfile
# Java 区块链开发环境镜像（完整自建）
# 预装：JDK 17 + Maven + Fabric Java SDK + FISCO Java SDK

FROM eclipse-temurin:17-jdk-jammy

ARG VERSION=v1.0.0
LABEL org.opencontainers.image.title="java-dev"
LABEL org.opencontainers.image.description="Java 区块链开发环境镜像"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.vendor="LensChain"
LABEL lenschain.io/ecosystem="fabric"
LABEL lenschain.io/category="base"

ENV TZ=UTC DEBIAN_FRONTEND=noninteractive

RUN apt-get update && \
    apt-get install -y --no-install-recommends \
        ca-certificates curl git maven gradle && \
    rm -rf /var/lib/apt/lists/*

RUN useradd -m -s /bin/bash -u 1001 developer

USER developer
WORKDIR /home/developer

CMD ["bash"]
