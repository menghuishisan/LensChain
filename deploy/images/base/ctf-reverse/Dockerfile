# base/ctf-reverse/Dockerfile
# CTF Reverse 方向基础镜像（多阶段构建）
# builder 阶段编译 Python C 扩展（capstone/keystone/unicorn），runtime 阶段拷贝产物
# 预装：radare2 + binwalk + pwntools + capstone + keystone + unicorn

# ============================
# 构建阶段：编译 Python C 扩展
# ============================
FROM python:3.11-slim-bookworm AS builder

ARG DEBIAN_MIRROR=mirrors.ustc.edu.cn

RUN sed -i "s|deb.debian.org|${DEBIAN_MIRROR}|g" /etc/apt/sources.list.d/debian.sources && \
    apt-get -o Acquire::Retries=5 update && \
    apt-get install -y --no-install-recommends gcc g++ cmake pkg-config && \
    pip install --no-cache-dir --prefix=/install \
        pwntools capstone keystone-engine unicorn && \
    rm -rf /var/lib/apt/lists/*

# ============================
# 运行阶段：CTF Reverse 环境
# ============================
FROM debian:bookworm-slim

ARG DEBIAN_MIRROR=mirrors.ustc.edu.cn
ARG VERSION=v1.0.0
LABEL org.opencontainers.image.title="ctf-reverse"
LABEL org.opencontainers.image.description="CTF Reverse 方向基础镜像"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.vendor="LensChain"
LABEL lenschain.io/ecosystem="general"
LABEL lenschain.io/category="base"
LABEL lenschain.io/scope="ctf"

ENV TZ=UTC DEBIAN_FRONTEND=noninteractive

RUN sed -i "s|deb.debian.org|${DEBIAN_MIRROR}|g" /etc/apt/sources.list && \
    apt-get -o Acquire::Retries=5 update && \
    apt-get install -y --no-install-recommends \
        ca-certificates curl git python3 \
        gdb ltrace strace file xxd binutils \
        radare2 binwalk && \
    rm -rf /var/lib/apt/lists/*

COPY --from=builder /install /usr/local

RUN useradd -m -s /bin/bash -u 1001 ctfplayer

USER ctfplayer
WORKDIR /home/ctfplayer

CMD ["sleep", "infinity"]
