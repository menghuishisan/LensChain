# chain-nodes/fabric-peer/Dockerfile
# Hyperledger Fabric Peer 节点（基于官方镜像薄封装）
# Fabric 网络核心组件，负责背书和提交交易
# 注意：Fabric 官方镜像以 root 运行，不做非 root 改造

ARG VERSION=2.5
FROM hyperledger/fabric-peer:${VERSION}

ARG VERSION
LABEL org.opencontainers.image.title="fabric-peer"
LABEL org.opencontainers.image.description="Hyperledger Fabric Peer 节点"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.source="https://github.com/hyperledger/fabric"
LABEL org.opencontainers.image.vendor="LensChain"
LABEL lenschain.io/ecosystem="fabric"
LABEL lenschain.io/category="chain-nodes"

ENV TZ=UTC

EXPOSE 7051 7053

HEALTHCHECK --interval=30s --timeout=10s --retries=3 --start-period=30s \
    CMD peer node status || exit 1
