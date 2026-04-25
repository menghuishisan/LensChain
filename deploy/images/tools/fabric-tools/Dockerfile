# tools/fabric-tools/Dockerfile
# Hyperledger Fabric CLI 工具集（基于官方镜像薄封装）
# 提供 peer、configtxgen、cryptogen 等 Fabric 命令行工具

ARG VERSION=2.5
FROM hyperledger/fabric-tools:${VERSION}

ARG VERSION
LABEL org.opencontainers.image.title="fabric-tools"
LABEL org.opencontainers.image.description="Hyperledger Fabric CLI 工具集"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.source="https://github.com/hyperledger/fabric"
LABEL org.opencontainers.image.vendor="LensChain"
LABEL lenschain.io/ecosystem="fabric"
LABEL lenschain.io/category="tools"

ENV TZ=UTC

CMD ["bash"]
