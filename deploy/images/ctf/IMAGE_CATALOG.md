# ctf/IMAGE_CATALOG.md
# 模块05 CTF 竞赛镜像目录总表

本文档对应 `docs/modules/05-CTF竞赛/01-功能需求说明.md` 中的题型、赛制、运行环境与链上战场要求。

## 1. 题目环境镜像（targets）

### 已实现首批镜像
- `targets/web/`

### 已预留目录（待补具体 Dockerfile 与元数据）
- `targets/crypto/`
- `targets/contract/`
- `targets/blockchain/`
- `targets/reverse/`
- `targets/misc/`

## 2. 链节点镜像（chain-nodes）

### 已实现首批镜像
- `chain-nodes/ganache/`

### 设计用途
- 解题赛：每队每题独立链环境
- 攻防赛：队伍链 / 裁判链运行环境基础

## 3. 中间件镜像（middleware）

### 已实现首批镜像
- `middleware/foundry/`

### 用途
- 智能合约题目构建
- 官方 PoC 预验证
- 断言验证脚本执行

## 4. 选手工具镜像（tools）

### 已实现首批镜像
- `tools/ctf-tools/`
- `tools/reverse-tools/`
- `tools/blockchain-tools/`
- `tools/crypto-tools/`
- `tools/team-tools/`

### 已预留目录（待补具体 Dockerfile 与元数据）
- `tools/reverse-tools/`
- `tools/blockchain-tools/`
- `tools/crypto-tools/`
- `tools/team-tools/`

## 5. 基础镜像（base）

### 已实现首批镜像
- `base/ctf-web/`
- `base/ctf-blockchain/`
- `base/ctf-reverse/`
- `base/ctf-crypto/`
- `base/ctf-misc/`

### 已预留目录（待补具体 Dockerfile 与元数据）
- `base/judge-chain/`
- `base/judge-service/`

## 6. 与赛制的映射

### 解题赛（Jeopardy）
- namespace：`ctf-{competition_id}-{team_id}-{challenge_id}`
- 基础资源：题目容器 + 选手工具容器
- 典型镜像组合：
  - web：`targets/web` + `tools/ctf-tools`
  - contract：`chain-nodes/ganache` + `middleware/foundry` + `base/ctf-blockchain`
  - reverse：`targets/reverse` + `tools/reverse-tools`

### 攻防对抗赛（Attack-Defense）
- namespace：`ctf-ad-{competition_id}-{group_id}`
- 基础资源：judge-chain + judge-service + team-{n}-chain + team-{n}-tools
- 典型镜像组合：
  - `base/judge-chain`
  - `base/judge-service`
  - `chain-nodes/ganache`
  - `tools/team-tools`

## 7. 元数据约束

所有镜像目录应最终包含：
- `Dockerfile`
- `image.yaml`

后续建议补充：
- Flag 注入模式（静态/动态）
- 断言验证契约
- Team/Challenge 生命周期钩子
- 防作弊限流侧车或策略标签

## 8. 说明

本目录已经按文档完整落下类型结构与运行位；当前首批对 Web、智能合约、链环境和通用工具做了直接实现，其余目录已预留为后续题型和赛制扩展位。
