# chain-nodes/fabric-ca/Dockerfile
# Hyperledger Fabric CA 证书颁发服务（基于官方镜像薄封装）
# 负责 Fabric 网络的身份管理和证书颁发

ARG VERSION=1.5
FROM hyperledger/fabric-ca:${VERSION}

ARG VERSION
LABEL org.opencontainers.image.title="fabric-ca"
LABEL org.opencontainers.image.description="Hyperledger Fabric CA 证书颁发服务"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.source="https://github.com/hyperledger/fabric-ca"
LABEL org.opencontainers.image.vendor="LensChain"
LABEL lenschain.io/ecosystem="fabric"
LABEL lenschain.io/category="chain-nodes"

ENV TZ=UTC

EXPOSE 7054

HEALTHCHECK --interval=30s --timeout=5s --retries=3 --start-period=15s \
    CMD curl -fsS http://localhost:7054/healthz || exit 1

CMD ["sh", "-c", "fabric-ca-server start -b admin:adminpw -d"]
