# tools/radare2/Dockerfile
# radare2 逆向分析工具（基于官方镜像薄封装）
# 开源逆向工程框架，支持反汇编、调试、分析

ARG VERSION=5.9.0
FROM radare/radare2:${VERSION}

ARG VERSION
LABEL org.opencontainers.image.title="radare2"
LABEL org.opencontainers.image.description="radare2 逆向分析工具"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.source="https://github.com/radareorg/radare2"
LABEL org.opencontainers.image.vendor="LensChain"
LABEL lenschain.io/ecosystem="general"
LABEL lenschain.io/category="tools"
LABEL lenschain.io/scope="ctf"

ENV TZ=UTC

CMD ["r2", "-h"]
