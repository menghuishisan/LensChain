# experiment/IMAGE_CATALOG.md
# 模块04 实验环境镜像目录总表

本文档对应 `docs/modules/04-实验环境/07-实验类型与环境配置.md` 的完整镜像库要求。

## 1. 链节点镜像

### 已实现首批镜像
- `chain-nodes/bitcoind/`
- `chain-nodes/geth-dev/`
- `chain-nodes/besu/`
- `chain-nodes/ganache/`
- `chain-nodes/hardhat-node/`
- `chain-nodes/fabric-peer/`
- `chain-nodes/fabric-orderer/`
- `chain-nodes/fabric-ca/`

### 已落地接入位（含元数据与明确失败的构建保护）
- `chain-nodes/chainmaker-node/`
- `chain-nodes/fisco-bcos/`
- `chain-nodes/solana-validator/`
- `chain-nodes/substrate-node/`
- `chain-nodes/cosmos-node/`
- `chain-nodes/aptos-node/`
- `chain-nodes/op-node/`

## 2. 中间件镜像

### 已实现首批镜像
- `middleware/foundry/`
- `middleware/couchdb/`
- `middleware/postgres/`
- `middleware/redis/`
- `middleware/kafka/`
- `middleware/zookeeper/`
- `middleware/ipfs/`
- `middleware/minio/`
- `middleware/nginx/`

### 已纳入目录规划
- `middleware/couchdb/`
- `middleware/postgres/`
- `middleware/redis/`
- `middleware/kafka/`
- `middleware/zookeeper/`
- `middleware/ipfs/`
- `middleware/minio/`
- `middleware/nginx/`

## 3. 工具镜像

### 已实现首批镜像
- `tools/code-server/`
- `tools/jupyter-notebook/`
- `tools/xterm-server/`
- `tools/novnc-desktop/`
- `tools/remix-ide/`
- `tools/fabric-tools/`
- `tools/slither/`
- `tools/mythril/`
- `tools/echidna/`
- `tools/caliper/`
- `tools/grafana/`
- `tools/prometheus/`
- `tools/collector-agent-ethereum/`
- `tools/collector-agent-fabric/`
- `tools/collector-agent-chainmaker/`
- `tools/collector-agent-fisco/`

### 已纳入目录规划
- `tools/blockscout/`
- `tools/fabric-explorer/`
- `tools/webase-front/`
- `tools/webase-web/`
- `tools/chainmaker-explorer/`

## 4. 基础环境镜像

### 已实现首批镜像
- `base/solidity-dev/`
- `base/dapp-dev/`
- `base/go-dev/`
- `base/java-dev/`
- `base/rust-dev/`
- `base/python-dev/`
- `base/circom-dev/`

### 已纳入目录规划
- `base/solana-dev/`
- `base/substrate-dev/`
- `base/cosmos-dev/`
- `base/move-dev/`

## 5. 元数据约束

所有镜像目录应最终包含：
- `Dockerfile`
- `image.yaml`

推荐扩展字段：
- `default_ports`
- `default_env_vars`
- `default_volumes`
- `typical_companions`
- `required_dependencies`
- `resource_recommendation`
- `documentation_url`

## 6. 说明

本目录已经按照文档完整落下镜像分类与交付位。当前目录中的镜像要么已经提供可构建定义，要么已经作为明确的目录规划项纳入镜像总表，后续新增实现时不得改变目录职责与分类口径。
