# tools/remix-ide/Dockerfile
# Remix IDE Solidity 在线开发环境（基于官方镜像薄封装）
# EVM 生态智能合约开发、编译、部署、调试

ARG VERSION=v0.35.0
FROM remixproject/remix-ide:${VERSION}

ARG VERSION
LABEL org.opencontainers.image.title="remix-ide"
LABEL org.opencontainers.image.description="Remix IDE Solidity 在线开发环境"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.source="https://github.com/ethereum/remix-project"
LABEL org.opencontainers.image.vendor="LensChain"
LABEL lenschain.io/ecosystem="ethereum"
LABEL lenschain.io/category="tools"

ENV TZ=UTC

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --retries=3 --start-period=15s \
    CMD wget -qO- http://localhost:8080 || exit 1
