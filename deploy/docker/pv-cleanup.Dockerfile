# pv-cleanup.Dockerfile
# 链镜平台实验卷清理执行器镜像
# 用途：由 CronJob 定期运行，对超过保留期的实验相关 PV 执行清理

FROM alpine:3.19

ARG VERSION=dev
ARG COMMIT_SHA=unknown
ARG KUBECTL_VERSION=v1.30.2

LABEL org.opencontainers.image.title="lenschain-pv-cleanup"
LABEL org.opencontainers.image.description="链镜平台实验卷清理执行器"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.revision="${COMMIT_SHA}"
LABEL org.opencontainers.image.vendor="LensChain"
LABEL lenschain.io/service="pv-cleanup"

ENV TZ=UTC

RUN for i in 1 2 3; do \
      apk add --no-cache bash ca-certificates curl jq coreutils tzdata && break; \
      echo ">>> apk add attempt $i failed, waiting 10s..."; sleep 10; \
    done && \
    curl -L "https://dl.k8s.io/release/${KUBECTL_VERSION}/bin/linux/amd64/kubectl" -o /usr/local/bin/kubectl && \
    chmod +x /usr/local/bin/kubectl && \
    addgroup -S -g 1001 lenschain && \
    adduser -S -u 1001 -G lenschain lenschain && \
    mkdir -p /app && \
    chown -R lenschain:lenschain /app /usr/local/bin/kubectl

WORKDIR /app

COPY --chown=lenschain:lenschain deploy/scripts/bash/pv-cleanup.sh /app/pv-cleanup.sh

USER lenschain

ENTRYPOINT ["bash", "/app/pv-cleanup.sh"]
