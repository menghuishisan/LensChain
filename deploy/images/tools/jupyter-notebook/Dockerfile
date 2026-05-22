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

# 预创建工作目录并交给 jovyan，与 manifest.yaml::default_volumes.mount_path 一一对应。
# runtime 会把 PVC 挂到该路径（详见 service/experiment/k8s_client.go::ensurePVC），
# 没挂时也保证目录存在，避免 jupyter 因为 root_dir 不存在直接退出。
USER root
RUN mkdir -p /home/jovyan/work && chown -R jovyan:users /home/jovyan/work
USER jovyan

# 覆写上游 CMD，做两件平台层标准化（与 code-server 同一模式）：
#
# 1. 关闭 jupyter 自带 token / password 鉴权
#    `--ServerApp.token=""` + `--ServerApp.password=""` 让 Web UI 无需登录即可使用。
#    平台层 `tool_proxy_cookie`（path-scoped HttpOnly JWT）已完成"学生本人 + 实例
#    Running + tool_kind=ide"三重校验，jupyter 自带的 token 是第二道门，且默认随机
#    生成，教学场景无法分发。这里关在镜像层，因为这是 jupyter 在本平台的标准运行方式，
#    不存在合法的"开"配置——任何引用 registry.lianjing.com/tools/jupyter-notebook
#    的模板（含教师克隆出的）都自动获得此行为，无需在 DB / 种子里重复声明。
#
# 2. 指定打开目录 `/home/jovyan/work`（root_dir）
#    上游默认 root_dir = /home/jovyan，会让学生看到 .ipynb_checkpoints / .jupyter
#    等系统配置文件。manifest.yaml 已声明 /home/jovyan/work 是 project_data 卷，
#    workdir 必须与其一致才能让"挂载点 = 学生工作区 = IDE 打开目录"三处统一。
#
# 还设了 `ip=0.0.0.0` 让 Pod 内所有接口都能接到反代的 SPDY 端口转发。
CMD ["start-notebook.sh", \
     "--ServerApp.ip=0.0.0.0", \
     "--ServerApp.port=8888", \
     "--ServerApp.token=", \
     "--ServerApp.password=", \
     "--ServerApp.root_dir=/home/jovyan/work", \
     "--ServerApp.allow_origin=*", \
     "--ServerApp.disable_check_xsrf=True"]
