# 系统管理与监控模块 — API接口设计

> 模块状态：✅ 已确认
> 文档版本：v1.0

---

## 一、接口总览

### API 前缀：`/api/v1/system`

> 所有接口仅超级管理员可访问。

| 分类 | 方法 | 路径 | 说明 |
|------|------|------|------|
| **统一审计** | GET | /audit/logs | 聚合审计日志查询 |
| | GET | /audit/logs/export | 导出审计日志 |
| **全局配置** | GET | /configs | 获取配置列表 |
| | GET | /configs/:group | 获取某分组配置 |
| | PUT | /configs/:group/:key | 更新单个配置 |
| | PUT | /configs/:group | 批量更新分组配置 |
| | GET | /configs/change-logs | 配置变更记录 |
| **告警规则** | POST | /alert-rules | 创建告警规则 |
| | GET | /alert-rules | 告警规则列表 |
| | GET | /alert-rules/:id | 告警规则详情 |
| | PUT | /alert-rules/:id | 更新告警规则 |
| | PATCH | /alert-rules/:id/toggle | 启用/禁用规则 |
| | DELETE | /alert-rules/:id | 删除告警规则 |
| **告警事件** | GET | /alert-events | 告警事件列表 |
| | GET | /alert-events/:id | 告警事件详情 |
| | POST | /alert-events/:id/handle | 处理告警 |
| | POST | /alert-events/:id/ignore | 忽略告警 |
| **运维仪表盘** | GET | /dashboard/health | 平台健康状态 |
| | GET | /dashboard/resources | 资源使用情况 |
| | GET | /dashboard/realtime | 实时指标 |
| **平台统计** | GET | /statistics/overview | 统计总览 |
| | GET | /statistics/trend | 趋势数据 |
| | GET | /statistics/schools | 学校活跃度排行 |
| **数据备份** | POST | /backups/trigger | 手动触发备份 |
| | GET | /backups | 备份列表 |
| | GET | /backups/:id/download | 下载备份文件 |
| | PUT | /backups/config | 更新备份配置 |
| | GET | /backups/config | 获取备份配置 |

---

## 二、接口详细定义

### 2.1 统一审计

#### GET /api/v1/system/audit/logs — 聚合审计日志查询

**查询参数：**
| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page | int | 否 | 页码，默认1 |
| page_size | int | 否 | 每页条数，默认20 |
| source | string | 否 | 日志来源：`login`/`operation`/`experiment`，不传则查询全部 |
| keyword | string | 否 | 关键词搜索（操作人姓名、操作类型、IP等） |
| operator_id | string | 否 | 操作人ID |
| action | string | 否 | 操作类型 |
| date_from | string | 否 | 开始时间 |
| date_to | string | 否 | 结束时间 |
| ip | string | 否 | IP地址 |

**成功响应：**
```json
{
  "code": 200,
  "data": {
    "list": [
      {
        "id": "1780000000001",
        "source": "login",
        "source_text": "登录日志",
        "operator_id": "1780000000500",
        "operator_name": "张三",
        "action": "login_success",
        "action_text": "登录成功",
        "target": null,
        "detail": { "login_method": "password" },
        "ip": "192.168.1.100",
        "user_agent": "Mozilla/5.0...",
        "created_at": "2026-04-09T10:00:00Z"
      },
      {
        "id": "1780000000002",
        "source": "operation",
        "source_text": "操作日志",
        "operator_id": "1780000000001",
        "operator_name": "管理员",
        "action": "import_students",
        "action_text": "批量导入学生",
        "target": { "type": "user", "id": null },
        "detail": { "total": 50, "success": 48, "fail": 2 },
        "ip": "192.168.1.1",
        "user_agent": null,
        "created_at": "2026-04-09T09:30:00Z"
      },
      {
        "id": "1780000000003",
        "source": "experiment",
        "source_text": "实验操作日志",
        "operator_id": "1780000000600",
        "operator_name": "李四",
        "action": "terminal_command",
        "action_text": "终端命令",
        "target": { "type": "experiment_instance", "id": "1790000000100" },
        "detail": { "command": "docker ps", "container": "eth-node" },
        "ip": null,
        "user_agent": null,
        "created_at": "2026-04-09T09:00:00Z"
      }
    ],
    "pagination": { "page": 1, "page_size": 20, "total": 1500, "total_pages": 75 },
    "source_counts": {
      "login": 800,
      "operation": 500,
      "experiment": 200
    }
  }
}
```

> **后端实现说明：** 此接口根据 `source` 参数决定查询哪些表。如果 `source` 为空，则并行查询三张表，合并后按 `created_at` 排序分页。为保证性能，建议前端默认选择某个日志来源，而非查询全部。

#### GET /api/v1/system/audit/logs/export — 导出审计日志

**查询参数：** 同审计日志查询，额外增加 `format`（`excel`/`csv`，默认`excel`）

**响应：** 文件下载（Content-Type: application/octet-stream）

---

### 2.2 全局配置

#### GET /api/v1/system/configs — 获取配置列表

**成功响应：**
```json
{
  "code": 200,
  "data": {
    "groups": [
      {
        "group": "platform",
        "group_text": "平台基本信息",
        "configs": [
          { "key": "name", "value": "链镜", "value_type": 1, "description": "平台名称", "is_sensitive": false },
          { "key": "logo_url", "value": "/static/logo.png", "value_type": 1, "description": "平台Logo", "is_sensitive": false },
          { "key": "icp_number", "value": "京ICP备XXXXXXXX号", "value_type": 1, "description": "ICP备案号", "is_sensitive": false }
        ]
      },
      {
        "group": "storage",
        "group_text": "存储配置",
        "configs": [
          { "key": "default_school_quota_gb", "value": "100", "value_type": 2, "description": "学校默认存储配额(GB)", "is_sensitive": false }
        ]
      },
      {
        "group": "security",
        "group_text": "安全配置",
        "configs": [
          { "key": "session_timeout_hours", "value": "24", "value_type": 2, "description": "会话超时(小时)", "is_sensitive": false },
          { "key": "max_login_fail_count", "value": "5", "value_type": 2, "description": "最大登录失败次数", "is_sensitive": false }
        ]
      }
    ]
  }
}
```

> 敏感配置（`is_sensitive=true`）的 `value` 返回 `"******"`。

#### PUT /api/v1/system/configs/:group/:key — 更新单个配置

**请求体：**
```json
{
  "value": "链镜教学平台"
}
```

**后端逻辑：**
1. 校验配置键存在
2. 校验值类型匹配
3. 记录变更到 `config_change_logs`（old_value, new_value, changed_by, ip）
4. 更新配置值
5. 刷新Redis缓存

#### GET /api/v1/system/configs/change-logs — 配置变更记录

**查询参数：** `config_group`, `config_key`, `date_from`, `date_to`, `page`, `page_size`

**成功响应：**
```json
{
  "code": 200,
  "data": {
    "list": [
      {
        "id": "1950000000001",
        "config_group": "platform",
        "config_key": "name",
        "old_value": "链镜",
        "new_value": "链镜教学平台",
        "changed_by": "1780000000001",
        "changed_by_name": "超级管理员",
        "changed_at": "2026-04-09T10:00:00Z",
        "ip": "192.168.1.1"
      }
    ],
    "pagination": { "page": 1, "page_size": 20, "total": 10, "total_pages": 1 }
  }
}
```

---

### 2.3 告警规则

#### POST /api/v1/system/alert-rules — 创建告警规则

**请求体：**
```json
{
  "name": "CPU使用率过高",
  "description": "K8s节点CPU使用率超过80%持续5分钟",
  "alert_type": 1,
  "level": 3,
  "condition": {
    "metric": "cpu_usage",
    "operator": ">",
    "value": 80,
    "duration": 300
  },
  "silence_period": 1800
}
```

#### GET /api/v1/system/alert-rules — 告警规则列表

**查询参数：** `alert_type`, `level`, `is_enabled`, `page`, `page_size`

#### GET /api/v1/system/alert-rules/:id — 告警规则详情

**成功响应：**
```json
{
  "code": 200,
  "data": {
    "id": "1950000000001",
    "name": "CPU使用率过高",
    "description": "K8s节点CPU使用率超过80%持续5分钟",
    "alert_type": 1,
    "alert_type_text": "阈值告警",
    "level": 3,
    "level_text": "严重",
    "condition": {
      "metric": "cpu_usage",
      "operator": ">",
      "value": 80,
      "duration": 300
    },
    "silence_period": 1800,
    "is_enabled": true,
    "created_at": "2026-04-09T10:00:00Z"
  }
}
```

#### PATCH /api/v1/system/alert-rules/:id/toggle — 启用/禁用规则

**请求体：**
```json
{
  "is_enabled": false
}
```

---

### 2.4 告警事件

#### GET /api/v1/system/alert-events — 告警事件列表

**查询参数：** `rule_id`, `level`, `status`, `date_from`, `date_to`, `page`, `page_size`

**成功响应：**
```json
{
  "code": 200,
  "data": {
    "list": [
      {
        "id": "1950000000100",
        "rule_id": "1950000000001",
        "rule_name": "CPU使用率过高",
        "level": 3,
        "level_text": "严重",
        "title": "K8s节点 k8s-node-01 CPU使用率 92.5%",
        "detail": {
          "metric": "cpu_usage",
          "current_value": 92.5,
          "threshold": 80,
          "node": "k8s-node-01"
        },
        "status": 1,
        "status_text": "待处理",
        "triggered_at": "2026-04-09T10:00:00Z"
      }
    ],
    "pagination": { "page": 1, "page_size": 20, "total": 15, "total_pages": 1 },
    "status_counts": { "pending": 3, "handled": 10, "ignored": 2 }
  }
}
```

#### GET /api/v1/system/alert-events/:id — 告警事件详情

**成功响应：**
```json
{
  "code": 200,
  "data": {
    "id": "1950000000100",
    "rule_id": "1950000000001",
    "rule_name": "CPU使用率过高",
    "level": 3,
    "level_text": "严重",
    "title": "K8s节点 k8s-node-01 CPU使用率 92.5%",
    "detail": {
      "metric": "cpu_usage",
      "current_value": 92.5,
      "threshold": 80,
      "duration_seconds": 320,
      "node": "k8s-node-01"
    },
    "status": 2,
    "status_text": "已处理",
    "handled_by": "1780000000001",
    "handled_by_name": "超级管理员",
    "handled_at": "2026-04-09T10:05:00Z",
    "handle_note": "已扩容K8s节点，CPU使用率已恢复正常。",
    "triggered_at": "2026-04-09T10:00:00Z"
  }
}
```

#### POST /api/v1/system/alert-events/:id/handle — 处理告警

**请求体：**
```json
{
  "handle_note": "已扩容K8s节点，CPU使用率已恢复正常。"
}
```

---

### 2.5 运维仪表盘

#### GET /api/v1/system/dashboard/health — 平台健康状态

**成功响应：**
```json
{
  "code": 200,
  "data": {
    "overall_status": "healthy",
    "services": [
      { "name": "API Server", "status": "healthy", "latency_ms": 5, "uptime": "15d 3h" },
      { "name": "PostgreSQL", "status": "healthy", "latency_ms": 2, "connections": { "active": 15, "max": 100 } },
      { "name": "Redis", "status": "healthy", "latency_ms": 1, "memory_used_mb": 256 },
      { "name": "NATS", "status": "healthy", "latency_ms": 1, "messages_in_queue": 0 },
      { "name": "MinIO", "status": "healthy", "latency_ms": 8, "storage_used_gb": 45.2 },
      { "name": "K8s Cluster", "status": "healthy", "nodes": 3, "pods_running": 42 }
    ],
    "last_check_at": "2026-04-09T10:00:00Z"
  }
}
```

#### GET /api/v1/system/dashboard/resources — 资源使用情况

**成功响应：**
```json
{
  "code": 200,
  "data": {
    "cpu": { "usage_percent": 45.2, "cores_total": 16, "cores_used": 7.2 },
    "memory": { "usage_percent": 62.5, "total_gb": 64, "used_gb": 40 },
    "storage": { "usage_percent": 35.0, "total_gb": 500, "used_gb": 175 },
    "k8s": {
      "nodes": 3,
      "pods_total": 42,
      "pods_running": 40,
      "pods_pending": 2,
      "namespaces": 15
    }
  }
}
```

#### GET /api/v1/system/dashboard/realtime — 实时指标

**成功响应：**
```json
{
  "code": 200,
  "data": {
    "online_users": 128,
    "active_experiments": 35,
    "active_competitions": 2,
    "api_requests_per_minute": 450,
    "pending_alerts": 3,
    "recent_alerts": [
      { "id": "...", "title": "CPU使用率过高", "level": 3, "triggered_at": "..." }
    ]
  }
}
```

---

### 2.6 平台统计

#### GET /api/v1/system/statistics/overview — 统计总览

**成功响应：**
```json
{
  "code": 200,
  "data": {
    "total_users": 5000,
    "total_schools": 10,
    "total_courses": 150,
    "total_experiments": 8500,
    "total_competitions": 25,
    "today": {
      "active_users": 320,
      "new_users": 15,
      "experiments_started": 85,
      "api_requests": 45000
    }
  }
}
```

#### GET /api/v1/system/statistics/trend — 趋势数据

**查询参数：** `metric`（`active_users`/`new_users`/`experiments`/`api_requests`）, `period`（`7d`/`30d`/`90d`/`365d`）

**成功响应：**
```json
{
  "code": 200,
  "data": {
    "metric": "active_users",
    "period": "30d",
    "data_points": [
      { "date": "2026-03-10", "value": 280 },
      { "date": "2026-03-11", "value": 310 },
      { "date": "2026-03-12", "value": 295 }
    ]
  }
}
```

#### GET /api/v1/system/statistics/schools — 学校活跃度排行

**成功响应：**
```json
{
  "code": 200,
  "data": {
    "list": [
      {
        "rank": 1,
        "school_id": "1780000000001",
        "school_name": "XX大学",
        "active_users": 320,
        "total_users": 500,
        "activity_score": 86.4
      },
      {
        "rank": 2,
        "school_id": "1780000000002",
        "school_name": "YY大学",
        "active_users": 280,
        "total_users": 460,
        "activity_score": 79.5
      }
    ]
  }
}
```

---

### 2.7 数据备份

#### POST /api/v1/system/backups/trigger — 手动触发备份

**成功响应：**
```json
{
  "code": 200,
  "data": {
    "id": "1950000000200",
    "backup_type": 2,
    "status": 1,
    "status_text": "进行中",
    "started_at": "2026-04-09T10:00:00Z"
  }
}
```

> 备份异步执行，前端可轮询备份列表查看状态。

#### GET /api/v1/system/backups — 备份列表

**查询参数：** `status`, `page`, `page_size`

**成功响应：**
```json
{
  "code": 200,
  "data": {
    "list": [
      {
        "id": "1950000000200",
        "backup_type": 2,
        "backup_type_text": "手动备份",
        "status": 2,
        "status_text": "成功",
        "database_name": "lenschain",
        "file_size": 1073741824,
        "file_size_text": "1.0 GB",
        "started_at": "2026-04-09T10:00:00Z",
        "completed_at": "2026-04-09T10:05:00Z",
        "duration_seconds": 300
      }
    ],
    "pagination": { "page": 1, "page_size": 20, "total": 30, "total_pages": 2 },
    "backup_config": {
      "auto_enabled": true,
      "cron": "0 2 * * *",
      "retention_count": 30
    }
  }
}
```

---

## 三、跨模块接口调用说明

### 3.1 本模块调用外部模块

| 调用目标 | 方式 | 场景 |
|---------|------|------|
| 模块01 `login_logs` | 数据库直接查询 | 统一审计 — 登录日志 |
| 模块01 `operation_logs` | 数据库直接查询 | 统一审计 — 操作日志 |
| 模块04 `instance_operation_logs` | 数据库直接查询 | 统一审计 — 实验操作日志 |
| 模块01 `users` | 数据库直接查询 | 审计日志中操作人姓名 |
| 模块02 `schools` | 数据库COUNT查询 | 平台统计 — 学校数量 |
| 模块03 `courses` | 数据库COUNT查询 | 平台统计 — 课程数量 |
| 模块04 `experiment_instances` | 数据库COUNT查询 | 平台统计 — 实验数量 |
| 模块05 `competitions` | 数据库COUNT查询 | 平台统计 — 竞赛数量 |
| 模块07 | 通知发送接口 | 告警通知、系统公告 |

### 3.2 外部模块调用本模块

无。模块08是最顶层聚合模块，不被其他模块调用。

---

*文档版本：v1.0*
*创建日期：2026-04-09*
*更新日期：2026-04-09*
