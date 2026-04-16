# images/README.md
# 运行时镜像总览

`deploy/images/` 是模块04（实验环境）与模块05（CTF竞赛）共享的运行时镜像目录。

- `experiment/`：实验环境镜像
- `ctf/`：CTF 竞赛镜像

设计原则：
- 平台服务镜像不进入本目录
- 运行时镜像必须与业务场景对齐
- 每个镜像目录至少包含 `Dockerfile` 与 `image.yaml`
- 镜像元数据应逐步扩展为后端编排服务可消费的模板配置来源
