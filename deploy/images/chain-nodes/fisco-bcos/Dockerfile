# chain-nodes/fisco-bcos/Dockerfile
# FISCO BCOS 联盟链节点（基于官方镜像薄封装）
# 国产金融级联盟链平台

ARG VERSION=v3.6.0
FROM fiscoorg/fiscobcos:${VERSION}

ARG VERSION
LABEL org.opencontainers.image.title="fisco-bcos"
LABEL org.opencontainers.image.description="FISCO BCOS 联盟链节点"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.source="https://github.com/FISCO-BCOS/FISCO-BCOS"
LABEL org.opencontainers.image.vendor="LensChain"
LABEL lenschain.io/ecosystem="fisco"
LABEL lenschain.io/category="chain-nodes"

ENV TZ=UTC

EXPOSE 20200 8545

HEALTHCHECK --interval=30s --timeout=10s --retries=3 --start-period=30s \
    CMD curl -fsS http://localhost:8545 -X POST -H 'Content-Type: application/json' \
        -d '{"jsonrpc":"2.0","id":1,"method":"getBlockNumber","params":[1]}' || exit 1
