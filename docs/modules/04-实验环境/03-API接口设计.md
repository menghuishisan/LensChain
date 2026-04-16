# 实验环境模块 — API 接口设计

> 模块状态：✅ 已确认
> 文档版本：v3.0
> 遵循规范：[API规范](../../standards/API规范.md)

---

## 一、接口总览

### 1.1 镜像管理（超级管理员）

| 方法 | 路径 | 说明 | 权限 |
|------|------|------|------|
| GET | /api/v1/images | 镜像列表 | 超级管理员/学校管理员/教师 |
| POST | /api/v1/images | 创建/上传镜像 | 超级管理员/教师 |
| GET | /api/v1/images/:id | 镜像详情 | 超级管理员/学校管理员/教师 |
| PUT | /api/v1/images/:id | 编辑镜像信息 | 超级管理员/镜像上传者 |
| DELETE | /api/v1/images/:id | 删除/下架镜像 | 超级管理员 |
| POST | /api/v1/images/:id/review | 审核镜像 | 超级管理员 |
| GET | /api/v1/image-categories | 镜像分类列表 | 超级管理员/教师 |

### 1.2 镜像版本管理

| 方法 | 路径 | 说明 | 权限 |
|------|------|------|------|
| GET | /api/v1/images/:id/versions | 镜像版本列表 | 超级管理员/教师 |
| POST | /api/v1/images/:id/versions | 添加镜像版本 | 超级管理员/镜像上传者 |
| PUT | /api/v1/image-versions/:id | 编辑镜像版本 | 超级管理员/镜像上传者 |
| DELETE | /api/v1/image-versions/:id | 删除镜像版本 | 超级管理员 |
| PATCH | /api/v1/image-versions/:id/default | 设为默认版本 | 超级管理员/镜像上传者 |

### 1.3 实验模板管理（教师）

| 方法 | 路径 | 说明 | 权限 |
|------|------|------|------|
| POST | /api/v1/experiment-templates | 创建实验模板 | 教师 |
| GET | /api/v1/experiment-templates | 实验模板列表（教师视角） | 教师 |
| GET | /api/v1/experiment-templates/:id | 实验模板详情 | 教师 |
| PUT | /api/v1/experiment-templates/:id | 编辑实验模板 | 模板创建教师 |
| DELETE | /api/v1/experiment-templates/:id | 删除实验模板 | 模板创建教师 |
| POST | /api/v1/experiment-templates/:id/publish | 发布实验模板 | 模板创建教师 |
| POST | /api/v1/experiment-templates/:id/clone | 克隆实验模板 | 教师 |
| PATCH | /api/v1/experiment-templates/:id/share | 设置共享状态 | 模板创建教师 |
| GET | /api/v1/experiment-templates/:id/k8s-config | 查看K8s编排配置 | 模板创建教师 |
| POST | /api/v1/experiment-templates/:id/k8s-config | 微调K8s编排配置 | 模板创建教师 |

### 1.4 模板容器配置

| 方法 | 路径 | 说明 | 权限 |
|------|------|------|------|
| GET | /api/v1/experiment-templates/:id/containers | 容器配置列表 | 模板创建教师 |
| POST | /api/v1/experiment-templates/:id/containers | 添加容器配置 | 模板创建教师 |
| PUT | /api/v1/template-containers/:id | 编辑容器配置 | 模板创建教师 |
| DELETE | /api/v1/template-containers/:id | 删除容器配置 | 模板创建教师 |
| PUT | /api/v1/experiment-templates/:id/containers/sort | 容器排序 | 模板创建教师 |

### 1.5 检查点管理

| 方法 | 路径 | 说明 | 权限 |
|------|------|------|------|
| GET | /api/v1/experiment-templates/:id/checkpoints | 检查点列表 | 模板创建教师 |
| POST | /api/v1/experiment-templates/:id/checkpoints | 添加检查点 | 模板创建教师 |
| PUT | /api/v1/template-checkpoints/:id | 编辑检查点 | 模板创建教师 |
| DELETE | /api/v1/template-checkpoints/:id | 删除检查点 | 模板创建教师 |
| PUT | /api/v1/experiment-templates/:id/checkpoints/sort | 检查点排序 | 模板创建教师 |

### 1.6 初始化脚本管理

| 方法 | 路径 | 说明 | 权限 |
|------|------|------|------|
| GET | /api/v1/experiment-templates/:id/init-scripts | 初始化脚本列表 | 模板创建教师 |
| POST | /api/v1/experiment-templates/:id/init-scripts | 添加初始化脚本 | 模板创建教师 |
| PUT | /api/v1/template-init-scripts/:id | 编辑初始化脚本 | 模板创建教师 |
| DELETE | /api/v1/template-init-scripts/:id | 删除初始化脚本 | 模板创建教师 |

### 1.7 仿真场景库

| 方法 | 路径 | 说明 | 权限 |
|------|------|------|------|
| GET | /api/v1/sim-scenarios | 仿真场景列表 | 超级管理员/教师 |
| POST | /api/v1/sim-scenarios | 上传自定义仿真场景 | 超级管理员/教师 |
| GET | /api/v1/sim-scenarios/:id | 场景详情 | 超级管理员/教师 |
| PUT | /api/v1/sim-scenarios/:id | 编辑场景信息 | 超级管理员/场景上传者 |
| DELETE | /api/v1/sim-scenarios/:id | 删除/下架场景 | 超级管理员 |
| POST | /api/v1/sim-scenarios/:id/review | 审核场景 | 超级管理员 |
| GET | /api/v1/sim-link-groups | 联动组列表 | 教师 |
| GET | /api/v1/sim-link-groups/:id | 联动组详情（含关联场景） | 教师 |

### 1.8 模板仿真场景配置

| 方法 | 路径 | 说明 | 权限 |
|------|------|------|------|
| GET | /api/v1/experiment-templates/:id/sim-scenes | 模板仿真场景配置列表 | 模板创建教师 |
| POST | /api/v1/experiment-templates/:id/sim-scenes | 添加仿真场景到模板 | 模板创建教师 |
| PUT | /api/v1/template-sim-scenes/:id | 编辑仿真场景配置 | 模板创建教师 |
| DELETE | /api/v1/template-sim-scenes/:id | 移除仿真场景 | 模板创建教师 |
| PUT | /api/v1/experiment-templates/:id/sim-scenes/layout | 更新仿真场景布局 | 模板创建教师 |

### 1.9 标签管理

| 方法 | 路径 | 说明 | 权限 |
|------|------|------|------|
| GET | /api/v1/tags | 标签列表 | 教师 |
| POST | /api/v1/tags | 创建自定义标签 | 教师 |
| PUT | /api/v1/experiment-templates/:id/tags | 设置模板标签 | 模板创建教师 |
| GET | /api/v1/experiment-templates/:id/tags | 获取模板标签 | 教师 |

### 1.10 多人实验角色管理

| 方法 | 路径 | 说明 | 权限 |
|------|------|------|------|
| GET | /api/v1/experiment-templates/:id/roles | 角色列表 | 模板创建教师 |
| POST | /api/v1/experiment-templates/:id/roles | 添加角色 | 模板创建教师 |
| PUT | /api/v1/template-roles/:id | 编辑角色 | 模板创建教师 |
| DELETE | /api/v1/template-roles/:id | 删除角色 | 模板创建教师 |

### 1.11 实验实例管理（学生）

| 方法 | 路径 | 说明 | 权限 |
|------|------|------|------|
| POST | /api/v1/experiment-instances | 启动实验环境 | 学生 |
| GET | /api/v1/experiment-instances | 我的实验实例列表 | 学生 |
| GET | /api/v1/experiment-instances/:id | 实验实例详情 | 学生/教师 |
| POST | /api/v1/experiment-instances/:id/pause | 暂停实验 | 实例所属学生 |
| POST | /api/v1/experiment-instances/:id/resume | 恢复实验 | 实例所属学生 |
| POST | /api/v1/experiment-instances/:id/restart | 重新开始实验 | 实例所属学生 |
| POST | /api/v1/experiment-instances/:id/submit | 提交实验 | 实例所属学生 |
| POST | /api/v1/experiment-instances/:id/destroy | 销毁实验环境 | 实例所属学生/课程教师/管理员 |
| GET | /api/v1/experiment-instances/:id/terminal | 学生 Web 终端（WebSocket升级） | 实例所属学生 |
| POST | /api/v1/experiment-instances/:id/heartbeat | 心跳上报 | 实例所属学生 |

### 1.12 检查点验证

| 方法 | 路径 | 说明 | 权限 |
|------|------|------|------|
| POST | /api/v1/experiment-instances/:id/checkpoints/verify | 触发检查点验证 | 实例所属学生 |
| GET | /api/v1/experiment-instances/:id/checkpoints | 检查点结果列表 | 学生/教师 |
| POST | /api/v1/checkpoint-results/:id/grade | 手动评分检查点 | 课程教师 |

### 1.13 快照管理

| 方法 | 路径 | 说明 | 权限 |
|------|------|------|------|
| GET | /api/v1/experiment-instances/:id/snapshots | 快照列表 | 学生/教师 |
| POST | /api/v1/experiment-instances/:id/snapshots | 手动创建快照 | 实例所属学生 |
| POST | /api/v1/experiment-instances/:id/snapshots/:snapshot_id/restore | 从快照恢复 | 实例所属学生 |

### 1.14 操作日志

| 方法 | 路径 | 说明 | 权限 |
|------|------|------|------|
| GET | /api/v1/experiment-instances/:id/operation-logs | 操作日志列表 | 学生（本人）/教师 |

### 1.15 实验报告

| 方法 | 路径 | 说明 | 权限 |
|------|------|------|------|
| POST | /api/v1/experiment-instances/:id/report | 提交实验报告 | 实例所属学生 |
| GET | /api/v1/experiment-instances/:id/report | 获取实验报告 | 学生/教师 |
| PUT | /api/v1/experiment-instances/:id/report | 更新实验报告 | 实例所属学生 |

### 1.16 实验分组（多人实验）

| 方法 | 路径 | 说明 | 权限 |
|------|------|------|------|
| POST | /api/v1/experiment-groups | 创建实验分组 | 课程教师 |
| GET | /api/v1/experiment-groups | 分组列表 | 课程教师/学生 |
| GET | /api/v1/experiment-groups/:id | 分组详情 | 课程教师/组内学生 |
| PUT | /api/v1/experiment-groups/:id | 编辑分组 | 课程教师 |
| DELETE | /api/v1/experiment-groups/:id | 删除分组 | 课程教师 |
| POST | /api/v1/experiment-groups/auto-assign | 系统随机分组 | 课程教师 |
| POST | /api/v1/experiment-groups/:id/join | 学生加入分组 | 学生 |
| DELETE | /api/v1/experiment-groups/:id/members/:student_id | 移除组员 | 课程教师 |
| GET | /api/v1/experiment-groups/:id/members | 组员列表 | 课程教师/组内学生 |
| GET | /api/v1/experiment-groups/:id/progress | 组内进度同步 | 组内学生/课程教师 |

### 1.17 组内通信

| 方法 | 路径 | 说明 | 权限 |
|------|------|------|------|
| POST | /api/v1/experiment-groups/:id/messages | 发送组内消息 | 组内学生 |
| GET | /api/v1/experiment-groups/:id/messages | 组内消息历史 | 组内学生/课程教师 |
| GET | /api/v1/experiment-groups/:id/members/:student_id/terminal-stream | 只读查看组员终端（WebSocket升级） | 组内学生 |

> 实时消息通过 WebSocket 推送，REST 接口用于发送和查询历史。

### 1.18 教师监控

| 方法 | 路径 | 说明 | 权限 |
|------|------|------|------|
| GET | /api/v1/courses/:id/experiment-monitor | 课程实验监控面板 | 课程教师 |
| GET | /api/v1/experiment-instances/:id/terminal-stream | 远程查看学生终端（WebSocket升级） | 课程教师 |
| POST | /api/v1/experiment-instances/:id/message | 向学生发送指导消息 | 课程教师 |
| POST | /api/v1/experiment-instances/:id/force-destroy | 强制回收实验环境 | 课程教师/管理员 |
| POST | /api/v1/experiment-instances/:id/manual-grade | 教师手动评分（整体） | 课程教师 |
| GET | /api/v1/courses/:id/experiment-statistics | 实验统计数据 | 课程教师 |

### 1.19 资源配额管理

| 方法 | 路径 | 说明 | 权限 |
|------|------|------|------|
| GET | /api/v1/resource-quotas | 资源配额列表 | 超级管理员/学校管理员 |
| POST | /api/v1/resource-quotas | 创建资源配额 | 超级管理员 |
| PUT | /api/v1/resource-quotas/:id | 编辑资源配额 | 超级管理员/学校管理员 |
| GET | /api/v1/resource-quotas/:id | 资源配额详情 | 超级管理员/学校管理员 |
| GET | /api/v1/schools/:id/resource-usage | 学校资源使用情况 | 超级管理员/学校管理员 |

### 1.20 全局监控（管理员）

| 方法 | 路径 | 说明 | 权限 |
|------|------|------|------|
| GET | /api/v1/admin/experiment-overview | 全平台实验概览 | 超级管理员 |
| GET | /api/v1/admin/container-resources | 全平台容器资源监控 | 超级管理员 |
| GET | /api/v1/admin/k8s-cluster-status | K8s集群状态 | 超级管理员 |
| GET | /api/v1/admin/experiment-instances | 全平台实验实例列表 | 超级管理员 |
| POST | /api/v1/admin/experiment-instances/:id/force-destroy | 强制回收任意实验环境 | 超级管理员 |

### 1.21 共享实验库

| 方法 | 路径 | 说明 | 权限 |
|------|------|------|------|
| GET | /api/v1/shared-experiment-templates | 共享实验模板列表 | 教师 |
| GET | /api/v1/shared-experiment-templates/:id | 共享实验模板详情 | 教师 |

### 1.22 学校管理员视角

| 方法 | 路径 | 说明 | 权限 |
|------|------|------|------|
| GET | /api/v1/school/images | 本校镜像列表 | 学校管理员 |
| GET | /api/v1/school/experiment-monitor | 本校实验监控 | 学校管理员 |
| PUT | /api/v1/school/course-quotas/:course_id | 课程资源配额分配 | 学校管理员 |

### 1.23 镜像配置模板

| 方法 | 路径 | 说明 | 权限 |
|------|------|------|------|
| GET | /api/v1/images/:id/config-template | 获取镜像配置模板 | 教师 |
| GET | /api/v1/images/:id/documentation | 获取镜像结构化文档 | 教师 |

### 1.24 模板配置验证

| 方法 | 路径 | 说明 | 权限 |
|------|------|------|------|
| POST | /api/v1/experiment-templates/:id/validate | 模板配置验证（5层） | 模板创建教师 |

### 1.25 镜像预拉取管理

| 方法 | 路径 | 说明 | 权限 |
|------|------|------|------|
| GET | /api/v1/admin/image-pull-status | 镜像预拉取状态 | 超级管理员 |
| POST | /api/v1/admin/image-pull | 触发镜像预拉取 | 超级管理员 |

---

## 二、核心接口详细定义

### 2.1 POST /api/v1/images — 创建/上传镜像

**权限：** 超级管理员（官方镜像）/ 教师（自定义镜像）

**请求体：**

```json
{
  "category_id": "1780000000100001",
  "name": "geth",
  "display_name": "Go-Ethereum",
  "description": "以太坊官方Go语言客户端",
  "icon_url": "https://oss.example.com/icons/geth.png",
  "ecosystem": "ethereum",
  "default_ports": [
    { "port": 8545, "protocol": "tcp", "name": "HTTP-RPC" },
    { "port": 30303, "protocol": "tcp", "name": "P2P" }
  ],
  "default_env_vars": [
    { "key": "NETWORK_ID", "value": "1337", "desc": "网络ID" }
  ],
  "default_volumes": [
    { "path": "/root/.ethereum", "desc": "数据目录" }
  ],
  "typical_companions": {
    "required": [],
    "recommended": [
      { "image": "blockscout", "reason": "本地区块链浏览器，方便查看交易和区块" }
    ],
    "optional": [
      { "image": "remix-ide", "reason": "Solidity在线IDE，可直接连接本地节点" },
      { "image": "prometheus", "reason": "采集geth指标数据" },
      { "image": "grafana", "reason": "可视化监控仪表盘" }
    ]
  },
  "required_dependencies": [],
  "resource_recommendation": {
    "cpu": "0.5",
    "memory": "1Gi",
    "disk": "2Gi"
  },
  "documentation_url": "/docs/images/geth",
  "versions": [
    {
      "version": "1.13",
      "registry_url": "registry.lianjing.com/geth:1.13",
      "min_cpu": "0.5",
      "min_memory": "512Mi",
      "min_disk": "1Gi",
      "is_default": true
    }
  ]
}
```

**响应：**

```json
{
  "code": 200,
  "message": "创建成功",
  "data": {
    "id": "1780000000200001",
    "name": "geth",
    "display_name": "Go-Ethereum",
    "status": 1,
    "status_text": "正常",
    "versions": [
      {
        "id": "1780000000200101",
        "version": "1.13",
        "is_default": true
      }
    ]
  }
}
```

> 超级管理员创建的镜像 source_type=1（官方），status 直接为正常。
> 教师创建的镜像 source_type=2（自定义），status 为待审核。

---

### 2.2 POST /api/v1/images/:id/review — 审核镜像

**权限：** 超级管理员

**请求体：**

```json
{
  "action": "approve",
  "comment": "安全扫描通过，资源限制合理"
}
```

| action 值 | 说明 |
|-----------|------|
| approve | 审核通过，状态变为正常 |
| reject | 审核拒绝，状态变为审核拒绝 |

**错误响应：**

| code | message | 场景 |
|------|---------|------|
| 40001 | action参数无效 | action 不是 approve/reject |
| 40401 | 镜像不存在 | 镜像ID无效 |
| 40002 | 该镜像不在待审核状态 | 镜像状态不是待审核 |

---

### 2.3 POST /api/v1/experiment-templates — 创建实验模板

**权限：** 教师

**请求体：**

```json
{
  "title": "以太坊智能合约开发入门",
  "description": "# 实验简介\n\n本实验将引导你完成第一个Solidity智能合约的编写、编译和部署...",
  "objectives": "1. 理解Solidity基本语法\n2. 掌握合约编译和部署流程\n3. 学会使用Remix IDE",
  "instructions": "## 步骤一：编写合约\n\n打开Remix IDE...",
  "references": "- [Solidity官方文档](https://docs.soliditylang.org)\n- [Remix IDE](https://remix.ethereum.org)",
  "experiment_type": 2,
  "topology_mode": 1,
  "judge_mode": 3,
  "auto_weight": 60.00,
  "manual_weight": 40.00,
  "total_score": 100,
  "max_duration": 120,
  "idle_timeout": 30,
  "cpu_limit": "2",
  "memory_limit": "4Gi",
  "disk_limit": "10Gi",
  "score_strategy": 1
}
```

**响应：**

```json
{
  "code": 200,
  "message": "创建成功",
  "data": {
    "id": "1780000000300001",
    "title": "以太坊智能合约开发入门",
    "experiment_type": 2,
    "experiment_type_text": "真实环境实验",
    "status": 1,
    "status_text": "草稿",
    "topology_mode": 1,
    "topology_mode_text": "单人单节点",
    "judge_mode": 3,
    "judge_mode_text": "混合"
  }
}
```

> 创建后为草稿状态，教师需继续配置容器、检查点、仿真场景等，最后发布。

---

### 2.4 GET /api/v1/experiment-templates/:id — 实验模板详情

**权限：** 教师（模板创建者或共享模板）

**响应：**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "id": "1780000000300001",
    "title": "以太坊智能合约开发入门",
    "description": "# 实验简介\n\n...",
    "objectives": "1. 理解Solidity基本语法...",
    "instructions": "## 步骤一：编写合约\n\n...",
    "references": "- [Solidity官方文档]...",
    "experiment_type": 2,
    "experiment_type_text": "真实环境实验",
    "topology_mode": 1,
    "topology_mode_text": "单人单节点",
    "judge_mode": 3,
    "judge_mode_text": "混合",
    "auto_weight": 60.00,
    "manual_weight": 40.00,
    "total_score": 100,
    "max_duration": 120,
    "idle_timeout": 30,
    "cpu_limit": "2",
    "memory_limit": "4Gi",
    "disk_limit": "10Gi",
    "score_strategy": 1,
    "is_shared": false,
    "status": 2,
    "status_text": "已发布",
    "teacher": {
      "id": "1780000000000001",
      "name": "李教授"
    },
    "containers": [
      {
        "id": "1780000000300101",
        "container_name": "geth-node",
        "image_version": {
          "id": "1780000000200101",
          "image_name": "geth",
          "image_display_name": "Go-Ethereum",
          "version": "1.13",
          "icon_url": "https://oss.example.com/icons/geth.png"
        },
        "env_vars": [{ "key": "NETWORK_ID", "value": "1337" }],
        "ports": [{ "container": 8545, "protocol": "tcp" }],
        "is_primary": true,
        "startup_order": 0
      },
      {
        "id": "1780000000300102",
        "container_name": "remix-ide",
        "image_version": {
          "id": "1780000000200201",
          "image_name": "remix-ide",
          "image_display_name": "Remix IDE",
          "version": "latest",
          "icon_url": "https://oss.example.com/icons/remix.png"
        },
        "is_primary": false,
        "startup_order": 1
      }
    ],
    "checkpoints": [
      {
        "id": "1780000000300201",
        "title": "合约编译成功",
        "check_type": 1,
        "check_type_text": "自动脚本",
        "score": 30,
        "sort_order": 1
      },
      {
        "id": "1780000000300202",
        "title": "合约部署到本地链",
        "check_type": 1,
        "check_type_text": "自动脚本",
        "score": 30,
        "sort_order": 2
      },
      {
        "id": "1780000000300203",
        "title": "代码质量与规范",
        "check_type": 2,
        "check_type_text": "手动评分",
        "score": 40,
        "sort_order": 3
      }
    ],
    "init_scripts": [
      {
        "id": "1780000000300301",
        "target_container": "geth-node",
        "script_language": "bash",
        "execution_order": 0,
        "timeout": 60
      }
    ],
    "sim_scenes": [
      {
        "id": "1780000000300401",
        "scenario": {
          "id": "1780000000400001",
          "name": "区块链结构与分叉",
          "code": "blockchain-structure-fork",
          "category": "data_structure",
          "category_text": "数据结构",
          "time_control_mode": "reactive",
          "container_image_url": "registry.lianjing.com/sim-scenes/blockchain-structure-fork:1.0",
          "container_image_size": "28MB"
        },
        "link_group_id": "1780000000700005",
        "link_group_name": "区块链完整性组",
        "scene_params": { "initial_blocks": 5, "fork_probability": 0.1 },
        "initial_state": {},
        "data_source_mode": 1,
        "data_source_mode_text": "仿真模式",
        "data_source_config": null,
        "layout_position": { "x": 0, "y": 0, "w": 6, "h": 4 }
      }
    ],
    "tags": [
      { "id": "1780000000500001", "name": "Ethereum/EVM", "category": "ecosystem" },
      { "id": "1780000000500002", "name": "智能合约开发", "category": "type" },
      { "id": "1780000000500003", "name": "入门", "category": "difficulty" }
    ],
    "roles": [],
    "created_at": "2026-04-08T10:00:00Z",
    "updated_at": "2026-04-08T12:00:00Z"
  }
}
```

---

### 2.5 POST /api/v1/experiment-templates/:id/containers — 添加容器配置

**权限：** 模板创建教师

**请求体：**

```json
{
  "image_version_id": "1780000000200101",
  "container_name": "geth-node",
  "role_id": null,
  "env_vars": [
    { "key": "NETWORK_ID", "value": "1337" },
    { "key": "RPC_ADDR", "value": "0.0.0.0" }
  ],
  "ports": [
    { "container": 8545, "protocol": "tcp" },
    { "container": 30303, "protocol": "tcp" }
  ],
  "volumes": [
    { "host_path": "", "container_path": "/root/.ethereum" }
  ],
  "cpu_limit": "1",
  "memory_limit": "2Gi",
  "depends_on": [],
  "startup_order": 0,
  "is_primary": true
}
```

**响应：**

```json
{
  "code": 200,
  "message": "添加成功",
  "data": {
    "id": "1780000000300101",
    "container_name": "geth-node",
    "image_version_id": "1780000000200101",
    "is_primary": true,
    "startup_order": 0
  }
}
```

**错误响应：**

| code | message | 场景 |
|------|---------|------|
| 40401 | 实验模板不存在 | 模板ID无效 |
| 40402 | 镜像版本不存在 | image_version_id 无效 |
| 40901 | 容器名称已存在 | 同一模板下容器名重复 |
| 40001 | 模板已发布，不可修改 | 模板状态非草稿 |

---

### 2.6 POST /api/v1/experiment-templates/:id/checkpoints — 添加检查点

**权限：** 模板创建教师

**请求体（自动检查点）：**

```json
{
  "title": "合约编译成功",
  "description": "检查学生是否成功编译了HelloWorld合约",
  "check_type": 1,
  "script_content": "#!/bin/bash\nif [ -f /workspace/artifacts/HelloWorld.json ]; then\n  echo 'PASS'\n  exit 0\nelse\n  echo 'FAIL: 未找到编译产物'\n  exit 1\nfi",
  "script_language": "bash",
  "target_container": "remix-ide",
  "score": 30,
  "scope": 1,
  "sort_order": 1
}
```

**请求体（手动评分项）：**

```json
{
  "title": "代码质量与规范",
  "description": "评估学生代码的可读性、注释、命名规范等",
  "check_type": 2,
  "score": 40,
  "scope": 1,
  "sort_order": 3
}
```

---

### 2.7 POST /api/v1/experiment-instances — 启动实验环境

**权限：** 学生

**请求体：**

```json
{
  "template_id": "1780000000300001",
  "course_id": "1780000000010001",
  "lesson_id": "1780000000010101",
  "assignment_id": null,
  "snapshot_id": null,
  "group_id": null
}
```

> `snapshot_id` 不为空时从快照恢复；`group_id` 不为空时为多人实验。

**响应（创建中）：**

```json
{
  "code": 200,
  "message": "实验环境创建中",
  "data": {
    "instance_id": "1780000000600001",
    "status": 1,
    "status_text": "创建中",
    "attempt_no": 1,
    "estimated_ready_seconds": 60,
    "queue_position": null
  }
}
```

**响应（排队中）：**

```json
{
  "code": 200,
  "message": "资源不足，已加入排队",
  "data": {
    "instance_id": null,
    "status": "queued",
    "queue_position": 3,
    "estimated_wait_seconds": 180
  }
}
```

**错误响应：**

| code | message | 场景 |
|------|---------|------|
| 40401 | 实验模板不存在 | template_id 无效 |
| 40001 | 实验模板未发布 | 模板状态非已发布 |
| 40901 | 已达个人并发实验上限 | 超过 max_per_student |
| 40902 | 课程并发实验数已满 | 超过课程级 max_concurrency |
| 40903 | 学校资源配额不足 | 超过学校级资源配额 |
| 40301 | 您未加入该课程 | 学生不在课程中 |

---

### 2.8 GET /api/v1/experiment-instances/:id — 实验实例详情

**权限：** 实例所属学生 / 课程教师

**响应：**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "id": "1780000000600001",
    "template": {
      "id": "1780000000300001",
      "title": "以太坊智能合约开发入门",
      "topology_mode": 1,
      "judge_mode": 3,
      "instructions": "## 步骤一：编写合约\n\n...",
      "max_duration": 120,
      "idle_timeout": 30,
      "total_score": 100
    },
    "student": {
      "id": "1780000000000001",
      "name": "张三",
      "student_no": "2024001"
    },
    "status": 3,
    "status_text": "运行中",
    "attempt_no": 1,
    "access_url": "https://lab.lianjing.com/instance/1780000000600001",
    "containers": [
      {
        "id": "1780000000600101",
        "container_name": "geth-node",
        "image_name": "geth",
        "image_version": "1.13",
        "status": 2,
        "status_text": "运行中",
        "internal_ip": "10.244.1.15",
        "cpu_usage": "0.3",
        "memory_usage": "256Mi"
      },
      {
        "id": "1780000000600102",
        "container_name": "remix-ide",
        "image_name": "remix-ide",
        "image_version": "latest",
        "status": 2,
        "status_text": "运行中",
        "internal_ip": "10.244.1.16"
      }
    ],
    "checkpoints": [
      {
        "checkpoint_id": "1780000000300201",
        "title": "合约编译成功",
        "check_type": 1,
        "score": 30,
        "result": {
          "is_passed": true,
          "score": 30,
          "checked_at": "2026-04-08T10:30:00Z"
        }
      },
      {
        "checkpoint_id": "1780000000300202",
        "title": "合约部署到本地链",
        "check_type": 1,
        "score": 30,
        "result": null
      },
      {
        "checkpoint_id": "1780000000300203",
        "title": "代码质量与规范",
        "check_type": 2,
        "score": 40,
        "result": null
      }
    ],
    "scores": {
      "auto_score": 30,
      "manual_score": null,
      "total_score": null
    },
    "started_at": "2026-04-08T10:00:00Z",
    "last_active_at": "2026-04-08T10:35:00Z",
    "created_at": "2026-04-08T09:59:00Z"
  }
}
```

---

### 2.9 POST /api/v1/experiment-instances/:id/pause — 暂停实验

**权限：** 实例所属学生

**响应：**

```json
{
  "code": 200,
  "message": "实验已暂停",
  "data": {
    "instance_id": "1780000000600001",
    "status": 4,
    "status_text": "暂停",
    "snapshot_id": "1780000000700001",
    "paused_at": "2026-04-08T11:00:00Z"
  }
}
```

> 暂停时自动创建快照并挂起容器。

**错误响应：**

| code | message | 场景 |
|------|---------|------|
| 40001 | 当前状态不可暂停 | 实例状态不是运行中 |
| 40301 | 无权操作该实验 | 非实例所属学生 |

---

### 2.10 POST /api/v1/experiment-instances/:id/resume — 恢复实验

**权限：** 实例所属学生

**请求体：**

```json
{
  "snapshot_id": "1780000000700001"
}
```

> `snapshot_id` 可选，不传则从最新快照恢复。

**响应：**

```json
{
  "code": 200,
  "message": "实验恢复中",
  "data": {
    "instance_id": "1780000000600001",
    "status": 2,
    "status_text": "初始化中",
    "estimated_ready_seconds": 30
  }
}
```

**错误响应：**

| code | message | 场景 |
|------|---------|------|
| 40001 | 当前状态不可恢复 | 实例状态不是暂停 |
| 40901 | 已达个人并发实验上限 | 恢复时超过并发限制 |

---

### 2.11 POST /api/v1/experiment-instances/:id/submit — 提交实验

**权限：** 实例所属学生

**响应：**

```json
{
  "code": 200,
  "message": "实验提交成功",
  "data": {
    "instance_id": "1780000000600001",
    "status": 5,
    "status_text": "已完成",
    "scores": {
      "auto_score": 60,
      "auto_total": 60,
      "manual_score": null,
      "manual_total": 40,
      "total_score": null,
      "details": [
        {
          "checkpoint_id": "1780000000300201",
          "title": "合约编译成功",
          "check_type": 1,
          "is_passed": true,
          "score": 30,
          "max_score": 30
        },
        {
          "checkpoint_id": "1780000000300202",
          "title": "合约部署到本地链",
          "check_type": 1,
          "is_passed": true,
          "score": 30,
          "max_score": 30
        },
        {
          "checkpoint_id": "1780000000300203",
          "title": "代码质量与规范",
          "check_type": 2,
          "status": "pending_review",
          "max_score": 40
        }
      ]
    },
    "completed_at": "2026-04-08T11:30:00Z"
  }
}
```

> 提交时自动触发所有自动检查点的最终验证，手动评分项标记为待评分。
> 提交后容器保留一段时间（供教师查看），之后自动回收。

**错误响应：**

| code | message | 场景 |
|------|---------|------|
| 40001 | 当前状态不可提交 | 实例状态不是运行中 |

---

### 2.12 POST /api/v1/experiment-instances/:id/checkpoints/verify — 触发检查点验证

**权限：** 实例所属学生

**请求体：**

```json
{
  "checkpoint_id": "1780000000300201"
}
```

> `checkpoint_id` 可选，不传则验证所有自动检查点。
> 接口限流：每用户每分钟最多触发 10 次检查点验证。

**响应：**

```json
{
  "code": 200,
  "message": "验证完成",
  "data": {
    "results": [
      {
        "checkpoint_id": "1780000000300201",
        "title": "合约编译成功",
        "is_passed": true,
        "score": 30,
        "check_output": "PASS",
        "checked_at": "2026-04-08T10:30:00Z"
      }
    ]
  }
}
```

**错误响应：**

| code | message | 场景 |
|------|---------|------|
| 40001 | 实验未在运行中 | 实例状态不是运行中 |
| 40401 | 检查点不存在 | checkpoint_id 无效 |
| 42911 | 检查点验证过于频繁，请稍后再试 | 超过每分钟 10 次限制 |

---

### 2.13 POST /api/v1/checkpoint-results/:id/grade — 手动评分检查点

**权限：** 课程教师

**请求体：**

```json
{
  "score": 35,
  "comment": "代码结构清晰，但缺少必要注释"
}
```

**错误响应：**

| code | message | 场景 |
|------|---------|------|
| 40001 | 评分不能超过检查点满分 | score > 检查点分值 |
| 40002 | 该检查点不是手动评分类型 | check_type != 2 |

---

### 2.14 POST /api/v1/experiment-instances/:id/manual-grade — 教师手动评分（整体）

**权限：** 课程教师

**请求体：**

```json
{
  "checkpoint_grades": [
    {
      "checkpoint_id": "1780000000300203",
      "score": 35,
      "comment": "代码结构清晰，但缺少必要注释"
    }
  ],
  "overall_comment": "整体完成度较好，建议加强代码注释习惯"
}
```

**响应：**

```json
{
  "code": 200,
  "message": "评分成功",
  "data": {
    "instance_id": "1780000000600001",
    "auto_score": 60,
    "manual_score": 35,
    "total_score": 50,
    "score_detail": "自动部分: 60/60 × 60% = 36, 手动部分: 35/40 × 40% = 14, 总分: 50"
  }
}
```

> 混合模式下总分 = 自动得分率 × auto_weight + 手动得分率 × manual_weight，映射到 total_score。

---

### 2.15 POST /api/v1/experiment-instances/:id/report — 提交实验报告

**权限：** 实例所属学生

**请求体（Markdown / 文件元数据）：**

```json
{
  "content": "# 实验报告\n\n## 实验目的\n\n...\n\n## 实验过程\n\n...\n\n## 实验总结\n\n...",
  "file_url": "https://oss.example.com/experiment-reports/report-1780000000600001.pdf",
  "file_name": "实验报告.pdf",
  "file_size": 1048576
}
```

> `content` 和报告文件元数据至少提供一种；当提交文件时，前端应先完成文件上传，再将 `file_url / file_name / file_size` 回传后端。  
> 报告文件仅允许 `pdf / doc / docx`，且文件大小不得超过 50MB。

**响应：**

```json
{
  "code": 200,
  "message": "报告提交成功",
  "data": {
    "id": "1780000000800001",
    "instance_id": "1780000000600001",
    "content": "# 实验报告\n\n...",
    "file_url": "https://oss.example.com/experiment-reports/report-1780000000600001.pdf",
    "file_name": "实验报告.pdf",
    "file_size": 1048576,
    "created_at": "2026-04-08T11:45:00Z",
    "updated_at": "2026-04-08T11:45:00Z"
  }
}
```

---

### 2.16 POST /api/v1/experiment-groups — 创建实验分组

**权限：** 课程教师

**请求体：**

```json
{
  "template_id": "1780000000300002",
  "course_id": "1780000000010001",
  "group_method": 1,
  "groups": [
    {
      "group_name": "第1组",
      "max_members": 3,
      "members": [
        { "student_id": "1780000000000001", "role_id": "1780000000300501" },
        { "student_id": "1780000000000002", "role_id": "1780000000300502" },
        { "student_id": "1780000000000003", "role_id": "1780000000300503" }
      ]
    },
    {
      "group_name": "第2组",
      "max_members": 3,
      "members": [
        { "student_id": "1780000000000004", "role_id": "1780000000300501" },
        { "student_id": "1780000000000005", "role_id": "1780000000300502" }
      ]
    }
  ]
}
```

> `group_method=1` 教师手动分组时需提供 members。
> `group_method=2` 学生自选时只需创建空分组。
> `group_method=3` 系统随机分组使用 `/auto-assign` 接口。

**响应：**

```json
{
  "code": 200,
  "message": "分组创建成功",
  "data": {
    "groups": [
      {
        "id": "1780000000900001",
        "group_name": "第1组",
        "member_count": 3,
        "max_members": 3,
        "status": 1,
        "status_text": "组建中"
      },
      {
        "id": "1780000000900002",
        "group_name": "第2组",
        "member_count": 2,
        "max_members": 3,
        "status": 1,
        "status_text": "组建中"
      }
    ]
  }
}
```

---

### 2.17 POST /api/v1/experiment-groups/:id/join — 学生加入分组

**权限：** 学生

**请求体：**

```json
{
  "role_id": "1780000000300502"
}
```

**错误响应：**

| code | message | 场景 |
|------|---------|------|
| 40901 | 该分组已满 | 超过 max_members |
| 40902 | 您已在其他分组中 | 同一实验模板下已加入其他组 |
| 40903 | 该角色已被占用 | 角色 max_members 已满 |
| 40001 | 分组不在组建中状态 | 分组状态不是组建中 |

---

### 2.18 GET /api/v1/courses/:id/experiment-monitor — 课程实验监控面板

**权限：** 课程教师

**查询参数：**

| 参数 | 类型 | 说明 |
|------|------|------|
| template_id | string | 筛选特定实验模板 |
| status | int | 筛选实例状态 |

**响应：**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "summary": {
      "total_students": 45,
      "running": 38,
      "paused": 2,
      "completed": 3,
      "not_started": 2,
      "avg_progress": 65.5,
      "resource_usage": {
        "cpu_used": "38",
        "cpu_total": "100",
        "memory_used": "76Gi",
        "memory_total": "200Gi"
      }
    },
    "students": [
      {
        "student_id": "1780000000000001",
        "student_name": "张三",
        "student_no": "2024001",
        "instance_id": "1780000000600001",
        "status": 3,
        "status_text": "运行中",
        "checkpoints_passed": 2,
        "checkpoints_total": 3,
        "progress_percent": 66.7,
        "cpu_usage": "0.5",
        "memory_usage": "512Mi",
        "started_at": "2026-04-08T10:00:00Z",
        "last_active_at": "2026-04-08T10:35:00Z"
      }
    ]
  }
}
```

---

### 2.19 GET /api/v1/experiment-instances/:id/terminal-stream — 远程查看学生终端

**权限：** 课程教师

**协议：** WebSocket 升级

**连接地址：**

```
wss://api.lianjing.com/api/v1/experiment-instances/:id/terminal-stream?token=<jwt>
```

**WebSocket 消息格式：**

```json
// 服务端推送终端输出
{
  "type": "terminal_output",
  "container": "geth-node",
  "data": "root@geth-node:~# geth --dev\n",
  "timestamp": "2026-04-08T10:30:00Z"
}

// 教师发送指导消息
{
  "type": "guidance_message",
  "content": "提示：请先检查网络ID是否正确配置"
}
```

> 教师端为只读模式，不可操作学生终端，仅可发送指导消息。

---

### 2.19A GET /api/v1/experiment-instances/:id/terminal — 学生 Web 终端

**权限：** 实例所属学生

**协议：** WebSocket 升级

**连接地址：**

```
wss://api.lianjing.com/api/v1/experiment-instances/:id/terminal?token=<jwt>&container=geth-node
```

**查询参数：**

| 参数 | 类型 | 说明 |
|------|------|------|
| container | string | 可选，目标容器名；不传时默认主容器 |

**WebSocket 消息格式：**

```json
// 学生发送终端命令
{
  "type": "terminal_command",
  "container": "geth-node",
  "command": "geth attach http://127.0.0.1:8545"
}

// 服务端返回命令执行结果
{
  "type": "terminal_output",
  "container": "geth-node",
  "command": "geth attach http://127.0.0.1:8545",
  "exit_code": 0,
  "stdout": "Welcome to the Geth JavaScript console!\n",
  "stderr": "",
  "timestamp": "2026-04-08T10:30:00Z"
}
```

> 学生 Web 终端以命令执行流方式工作，服务端负责在目标容器内执行命令并回传输出。  
> 每次成功接收终端命令时更新 `last_active_at`。  
> 每条终端命令必须记录到实例操作日志，`action=terminal_command`，用于教师回查与评分审计。

---

### 2.20 POST /api/v1/resource-quotas — 创建资源配额

**权限：** 超级管理员

**请求体（学校级）：**

```json
{
  "quota_level": 1,
  "school_id": "1780000000000001",
  "max_cpu": "100",
  "max_memory": "200Gi",
  "max_storage": "500Gi",
  "max_concurrency": 200,
  "max_per_student": 2
}
```

**请求体（课程级）：**

```json
{
  "quota_level": 2,
  "school_id": "1780000000000001",
  "course_id": "1780000000010001",
  "max_concurrency": 50,
  "max_per_student": 2
}
```

**错误响应：**

| code | message | 场景 |
|------|---------|------|
| 40901 | 该学校已存在资源配额 | 学校级配额重复 |
| 40902 | 该课程已存在资源配额 | 课程级配额重复 |
| 40001 | 课程级配额不能超过学校级 | 课程配额超出学校总配额 |

---

### 2.21 GET /api/v1/schools/:id/resource-usage — 学校资源使用情况

**权限：** 超级管理员 / 学校管理员

**响应：**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "school_id": "1780000000000001",
    "school_name": "XX大学",
    "quota": {
      "max_cpu": "100",
      "max_memory": "200Gi",
      "max_storage": "500Gi",
      "max_concurrency": 200
    },
    "usage": {
      "used_cpu": "45.5",
      "used_memory": "98Gi",
      "used_storage": "120Gi",
      "current_concurrency": 85,
      "cpu_usage_percent": 45.5,
      "memory_usage_percent": 49.0,
      "storage_usage_percent": 24.0,
      "concurrency_usage_percent": 42.5
    },
    "course_breakdown": [
      {
        "course_id": "1780000000010001",
        "course_title": "区块链技术与应用",
        "current_concurrency": 38,
        "max_concurrency": 50,
        "cpu_used": "19",
        "memory_used": "38Gi"
      }
    ]
  }
}
```

---

### 2.22 GET /api/v1/admin/experiment-overview — 全平台实验概览

**权限：** 超级管理员

**响应：**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "total_instances": 1250,
    "running_instances": 320,
    "total_templates": 85,
    "total_images": 42,
    "pending_reviews": 3,
    "cluster_status": {
      "nodes": 10,
      "healthy_nodes": 10,
      "total_cpu": "400",
      "used_cpu": "180",
      "total_memory": "800Gi",
      "used_memory": "350Gi"
    },
    "school_usage": [
      {
        "school_id": "1780000000000001",
        "school_name": "XX大学",
        "running_instances": 85,
        "cpu_used": "45.5",
        "memory_used": "98Gi",
        "quota_cpu": "100",
        "quota_memory": "200Gi"
      }
    ]
  }
}
```

---

### 2.23 GET /api/v1/courses/:id/experiment-statistics — 实验统计数据

**权限：** 课程教师

**查询参数：**

| 参数 | 类型 | 说明 |
|------|------|------|
| template_id | string | 筛选特定实验模板（可选） |

**响应：**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "templates": [
      {
        "template_id": "1780000000300001",
        "template_title": "以太坊智能合约开发入门",
        "statistics": {
          "total_students": 45,
          "started_count": 43,
          "completed_count": 38,
          "completion_rate": 84.4,
          "avg_score": 78.5,
          "max_score": 98,
          "min_score": 42,
          "avg_duration_minutes": 85,
          "avg_attempts": 1.3,
          "checkpoint_pass_rates": [
            {
              "checkpoint_id": "1780000000300201",
              "title": "合约编译成功",
              "pass_rate": 95.6
            },
            {
              "checkpoint_id": "1780000000300202",
              "title": "合约部署到本地链",
              "pass_rate": 88.9
            },
            {
              "checkpoint_id": "1780000000300203",
              "title": "代码质量与规范",
              "avg_score": 32.5,
              "max_score": 40
            }
          ],
          "score_distribution": {
            "90_100": 8,
            "80_89": 12,
            "70_79": 10,
            "60_69": 5,
            "below_60": 3
          }
        }
      }
    ]
  }
}
```

---

### 2.24 POST /api/v1/experiment-instances/:id/heartbeat — 心跳上报

**权限：** 实例所属学生

**请求体：**

```json
{
  "active_container": "geth-node"
}
```

> 前端每60秒上报一次心跳，后端仅更新 Redis 心跳缓存与剩余时间信息，用于判断连接仍然在线。  
> `heartbeat` **不更新** `last_active_at`；`last_active_at` 仅在终端命令、检查点验证、快照恢复、暂停/恢复、提交、SimEngine 场景交互等真实用户操作发生时更新。  
> 接口限流：每用户每分钟最多上报 2 次心跳。

**响应：**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "status": 2,
    "remaining_minutes": 85,
    "idle_warning": false
  }
}
```

> `remaining_minutes` 为距离最长运行时间的剩余分钟数。
> `idle_warning` 为 true 时表示即将因空闲超时被回收（剩余5分钟内）。
>
> **错误响应：**
>
> | code | message | 场景 |
> |------|---------|------|
> | 42910 | 心跳上报过于频繁，请稍后再试 | 超过每分钟 2 次限制 |

---

### 2.25 POST /api/v1/experiment-instances/:id/message — 向学生发送指导消息

**权限：** 课程教师

**请求体：**

```json
{
  "content": "提示：请检查合约的构造函数参数是否正确"
}
```

> 消息通过 WebSocket 实时推送到学生实验界面。

---

### 2.26 GET /api/v1/experiment-instances/:id/operation-logs — 操作日志列表

**权限：** 实例所属学生（本人）/ 课程教师

**查询参数：**

| 参数 | 类型 | 说明 |
|------|------|------|
| action | string | 筛选操作类型 |
| target_container | string | 筛选目标容器 |
| page | int | 页码 |
| page_size | int | 每页条数 |

**响应：**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "list": [
      {
        "id": "1780000001000001",
        "action": "terminal_command",
        "target_container": "geth-node",
        "command": "geth --dev --http --http.addr 0.0.0.0",
        "detail": {
          "exit_code": 0,
          "stdout": "INFO [04-08|10:05:00] Starting Geth in dev mode...",
          "stderr": ""
        },
        "created_at": "2026-04-08T10:05:00Z"
      },
      {
        "id": "1780000001000002",
        "action": "checkpoint_check",
        "detail": {
          "checkpoint_id": "1780000000300201",
          "checkpoint_title": "合约编译成功",
          "is_passed": true
        },
        "created_at": "2026-04-08T10:30:00Z"
      }
    ],
    "pagination": {
      "page": 1,
      "page_size": 20,
      "total": 156,
      "total_pages": 8
    }
  }
}
```

---

### 2.26A GET /api/v1/experiment-groups/:id/members/:student_id/terminal-stream — 只读查看组员终端

**权限：** 组内学生

**协议：** WebSocket 升级

**连接地址：**

```
wss://api.lianjing.com/api/v1/experiment-groups/:id/members/:student_id/terminal-stream?token=<jwt>
```

**WebSocket 消息格式：**

```json
// 服务端推送组员终端输出
{
  "type": "terminal_output",
  "container": "geth-node",
  "data": "root@geth-node:~# geth --dev\n",
  "timestamp": "2026-04-08T10:30:00Z"
}
```

> 仅允许查看同组成员的终端输出，不可输入命令，也不可发送教师指导消息。  
> 权限校验必须同时满足：请求方为该实验分组成员，且 `student_id` 对应实例属于同一分组。

---

### 2.27 POST /api/v1/experiment-groups/auto-assign — 系统随机分组

**权限：** 课程教师

**请求体：**

```json
{
  "template_id": "1780000000300002",
  "course_id": "1780000000010001",
  "group_size": 3,
  "group_name_prefix": "第"
}
```

> 系统根据课程学生名单自动随机分组，每组 group_size 人，角色随机分配。

**响应：**

```json
{
  "code": 200,
  "message": "随机分组完成",
  "data": {
    "total_groups": 15,
    "total_students": 45,
    "groups": [
      {
        "id": "1780000000900001",
        "group_name": "第1组",
        "members": [
          { "student_id": "1780000000000001", "student_name": "张三", "role_name": "Org1管理员" },
          { "student_id": "1780000000000002", "student_name": "李四", "role_name": "Org2管理员" },
          { "student_id": "1780000000000003", "student_name": "王五", "role_name": "Orderer运维" }
        ]
      }
    ]
  }
}
```

---

### 2.28 GET /api/v1/experiment-groups/:id/progress — 组内进度同步

**权限：** 组内学生 / 课程教师

**响应：**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "group_id": "1780000000900001",
    "group_name": "第1组",
    "group_status": 3,
    "group_status_text": "实验中",
    "members": [
      {
        "student_id": "1780000000000001",
        "student_name": "张三",
        "role_name": "Org1管理员",
        "instance_id": "1780000000600001",
        "instance_status": 3,
        "instance_status_text": "运行中",
        "checkpoints_passed": 2,
        "checkpoints_total": 3,
        "personal_score": 60
      },
      {
        "student_id": "1780000000000002",
        "student_name": "李四",
        "role_name": "Org2管理员",
        "instance_id": "1780000000600002",
        "instance_status": 3,
        "instance_status_text": "运行中",
        "checkpoints_passed": 1,
        "checkpoints_total": 3,
        "personal_score": 30
      }
    ],
    "group_checkpoints": [
      {
        "checkpoint_id": "1780000000300210",
        "title": "全网络联通验证",
        "scope": 2,
        "is_passed": true,
        "checked_at": "2026-04-08T10:20:00Z"
      }
    ]
  }
}
```

---

### 2.29 GET /api/v1/images/:id/config-template — 获取镜像配置模板

**权限：** 教师

**响应：**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "image_id": "1780000000200001",
    "name": "geth",
    "display_name": "Go-Ethereum",
    "ecosystem": "ethereum",
    "default_ports": [
      { "port": 8545, "protocol": "tcp", "name": "HTTP RPC" },
      { "port": 8546, "protocol": "tcp", "name": "WebSocket RPC" },
      { "port": 30303, "protocol": "tcp", "name": "P2P Discovery" }
    ],
    "default_env_vars": [
      { "key": "NETWORK_ID", "value": "1337", "desc": "网络ID（开发链）", "conditions": null },
      { "key": "HTTP_API", "value": "eth,net,web3,personal,debug", "desc": "开放的RPC API", "conditions": null },
      { "key": "MINE", "value": "true", "desc": "是否启用挖矿", "conditions": null },
      { "key": "ALLOW_INSECURE_UNLOCK", "value": "true", "desc": "允许HTTP解锁账户（教学环境）", "conditions": null }
    ],
    "default_volumes": [
      { "path": "/root/.ethereum", "desc": "链数据目录" }
    ],
    "typical_companions": {
      "required": [],
      "recommended": [
        { "image": "blockscout", "reason": "本地区块链浏览器，方便查看交易和区块" }
      ],
      "optional": [
        { "image": "remix-ide", "reason": "Solidity在线IDE，可直接连接本地节点" },
        { "image": "prometheus", "reason": "采集geth指标数据" },
        { "image": "grafana", "reason": "可视化监控仪表盘" }
      ]
    },
    "required_dependencies": [],
    "resource_recommendation": {
      "cpu": "0.5",
      "memory": "1Gi",
      "disk": "2Gi"
    },
    "conditional_env_vars_example": [
      {
        "key": "CORE_LEDGER_STATE_STATEDATABASE",
        "default_value": "goleveldb",
        "conditions": [
          {
            "when": "container_exists:couchdb",
            "value": "CouchDB",
            "inject_vars": [
              { "key": "CORE_LEDGER_STATE_COUCHDBCONFIG_COUCHDBADDRESS", "value": "${COUCHDB_HOST}:5984" }
            ]
          }
        ],
        "description": "状态数据库类型，检测到 CouchDB 容器时自动切换"
      }
    ]
  }
}
```

> `conditional_env_vars_example` 仅在镜像有条件环境变量时返回（如 fabric-peer），geth 等无条件变量的镜像该字段为空数组。

---

### 2.30 GET /api/v1/images/:id/documentation — 获取镜像结构化文档

**权限：** 教师

**响应：**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "image_id": "1780000000200001",
    "name": "geth",
    "display_name": "Go-Ethereum",
    "sections": {
      "overview": "Go-Ethereum（geth）是以太坊官方Go语言客户端，支持完整的以太坊协议...",
      "version_notes": "- 1.14：支持 Cancun 升级，新增 EIP-4844 blob 交易\n- 1.13：稳定版本，推荐教学使用",
      "default_config": "默认以开发模式启动，网络ID 1337，开放 eth/net/web3/personal/debug API...",
      "typical_companions": "- **推荐搭配 blockscout**：本地区块链浏览器\n- **可选搭配 remix-ide**：Solidity 在线 IDE\n- **可选搭配 prometheus + grafana**：监控",
      "env_vars_reference": "| 变量名 | 默认值 | 说明 |\n|--------|--------|------|\n| NETWORK_ID | 1337 | 网络ID |\n| HTTP_API | eth,net,web3,personal,debug | 开放的RPC API |\n| MINE | true | 是否启用挖矿 |",
      "usage_examples": "### 单节点开发实验\n选择 geth + solidity-dev + remix-ide，适合智能合约入门...\n\n### 多节点共识实验\n选择 3 个 geth 节点，配置不同 NETWORK_ID...",
      "notes": "- 教学环境默认开启 personal API 和 insecure unlock，生产环境请勿使用\n- 开发模式下自动挖矿，无需配置矿工账户"
    }
  }
}
```

> 文档内容为 Markdown 格式字符串，前端渲染为富文本展示。

---

### 2.31 POST /api/v1/experiment-templates/:id/validate — 模板配置验证（5层）

**权限：** 模板创建教师

**请求体：**

```json
{
  "levels": [1, 2, 3, 4, 5]
}
```

> `levels` 可选，指定要执行的验证层级。默认执行全部5层。

**响应：**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "template_id": "1780000000300001",
    "is_publishable": false,
    "summary": {
      "errors": 1,
      "warnings": 1,
      "hints": 1,
      "infos": 0
    },
    "results": [
      {
        "level": 1,
        "level_name": "依赖完整性检查",
        "severity": "error",
        "passed": false,
        "issues": [
          {
            "code": "L1_MISSING_DEPENDENCY",
            "message": "kafka 需要 zookeeper 作为依赖，请添加 zookeeper 容器",
            "source_container": "kafka",
            "missing_dependency": "zookeeper",
            "suggestion": {
              "action": "add_container",
              "image": "zookeeper",
              "reason": "Kafka 必须依赖 ZooKeeper 进行集群协调"
            }
          }
        ]
      },
      {
        "level": 2,
        "level_name": "端口冲突检测",
        "severity": "error",
        "passed": true,
        "issues": []
      },
      {
        "level": 3,
        "level_name": "资源合理性检查",
        "severity": "warning",
        "passed": false,
        "issues": [
          {
            "code": "L3_TOTAL_CPU_EXCEEDS_QUOTA",
            "message": "当前配置总 CPU 需求 4 核，超过课程配额单实验上限 2 核，可能导致启动失败",
            "current_total_cpu": "4",
            "quota_limit_cpu": "2"
          }
        ]
      },
      {
        "level": 4,
        "level_name": "生态一致性提示",
        "severity": "hint",
        "passed": false,
        "issues": [
          {
            "code": "L4_TOOL_ECOSYSTEM_MISMATCH",
            "message": "fabric-explorer 适用于 Fabric 生态，当前实验未包含 Fabric 链节点",
            "tool_image": "fabric-explorer",
            "expected_ecosystem": "fabric",
            "current_ecosystems": ["ethereum"]
          }
        ]
      },
      {
        "level": 5,
        "level_name": "连通性预检",
        "severity": "info",
        "passed": true,
        "issues": []
      }
    ]
  }
}
```

> `is_publishable` 为 false 时表示存在 L1 或 L2 级别的错误，阻断发布。L3-L5 的问题不影响发布。

---

### 2.32 GET /api/v1/admin/image-pull-status — 镜像预拉取状态

**权限：** 超级管理员

**查询参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| node_name | string | 否 | 按K8s节点名筛选 |
| image_name | string | 否 | 按镜像名筛选 |
| status | int | 否 | 1=已拉取 2=拉取中 3=拉取失败 4=未拉取 |
| page | int | 否 | 页码，默认1 |
| page_size | int | 否 | 每页条数，默认20 |

**响应：**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "summary": {
      "total_images": 57,
      "total_nodes": 5,
      "fully_pulled": 52,
      "partially_pulled": 3,
      "not_pulled": 2,
      "completion_rate": 96.49
    },
    "items": [
      {
        "image_name": "geth",
        "image_version": "1.14",
        "registry_url": "registry.lianjing.com/geth:1.14",
        "source_type": 1,
        "source_type_text": "官方",
        "nodes": [
          { "node_name": "node-01", "status": 1, "status_text": "已拉取", "pulled_at": "2026-04-08T06:00:00Z", "node_cache_size": "12.3Gi" },
          { "node_name": "node-02", "status": 1, "status_text": "已拉取", "pulled_at": "2026-04-08T06:01:00Z", "node_cache_size": "11.8Gi" },
          { "node_name": "node-03", "status": 2, "status_text": "拉取中", "pulled_at": null, "node_cache_size": "8.4Gi" }
        ]
      }
    ],
    "pagination": {
      "page": 1,
      "page_size": 20,
      "total": 57
    }
  }
}
```

> `completion_rate` 表示当前筛选条件下，所有“镜像版本 × 节点”组合中状态为“已拉取”的占比，范围 0~100，保留两位小数。  
> `node_cache_size` 表示该节点当前镜像缓存总大小，便于管理员判断节点磁盘占用情况。

---

### 2.33 POST /api/v1/admin/image-pull — 触发镜像预拉取

**权限：** 超级管理员

**请求体：**

```json
{
  "image_ids": ["1780000000200001", "1780000000200002"],
  "target_nodes": ["node-01", "node-02", "node-03"],
  "force": false
}
```

> `image_ids` 为空时拉取所有官方镜像。`target_nodes` 为空时拉取到所有节点。`force` 为 true 时即使已拉取也重新拉取。

**响应：**

```json
{
  "code": 200,
  "message": "预拉取任务已创建",
  "data": {
    "task_id": "pull-task-20260408-001",
    "total_jobs": 6,
    "images": ["geth:1.14", "fabric-peer:2.5"],
    "target_nodes": ["node-01", "node-02", "node-03"],
    "status": "running",
    "created_at": "2026-04-08T10:00:00Z"
  }
}
```

> 预拉取为异步任务，可通过 GET /api/v1/admin/image-pull-status 查看进度。

---

## 三、WebSocket 接口

### 3.1 实验实例状态推送

**连接地址：**

```
wss://api.lianjing.com/api/v1/ws/experiment-instances/:id?token=<jwt>
```

**服务端推送消息类型：**

```json
// 状态变更
{
  "type": "status_change",
  "data": {
    "status": 3,
    "status_text": "运行中",
    "access_url": "https://lab.lianjing.com/instance/xxx"
  }
}

// 检查点结果
{
  "type": "checkpoint_result",
  "data": {
    "checkpoint_id": "1780000000300201",
    "title": "合约编译成功",
    "is_passed": true,
    "score": 30
  }
}

// 教师指导消息
{
  "type": "guidance_message",
  "data": {
    "teacher_name": "李教授",
    "content": "提示：请检查合约的构造函数参数是否正确",
    "sent_at": "2026-04-08T10:35:00Z"
  }
}

// 空闲超时警告
{
  "type": "idle_warning",
  "data": {
    "remaining_minutes": 5,
    "message": "您的实验环境将在5分钟后因空闲超时被回收，请继续操作或手动暂停"
  }
}

// 时长超限警告
{
  "type": "duration_warning",
  "data": {
    "remaining_minutes": 10,
    "message": "实验剩余时间10分钟，请尽快完成并提交"
  }
}

// 容器状态变更
{
  "type": "container_status",
  "data": {
    "container_name": "geth-node",
    "status": 2,
    "status_text": "运行中"
  }
}
```

### 3.2 组内实时消息

**连接地址：**

```
wss://api.lianjing.com/api/v1/ws/experiment-groups/:id/chat?token=<jwt>
```

**消息格式：**

```json
// 客户端发送
{
  "type": "chat_message",
  "content": "我这边Org1的Peer已经启动了，你那边呢？"
}

// 服务端广播
{
  "type": "chat_message",
  "data": {
    "sender_id": "1780000000000001",
    "sender_name": "张三",
    "role_name": "Org1管理员",
    "content": "我这边Org1的Peer已经启动了，你那边呢？",
    "sent_at": "2026-04-08T10:15:00Z"
  }
}

// 系统通知
{
  "type": "system_notification",
  "data": {
    "content": "组员李四已完成检查点「Peer节点启动」",
    "sent_at": "2026-04-08T10:16:00Z"
  }
}
```

### 3.3 教师监控面板实时推送

**连接地址：**

```
wss://api.lianjing.com/api/v1/ws/courses/:id/experiment-monitor?token=<jwt>&template_id=xxx
```

**服务端推送消息类型：**

```json
// 学生状态变更
{
  "type": "student_status_change",
  "data": {
    "student_id": "1780000000000001",
    "student_name": "张三",
    "instance_id": "1780000000600001",
    "old_status": 1,
    "new_status": 3,
    "new_status_text": "运行中"
  }
}

// 检查点完成通知
{
  "type": "checkpoint_completed",
  "data": {
    "student_id": "1780000000000001",
    "student_name": "张三",
    "checkpoint_title": "合约编译成功",
    "is_passed": true
  }
}

// 实验提交通知
{
  "type": "experiment_submitted",
  "data": {
    "student_id": "1780000000000001",
    "student_name": "张三",
    "auto_score": 60,
    "has_manual_items": true
  }
}

// 异常告警
{
  "type": "instance_error",
  "data": {
    "student_id": "1780000000000001",
    "student_name": "张三",
    "instance_id": "1780000000600001",
    "error_message": "容器 geth-node OOMKilled"
  }
}
```

---

## 四、SimEngine 仿真数据通道

### 4.1 SimEngine WebSocket

前端通过 WebSocket 与 SimEngine Core 微服务通信，SimEngine Core 通过 gRPC 调度场景算法容器。

**连接地址：**

```
wss://api.lianjing.com/api/v1/ws/sim-engine/:session_id?token=<jwt>
```

> `session_id` 为 SimEngine 会话ID，在启动实验时由 SimEngine Core 分配。
> 前端浏览器 WebSocket 连接通过 query token 完成鉴权；服务端实现时需校验该 token 与实验实例、会话归属关系。

**消息格式：**

```json
// 通用消息格式
{
  "type": "state_diff | state_full | event | link_update | control_ack | snapshot | action | control | rewind_to",
  "scene_code": "pbft-consensus",
  "tick": 42,
  "timestamp": 1712500000000,
  "payload": {}
}
```

**后端 → 前端消息：**

| type | 说明 | payload 示例 |
|------|------|-------------|
| `state_diff` | 状态增量更新（每 tick） | `{"nodes":{"node-3":{"status":"byzantine"}},"messages":[...]}` |
| `state_full` | 完整状态快照（初始化/回退时） | 完整状态树 |
| `event` | 仿真事件通知 | `{"event":"view_change","data":{"new_view":2,"reason":"timeout"}}` |
| `link_update` | 联动状态变更 | `{"source_scene":"pow-mining","affected_scenes":["51-percent-attack"],"changed_keys":["nodes.attacker.hashrate"]}` |
| `control_ack` | 控制指令确认 | `{"command":"pause","success":true}` |
| `snapshot` | 快照通知 | `{"snapshot_type":"keyframe","snapshot_id":"snap-001"}` |

**前端 → 后端消息：**

| type | 说明 | payload 示例 |
|------|------|-------------|
| `action` | 用户交互操作 | `{"action_code":"crash_node","params":{"node_id":"node-3"}}` |
| `control` | 仿真控制 | `{"command":"play" | "pause" | "step" | "set_speed" | "reset" | "resume","value":1.5}` |
| `rewind_to` | 回退到指定 tick | `{"target_tick":30}` |

**消息示例：**

```json
// 状态增量推送（SimEngine Core → 前端）
{
  "type": "state_diff",
  "scene_code": "blockchain-structure-fork",
  "tick": 150,
  "timestamp": 1712500000000,
  "payload": {
    "blocks": [...],
    "latest_block": 42,
    "fork_detected": false,
    "nodes": [...]
  }
}

// 用户交互操作（前端 → SimEngine Core → 场景算法容器 gRPC）
{
  "type": "action",
  "scene_code": "pbft-consensus",
  "tick": 150,
  "timestamp": 1712500000000,
  "payload": {
    "action_code": "inject_byzantine",
    "params": {
      "node_id": "node-3",
      "behavior": "send_conflicting"
    }
  }
}

// 时间控制指令（前端 → SimEngine Core）
{
  "type": "control",
  "scene_code": "pbft-consensus",
  "tick": 150,
  "timestamp": 1712500000000,
  "payload": {
    "command": "set_speed",
    "value": 1.5
  }
}

// 联动事件广播（SimEngine Core → 同联动组所有场景前端）
{
  "type": "link_update",
  "scene_code": "pbft-consensus",
  "tick": 150,
  "timestamp": 1712500000000,
  "payload": {
    "link_group_id": "1780000000700002",
    "source_scene": "pbft-consensus",
    "event": "byzantine_injected",
    "data": {
      "node_id": "node-3",
      "behavior": "send_conflicting",
      "affected_scenes": ["pbft-consensus", "byzantine-attack", "network-partition"]
    }
  }
}

// 快照通知（SimEngine Core → 前端）
{
  "type": "snapshot",
  "scene_code": "pbft-consensus",
  "tick": 150,
  "timestamp": 1712500000000,
  "payload": {
    "snapshot_type": "keyframe",
    "snapshot_id": "snap-001"
  }
}
```

**时间控制指令（command 枚举）：**

| command | 适用模式 | 说明 |
|---------|----------|------|
| play | process | 播放 |
| pause | process, continuous | 暂停 |
| step | process | 单步推进 |
| set_speed | process, continuous | 变速（0.5x / 1x / 1.5x / 2x） |
| reset | process | 重置到初始状态 |
| resume | continuous | 恢复持续运行 |

> 回退到指定 tick 不通过 `control.command` 传递，统一使用独立消息 `rewind_to`，负载为 `{"target_tick":30}`。
> process 模式使用 `play` 启动播放，不使用 `resume`；continuous 模式使用 `resume` 恢复持续运行，不使用 `play`。

> reactive 模式无时间控件，操作即响应。

---

*文档版本：v3.0*
*创建日期：2026-04-08*
*更新日期：2026-04-08*
*更新说明：v3.0 — 新增镜像配置模板接口(2.29)、镜像结构化文档接口(2.30)、5层模板配置验证接口(2.31)、镜像预拉取状态接口(2.32)、触发预拉取接口(2.33)；镜像创建接口(2.1)请求体增加typical_companions/required_dependencies/resource_recommendation/documentation_url字段；概览表新增1.23-1.25节*
