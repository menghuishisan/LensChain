# middleware/zookeeper/Dockerfile
# Apache ZooKeeper 分布式协调服务（基于 Confluent 官方镜像薄封装）
# Kafka 依赖组件

ARG VERSION=7.6.0
FROM confluentinc/cp-zookeeper:${VERSION}

ARG VERSION
LABEL org.opencontainers.image.title="zookeeper"
LABEL org.opencontainers.image.description="Apache ZooKeeper 分布式协调服务"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.source="https://github.com/confluentinc/kafka-images"
LABEL org.opencontainers.image.vendor="LensChain"
LABEL lenschain.io/ecosystem="general"
LABEL lenschain.io/category="middleware"

ENV TZ=UTC
EXPOSE 2181

HEALTHCHECK --interval=30s --timeout=5s --retries=3 --start-period=15s \
    CMD echo ruok | nc localhost 2181 | grep imok || exit 1
