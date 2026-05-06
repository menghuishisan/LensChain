# tools/zaproxy/Dockerfile
# OWASP ZAP Web 安全测试代理（基于官方镜像薄封装）
# 开源 Web 应用安全扫描器，功能覆盖代理抓包/漏洞扫描/Fuzzer/重放
# CTF Web 方向安全测试

ARG VERSION=2.15.0
FROM ghcr.io/zaproxy/zaproxy:${VERSION}

ARG VERSION
LABEL org.opencontainers.image.title="zaproxy"
LABEL org.opencontainers.image.description="OWASP ZAP Web 安全测试代理"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.source="https://github.com/zaproxy/zaproxy"
LABEL org.opencontainers.image.vendor="LensChain"
LABEL lenschain.io/ecosystem="general"
LABEL lenschain.io/category="tools"
LABEL lenschain.io/scope="ctf"

ENV TZ=UTC

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --retries=3 --start-period=60s \
    CMD curl -fsS http://localhost:8080/ || exit 1
