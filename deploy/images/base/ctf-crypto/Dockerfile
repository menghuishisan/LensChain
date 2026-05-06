# base/ctf-crypto/Dockerfile
# CTF Crypto 方向基础镜像（多阶段构建）
# builder 阶段编译 C 扩展依赖（gmpy2 等），runtime 阶段拷贝 Python 包
# 预装：Python + pycryptodome + gmpy2 + z3 + sympy + pwntools + SageMath

# ============================
# 构建阶段：编译 Python C 扩展
# ============================
FROM python:3.11-slim-bookworm AS builder

ARG DEBIAN_MIRROR=mirrors.tuna.tsinghua.edu.cn
ENV PIP_INDEX_URL=https://mirrors.aliyun.com/pypi/simple/ \
    PIP_TRUSTED_HOST=mirrors.aliyun.com

RUN sed -i "s|deb.debian.org|${DEBIAN_MIRROR}|g" /etc/apt/sources.list.d/debian.sources && \
    apt-get -o Acquire::Retries=5 update && \
    for i in 1 2 3; do \
        apt-get -o Acquire::Retries=5 install -y --no-install-recommends --fix-missing \
            gcc g++ libgmp-dev libmpfr-dev libmpc-dev && break; \
        echo ">>> apt-get install attempt $i failed, waiting 10s..."; \
        sleep 10; apt-get clean; apt-get -o Acquire::Retries=5 update; \
    done && \
    pip install --no-cache-dir --prefix=/install \
        pycryptodome gmpy2 z3-solver sympy pwntools && \
    rm -rf /var/lib/apt/lists/*

# ============================
# 运行阶段：CTF Crypto 环境
# ============================
FROM python:3.11-slim-bookworm

ARG DEBIAN_MIRROR=mirrors.tuna.tsinghua.edu.cn
ARG VERSION=v1.0.0
LABEL org.opencontainers.image.title="ctf-crypto"
LABEL org.opencontainers.image.description="CTF Crypto 方向基础镜像"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.vendor="LensChain"
LABEL lenschain.io/ecosystem="general"
LABEL lenschain.io/category="base"
LABEL lenschain.io/scope="ctf"

ENV TZ=UTC DEBIAN_FRONTEND=noninteractive \
    PIP_INDEX_URL=https://mirrors.aliyun.com/pypi/simple/ \
    PIP_TRUSTED_HOST=mirrors.aliyun.com

RUN sed -i "s|deb.debian.org|${DEBIAN_MIRROR}|g" /etc/apt/sources.list.d/debian.sources && \
    apt-get -o Acquire::Retries=5 update && \
    for i in 1 2 3; do \
        apt-get -o Acquire::Retries=5 install -y --no-install-recommends --fix-missing \
            ca-certificates curl git libgmp10 libmpfr6 libmpc3 \
            sagemath && break; \
        echo ">>> apt-get install attempt $i failed, waiting 10s..."; \
        sleep 10; apt-get clean; apt-get -o Acquire::Retries=5 update; \
    done && \
    rm -rf /var/lib/apt/lists/*

COPY --from=builder /install /usr/local

RUN useradd -m -s /bin/bash -u 1001 ctfplayer

USER ctfplayer
WORKDIR /home/ctfplayer

CMD ["sleep", "infinity"]
