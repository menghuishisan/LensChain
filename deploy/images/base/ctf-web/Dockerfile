# base/ctf-web/Dockerfile
# CTF Web 方向基础镜像（多阶段构建）
# builder 阶段安装 Node.js 全局包，runtime 阶段拷贝产物
# 预装：Nginx + PHP 8 + Python Flask + Node.js + 常用 Web 框架
# 允许 root 运行（部分漏洞题目需要 root 权限演示提权）

# ============================
# 构建阶段：安装 Node.js 全局包
# ============================
FROM node:20-bullseye-slim AS node-builder

RUN npm install -g express && \
    npm cache clean --force

# ============================
# 运行阶段：CTF Web 环境
# ============================
FROM debian:bookworm-slim

ARG VERSION=v1.0.0
LABEL org.opencontainers.image.title="ctf-web"
LABEL org.opencontainers.image.description="CTF Web 方向基础镜像"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.vendor="LensChain"
LABEL lenschain.io/ecosystem="general"
LABEL lenschain.io/category="base"
LABEL lenschain.io/scope="ctf"

ENV TZ=UTC DEBIAN_FRONTEND=noninteractive

RUN apt-get update && \
    apt-get install -y --no-install-recommends \
        ca-certificates curl git nginx \
        php8.2 php8.2-fpm php8.2-mysql php8.2-sqlite3 php8.2-curl \
        python3 python3-pip python3-venv \
        nodejs npm && \
    pip3 install --no-cache-dir --break-system-packages \
        flask fastapi uvicorn jinja2 requests && \
    rm -rf /var/lib/apt/lists/*

COPY --from=node-builder /usr/local/lib/node_modules /usr/local/lib/node_modules
COPY --from=node-builder /usr/local/bin/express /usr/local/bin/express

EXPOSE 80 8080

HEALTHCHECK --interval=30s --timeout=5s --retries=3 --start-period=10s \
    CMD curl -fsS http://localhost:80/ || curl -fsS http://localhost:8080/ || exit 1

CMD ["nginx", "-g", "daemon off;"]
