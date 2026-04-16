# ctf/README.md
# 模块05 CTF 竞赛镜像目录

本目录用于存放 CTF 竞赛运行时镜像，覆盖模块05文档要求的解题赛与攻防赛场景。

当前目录分层：
- `targets/`：题目环境/靶机镜像
- `chain-nodes/`：链节点与队伍链/裁判链运行环境
- `middleware/`：验证、构建、预验证等中间件工具链
- `tools/`：选手工具镜像
- `base/`：比赛基础镜像

每个镜像目录统一包含：
- `Dockerfile`
- `image.yaml`

后续可以继续补充：
- judge-chain / judge-service 镜像
- reverse / misc / crypto 题型镜像
- 动态 Flag 注入脚本与校验侧车
