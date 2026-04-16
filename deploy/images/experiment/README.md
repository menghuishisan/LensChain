# experiment/README.md
# 模块04 实验环境镜像目录

本目录用于存放实验环境模块运行时镜像定义，严格对应文档中的四类镜像：
- `chain-nodes/`：链节点镜像
- `middleware/`：中间件与工具链镜像
- `tools/`：实验操作工具镜像
- `base/`：基础开发环境镜像

每个镜像目录统一包含：
- `Dockerfile`
- `image.yaml`（镜像元数据）

其中 `tools/collector-agent-ethereum/`、`tools/collector-agent-fabric/`、`tools/collector-agent-chainmaker/`、`tools/collector-agent-fisco/`
用于混合实验数据采集 sidecar，分别对应模块 04 文档规定的四类内置 Collector Agent 镜像。
