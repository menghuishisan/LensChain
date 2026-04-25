# tools/foundry/Dockerfile
# Foundry 工具套件（基于官方镜像薄封装）
# 包含 forge、cast、anvil、chisel 等 EVM 开发工具

ARG VERSION=nightly-5b7e4cb3c882b28f3c32ba580de27ce7381f415a
FROM ghcr.io/foundry-rs/foundry:${VERSION}

ARG VERSION
LABEL org.opencontainers.image.title="foundry"
LABEL org.opencontainers.image.description="Foundry 工具套件（forge/cast/anvil/chisel）"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.source="https://github.com/foundry-rs/foundry"
LABEL org.opencontainers.image.vendor="LensChain"
LABEL lenschain.io/ecosystem="ethereum"
LABEL lenschain.io/category="tools"
LABEL lenschain.io/scope="ctf"

ENV TZ=UTC

EXPOSE 8545

HEALTHCHECK --interval=30s --timeout=5s --retries=3 --start-period=10s \
    CMD forge --version || exit 1

CMD ["anvil", "--host", "0.0.0.0"]
