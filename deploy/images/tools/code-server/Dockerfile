# tools/code-server/Dockerfile
# VS Code Web 在线编辑器（基于官方镜像薄封装）
# 提供浏览器内 VS Code 开发环境，支持所有链生态

ARG VERSION=4.89.1
FROM codercom/code-server:${VERSION}

ARG VERSION
LABEL org.opencontainers.image.title="code-server"
LABEL org.opencontainers.image.description="VS Code Web 在线编辑器"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.source="https://github.com/coder/code-server"
LABEL org.opencontainers.image.vendor="LensChain"
LABEL lenschain.io/ecosystem="general"
LABEL lenschain.io/category="tools"

ENV TZ=UTC

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --retries=3 --start-period=15s \
    CMD curl -fsS http://localhost:8080/healthz || exit 1

# 预创建项目工作目录并交给 coder 用户，与 manifest.yaml::default_volumes.mount_path
# 一一对应。runtime 会把 emptyDir / PVC 挂到该路径，没挂时也保证存在，避免 IDE 因为
# 目录不存在而 fallback 到 /home/coder（露出 .bashrc/.profile 等系统 dotfiles）。
USER root
RUN mkdir -p /home/coder/project && chown -R coder:coder /home/coder/project
USER coder

# 覆写上游 ENTRYPOINT，做两件平台层标准化：
#
# 1. 关闭 code-server 自带密码层（`--auth none`）
#    链镜平台在 iframe 加载前已通过 `tool_proxy_cookie`（path-scoped HttpOnly JWT）完成
#    "学生本人 + 实例 Running + tool_kind 匹配"三重校验（详见
#    backend/internal/middleware/tool_proxy_auth.go 与 backend/internal/handler/experiment/
#    tool_proxy.go）。code-server 自带的密码层对学生而言是第二道门，且密码默认随机生成，
#    教学场景下无法分发。
#
# 2. 显式指定打开目录 `/home/coder/project`
#    上游 ENTRYPOINT 末尾的 `.` = CWD = /home/coder（基础镜像 WORKDIR），会让 IDE 打开
#    用户家目录，学生看到的是 .bashrc/.config 这类系统文件，而非项目工作区——既不教学
#    友好，也与 manifest.yaml::default_volumes 声明的 project_data 卷不对齐。
#
# 这两件事都属于 code-server 在本平台的"标准运行方式"，没有合法的"开"配置，因此放在
# 镜像层；任何引用 registry.lianjing.com/tools/code-server 的模板（含教师克隆出的模板）
# 都自动获得正确行为，无需在 DB / 种子 / template_containers 里逐个重复声明。
#
# 上游 codercom/code-server ENTRYPOINT 为
#   ["/usr/bin/entrypoint.sh", "--bind-addr", "0.0.0.0:8080", "."]
ENTRYPOINT ["/usr/bin/entrypoint.sh", "--auth", "none", "--bind-addr", "0.0.0.0:8080", "/home/coder/project"]
