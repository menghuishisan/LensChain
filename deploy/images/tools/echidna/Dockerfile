# tools/echidna/Dockerfile
# Echidna 智能合约模糊测试工具（基于官方镜像薄封装）
# 基于属性的 Solidity 模糊测试框架

ARG VERSION=v2.2.4
FROM trailofbits/echidna:${VERSION}

ARG VERSION
LABEL org.opencontainers.image.title="echidna"
LABEL org.opencontainers.image.description="Echidna 智能合约模糊测试工具"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.source="https://github.com/crytic/echidna"
LABEL org.opencontainers.image.vendor="LensChain"
LABEL lenschain.io/ecosystem="ethereum"
LABEL lenschain.io/category="tools"

ENV TZ=UTC

CMD ["sleep", "infinity"]
