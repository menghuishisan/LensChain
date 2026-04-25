# tools/jupyter-notebook/Dockerfile
# Jupyter Notebook 交互式编程环境（基于官方镜像薄封装）
# 支持 Python、数据分析、区块链交互式教学

ARG VERSION=python-3.11
FROM jupyter/base-notebook:${VERSION}

ARG VERSION
LABEL org.opencontainers.image.title="jupyter-notebook"
LABEL org.opencontainers.image.description="Jupyter Notebook 交互式编程环境"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.source="https://github.com/jupyter/docker-stacks"
LABEL org.opencontainers.image.vendor="LensChain"
LABEL lenschain.io/ecosystem="general"
LABEL lenschain.io/category="tools"

ENV TZ=UTC

EXPOSE 8888

HEALTHCHECK --interval=30s --timeout=5s --retries=3 --start-period=20s \
    CMD curl -fsS http://localhost:8888/api || exit 1
