# tools/ctf-tools/Dockerfile
# CTF 常用工具集（自建镜像）
# 集成 pwntools、binwalk、stegsolve、foremost 等 CTF 常用工具

FROM python:3.11-slim-bookworm

ARG VERSION=v1.0.0
LABEL org.opencontainers.image.title="ctf-tools"
LABEL org.opencontainers.image.description="CTF 常用工具集"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.vendor="LensChain"
LABEL lenschain.io/ecosystem="general"
LABEL lenschain.io/category="tools"
LABEL lenschain.io/scope="ctf"

ENV TZ=UTC DEBIAN_FRONTEND=noninteractive

RUN apt-get update && \
    apt-get install -y --no-install-recommends \
        ca-certificates curl git wget netcat-openbsd nmap \
        binwalk foremost steghide exiftool john hashcat \
        gdb ltrace strace file xxd && \
    pip install --no-cache-dir pwntools z3-solver requests && \
    rm -rf /var/lib/apt/lists/*

RUN useradd -m -s /bin/bash -u 1001 ctfplayer

USER ctfplayer
WORKDIR /home/ctfplayer

CMD ["bash"]
