# tools/sagemath/Dockerfile
# SageMath 数学计算环境（基于官方镜像薄封装）
# CTF Crypto 方向密码学分析

ARG VERSION=10.3
FROM sagemath/sagemath:${VERSION}

ARG VERSION
LABEL org.opencontainers.image.title="sagemath"
LABEL org.opencontainers.image.description="SageMath 数学计算环境"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.source="https://github.com/sagemath/sage"
LABEL org.opencontainers.image.vendor="LensChain"
LABEL lenschain.io/ecosystem="general"
LABEL lenschain.io/category="tools"
LABEL lenschain.io/scope="ctf"

ENV TZ=UTC

EXPOSE 8888

HEALTHCHECK --interval=30s --timeout=10s --retries=3 --start-period=30s \
    CMD sage --version || exit 1

CMD ["sage", "-notebook=jupyter", "--ip=0.0.0.0", "--port=8888", "--no-browser"]
