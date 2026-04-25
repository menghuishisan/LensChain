# tools/caliper/Dockerfile
# Hyperledger Caliper 区块链性能测试工具（基于 Node.js 自建）
# 支持 Fabric / FISCO / Ethereum 等多链性能基准测试

FROM node:20-alpine

ARG VERSION=v0.6.0
LABEL org.opencontainers.image.title="caliper"
LABEL org.opencontainers.image.description="Hyperledger Caliper 区块链性能测试工具"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.source="https://github.com/hyperledger/caliper"
LABEL org.opencontainers.image.vendor="LensChain"
LABEL lenschain.io/ecosystem="general"
LABEL lenschain.io/category="tools"

ENV TZ=UTC

RUN npm install -g @hyperledger/caliper-cli@0.6.0 && \
    addgroup -S lenschain && adduser -S lenschain -G lenschain

USER lenschain
WORKDIR /home/lenschain

CMD ["sleep", "infinity"]
