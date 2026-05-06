# base/ctf-reverse/Dockerfile
# CTF Reverse 方向基础镜像（多阶段构建）
# builder 阶段编译 Python C 扩展（capstone/keystone/unicorn），runtime 阶段拷贝产物
# 预装：radare2 + rizin + rz-ghidra + Ghidra + binwalk + pwntools + capstone + keystone + unicorn

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
        apt-get -o Acquire::Retries=5 install -y --no-install-recommends --fix-missing gcc g++ cmake pkg-config && break; \
        echo ">>> apt-get install attempt $i failed, waiting 10s..."; \
        sleep 10; apt-get clean; apt-get -o Acquire::Retries=5 update; \
    done && \
    pip install --no-cache-dir --prefix=/install \
        pwntools capstone keystone-engine unicorn && \
    rm -rf /var/lib/apt/lists/*

# ============================
# 运行阶段：CTF Reverse 环境
# ============================
FROM debian:bookworm-slim

ARG DEBIAN_MIRROR=mirrors.tuna.tsinghua.edu.cn
ARG VERSION=v1.0.0
ARG RADARE2_VERSION=6.1.4
ARG GHIDRA_VERSION=11.0.3
ARG RIZIN_VERSION=0.7.3
LABEL org.opencontainers.image.title="ctf-reverse"
LABEL org.opencontainers.image.description="CTF Reverse 方向基础镜像"
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
            ca-certificates curl git python3 \
            gdb ltrace strace file xxd binutils \
            binwalk openjdk-17-jre-headless unzip && break; \
        echo ">>> apt-get install attempt $i failed, waiting 10s..."; \
        sleep 10; apt-get clean; apt-get -o Acquire::Retries=5 update; \
    done && \
    # --- radare2 ---
    curl -fsSL "https://github.com/radareorg/radare2/releases/download/${RADARE2_VERSION}/radare2_${RADARE2_VERSION}_amd64.deb" \
        -o /tmp/radare2.deb && \
    dpkg -i /tmp/radare2.deb || apt-get install -f -y --no-install-recommends && \
    rm -f /tmp/radare2.deb && \
    # --- rizin + rz-ghidra（Cutter 核心引擎） ---
    curl -fsSL "https://github.com/rizinorg/rizin/releases/download/v${RIZIN_VERSION}/rizin-v${RIZIN_VERSION}-debian-bookworm-x86_64.deb" \
        -o /tmp/rizin.deb && \
    dpkg -i /tmp/rizin.deb || apt-get install -f -y --no-install-recommends && \
    rm -f /tmp/rizin.deb && \
    curl -fsSL "https://github.com/rizinorg/rz-ghidra/releases/download/v${RIZIN_VERSION}/rz-ghidra-v${RIZIN_VERSION}-debian-bookworm-x86_64.deb" \
        -o /tmp/rz-ghidra.deb && \
    dpkg -i /tmp/rz-ghidra.deb || apt-get install -f -y --no-install-recommends && \
    rm -f /tmp/rz-ghidra.deb && \
    # --- Ghidra ---
    curl -fsSL "https://github.com/NationalSecurityAgency/ghidra/releases/download/Ghidra_${GHIDRA_VERSION}_build/ghidra_${GHIDRA_VERSION}_PUBLIC_20240410.zip" \
        -o /tmp/ghidra.zip && \
    unzip /tmp/ghidra.zip -d /opt && \
    mv /opt/ghidra_* /opt/ghidra && \
    rm /tmp/ghidra.zip && \
    rm -rf /var/lib/apt/lists/*

COPY --from=builder /install /usr/local

ENV PATH="/opt/ghidra:${PATH}"

RUN useradd -m -s /bin/bash -u 1001 ctfplayer && \
    chown -R ctfplayer:ctfplayer /opt/ghidra

USER ctfplayer
WORKDIR /home/ctfplayer

CMD ["sleep", "infinity"]
