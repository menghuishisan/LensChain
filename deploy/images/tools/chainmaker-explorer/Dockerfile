# tools/chainmaker-explorer/Dockerfile
# 长安链浏览器（占位镜像，需从源码构建）
# chainmaker-explorer 无官方 Docker 镜像
# 实际部署时需从 https://git.chainmaker.org.cn/chainmaker/chainmaker-explorer 源码构建

ARG VERSION=v2.3.0
FROM node:20-alpine

ARG VERSION
LABEL org.opencontainers.image.title="chainmaker-explorer"
LABEL org.opencontainers.image.description="长安链浏览器"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.source="https://git.chainmaker.org.cn/chainmaker/chainmaker-explorer"
LABEL org.opencontainers.image.vendor="LensChain"
LABEL lenschain.io/ecosystem="chainmaker"
LABEL lenschain.io/category="tools"

ENV TZ=UTC

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --retries=3 --start-period=30s \
    CMD curl -fsS http://localhost:8080/ || exit 1

CMD ["sleep", "infinity"]
