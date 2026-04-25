# tools/slither/Dockerfile
# Slither 智能合约静态分析工具（基于官方镜像薄封装）
# 检测 Solidity 合约安全漏洞

ARG VERSION=0.10.1
FROM trailofbits/slither:${VERSION}

ARG VERSION
LABEL org.opencontainers.image.title="slither"
LABEL org.opencontainers.image.description="Slither 智能合约静态分析工具"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.source="https://github.com/crytic/slither"
LABEL org.opencontainers.image.vendor="LensChain"
LABEL lenschain.io/ecosystem="ethereum"
LABEL lenschain.io/category="tools"

ENV TZ=UTC

CMD ["slither", "--help"]
