# tools/ctf-tools/Dockerfile
# CTF 常用工具集（自建镜像）
# 集成 pwntools、binwalk、stegsolve、foremost 等 CTF 常用工具

FROM python:3.11-slim-bookworm

ARG DEBIAN_MIRROR=mirrors.tuna.tsinghua.edu.cn
ARG VERSION=v1.0.0
LABEL org.opencontainers.image.title="ctf-tools"
LABEL org.opencontainers.image.description="CTF 常用工具集"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.vendor="LensChain"
LABEL lenschain.io/ecosystem="general"
LABEL lenschain.io/category="tools"
LABEL lenschain.io/scope="ctf"

ENV TZ=UTC DEBIAN_FRONTEND=noninteractive \
    PIP_INDEX_URL=https://mirrors.aliyun.com/pypi/simple/ \
    PIP_TRUSTED_HOST=mirrors.aliyun.com

RUN sed -i "s|deb.debian.org|${DEBIAN_MIRROR}|g" /etc/apt/sources.list.d/debian.sources && \
    apt-get -o Acquire::Retries=5 update && \
    for i in 1 2 3; do \
        apt-get -o Acquire::Retries=5 install -y --no-install-recommends --fix-missing \
            ca-certificates curl git wget netcat-openbsd nmap \
            binwalk foremost steghide exiftool john hashcat \
            gdb ltrace strace file xxd && break; \
        echo ">>> apt-get install attempt $i failed, waiting 10s..."; \
        sleep 10; apt-get clean; apt-get -o Acquire::Retries=5 update; \
    done && \
    pip install --no-cache-dir pwntools z3-solver requests && \
    rm -rf /var/lib/apt/lists/*

RUN useradd -m -s /bin/bash -u 1001 ctfplayer

USER ctfplayer
WORKDIR /home/ctfplayer

CMD ["sleep", "infinity"]
