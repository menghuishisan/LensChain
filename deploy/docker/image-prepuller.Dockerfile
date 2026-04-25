# image-prepuller.Dockerfile
# 链镜平台镜像预拉取执行器镜像
# 用途：运行在 DaemonSet 中，读取 image-manifest 清单并通过 crictl 在节点侧拉取镜像

FROM alpine:3.19

ARG VERSION=dev
ARG COMMIT_SHA=unknown
ARG CRICTL_VERSION=1.30.0

LABEL org.opencontainers.image.title="lenschain-image-prepuller"
LABEL org.opencontainers.image.description="链镜平台镜像预拉取执行器"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.revision="${COMMIT_SHA}"
LABEL org.opencontainers.image.vendor="LensChain"
LABEL lenschain.io/service="image-prepuller"

ENV TZ=UTC

RUN apk add --no-cache bash ca-certificates curl tar tzdata && \
    curl -L "https://github.com/kubernetes-sigs/cri-tools/releases/download/v${CRICTL_VERSION}/crictl-v${CRICTL_VERSION}-linux-amd64.tar.gz" \
      | tar -xz -C /usr/local/bin crictl && \
    addgroup -S -g 1001 lenschain && \
    adduser -S -u 1001 -G lenschain lenschain && \
    mkdir -p /app /etc/image-manifest && \
    chown -R lenschain:lenschain /app /etc/image-manifest /usr/local/bin/crictl

WORKDIR /app

COPY --chown=lenschain:lenschain deploy/scripts/image-prepuller.sh /app/image-prepuller.sh

USER lenschain

ENTRYPOINT ["bash", "/app/image-prepuller.sh"]
