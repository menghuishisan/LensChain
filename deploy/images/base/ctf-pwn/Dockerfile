# base/ctf-pwn/Dockerfile
# CTF Pwn 方向基础镜像（多阶段构建）
# builder 阶段编译 Python/Ruby 扩展，runtime 阶段拷贝产物
# 预装：gdb + pwntools + ROPgadget + one_gadget

# ============================
# 构建阶段：编译 Python/Ruby 扩展
# ============================
FROM debian:bookworm-slim AS builder

ARG DEBIAN_MIRROR=mirrors.ustc.edu.cn
ENV DEBIAN_FRONTEND=noninteractive

RUN sed -i "s|deb.debian.org|${DEBIAN_MIRROR}|g" /etc/apt/sources.list && \
    apt-get -o Acquire::Retries=5 update && \
    apt-get install -y --no-install-recommends \
        python3 python3-pip gcc g++ ruby ruby-dev make && \
    pip3 install --no-cache-dir --break-system-packages --prefix=/install-py \
        pwntools ROPGadget && \
    gem install --install-dir /install-gem one_gadget && \
    rm -rf /var/lib/apt/lists/*

# ============================
# 运行阶段：CTF Pwn 环境
# ============================
FROM debian:bookworm-slim

ARG DEBIAN_MIRROR=mirrors.ustc.edu.cn
ARG VERSION=v1.0.0
LABEL org.opencontainers.image.title="ctf-pwn"
LABEL org.opencontainers.image.description="CTF Pwn 方向基础镜像"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.vendor="LensChain"
LABEL lenschain.io/ecosystem="general"
LABEL lenschain.io/category="base"
LABEL lenschain.io/scope="ctf"

ENV TZ=UTC DEBIAN_FRONTEND=noninteractive

RUN sed -i "s|deb.debian.org|${DEBIAN_MIRROR}|g" /etc/apt/sources.list && \
    apt-get -o Acquire::Retries=5 update && \
    apt-get install -y --no-install-recommends \
        ca-certificates curl git python3 ruby \
        gdb gdbserver gcc g++ make nasm \
        libc6-dbg libc6-dev \
        ltrace strace file xxd netcat-openbsd socat && \
        rm -rf /var/lib/apt/lists/*

COPY --from=builder /install-py /usr/local
COPY --from=builder /install-gem /usr/local/lib/ruby/gems

RUN useradd -m -s /bin/bash -u 1001 ctfplayer

USER ctfplayer
WORKDIR /home/ctfplayer

CMD ["sleep", "infinity"]
