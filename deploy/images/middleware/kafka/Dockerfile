# middleware/kafka/Dockerfile
# Apache Kafka 消息队列（基于 Confluent 官方镜像薄封装）
# Fabric 排序服务（旧版本）和通用消息队列

ARG VERSION=7.6.0
FROM confluentinc/cp-kafka:${VERSION}

ARG VERSION
LABEL org.opencontainers.image.title="kafka"
LABEL org.opencontainers.image.description="Apache Kafka 消息队列"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.source="https://github.com/confluentinc/kafka-images"
LABEL org.opencontainers.image.vendor="LensChain"
LABEL lenschain.io/ecosystem="general"
LABEL lenschain.io/category="middleware"

ENV TZ=UTC
EXPOSE 9092

HEALTHCHECK --interval=30s --timeout=10s --retries=3 --start-period=30s \
    CMD kafka-broker-api-versions --bootstrap-server localhost:9092 || exit 1
