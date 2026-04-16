# frontend.Dockerfile
# 平台前端服务镜像构建接入位
# 当前仓库尚未落地 frontend/ 代码目录，因此不提供伪可用镜像实现。
# 使用一个明确失败的构建步骤，避免误把占位文件当成可交付镜像。

FROM alpine:3.21
RUN printf '%s\n' 'frontend/ 代码目录尚未存在，当前只能保留 deploy 接入位，不能构建真实前端镜像。' >&2 && exit 1
