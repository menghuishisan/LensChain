# tools/mythril/Dockerfile
# Mythril 智能合约安全分析工具（基于官方镜像薄封装）
# 使用符号执行检测 EVM 字节码安全漏洞

ARG VERSION=0.24.7
FROM mythril/myth:${VERSION}

ARG VERSION
LABEL org.opencontainers.image.title="mythril"
LABEL org.opencontainers.image.description="Mythril 智能合约安全分析工具"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.source="https://github.com/Consensys/mythril"
LABEL org.opencontainers.image.vendor="LensChain"
LABEL lenschain.io/ecosystem="ethereum"
LABEL lenschain.io/category="tools"

ENV TZ=UTC

CMD ["myth", "--help"]
