# image-gc.Dockerfile
# 链镜平台镜像版本清理执行器镜像
# 用途：由 CronJob 定期运行，调用 OCI Registry API 清理超出保留窗口的旧标签

FROM alpine:3.19

ARG VERSION=dev
ARG COMMIT_SHA=unknown

LABEL org.opencontainers.image.title="lenschain-image-gc"
LABEL org.opencontainers.image.description="链镜平台镜像版本清理执行器"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.revision="${COMMIT_SHA}"
LABEL org.opencontainers.image.vendor="LensChain"
LABEL lenschain.io/service="image-gc"

ENV TZ=UTC

RUN for i in 1 2 3; do \
      apk add --no-cache bash ca-certificates curl jq coreutils tzdata && break; \
      echo ">>> apk add attempt $i failed, waiting 10s..."; sleep 10; \
    done && \
    addgroup -S -g 1001 lenschain && \
    adduser -S -u 1001 -G lenschain lenschain && \
    mkdir -p /app && \
    chown -R lenschain:lenschain /app

WORKDIR /app

COPY --chown=lenschain:lenschain deploy/scripts/bash/image-gc.sh /app/image-gc.sh

USER lenschain

ENTRYPOINT ["bash", "/app/image-gc.sh"]
