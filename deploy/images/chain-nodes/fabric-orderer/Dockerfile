# chain-nodes/fabric-orderer/Dockerfile
# Hyperledger Fabric Orderer 排序节点（基于官方镜像薄封装）
# Fabric 网络排序服务，负责交易排序和区块生成

ARG VERSION=2.5
FROM hyperledger/fabric-orderer:${VERSION}

ARG VERSION
LABEL org.opencontainers.image.title="fabric-orderer"
LABEL org.opencontainers.image.description="Hyperledger Fabric Orderer 排序节点"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.source="https://github.com/hyperledger/fabric"
LABEL org.opencontainers.image.vendor="LensChain"
LABEL lenschain.io/ecosystem="fabric"
LABEL lenschain.io/category="chain-nodes"

ENV TZ=UTC

EXPOSE 7050

HEALTHCHECK --interval=30s --timeout=10s --retries=3 --start-period=30s \
    CMD curl -fsS http://localhost:8443/healthz || exit 1
