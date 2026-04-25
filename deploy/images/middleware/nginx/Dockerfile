# middleware/nginx/Dockerfile
# Nginx Web 服务器和反向代理（基于官方镜像薄封装）

ARG VERSION=1.25-alpine
FROM nginx:${VERSION}

ARG VERSION
LABEL org.opencontainers.image.title="nginx"
LABEL org.opencontainers.image.description="Nginx Web 服务器和反向代理"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.source="https://github.com/nginx/nginx"
LABEL org.opencontainers.image.vendor="LensChain"
LABEL lenschain.io/ecosystem="general"
LABEL lenschain.io/category="middleware"

ENV TZ=UTC
EXPOSE 80 443

HEALTHCHECK --interval=30s --timeout=5s --retries=3 \
    CMD curl -fsS http://localhost/ || exit 1
