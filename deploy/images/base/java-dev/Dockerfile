# base/java-dev/Dockerfile
# Java 区块链开发环境镜像（完整自建）
# 预装：JDK 17 + Maven + Gradle + Fabric Java SDK + FISCO BCOS Java SDK（预拉取至本地 Maven 仓库）

FROM eclipse-temurin:17-jdk-jammy

ARG UBUNTU_MIRROR=mirrors.tuna.tsinghua.edu.cn
ARG VERSION=v1.0.0
LABEL org.opencontainers.image.title="java-dev"
LABEL org.opencontainers.image.description="Java 区块链开发环境镜像"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.vendor="LensChain"
LABEL lenschain.io/ecosystem="fabric"
LABEL lenschain.io/category="base"

ENV TZ=UTC DEBIAN_FRONTEND=noninteractive

RUN sed -i "s|archive.ubuntu.com|${UBUNTU_MIRROR}|g; s|security.ubuntu.com|${UBUNTU_MIRROR}|g" /etc/apt/sources.list && \
    apt-get -o Acquire::Retries=5 update && \
    for i in 1 2 3; do \
        apt-get -o Acquire::Retries=5 install -y --no-install-recommends --fix-missing \
            ca-certificates curl git maven gradle && break; \
        echo ">>> apt-get install attempt $i failed, waiting 10s..."; \
        sleep 10; apt-get clean; apt-get -o Acquire::Retries=5 update; \
    done && \
    rm -rf /var/lib/apt/lists/*

# 预拉取 Fabric Java SDK 和 FISCO BCOS Java SDK 依赖到本地 Maven 仓库
# 用户创建项目时无需重新下载
RUN mvn dependency:get -Dartifact=org.hyperledger.fabric-sdk-java:fabric-sdk-java:2.2.20 && \
    mvn dependency:get -Dartifact=org.fisco-bcos.java-sdk:fisco-bcos-java-sdk:3.6.0

RUN useradd -m -s /bin/bash -u 1001 developer

USER developer
WORKDIR /home/developer

CMD ["sleep", "infinity"]
