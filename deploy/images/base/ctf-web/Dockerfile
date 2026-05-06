# base/ctf-web/Dockerfile
# CTF Web 方向基础镜像
# 预装：Nginx + PHP 8 + Python Flask/FastAPI + Node.js + Express
# 允许 root 运行（部分漏洞题目需要 root 权限演示提权）

FROM debian:bookworm-slim

ARG DEBIAN_MIRROR=mirrors.tuna.tsinghua.edu.cn
ARG VERSION=v1.0.0
LABEL org.opencontainers.image.title="ctf-web"
LABEL org.opencontainers.image.description="CTF Web 方向基础镜像"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.vendor="LensChain"
LABEL lenschain.io/ecosystem="general"
LABEL lenschain.io/category="base"
LABEL lenschain.io/scope="ctf"

ENV TZ=UTC DEBIAN_FRONTEND=noninteractive \
    PIP_INDEX_URL=https://mirrors.aliyun.com/pypi/simple/ \
    PIP_TRUSTED_HOST=mirrors.aliyun.com

# 安装系统包 + NodeSource Node.js 20
RUN sed -i "s|deb.debian.org|${DEBIAN_MIRROR}|g" /etc/apt/sources.list.d/debian.sources && \
    apt-get -o Acquire::Retries=5 update && \
    for i in 1 2 3; do \
        apt-get -o Acquire::Retries=5 install -y --no-install-recommends --fix-missing \
            ca-certificates curl git gnupg nginx \
            php8.2 php8.2-fpm php8.2-mysql php8.2-sqlite3 php8.2-curl \
            python3 python3-pip python3-venv && break; \
        echo ">>> apt-get install attempt $i failed, waiting 10s..."; \
        sleep 10; apt-get clean; apt-get -o Acquire::Retries=5 update; \
    done && \
    curl -fsSL https://deb.nodesource.com/setup_20.x | bash - && \
    for i in 1 2 3; do \
        apt-get -o Acquire::Retries=5 install -y --no-install-recommends --fix-missing nodejs && break; \
        echo ">>> apt-get install nodejs attempt $i failed, waiting 10s..."; \
        sleep 10; apt-get clean; apt-get -o Acquire::Retries=5 update; \
    done && \
    npm install -g express express-generator && \
    pip3 install --no-cache-dir --break-system-packages \
        flask fastapi uvicorn jinja2 requests && \
    rm -rf /var/lib/apt/lists/*

EXPOSE 80 8080

HEALTHCHECK --interval=30s --timeout=5s --retries=3 --start-period=10s \
    CMD curl -fsS http://localhost:80/ || curl -fsS http://localhost:8080/ || exit 1

CMD ["nginx", "-g", "daemon off;"]
