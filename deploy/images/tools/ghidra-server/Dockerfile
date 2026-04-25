# tools/ghidra-server/Dockerfile
# Ghidra 逆向工程 Web 服务（自建镜像）
# NSA 开源逆向分析工具，提供 Web 访问界面

FROM eclipse-temurin:17-jdk-jammy

ARG VERSION=v1.0.0
ARG GHIDRA_VERSION=11.0.3
LABEL org.opencontainers.image.title="ghidra-server"
LABEL org.opencontainers.image.description="Ghidra 逆向工程 Web 服务"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.source="https://github.com/NationalSecurityAgency/ghidra"
LABEL org.opencontainers.image.vendor="LensChain"
LABEL lenschain.io/ecosystem="general"
LABEL lenschain.io/category="tools"
LABEL lenschain.io/scope="ctf"

ENV TZ=UTC DEBIAN_FRONTEND=noninteractive

RUN apt-get update && \
    apt-get install -y --no-install-recommends \
        ca-certificates curl unzip && \
    curl -fsSL "https://github.com/NationalSecurityAgency/ghidra/releases/download/Ghidra_${GHIDRA_VERSION}_build/ghidra_${GHIDRA_VERSION}_PUBLIC_20240410.zip" \
        -o /tmp/ghidra.zip && \
    unzip /tmp/ghidra.zip -d /opt && \
    mv /opt/ghidra_* /opt/ghidra && \
    rm /tmp/ghidra.zip && \
    rm -rf /var/lib/apt/lists/*

RUN useradd -m -s /bin/bash -u 1001 ghidra && \
    chown -R ghidra:ghidra /opt/ghidra

USER ghidra
WORKDIR /opt/ghidra

EXPOSE 18080

HEALTHCHECK --interval=30s --timeout=10s --retries=3 --start-period=60s \
    CMD curl -fsS http://localhost:18080/ || exit 1

CMD ["/opt/ghidra/server/ghidraSvr", "start"]
