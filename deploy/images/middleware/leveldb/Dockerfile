# middleware/leveldb/Dockerfile
# LevelDB 工具镜像（自建）
# 提供 LevelDB 命令行工具，用于数据查看和调试

FROM alpine:3.19

ARG VERSION=v1.0.0
LABEL org.opencontainers.image.title="leveldb"
LABEL org.opencontainers.image.description="LevelDB 工具镜像"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.vendor="LensChain"
LABEL lenschain.io/ecosystem="general"
LABEL lenschain.io/category="middleware"

ENV TZ=UTC

RUN apk add --no-cache leveldb leveldb-tools ca-certificates tzdata && \
    addgroup -S lenschain && adduser -S lenschain -G lenschain

USER lenschain

CMD ["sh"]
