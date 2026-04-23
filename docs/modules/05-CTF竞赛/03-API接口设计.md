# CTF竞赛模块 — API 接口设计

> 模块状态：✅ 已确认
> 文档版本：v1.0
> 遵循规范：[API规范](../../standards/API规范.md)

---

## 一、接口总览

### 1.1 竞赛管理

| 方法 | 路径 | 说明 | 权限 |
|------|------|------|------|
| POST | /api/v1/ctf/competitions | 创建竞赛 | 超级管理员/学校管理员 |
| GET | /api/v1/ctf/competitions | 竞赛列表 | 全部角色 |
| GET | /api/v1/ctf/competitions/:id | 竞赛详情 | 全部角色 |
| PUT | /api/v1/ctf/competitions/:id | 编辑竞赛 | 竞赛创建者 |
| DELETE | /api/v1/ctf/competitions/:id | 删除竞赛 | 竞赛创建者 |
| POST | /api/v1/ctf/competitions/:id/publish | 发布竞赛（草稿→报名中） | 竞赛创建者 |
| POST | /api/v1/ctf/competitions/:id/archive | 归档竞赛（已结束→已归档） | 竞赛创建者 |
| POST | /api/v1/ctf/competitions/:id/terminate | 强制终止竞赛 | 超级管理员 |

### 1.2 题目管理

| 方法 | 路径 | 说明 | 权限 |
|------|------|------|------|
| POST | /api/v1/ctf/challenges | 创建题目 | 教师 |
| GET | /api/v1/ctf/challenges | 题目列表（题库） | 超级管理员/教师 |
| GET | /api/v1/ctf/challenges/:id | 题目详情 | 超级管理员/题目作者 |
| PUT | /api/v1/ctf/challenges/:id | 编辑题目 | 题目作者 |
| DELETE | /api/v1/ctf/challenges/:id | 删除题目 | 题目作者 |
| POST | /api/v1/ctf/challenges/:id/submit-review | 提交审核（草稿→待审核） | 题目作者 |

### 1.3 题目合约管理

| 方法 | 路径 | 说明 | 权限 |
|------|------|------|------|
| GET | /api/v1/ctf/challenges/:id/contracts | 合约列表 | 题目作者 |
| POST | /api/v1/ctf/challenges/:id/contracts | 添加合约 | 题目作者 |
| PUT | /api/v1/ctf/challenge-contracts/:id | 编辑合约 | 题目作者 |
| DELETE | /api/v1/ctf/challenge-contracts/:id | 删除合约 | 题目作者 |

### 1.4 题目断言管理

| 方法 | 路径 | 说明 | 权限 |
|------|------|------|------|
| GET | /api/v1/ctf/challenges/:id/assertions | 断言列表 | 题目作者 |
| POST | /api/v1/ctf/challenges/:id/assertions | 添加断言 | 题目作者 |
| PUT | /api/v1/ctf/challenge-assertions/:id | 编辑断言 | 题目作者 |
| DELETE | /api/v1/ctf/challenge-assertions/:id | 删除断言 | 题目作者 |
| PUT | /api/v1/ctf/challenges/:id/assertions/sort | 断言排序 | 题目作者 |

### 1.5 漏洞转化

| 方法 | 路径 | 说明 | 权限 |
|------|------|------|------|
| GET | /api/v1/ctf/swc-registry | SWC Registry 列表 | 教师 |
| POST | /api/v1/ctf/challenges/import-swc | 从SWC导入生成题目 | 教师 |
| GET | /api/v1/ctf/challenge-templates | 参数化模板列表 | 教师 |
| GET | /api/v1/ctf/challenge-templates/:id | 模板详情（含参数定义和变体） | 教师 |
| POST | /api/v1/ctf/challenges/generate-from-template | 从模板生成题目 | 教师 |

### 1.6 题目预验证

| 方法 | 路径 | 说明 | 权限 |
|------|------|------|------|
| POST | /api/v1/ctf/challenges/:id/verify | 发起预验证（6步） | 题目作者 |
| GET | /api/v1/ctf/challenges/:id/verifications | 预验证记录列表 | 题目作者 |
| GET | /api/v1/ctf/challenge-verifications/:id | 预验证详情（含各步结果） | 题目作者 |

### 1.7 题目审核

| 方法 | 路径 | 说明 | 权限 |
|------|------|------|------|
| GET | /api/v1/ctf/challenge-reviews/pending | 待审核题目列表 | 超级管理员 |
| POST | /api/v1/ctf/challenges/:id/review | 审核题目 | 超级管理员 |
| GET | /api/v1/ctf/challenges/:id/reviews | 题目审核记录 | 超级管理员/题目作者 |

### 1.8 竞赛题目配置

| 方法 | 路径 | 说明 | 权限 |
|------|------|------|------|
| GET | /api/v1/ctf/competitions/:id/challenges | 竞赛题目列表 | 竞赛创建者/参赛选手 |
| POST | /api/v1/ctf/competitions/:id/challenges | 添加题目到竞赛 | 竞赛创建者 |
| DELETE | /api/v1/ctf/competition-challenges/:id | 移除竞赛题目 | 竞赛创建者 |
| PUT | /api/v1/ctf/competitions/:id/challenges/sort | 竞赛题目排序 | 竞赛创建者 |

### 1.9 团队管理

| 方法 | 路径 | 说明 | 权限 |
|------|------|------|------|
| POST | /api/v1/ctf/competitions/:id/teams | 创建团队 | 学生 |
| GET | /api/v1/ctf/competitions/:id/teams | 竞赛团队列表 | 全部角色 |
| GET | /api/v1/ctf/teams/:id | 团队详情 | 团队成员/管理员 |
| PUT | /api/v1/ctf/teams/:id | 编辑团队信息 | 队长 |
| POST | /api/v1/ctf/teams/:id/disband | 解散团队 | 队长 |
| POST | /api/v1/ctf/teams/join | 通过邀请码加入团队 | 学生 |
| DELETE | /api/v1/ctf/teams/:id/members/:student_id | 移除队员 | 队长 |
| POST | /api/v1/ctf/teams/:id/leave | 退出团队 | 队员 |

### 1.10 竞赛报名

| 方法 | 路径 | 说明 | 权限 |
|------|------|------|------|
| POST | /api/v1/ctf/competitions/:id/register | 报名竞赛 | 队长/个人 |
| DELETE | /api/v1/ctf/competitions/:id/register | 取消报名 | 队长/个人 |
| GET | /api/v1/ctf/competitions/:id/registrations | 报名列表 | 竞赛创建者 |
| GET | /api/v1/ctf/competitions/:id/my-registration | 我的报名状态 | 学生 |

### 1.11 解题赛提交与验证

| 方法 | 路径 | 说明 | 权限 |
|------|------|------|------|
| POST | /api/v1/ctf/competitions/:id/submissions | 提交Flag/攻击交易 | 参赛选手 |
| GET | /api/v1/ctf/competitions/:id/submissions | 团队提交记录 | 参赛选手 |
| GET | /api/v1/ctf/competitions/:id/submissions/statistics | 提交统计 | 竞赛创建者 |

### 1.12 攻防赛分组管理

| 方法 | 路径 | 说明 | 权限 |
|------|------|------|------|
| POST | /api/v1/ctf/competitions/:id/ad-groups | 创建攻防赛分组 | 竞赛创建者 |
| GET | /api/v1/ctf/competitions/:id/ad-groups | 分组列表 | 竞赛创建者/参赛选手 |
| GET | /api/v1/ctf/ad-groups/:id | 分组详情 | 竞赛创建者/分组内选手 |
| POST | /api/v1/ctf/competitions/:id/ad-groups/auto-assign | 自动分组 | 竞赛创建者 |

### 1.13 攻防赛回合管理

| 方法 | 路径 | 说明 | 权限 |
|------|------|------|------|
| GET | /api/v1/ctf/ad-groups/:id/rounds | 回合列表 | 竞赛创建者/分组内选手 |
| GET | /api/v1/ctf/ad-rounds/:id | 回合详情 | 竞赛创建者/分组内选手 |
| GET | /api/v1/ctf/ad-groups/:id/current-round | 当前回合状态 | 分组内选手 |

### 1.14 攻防赛攻击

| 方法 | 路径 | 说明 | 权限 |
|------|------|------|------|
| POST | /api/v1/ctf/ad-rounds/:id/attacks | 提交攻击交易 | 参赛选手 |
| GET | /api/v1/ctf/ad-rounds/:id/attacks | 本回合攻击记录 | 参赛选手 |
| GET | /api/v1/ctf/ad-groups/:id/attacks | 分组全部攻击记录 | 竞赛创建者/分组内选手 |

### 1.15 攻防赛防守

| 方法 | 路径 | 说明 | 权限 |
|------|------|------|------|
| POST | /api/v1/ctf/ad-rounds/:id/defenses | 提交补丁合约 | 参赛选手 |
| GET | /api/v1/ctf/ad-rounds/:id/defenses | 本回合防守记录 | 参赛选手 |

### 1.16 攻防赛Token流水

| 方法 | 路径 | 说明 | 权限 |
|------|------|------|------|
| GET | /api/v1/ctf/competitions/:id/token-ledger | Token流水记录 | 竞赛创建者 |
| GET | /api/v1/ctf/teams/:id/token-ledger | 团队Token流水 | 团队成员 |

### 1.17 排行榜

| 方法 | 路径 | 说明 | 权限 |
|------|------|------|------|
| GET | /api/v1/ctf/competitions/:id/leaderboard | 实时排行榜 | 全部角色 |
| GET | /api/v1/ctf/competitions/:id/leaderboard/history | 排行榜历史快照 | 全部角色 |
| GET | /api/v1/ctf/competitions/:id/leaderboard/final | 最终排名 | 全部角色 |

### 1.18 公告管理

| 方法 | 路径 | 说明 | 权限 |
|------|------|------|------|
| POST | /api/v1/ctf/competitions/:id/announcements | 发布公告 | 竞赛创建者 |
| GET | /api/v1/ctf/competitions/:id/announcements | 公告列表 | 全部角色 |
| GET | /api/v1/ctf/announcements/:id | 公告详情 | 全部角色 |
| DELETE | /api/v1/ctf/announcements/:id | 删除公告 | 竞赛创建者 |

### 1.19 CTF资源配额

| 方法 | 路径 | 说明 | 权限 |
|------|------|------|------|
| GET | /api/v1/ctf/competitions/:id/resource-quota | 竞赛资源配额详情 | 超级管理员/竞赛创建者 |
| PUT | /api/v1/ctf/competitions/:id/resource-quota | 设置竞赛资源配额 | 超级管理员 |

### 1.20 题目环境管理

| 方法 | 路径 | 说明 | 权限 |
|------|------|------|------|
| POST | /api/v1/ctf/competitions/:comp_id/challenges/:challenge_id/environment | 启动题目环境 | 参赛选手 |
| GET | /api/v1/ctf/challenge-environments/:id | 环境详情 | 参赛选手 |
| POST | /api/v1/ctf/challenge-environments/:id/reset | 重置题目环境 | 参赛选手 |
| POST | /api/v1/ctf/challenge-environments/:id/destroy | 销毁题目环境 | 参赛选手/管理员 |
| GET | /api/v1/ctf/competitions/:id/my-environments | 我的所有题目环境 | 参赛选手 |

### 1.21 攻防赛队伍链

| 方法 | 路径 | 说明 | 权限 |
|------|------|------|------|
| GET | /api/v1/ctf/teams/:id/chain | 队伍链信息 | 团队成员 |
| GET | /api/v1/ctf/ad-groups/:id/chains | 分组所有队伍链 | 分组内选手 |

### 1.22 竞赛监控

| 方法 | 路径 | 说明 | 权限 |
|------|------|------|------|
| GET | /api/v1/ctf/admin/competitions/overview | 全平台竞赛概览 | 超级管理员 |
| GET | /api/v1/ctf/competitions/:id/monitor | 竞赛运行监控 | 竞赛创建者/超级管理员 |
| GET | /api/v1/ctf/competitions/:id/environments | 竞赛环境资源列表 | 竞赛创建者/超级管理员 |
| POST | /api/v1/ctf/challenge-environments/:id/force-destroy | 强制回收环境 | 超级管理员 |

### 1.23 竞赛统计

| 方法 | 路径 | 说明 | 权限 |
|------|------|------|------|
| GET | /api/v1/ctf/competitions/:id/statistics | 竞赛统计数据 | 竞赛创建者/超级管理员 |
| GET | /api/v1/ctf/competitions/:id/results | 竞赛最终结果 | 全部角色 |

---

## 二、核心接口详细定义

### 2.1 POST /api/v1/ctf/competitions — 创建竞赛

**权限：** 超级管理员（平台级竞赛）/ 学校管理员（校级竞赛）

**请求体：**

```json
{
  "title": "链镜杯·2026春季区块链安全挑战赛",
  "description": "# 赛事简介\n\n面向全平台的区块链安全CTF竞赛...",
  "banner_url": "https://oss.example.com/banners/ctf-2026-spring.png",
  "competition_type": 1,
  "scope": 1,
  "school_id": null,
  "team_mode": 2,
  "max_team_size": 4,
  "min_team_size": 1,
  "max_teams": 100,
  "registration_start_at": "2026-04-10T00:00:00Z",
  "registration_end_at": "2026-04-18T23:59:59Z",
  "start_at": "2026-04-20T09:00:00Z",
  "end_at": "2026-04-20T17:00:00Z",
  "freeze_at": "2026-04-20T16:00:00Z",
  "rules": "## 竞赛规则\n\n1. 禁止攻击平台基础设施...",
  "jeopardy_config": {
    "scoring": {
      "decay_factor": 0.95,
      "min_score_ratio": 0.2,
      "first_blood_bonus": 0.1
    },
    "submission_limit": {
      "max_per_minute": 5,
      "cooldown_threshold": 10,
      "cooldown_minutes": 5
    }
  }
}
```

> `competition_type=1` 时填 `jeopardy_config`，`competition_type=2` 时填 `ad_config`，不可同时填写。
> `scope=2`（校级竞赛）时 `school_id` 必填。

**响应：**

```json
{
  "code": 200,
  "message": "创建成功",
  "data": {
    "id": "1780000000500001",
    "title": "链镜杯·2026春季区块链安全挑战赛",
    "competition_type": 1,
    "competition_type_text": "解题赛",
    "scope": 1,
    "scope_text": "平台级",
    "team_mode": 2,
    "team_mode_text": "自由组队",
    "status": 1,
    "status_text": "草稿",
    "created_at": "2026-04-08T10:00:00Z"
  }
}
```

**错误响应：**

| code | message | 场景 |
|------|---------|------|
| 40001 | 竞赛名称不能为空 | title 为空 |
| 40002 | 竞赛类型无效 | competition_type 不是 1 或 2 |
| 40003 | 校级竞赛必须指定学校ID | scope=2 但 school_id 为空 |
| 40004 | 报名时间必须早于竞赛开始时间 | 时间校验失败 |
| 40005 | 解题赛必须配置jeopardy_config | competition_type=1 但配置缺失 |
| 40301 | 学校管理员只能创建校级竞赛 | 学校管理员尝试创建平台级竞赛 |

---

### 2.2 GET /api/v1/ctf/competitions — 竞赛列表

**权限：** 全部角色

**查询参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page | int | 否 | 页码，默认1 |
| page_size | int | 否 | 每页条数，默认20 |
| competition_type | int | 否 | 竞赛类型：1解题赛 2攻防赛 |
| scope | int | 否 | 竞赛范围：1平台级 2校级 |
| status | int | 否 | 状态筛选 |
| keyword | string | 否 | 关键词搜索（标题） |

**响应：**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "list": [
      {
        "id": "1780000000500001",
        "title": "链镜杯·2026春季区块链安全挑战赛",
        "banner_url": "https://oss.example.com/banners/ctf-2026-spring.png",
        "competition_type": 1,
        "competition_type_text": "解题赛",
        "scope": 1,
        "scope_text": "平台级",
        "team_mode": 2,
        "team_mode_text": "自由组队",
        "max_team_size": 4,
        "status": 2,
        "status_text": "报名中",
        "registered_teams": 45,
        "max_teams": 100,
        "challenge_count": 12,
        "registration_start_at": "2026-04-10T00:00:00Z",
        "registration_end_at": "2026-04-18T23:59:59Z",
        "start_at": "2026-04-20T09:00:00Z",
        "end_at": "2026-04-20T17:00:00Z",
        "created_by_name": "平台管理员"
      }
    ],
    "pagination": {
      "page": 1,
      "page_size": 20,
      "total": 8,
      "total_pages": 1
    }
  }
}
```

> 学生看到的列表根据 scope 和 school_id 自动过滤：平台级竞赛全员可见，校级竞赛仅本校学生和被邀请的外校学生可见。

---

### 2.3 GET /api/v1/ctf/competitions/:id — 竞赛详情

**权限：** 全部角色

**响应：**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "id": "1780000000500001",
    "title": "链镜杯·2026春季区块链安全挑战赛",
    "description": "# 赛事简介\n\n...",
    "banner_url": "https://oss.example.com/banners/ctf-2026-spring.png",
    "competition_type": 1,
    "competition_type_text": "解题赛",
    "scope": 1,
    "scope_text": "平台级",
    "team_mode": 2,
    "team_mode_text": "自由组队",
    "max_team_size": 4,
    "min_team_size": 1,
    "max_teams": 100,
    "status": 3,
    "status_text": "进行中",
    "registration_start_at": "2026-04-10T00:00:00Z",
    "registration_end_at": "2026-04-18T23:59:59Z",
    "start_at": "2026-04-20T09:00:00Z",
    "end_at": "2026-04-20T17:00:00Z",
    "freeze_at": "2026-04-20T16:00:00Z",
    "rules": "## 竞赛规则\n\n...",
    "jeopardy_config": {
      "scoring": {
        "decay_factor": 0.95,
        "min_score_ratio": 0.2,
        "first_blood_bonus": 0.1
      },
      "submission_limit": {
        "max_per_minute": 5,
        "cooldown_threshold": 10,
        "cooldown_minutes": 5
      }
    },
    "ad_config": null,
    "registered_teams": 78,
    "challenge_count": 12,
    "created_by": {
      "id": "1780000000000001",
      "name": "平台管理员"
    },
    "created_at": "2026-04-08T10:00:00Z"
  }
}
```

**错误响应：**

| code | message | 场景 |
|------|---------|------|
| 40401 | 竞赛不存在 | 竞赛ID无效或已删除 |

---

### 2.4 POST /api/v1/ctf/competitions/:id/publish — 发布竞赛

**权限：** 竞赛创建者

**请求体：** 无

**响应：**

```json
{
  "code": 200,
  "message": "发布成功",
  "data": {
    "id": "1780000000500001",
    "status": 2,
    "status_text": "报名中"
  }
}
```

**错误响应：**

| code | message | 场景 |
|------|---------|------|
| 40401 | 竞赛不存在 | 竞赛ID无效 |
| 40001 | 竞赛必须处于草稿状态才能发布 | 状态不是草稿 |
| 40002 | 竞赛至少需要1道题目 | 未配置题目 |
| 40003 | 请设置完整的竞赛时间 | 时间字段缺失 |

---

### 2.5 POST /api/v1/ctf/competitions/:id/terminate — 强制终止竞赛

**权限：** 超级管理员

**请求体：**

```json
{
  "reason": "发现严重安全问题，需要紧急终止"
}
```

**响应：**

```json
{
  "code": 200,
  "message": "竞赛已强制终止",
  "data": {
    "id": "1780000000500001",
    "status": 4,
    "status_text": "已结束",
    "environments_destroyed": 156
  }
}
```

> 强制终止会立即回收所有竞赛环境资源，排行榜以终止时刻的状态为最终结果。

---

### 2.6 POST /api/v1/ctf/challenges — 创建题目

**权限：** 教师

**请求体：**

```json
{
  "title": "重入攻击：银行合约",
  "description": "# 题目描述\n\n一个存在重入漏洞的银行合约...",
  "category": "contract",
  "difficulty": 2,
  "base_score": 300,
  "flag_type": 3,
  "runtime_mode": 1,
  "chain_config": {
    "chain_type": "evm",
    "chain_version": "london",
    "block_number": 0,
    "accounts": [
      {"name": "deployer", "balance": "100 ether"},
      {"name": "attacker", "balance": "10 ether"}
    ]
  },
  "setup_transactions": [
    {
      "from": "deployer",
      "to": "VulnerableBank",
      "function": "deposit",
      "args": [],
      "value": "10 ether"
    }
  ],
  "source_path": 3,
  "attachment_urls": []
}
```

> `flag_type=1`（静态Flag）时需填 `static_flag`；`flag_type=2`（动态Flag）时需填 `dynamic_flag_secret`；`flag_type=3`（链上验证）时需配置 contracts 和 assertions（通过子接口）。
> `category` 非智能合约类型时使用 `environment_config` 代替 `chain_config`。
> `runtime_mode=1` 表示独立链模式；`runtime_mode=2` 表示 Fork 模式。Fork 模式要求 `chain_config.fork.rpc_url` 和 `chain_config.fork.block_number` 均已配置。

**响应：**

```json
{
  "code": 200,
  "message": "创建成功",
  "data": {
    "id": "1780000000510001",
    "title": "重入攻击：银行合约",
    "category": "contract",
    "category_text": "智能合约安全",
    "difficulty": 2,
    "difficulty_text": "Easy",
    "base_score": 300,
    "flag_type": 3,
    "flag_type_text": "链上状态验证",
    "runtime_mode": 1,
    "runtime_mode_text": "独立链模式",
    "source_path": 3,
    "source_path_text": "完全自定义",
    "status": 1,
    "status_text": "草稿",
    "created_at": "2026-04-08T11:00:00Z"
  }
}
```

**错误响应：**

| code | message | 场景 |
|------|---------|------|
| 40001 | 题目名称不能为空 | title 为空 |
| 40002 | 题目类型无效 | category 不在 6 种类型中 |
| 40003 | 难度等级无效 | difficulty 不在 1-5 范围 |
| 40004 | 基础分值超出难度范围 | base_score 与 difficulty 不匹配 |
| 40005 | 静态Flag题目必须填写flag值 | flag_type=1 但 static_flag 为空 |

---

### 2.7 GET /api/v1/ctf/challenges — 题目列表（题库）

**权限：** 超级管理员 / 教师

**查询参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page | int | 否 | 页码，默认1 |
| page_size | int | 否 | 每页条数，默认20 |
| category | string | 否 | 题目类型筛选 |
| difficulty | int | 否 | 难度筛选 |
| flag_type | int | 否 | Flag类型筛选 |
| status | int | 否 | 状态筛选 |
| is_public | bool | 否 | 是否公共题库 |
| keyword | string | 否 | 关键词搜索 |
| author_id | string | 否 | 按作者筛选 |

**响应：**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "list": [
      {
        "id": "1780000000510001",
        "title": "重入攻击：银行合约",
        "category": "contract",
        "category_text": "智能合约安全",
        "difficulty": 2,
        "difficulty_text": "Easy",
        "base_score": 300,
        "flag_type": 3,
        "flag_type_text": "链上状态验证",
        "runtime_mode": 1,
        "runtime_mode_text": "独立链模式",
        "source_path": 3,
        "source_path_text": "完全自定义",
        "status": 3,
        "status_text": "已通过",
        "is_public": true,
        "usage_count": 5,
        "author": {
          "id": "1780000000000010",
          "name": "张教授"
        },
        "created_at": "2026-04-08T11:00:00Z"
      }
    ],
    "pagination": {
      "page": 1,
      "page_size": 20,
      "total": 35,
      "total_pages": 2
    }
  }
}
```

> 教师默认只看到自己创建的题目和公共题库中已通过的题目。超级管理员可看到全部题目。

---

### 2.8 GET /api/v1/ctf/challenges/:id — 题目详情

**权限：** 超级管理员 / 题目作者

**响应：**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "id": "1780000000510001",
    "title": "重入攻击：银行合约",
    "description": "# 题目描述\n\n...",
    "category": "contract",
    "category_text": "智能合约安全",
    "difficulty": 2,
    "difficulty_text": "Easy",
    "base_score": 300,
    "flag_type": 3,
    "flag_type_text": "链上状态验证",
    "runtime_mode": 1,
    "runtime_mode_text": "独立链模式",
    "chain_config": {
      "chain_type": "evm",
      "chain_version": "london",
      "block_number": 0,
      "accounts": [
        {"name": "deployer", "balance": "100 ether"},
        {"name": "attacker", "balance": "10 ether"}
      ]
    },
    "setup_transactions": [
      {
        "from": "deployer",
        "to": "VulnerableBank",
        "function": "deposit",
        "args": [],
        "value": "10 ether"
      }
    ],
    "source_path": 3,
    "source_path_text": "完全自定义",
    "swc_id": null,
    "template_id": null,
    "environment_config": null,
    "attachment_urls": [],
    "status": 3,
    "status_text": "已通过",
    "is_public": true,
    "usage_count": 5,
    "contracts": [
      {
        "id": "1780000000511001",
        "name": "VulnerableBank",
        "source_code": "// SPDX-License-Identifier: MIT\npragma solidity ^0.8.0;\n\ncontract VulnerableBank {\n    mapping(address => uint256) public balances;\n    \n    function deposit() external payable {\n        balances[msg.sender] += msg.value;\n    }\n    \n    function withdraw() external {\n        uint256 bal = balances[msg.sender];\n        require(bal > 0);\n        (bool sent, ) = msg.sender.call{value: bal}(\"\");\n        require(sent, \"Failed to send Ether\");\n        balances[msg.sender] = 0;\n    }\n}",
        "deploy_order": 1
      }
    ],
    "assertions": [
      {
        "id": "1780000000512001",
        "assertion_type": "balance_check",
        "target": "VulnerableBank",
        "operator": "lt",
        "expected_value": "1 ether",
        "description": "合约ETH余额被掏空",
        "sort_order": 1
      }
    ],
    "latest_verification": {
      "id": "1780000000513001",
      "status": 2,
      "status_text": "通过",
      "completed_at": "2026-04-08T12:00:00Z"
    },
    "author": {
      "id": "1780000000000010",
      "name": "张教授"
    },
    "created_at": "2026-04-08T11:00:00Z",
    "updated_at": "2026-04-08T12:30:00Z"
  }
}
```

> 参赛选手通过竞赛题目接口（2.17）查看题目信息，不能直接访问此接口。参赛选手看到的题目不含合约源码、断言详情等敏感信息。

---

### 2.9 POST /api/v1/ctf/challenges/:id/contracts — 添加合约

**权限：** 题目作者

**请求体：**

```json
{
  "name": "VulnerableBank",
  "source_code": "// SPDX-License-Identifier: MIT\npragma solidity ^0.8.0;\n\ncontract VulnerableBank { ... }",
  "abi": [{"inputs":[],"name":"deposit","outputs":[],"stateMutability":"payable","type":"function"}],
  "bytecode": "0x608060...",
  "constructor_args": [],
  "deploy_order": 1
}
```

**响应：**

```json
{
  "code": 200,
  "message": "添加成功",
  "data": {
    "id": "1780000000511001",
    "challenge_id": "1780000000510001",
    "name": "VulnerableBank",
    "deploy_order": 1
  }
}
```

**错误响应：**

| code | message | 场景 |
|------|---------|------|
| 40401 | 题目不存在 | challenge_id 无效 |
| 40001 | 题目必须处于草稿状态才能修改合约 | 题目已提交审核 |
| 40002 | 合约名称不能为空 | name 为空 |
| 40003 | 源码不能为空 | source_code 为空 |

---

### 2.10 POST /api/v1/ctf/challenges/:id/assertions — 添加断言

**权限：** 题目作者

**请求体：**

```json
{
  "assertion_type": "balance_check",
  "target": "VulnerableBank",
  "operator": "lt",
  "expected_value": "1 ether",
  "description": "合约ETH余额被掏空",
  "extra_params": null,
  "sort_order": 1
}
```

**响应：**

```json
{
  "code": 200,
  "message": "添加成功",
  "data": {
    "id": "1780000000512001",
    "challenge_id": "1780000000510001",
    "assertion_type": "balance_check",
    "target": "VulnerableBank",
    "operator": "lt",
    "expected_value": "1 ether",
    "sort_order": 1
  }
}
```

**错误响应：**

| code | message | 场景 |
|------|---------|------|
| 40401 | 题目不存在 | challenge_id 无效 |
| 40001 | 断言类型无效 | assertion_type 不在 7 种类型中 |
| 40002 | 比较运算符无效 | operator 不在有效值中 |
| 40003 | 题目必须处于草稿状态才能修改断言 | 题目已提交审核 |

---

### 2.11 GET /api/v1/ctf/swc-registry — SWC Registry 列表

**权限：** 教师

**查询参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| keyword | string | 否 | 关键词搜索（SWC编号或名称） |

**响应：**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "list": [
      {
        "swc_id": "SWC-107",
        "title": "Reentrancy",
        "description": "One of the major dangers of calling external contracts is that they can take over the control flow...",
        "severity": "High",
        "has_example": true,
        "suggested_difficulty": 2
      },
      {
        "swc_id": "SWC-101",
        "title": "Integer Overflow and Underflow",
        "description": "An overflow/underflow happens when an arithmetic operation reaches the maximum or minimum size of a type...",
        "severity": "High",
        "has_example": true,
        "suggested_difficulty": 2
      }
    ]
  }
}
```

---

### 2.12 POST /api/v1/ctf/challenges/import-swc — 从SWC导入生成题目

**权限：** 教师

**请求体：**

```json
{
  "swc_id": "SWC-107",
  "title": "重入攻击入门",
  "difficulty": 2,
  "base_score": 300
}
```

**响应：**

```json
{
  "code": 200,
  "message": "导入成功",
  "data": {
    "id": "1780000000510002",
    "title": "重入攻击入门",
    "category": "contract",
    "difficulty": 2,
    "difficulty_text": "Easy",
    "base_score": 300,
    "flag_type": 3,
    "flag_type_text": "链上状态验证",
    "source_path": 1,
    "source_path_text": "SWC导入",
    "swc_id": "SWC-107",
    "status": 1,
    "status_text": "草稿",
    "contracts_generated": 1,
    "assertions_generated": 1
  }
}
```

> 平台自动根据 SWC 示例代码生成 contracts 和 assertions，教师可后续修改调整。
> SWC 导入默认生成 `runtime_mode=1` 的独立链草稿题；如需复现依赖真实历史状态的案例，教师后续改为 Fork 模式并补充 setup_transactions。

**错误响应：**

| code | message | 场景 |
|------|---------|------|
| 40001 | SWC编号无效 | swc_id 不存在 |
| 40002 | 该SWC条目无示例代码 | has_example=false |

---

### 2.13 GET /api/v1/ctf/challenge-templates — 参数化模板列表

**权限：** 教师

**查询参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| vulnerability_type | string | 否 | 漏洞类型筛选 |
| keyword | string | 否 | 关键词搜索 |

**响应：**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "list": [
      {
        "id": "1780000000520001",
        "name": "重入攻击模板",
        "code": "reentrancy-template",
        "vulnerability_type": "reentrancy",
        "description": "基于真实重入攻击事件的参数化模板，支持单函数/跨函数/跨合约三种变体",
        "difficulty_range": {"min": 2, "max": 4},
        "variant_count": 3,
        "usage_count": 12
      }
    ]
  }
}
```

---

### 2.14 GET /api/v1/ctf/challenge-templates/:id — 模板详情

**权限：** 教师

**响应：**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "id": "1780000000520001",
    "name": "重入攻击模板",
    "code": "reentrancy-template",
    "description": "基于真实重入攻击事件的参数化模板...",
    "vulnerability_type": "reentrancy",
    "difficulty_range": {"min": 2, "max": 4},
    "parameters": {
      "params": [
        {
          "key": "initial_deposit",
          "label": "初始存款金额",
          "type": "string",
          "default": "10 ether",
          "options": ["1 ether", "5 ether", "10 ether", "100 ether"]
        },
        {
          "key": "variant",
          "label": "重入变体",
          "type": "enum",
          "default": "single_function",
          "options": ["single_function", "cross_function", "cross_contract"]
        },
        {
          "key": "has_guard",
          "label": "是否有部分防护",
          "type": "boolean",
          "default": false
        }
      ]
    },
    "variants": [
      {
        "name": "单函数重入",
        "params": {"initial_deposit": "10 ether", "variant": "single_function", "has_guard": false},
        "suggested_difficulty": 2
      },
      {
        "name": "跨函数重入",
        "params": {"initial_deposit": "10 ether", "variant": "cross_function", "has_guard": false},
        "suggested_difficulty": 3
      },
      {
        "name": "跨合约重入（带部分防护）",
        "params": {"initial_deposit": "50 ether", "variant": "cross_contract", "has_guard": true},
        "suggested_difficulty": 4
      }
    ],
    "reference_events": [
      {"name": "The DAO Hack", "date": "2016-06-17", "loss": "$60M"},
      {"name": "Cream Finance", "date": "2021-08-30", "loss": "$18.8M"}
    ],
    "usage_count": 12
  }
}
```

---

### 2.15 POST /api/v1/ctf/challenges/generate-from-template — 从模板生成题目

**权限：** 教师

**请求体：**

```json
{
  "template_id": "1780000000520001",
  "title": "跨函数重入攻击",
  "difficulty": 3,
  "base_score": 500,
  "template_params": {
    "initial_deposit": "10 ether",
    "variant": "cross_function",
    "has_guard": false
  }
}
```

**响应：**

```json
{
  "code": 200,
  "message": "生成成功",
  "data": {
    "id": "1780000000510003",
    "title": "跨函数重入攻击",
    "category": "contract",
    "difficulty": 3,
    "difficulty_text": "Medium",
    "base_score": 500,
    "flag_type": 3,
    "flag_type_text": "链上状态验证",
    "runtime_mode": 1,
    "runtime_mode_text": "独立链模式",
    "source_path": 2,
    "source_path_text": "参数化模板",
    "template_id": "1780000000520001",
    "status": 1,
    "status_text": "草稿",
    "contracts_generated": 1,
    "assertions_generated": 2
  }
}
```

**错误响应：**

| code | message | 场景 |
|------|---------|------|
| 40401 | 模板不存在 | template_id 无效 |
| 40001 | 难度超出模板适用范围 | difficulty 不在 difficulty_range 内 |
| 40002 | 模板参数不完整 | 必填参数缺失 |

---

### 2.16 POST /api/v1/ctf/challenges/:id/verify — 发起预验证

**权限：** 题目作者

**请求体：**

```json
{
  "poc_content": "const { ethers } = require('ethers');\n\nasync function exploit(provider, bankAddress) {\n  // 部署攻击合约并执行重入攻击...\n}",
  "poc_language": "javascript"
}
```

**响应：**

```json
{
  "code": 200,
  "message": "预验证已启动",
  "data": {
    "verification_id": "1780000000513001",
    "status": 1,
    "status_text": "进行中",
    "challenge_id": "1780000000510001",
    "started_at": "2026-04-08T12:00:00Z"
  }
}
```

> 预验证为异步任务，通过轮询 GET /api/v1/ctf/challenge-verifications/:id 获取进度。6步流程：部署测试环境 → 提交PoC → 正向验证 → 反向验证 → 通过/失败。
> Fork 模式下，“部署测试环境”表示创建固定历史区块的临时 Fork 链；“反向验证”表示重置回同一 Fork 快照后重新执行断言。

**错误响应：**

| code | message | 场景 |
|------|---------|------|
| 40401 | 题目不存在 | challenge_id 无效 |
| 40001 | 链上验证题目才需要预验证 | flag_type 不是 3 |
| 40002 | 题目至少需要1个合约 | contracts 为空 |
| 40003 | 题目至少需要1个断言 | assertions 为空 |
| 40004 | PoC内容不能为空 | poc_content 为空 |
| 40901 | 存在进行中的预验证 | 上一次预验证尚未完成 |

---

### 2.17 GET /api/v1/ctf/challenge-verifications/:id — 预验证详情

**权限：** 题目作者

**响应：**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "id": "1780000000513001",
    "challenge_id": "1780000000510001",
    "status": 2,
    "status_text": "通过",
    "poc_language": "javascript",
    "step_results": [
      {
        "step": 1,
        "name": "部署测试环境",
        "status": "passed",
        "detail": "链节点启动成功，合约部署完成",
        "duration_ms": 5200
      },
      {
        "step": 2,
        "name": "提交官方PoC",
        "status": "passed",
        "detail": "PoC脚本已接收",
        "duration_ms": 100
      },
      {
        "step": 3,
        "name": "正向验证",
        "status": "passed",
        "detail": "执行PoC后所有断言通过",
        "assertions": [
          {"type": "balance_check", "passed": true, "actual": "0 ether", "expected": "< 1 ether"}
        ],
        "duration_ms": 3100
      },
      {
        "step": 4,
        "name": "反向验证",
        "status": "passed",
        "detail": "未执行PoC时所有断言失败（符合预期）",
        "assertions": [
          {"type": "balance_check", "passed": false, "actual": "10 ether", "expected": "< 1 ether"}
        ],
        "duration_ms": 2800
      },
      {
        "step": 5,
        "name": "验证通过",
        "status": "passed",
        "detail": "题目预验证全部通过，可提交审核"
      }
    ],
    "environment_id": "ctf-verify-1780000000513001",
    "started_at": "2026-04-08T12:00:00Z",
    "completed_at": "2026-04-08T12:00:12Z"
  }
}
```

---

### 2.18 POST /api/v1/ctf/challenges/:id/review — 审核题目

**权限：** 超级管理员

**请求体：**

```json
{
  "action": 1,
  "comment": "题目设计合理，预验证通过，审核通过"
}
```

| action 值 | 说明 |
|-----------|------|
| 1 | 通过（题目进入公共题库） |
| 2 | 拒绝（返回修改） |

**响应：**

```json
{
  "code": 200,
  "message": "审核完成",
  "data": {
    "challenge_id": "1780000000510001",
    "status": 3,
    "status_text": "已通过",
    "review_id": "1780000000514001"
  }
}
```

**错误响应：**

| code | message | 场景 |
|------|---------|------|
| 40401 | 题目不存在 | challenge_id 无效 |
| 40001 | action参数无效 | action 不是 1 或 2 |
| 40002 | 题目不在待审核状态 | 题目状态不是 2(待审核) |
| 40003 | 链上验证题目必须有通过的预验证记录 | 无通过的预验证 |

---

### 2.19 POST /api/v1/ctf/competitions/:id/challenges — 添加题目到竞赛

**权限：** 竞赛创建者

**请求体：**

```json
{
  "challenge_ids": [
    "1780000000510001",
    "1780000000510002",
    "1780000000510003"
  ]
}
```

**响应：**

```json
{
  "code": 200,
  "message": "添加成功",
  "data": {
    "added_count": 3,
    "competition_id": "1780000000500001",
    "challenges": [
      {"id": "1780000000510001", "title": "重入攻击：银行合约", "sort_order": 1},
      {"id": "1780000000510002", "title": "重入攻击入门", "sort_order": 2},
      {"id": "1780000000510003", "title": "跨函数重入攻击", "sort_order": 3}
    ]
  }
}
```

**错误响应：**

| code | message | 场景 |
|------|---------|------|
| 40401 | 竞赛不存在 | competition_id 无效 |
| 40001 | 竞赛必须处于草稿状态才能添加题目 | 竞赛已发布 |
| 40002 | 题目未通过审核 | 存在未审核通过的题目 |
| 40901 | 题目已在竞赛中 | 重复添加 |

---

### 2.20 GET /api/v1/ctf/competitions/:id/challenges — 竞赛题目列表

**权限：** 竞赛创建者 / 参赛选手

**响应（参赛选手视角，竞赛进行中）：**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "list": [
      {
        "id": "1780000000530001",
        "challenge": {
          "id": "1780000000510001",
          "title": "重入攻击：银行合约",
          "description": "# 题目描述\n\n一个存在重入漏洞的银行合约...",
          "category": "contract",
          "category_text": "智能合约安全",
          "difficulty": 2,
          "difficulty_text": "Easy",
          "flag_type": 3,
          "flag_type_text": "链上状态验证",
          "attachment_urls": []
        },
        "current_score": 285,
        "base_score": 300,
        "solve_count": 3,
        "first_blood_team": {
          "id": "1780000000540001",
          "name": "BlockSec"
        },
        "first_blood_at": "2026-04-20T09:45:00Z",
        "my_team_solved": false,
        "my_team_environment": null,
        "sort_order": 1
      }
    ]
  }
}
```

> 参赛选手视角不包含合约源码、断言详情、Flag值等敏感信息。`my_team_solved` 和 `my_team_environment` 根据当前用户所在团队自动填充。
> 竞赛创建者视角包含完整信息。

---

### 2.21 POST /api/v1/ctf/competitions/:id/teams — 创建团队

**权限：** 学生

**请求体：**

```json
{
  "name": "BlockSec"
}
```

**响应：**

```json
{
  "code": 200,
  "message": "创建成功",
  "data": {
    "id": "1780000000540001",
    "competition_id": "1780000000500001",
    "name": "BlockSec",
    "captain_id": "1780000000000020",
    "invite_code": "CTF-A3B7K9",
    "status": 1,
    "status_text": "组建中",
    "members": [
      {
        "student_id": "1780000000000020",
        "name": "李同学",
        "role": 1,
        "role_text": "队长",
        "joined_at": "2026-04-12T10:00:00Z"
      }
    ]
  }
}
```

> 个人赛（team_mode=1）时系统自动创建单人团队，无需调用此接口。
> 自由组队（team_mode=2）时队长创建团队并获得邀请码。

**错误响应：**

| code | message | 场景 |
|------|---------|------|
| 40401 | 竞赛不存在 | competition_id 无效 |
| 40001 | 竞赛不在报名阶段 | 竞赛状态不是 2(报名中) |
| 40002 | 个人赛无需创建团队 | team_mode=1 |
| 40901 | 您已在该竞赛的其他团队中 | 已加入其他团队 |

---

### 2.22 POST /api/v1/ctf/teams/join — 通过邀请码加入团队

**权限：** 学生

**请求体：**

```json
{
  "invite_code": "CTF-A3B7K9"
}
```

**响应：**

```json
{
  "code": 200,
  "message": "加入成功",
  "data": {
    "team_id": "1780000000540001",
    "team_name": "BlockSec",
    "competition_id": "1780000000500001",
    "role": 2,
    "role_text": "队员",
    "current_members": 3,
    "max_team_size": 4
  }
}
```

**错误响应：**

| code | message | 场景 |
|------|---------|------|
| 40401 | 邀请码无效 | invite_code 不存在 |
| 40001 | 竞赛不在报名阶段 | 竞赛状态不是 2(报名中) |
| 40002 | 团队已满 | 成员数已达 max_team_size |
| 40901 | 您已在该竞赛的其他团队中 | 已加入其他团队 |
| 40003 | 团队已锁定 | 竞赛已开始，团队不可变更 |

---

### 2.23 POST /api/v1/ctf/competitions/:id/register — 报名竞赛

**权限：** 队长 / 个人

**请求体：**

```json
{
  "team_id": "1780000000540001"
}
```

> 个人赛时 team_id 可省略，系统自动创建单人团队并报名。

**响应：**

```json
{
  "code": 200,
  "message": "报名成功",
  "data": {
    "registration_id": "1780000000550001",
    "competition_id": "1780000000500001",
    "team_id": "1780000000540001",
    "team_name": "BlockSec",
    "status": 1,
    "status_text": "已报名",
    "registered_at": "2026-04-12T10:30:00Z"
  }
}
```

**错误响应：**

| code | message | 场景 |
|------|---------|------|
| 40401 | 竞赛不存在 | competition_id 无效 |
| 40001 | 竞赛不在报名阶段 | 竞赛状态不是 2(报名中) |
| 40002 | 只有队长可以报名 | 非队长尝试报名 |
| 40003 | 团队人数不满足最低要求 | 成员数 < min_team_size |
| 40004 | 报名队伍数已达上限 | 已达 max_teams |
| 40901 | 该团队已报名 | 重复报名 |

---

### 2.24 POST /api/v1/ctf/competitions/:id/submissions — 提交Flag/攻击交易

**权限：** 参赛选手

**请求体（静态/动态Flag）：**

```json
{
  "challenge_id": "1780000000510001",
  "submission_type": 1,
  "content": "flag{r33ntrancy_1s_d4ng3r0us}"
}
```

**请求体（攻击交易 — 链上验证）：**

```json
{
  "challenge_id": "1780000000510001",
  "submission_type": 3,
  "content": "0x608060405234801561001057600080fd5b5060..."
}
```

> `submission_type=3`（攻击交易）时，`content` 为攻击合约字节码或交易数据，平台在选手的题目环境中执行并验证断言。

**响应（正确提交）：**

```json
{
  "code": 200,
  "message": "提交正确",
  "data": {
    "submission_id": "1780000000560001",
    "is_correct": true,
    "score_awarded": 285,
    "is_first_blood": false,
    "challenge_new_score": 270,
    "team_total_score": 855,
    "team_rank": 3,
    "assertion_results": {
      "all_passed": true,
      "results": [
        {
          "type": "balance_check",
          "target": "VulnerableBank",
          "expected": "< 1 ether",
          "actual": "0 ether",
          "passed": true
        }
      ],
      "execution_time_ms": 1200,
      "tx_hash": "0xabc..."
    }
  }
}
```

**响应（错误提交）：**

```json
{
  "code": 200,
  "message": "提交错误",
  "data": {
    "submission_id": "1780000000560002",
    "is_correct": false,
    "error_message": "Flag不正确",
    "remaining_attempts": 3,
    "cooldown_until": null
  }
}
```

> 提交错误不返回 HTTP 错误码，而是在 data 中标记 `is_correct: false`。

**错误响应：**

| code | message | 场景 |
|------|---------|------|
| 40401 | 竞赛不存在 | competition_id 无效 |
| 40402 | 题目不存在 | challenge_id 不在竞赛中 |
| 40001 | 竞赛不在进行中 | 竞赛状态不是 3(进行中) |
| 40002 | 您的团队已解出此题 | 重复提交已解出的题目 |
| 42901 | 提交过于频繁，请稍后再试 | 触发限流（5次/分钟） |
| 42902 | 提交已冷却，请等待N分钟 | 触发冷却期（连续10次失败） |
| 40003 | 题目环境未启动 | 链上验证题目但环境未启动 |

---

### 2.25 POST /api/v1/ctf/competitions/:id/ad-groups — 创建攻防赛分组

**权限：** 竞赛创建者

**请求体：**

```json
{
  "group_name": "A组",
  "team_ids": [
    "1780000000540001",
    "1780000000540002",
    "1780000000540003",
    "1780000000540004"
  ]
}
```

**响应：**

```json
{
  "code": 200,
  "message": "创建成功",
  "data": {
    "id": "1780000000570001",
    "competition_id": "1780000000500002",
    "group_name": "A组",
    "namespace": "ctf-ad-1780000000500002-1780000000570001",
    "status": 1,
    "status_text": "准备中",
    "teams": [
      {"id": "1780000000540001", "name": "BlockSec"},
      {"id": "1780000000540002", "name": "ChainGuard"},
      {"id": "1780000000540003", "name": "SmartAudit"},
      {"id": "1780000000540004", "name": "DeFiHunter"}
    ]
  }
}
```

**错误响应：**

| code | message | 场景 |
|------|---------|------|
| 40401 | 竞赛不存在 | competition_id 无效 |
| 40001 | 仅攻防对抗赛需要分组 | competition_type 不是 2 |
| 40002 | 竞赛必须处于草稿或报名状态 | 竞赛已开始 |
| 40003 | 分组队伍数超过上限 | 超过 ad_config.max_teams_per_group |

---

### 2.26 POST /api/v1/ctf/competitions/:id/ad-groups/auto-assign — 自动分组

**权限：** 竞赛创建者

**请求体：**

```json
{
  "teams_per_group": 4
}
```

**响应：**

```json
{
  "code": 200,
  "message": "自动分组完成",
  "data": {
    "groups": [
      {
        "id": "1780000000570001",
        "group_name": "A组",
        "team_count": 4
      },
      {
        "id": "1780000000570002",
        "group_name": "B组",
        "team_count": 4
      },
      {
        "id": "1780000000570003",
        "group_name": "C组",
        "team_count": 3
      }
    ],
    "total_teams": 11,
    "total_groups": 3
  }
}
```

---

### 2.27 GET /api/v1/ctf/ad-groups/:id/current-round — 当前回合状态

**权限：** 分组内选手

**响应：**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "group_id": "1780000000570001",
    "round_id": "1780000000580003",
    "round_number": 3,
    "total_rounds": 10,
    "phase": 1,
    "phase_text": "攻击阶段",
    "phase_start_at": "2026-04-20T10:30:00Z",
    "phase_end_at": "2026-04-20T10:40:00Z",
    "remaining_seconds": 342,
    "my_team": {
      "id": "1780000000540001",
      "name": "BlockSec",
      "token_balance": 10350,
      "rank": 2
    }
  }
}
```

---

### 2.28 POST /api/v1/ctf/ad-rounds/:id/attacks — 提交攻击交易

**权限：** 参赛选手（攻击阶段）

**请求体：**

```json
{
  "target_team_id": "1780000000540002",
  "challenge_id": "1780000000510001",
  "attack_tx_data": "0x608060405234801561001057600080fd5b5060..."
}
```

**响应（攻击成功）：**

```json
{
  "code": 200,
  "message": "攻击成功",
  "data": {
    "attack_id": "1780000000590001",
    "is_successful": true,
    "token_reward": 400,
    "is_first_blood": true,
    "exploit_count": 1,
    "assertion_results": {
      "all_passed": true,
      "results": [
        {
          "type": "balance_check",
          "target": "VulnerableBank",
          "expected": "< 1 ether",
          "actual": "0 ether",
          "passed": true
        }
      ]
    },
    "attacker_balance_after": 10900,
    "target_balance_after": 9600
  }
}
```

**响应（攻击失败）：**

```json
{
  "code": 200,
  "message": "攻击失败",
  "data": {
    "attack_id": "1780000000590002",
    "is_successful": false,
    "error_message": "断言验证未通过：合约余额未被掏空",
    "assertion_results": {
      "all_passed": false,
      "results": [
        {
          "type": "balance_check",
          "target": "VulnerableBank",
          "expected": "< 1 ether",
          "actual": "10 ether",
          "passed": false
        }
      ]
    }
  }
}
```

**错误响应：**

| code | message | 场景 |
|------|---------|------|
| 40401 | 回合不存在 | round_id 无效 |
| 40001 | 当前不在攻击阶段 | phase 不是 1(攻击阶段) |
| 40002 | 不能攻击自己的队伍 | target_team_id 是自己的团队 |
| 40003 | 目标队伍不在本分组 | target_team_id 不在同一分组 |
| 40004 | 目标漏洞已被修复 | 目标队伍已成功提交该漏洞的补丁 |

---

### 2.29 POST /api/v1/ctf/ad-rounds/:id/defenses — 提交补丁合约

**权限：** 参赛选手（防守阶段）

**请求体：**

```json
{
  "challenge_id": "1780000000510001",
  "patch_source_code": "// SPDX-License-Identifier: MIT\npragma solidity ^0.8.0;\n\ncontract VulnerableBank {\n    mapping(address => uint256) public balances;\n    bool private locked;\n    \n    modifier noReentrant() {\n        require(!locked, \"No re-entrancy\");\n        locked = true;\n        _;\n        locked = false;\n    }\n    \n    function deposit() external payable {\n        balances[msg.sender] += msg.value;\n    }\n    \n    function withdraw() external noReentrant {\n        uint256 bal = balances[msg.sender];\n        require(bal > 0);\n        balances[msg.sender] = 0;\n        (bool sent, ) = msg.sender.call{value: bal}(\"\");\n        require(sent, \"Failed to send Ether\");\n    }\n}"
}
```

**响应（补丁接受）：**

```json
{
  "code": 200,
  "message": "补丁验证通过",
  "data": {
    "defense_id": "1780000000600001",
    "is_accepted": true,
    "functionality_passed": true,
    "vulnerability_fixed": true,
    "is_first_patch": true,
    "token_reward": 200,
    "team_balance_after": 10550
  }
}
```

**响应（补丁拒绝）：**

```json
{
  "code": 200,
  "message": "补丁验证未通过",
  "data": {
    "defense_id": "1780000000600002",
    "is_accepted": false,
    "functionality_passed": true,
    "vulnerability_fixed": false,
    "rejection_reason": "漏洞修复验证失败：官方PoC仍可成功执行"
  }
}
```

**错误响应：**

| code | message | 场景 |
|------|---------|------|
| 40401 | 回合不存在 | round_id 无效 |
| 40001 | 当前不在防守阶段 | phase 不是 2(防守阶段) |
| 40002 | 该漏洞已修复 | 本队已成功提交该漏洞的补丁 |
| 40003 | 补丁源码不能为空 | patch_source_code 为空 |

---

### 2.30 GET /api/v1/ctf/teams/:id/token-ledger — 团队Token流水

**权限：** 团队成员

**查询参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page | int | 否 | 页码，默认1 |
| page_size | int | 否 | 每页条数，默认20 |
| round_id | string | 否 | 按回合筛选 |
| change_type | int | 否 | 按变动类型筛选 |

**响应：**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "list": [
      {
        "id": "1780000000610001",
        "round_number": 3,
        "change_type": 2,
        "change_type_text": "攻击窃取",
        "amount": 400,
        "balance_after": 10900,
        "description": "攻击 ChainGuard 队伍的 VulnerableBank 漏洞",
        "related_attack_id": "1780000000590001",
        "created_at": "2026-04-20T10:35:00Z"
      },
      {
        "id": "1780000000610002",
        "round_number": 3,
        "change_type": 3,
        "change_type_text": "攻击奖励",
        "amount": 20,
        "balance_after": 10920,
        "description": "攻击奖励（窃取金额的5%）",
        "related_attack_id": "1780000000590001",
        "created_at": "2026-04-20T10:35:00Z"
      },
      {
        "id": "1780000000610003",
        "round_number": 3,
        "change_type": 7,
        "change_type_text": "First Blood奖励",
        "amount": 40,
        "balance_after": 10960,
        "description": "首次攻破 VulnerableBank 漏洞的额外奖励（10%）",
        "related_attack_id": "1780000000590001",
        "created_at": "2026-04-20T10:35:00Z"
      }
    ],
    "pagination": {
      "page": 1,
      "page_size": 20,
      "total": 15,
      "total_pages": 1
    }
  }
}
```

---

### 2.31 GET /api/v1/ctf/competitions/:id/leaderboard — 实时排行榜

**权限：** 全部角色

**查询参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| group_id | string | 否 | 攻防赛分组ID（攻防赛时必填） |
| top | int | 否 | 返回前N名，默认50 |

**响应（解题赛）：**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "competition_id": "1780000000500001",
    "competition_type": 1,
    "is_frozen": false,
    "frozen_at": "2026-04-20T16:00:00Z",
    "updated_at": "2026-04-20T14:30:00Z",
    "rankings": [
      {
        "rank": 1,
        "team_id": "1780000000540003",
        "team_name": "SmartAudit",
        "score": 1250,
        "solve_count": 5,
        "last_solve_at": "2026-04-20T13:20:00Z"
      },
      {
        "rank": 2,
        "team_id": "1780000000540001",
        "team_name": "BlockSec",
        "score": 1100,
        "solve_count": 4,
        "last_solve_at": "2026-04-20T14:10:00Z"
      },
      {
        "rank": 3,
        "team_id": "1780000000540002",
        "team_name": "ChainGuard",
        "score": 950,
        "solve_count": 4,
        "last_solve_at": "2026-04-20T14:25:00Z"
      }
    ]
  }
}
```

**响应（攻防赛）：**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "competition_id": "1780000000500002",
    "competition_type": 2,
    "group_id": "1780000000570001",
    "group_name": "A组",
    "current_round": 5,
    "total_rounds": 10,
    "is_frozen": false,
    "rankings": [
      {
        "rank": 1,
        "team_id": "1780000000540001",
        "team_name": "BlockSec",
        "token_balance": 12350,
        "attacks_successful": 8,
        "defenses_successful": 3,
        "patches_accepted": 2
      },
      {
        "rank": 2,
        "team_id": "1780000000540003",
        "team_name": "SmartAudit",
        "token_balance": 11200,
        "attacks_successful": 5,
        "defenses_successful": 4,
        "patches_accepted": 3
      }
    ]
  }
}
```

> 排行榜冻结期间（`is_frozen=true`），排名数据停止更新，显示冻结时刻的快照。竞赛结束后解冻显示最终排名。

---

### 2.32 GET /api/v1/ctf/competitions/:id/leaderboard/final — 最终排名

**权限：** 全部角色

**响应：**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "competition_id": "1780000000500001",
    "competition_type": 1,
    "ended_at": "2026-04-20T17:00:00Z",
    "rankings": [
      {
        "rank": 1,
        "team_id": "1780000000540003",
        "team_name": "SmartAudit",
        "score": 2450,
        "solve_count": 9,
        "last_solve_at": "2026-04-20T16:45:00Z",
        "members": [
          {"name": "王同学", "role_text": "队长"},
          {"name": "赵同学", "role_text": "队员"}
        ],
        "solved_challenges": [
          {"challenge_id": "1780000000510001", "title": "重入攻击：银行合约", "score": 285, "solved_at": "2026-04-20T09:45:00Z", "is_first_blood": true}
        ]
      }
    ],
    "total_teams": 78,
    "total_solves": 312
  }
}
```

**错误响应：**

| code | message | 场景 |
|------|---------|------|
| 40401 | 竞赛不存在 | competition_id 无效 |
| 40001 | 竞赛尚未结束 | 竞赛状态不是 4(已结束) 或 5(已归档) |

---

### 2.33 POST /api/v1/ctf/competitions/:id/announcements — 发布公告

**权限：** 竞赛创建者

**请求体：**

```json
{
  "title": "题目勘误：重入攻击题目补充说明",
  "content": "## 补充说明\n\n重入攻击题目中的初始存款为10 ETH，请注意...",
  "announcement_type": 2,
  "challenge_id": "1780000000510001"
}
```

| announcement_type 值 | 说明 |
|----------------------|------|
| 1 | 信息通知 |
| 2 | 题目勘误 |
| 3 | 规则说明 |

> `challenge_id` 为可选字段，填写时为单题公告，不填时为全局公告。

**响应：**

```json
{
  "code": 200,
  "message": "发布成功",
  "data": {
    "id": "1780000000620001",
    "title": "题目勘误：重入攻击题目补充说明",
    "announcement_type": 2,
    "announcement_type_text": "题目勘误",
    "challenge_id": "1780000000510001",
    "published_by_name": "平台管理员",
    "created_at": "2026-04-20T11:00:00Z"
  }
}
```

> 公告发布后通过 WebSocket 实时推送到所有参赛选手。

---

### 2.34 GET /api/v1/ctf/competitions/:id/announcements — 公告列表

**权限：** 全部角色

**响应：**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "list": [
      {
        "id": "1780000000620001",
        "title": "题目勘误：重入攻击题目补充说明",
        "content": "## 补充说明\n\n...",
        "announcement_type": 2,
        "announcement_type_text": "题目勘误",
        "challenge_id": "1780000000510001",
        "challenge_title": "重入攻击：银行合约",
        "published_by_name": "平台管理员",
        "created_at": "2026-04-20T11:00:00Z"
      }
    ]
  }
}
```

---

### 2.35 POST /api/v1/ctf/competitions/:comp_id/challenges/:challenge_id/environment — 启动题目环境

**权限：** 参赛选手

**请求体：** 无

**响应：**

```json
{
  "code": 200,
  "message": "环境创建中",
  "data": {
    "environment_id": "1780000000630001",
    "competition_id": "1780000000500001",
    "challenge_id": "1780000000510001",
    "team_id": "1780000000540001",
    "namespace": "ctf-1780000000500001-1780000000540001-1780000000510001",
    "status": 1,
    "status_text": "创建中",
    "chain_rpc_url": null,
    "created_at": "2026-04-20T10:00:00Z"
  }
}
```

> 环境创建为异步操作，选手通过轮询 GET /api/v1/ctf/challenge-environments/:id 获取状态更新。环境就绪后 `chain_rpc_url` 会填充链节点RPC地址。

**错误响应：**

| code | message | 场景 |
|------|---------|------|
| 40401 | 竞赛不存在 | competition_id 无效 |
| 40402 | 题目不在竞赛中 | challenge_id 不在竞赛中 |
| 40001 | 竞赛不在进行中 | 竞赛状态不是 3(进行中) |
| 40901 | 该题目环境已存在 | 已有运行中的环境 |
| 40002 | 竞赛资源配额不足 | 资源配额已用尽 |

---

### 2.36 GET /api/v1/ctf/challenge-environments/:id — 环境详情

**权限：** 参赛选手

**响应：**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "id": "1780000000630001",
    "competition_id": "1780000000500001",
    "challenge_id": "1780000000510001",
    "team_id": "1780000000540001",
    "namespace": "ctf-1780000000500001-1780000000540001-1780000000510001",
    "chain_rpc_url": "http://10.0.5.12:8545",
    "container_status": {
      "chain_node": {"status": "running", "image": "ganache:latest"},
      "tools": {"status": "running", "image": "ctf-blockchain:latest"}
    },
    "status": 2,
    "status_text": "运行中",
    "started_at": "2026-04-20T10:00:30Z",
    "created_at": "2026-04-20T10:00:00Z"
  }
}
```

---

### 2.37 POST /api/v1/ctf/challenge-environments/:id/reset — 重置题目环境

**权限：** 参赛选手

**请求体：** 无

**响应：**

```json
{
  "code": 200,
  "message": "环境重置中",
  "data": {
    "environment_id": "1780000000630001",
    "status": 1,
    "status_text": "创建中"
  }
}
```

> 重置会销毁当前环境并重新创建，合约和链状态恢复到初始状态。

---

### 2.38 PUT /api/v1/ctf/competitions/:id/resource-quota — 设置竞赛资源配额

**权限：** 超级管理员

**请求体：**

```json
{
  "max_cpu": "32",
  "max_memory": "64Gi",
  "max_storage": "100Gi",
  "max_namespaces": 200
}
```

**响应：**

```json
{
  "code": 200,
  "message": "设置成功",
  "data": {
    "competition_id": "1780000000500001",
    "max_cpu": "32",
    "max_memory": "64Gi",
    "max_storage": "100Gi",
    "max_namespaces": 200,
    "used_cpu": "8.5",
    "used_memory": "17Gi",
    "used_storage": "25Gi",
    "current_namespaces": 45
  }
}
```

---

### 2.39 GET /api/v1/ctf/competitions/:id/monitor — 竞赛运行监控

**权限：** 竞赛创建者 / 超级管理员

**响应：**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "competition_id": "1780000000500001",
    "competition_type": 1,
    "status": 3,
    "status_text": "进行中",
    "overview": {
      "registered_teams": 78,
      "active_teams": 65,
      "total_submissions": 456,
      "correct_submissions": 123,
      "total_environments": 89,
      "running_environments": 67
    },
    "resource_usage": {
      "cpu_used": "8.5",
      "cpu_max": "32",
      "memory_used": "17Gi",
      "memory_max": "64Gi",
      "namespaces_used": 67,
      "namespaces_max": 200
    },
    "challenge_stats": [
      {
        "challenge_id": "1780000000510001",
        "title": "重入攻击：银行合约",
        "category": "contract",
        "solve_count": 15,
        "attempt_count": 89,
        "solve_rate": 0.192,
        "current_score": 220,
        "environments_running": 12
      }
    ],
    "recent_submissions": [
      {
        "team_name": "BlockSec",
        "challenge_title": "重入攻击：银行合约",
        "is_correct": true,
        "submitted_at": "2026-04-20T14:30:00Z"
      }
    ]
  }
}
```

---

### 2.40 GET /api/v1/ctf/competitions/:id/statistics — 竞赛统计数据

**权限：** 竞赛创建者 / 超级管理员

**响应：**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "competition_id": "1780000000500001",
    "summary": {
      "total_teams": 78,
      "total_participants": 245,
      "total_submissions": 1234,
      "total_correct": 456,
      "overall_solve_rate": 0.369,
      "average_score": 780,
      "highest_score": 2450,
      "lowest_score": 0
    },
    "challenge_statistics": [
      {
        "challenge_id": "1780000000510001",
        "title": "重入攻击：银行合约",
        "category": "contract",
        "difficulty": 2,
        "difficulty_text": "Easy",
        "solve_count": 45,
        "attempt_count": 234,
        "solve_rate": 0.577,
        "first_blood_team": "SmartAudit",
        "first_blood_time_minutes": 12,
        "average_solve_time_minutes": 35
      }
    ],
    "score_distribution": {
      "ranges": [
        {"label": "0-500", "count": 20},
        {"label": "500-1000", "count": 25},
        {"label": "1000-1500", "count": 18},
        {"label": "1500-2000", "count": 10},
        {"label": "2000+", "count": 5}
      ]
    },
    "timeline": {
      "submissions_per_hour": [
        {"hour": "09:00", "count": 45},
        {"hour": "10:00", "count": 120},
        {"hour": "11:00", "count": 98}
      ]
    }
  }
}
```

---

### 2.41 GET /api/v1/ctf/teams/:id/chain — 队伍链信息

**权限：** 团队成员

**响应：**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "id": "1780000000640001",
    "competition_id": "1780000000500002",
    "group_id": "1780000000570001",
    "team_id": "1780000000540001",
    "chain_rpc_url": "http://10.0.6.10:8545",
    "chain_ws_url": "ws://10.0.6.10:8546",
    "deployed_contracts": [
      {
        "challenge_id": "1780000000510001",
        "contract_name": "VulnerableBank",
        "address": "0x5FbDB2315678afecb367f032d93F642f64180aa3",
        "patch_version": 0,
        "is_patched": false
      },
      {
        "challenge_id": "1780000000510004",
        "contract_name": "TokenSwap",
        "address": "0xe7f1725E7734CE288F8367e1Bb143E90bb3F0512",
        "patch_version": 1,
        "is_patched": true
      }
    ],
    "current_patch_version": 1,
    "status": 2,
    "status_text": "运行中"
  }
}
```

---

### 2.42 GET /api/v1/ctf/admin/competitions/overview — 全平台竞赛概览

**权限：** 超级管理员

**响应：**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "total_competitions": 15,
    "running_competitions": 2,
    "upcoming_competitions": 3,
    "total_participants": 1200,
    "total_resource_usage": {
      "cpu_used": "45.5",
      "memory_used": "92Gi",
      "namespaces_active": 234
    },
    "running_competitions_list": [
      {
        "id": "1780000000500001",
        "title": "链镜杯·2026春季区块链安全挑战赛",
        "competition_type": 1,
        "competition_type_text": "解题赛",
        "status": 3,
        "status_text": "进行中",
        "teams": 78,
        "environments_running": 67,
        "start_at": "2026-04-20T09:00:00Z",
        "end_at": "2026-04-20T17:00:00Z"
      }
    ],
    "alerts": [
      {
        "type": "resource_warning",
        "message": "竞赛 '链镜杯' 资源使用率超过80%",
        "competition_id": "1780000000500001",
        "created_at": "2026-04-20T14:00:00Z"
      }
    ]
  }
}
```

---

### 2.43 GET /api/v1/ctf/competitions/:id/leaderboard/history — 排行榜历史快照

**权限：** 全部角色

**查询参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page | int | 否 | 页码，默认1 |
| page_size | int | 否 | 每页条数，默认20 |
| group_id | string | 否 | 攻防赛分组ID（攻防赛时可传） |

**响应：**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "list": [
      {
        "snapshot_at": "2026-04-20T14:30:00Z",
        "rankings": [
          {
            "rank": 1,
            "team_id": "1780000000540003",
            "team_name": "SmartAudit",
            "score": 1250,
            "solve_count": 5,
            "last_solve_at": "2026-04-20T13:20:00Z"
          },
          {
            "rank": 2,
            "team_id": "1780000000540001",
            "team_name": "BlockSec",
            "score": 1100,
            "solve_count": 4,
            "last_solve_at": "2026-04-20T14:10:00Z"
          }
        ]
      }
    ],
    "pagination": {
      "page": 1,
      "page_size": 20,
      "total": 12,
      "total_pages": 1
    }
  }
}
```

> 解题赛返回 `score / solve_count / last_solve_at`；攻防赛返回 `token_balance / attacks_successful / defenses_successful / patches_accepted`。快照数据来源于定时归档的排行榜快照表。

---

### 2.44 GET /api/v1/ctf/competitions/:id/environments — 竞赛环境资源列表

**权限：** 竞赛创建者 / 超级管理员

**查询参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page | int | 否 | 页码，默认1 |
| page_size | int | 否 | 每页条数，默认20 |
| status | int | 否 | 环境状态：1创建中 2运行中 3已停止 4异常 5已销毁 |
| challenge_id | string | 否 | 按题目筛选 |
| team_id | string | 否 | 按队伍筛选 |

**响应：**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "list": [
      {
        "id": "1780000000630001",
        "competition_id": "1780000000500001",
        "challenge_id": "1780000000510001",
        "challenge_title": "重入攻击：银行合约",
        "team_id": "1780000000540001",
        "team_name": "BlockSec",
        "namespace": "ctf-1780000000500001-1780000000540001-1780000000510001",
        "status": 2,
        "status_text": "运行中",
        "chain_rpc_url": "http://10.0.5.12:8545",
        "started_at": "2026-04-20T10:00:30Z",
        "created_at": "2026-04-20T10:00:00Z"
      }
    ],
    "pagination": {
      "page": 1,
      "page_size": 20,
      "total": 67,
      "total_pages": 4
    }
  }
}
```

---

### 2.45 POST /api/v1/ctf/challenge-environments/:id/force-destroy — 强制回收环境

**权限：** 超级管理员

**请求体：**

```json
{
  "reason": "竞赛已结束，统一回收残留环境"
}
```

**响应：**

```json
{
  "code": 200,
  "message": "强制回收成功",
  "data": {
    "environment_id": "1780000000630001",
    "status": 5,
    "status_text": "已销毁"
  }
}
```

> 该接口用于管理员绕过参赛者权限直接回收异常或残留环境，并记录审计原因。

---

## 三、WebSocket 接口

### 3.1 连接地址

```
ws://api.lianjing.com/api/v1/ctf/ws?token={jwt_token}&competition_id={competition_id}
```

> 连接时需携带 JWT Token 进行身份验证。连接成功后根据用户角色和竞赛类型自动订阅相关频道。

### 3.2 消息格式

**客户端 → 服务端：**

```json
{
  "type": "subscribe",
  "channel": "leaderboard",
  "params": {
    "competition_id": "1780000000500001"
  }
}
```

**服务端 → 客户端：**

```json
{
  "type": "message",
  "channel": "leaderboard",
  "data": { ... },
  "timestamp": "2026-04-20T14:30:00Z"
}
```

### 3.3 频道定义

#### 3.3.1 排行榜频道 — ctf:ws:leaderboard:{competition_id}

**推送时机：** 有新的正确提交时（冻结期间停止推送）

**推送数据：**

```json
{
  "type": "message",
  "channel": "leaderboard",
  "data": {
    "event": "rank_update",
    "competition_id": "1780000000500001",
    "is_frozen": false,
    "rankings": [
      {
        "rank": 1,
        "team_id": "1780000000540003",
        "team_name": "SmartAudit",
        "score": 1250,
        "solve_count": 5,
        "last_solve_at": "2026-04-20T13:20:00Z",
        "rank_change": 0
      },
      {
        "rank": 2,
        "team_id": "1780000000540001",
        "team_name": "BlockSec",
        "score": 1100,
        "solve_count": 4,
        "last_solve_at": "2026-04-20T14:10:00Z",
        "rank_change": 1
      }
    ],
    "trigger": {
      "team_name": "BlockSec",
      "challenge_title": "重入攻击：银行合约",
      "is_first_blood": false
    }
  },
  "timestamp": "2026-04-20T14:10:01Z"
}
```

#### 3.3.2 公告频道 — ctf:ws:announcement:{competition_id}

**推送时机：** 管理员发布新公告时

**推送数据：**

```json
{
  "type": "message",
  "channel": "announcement",
  "data": {
    "event": "new_announcement",
    "announcement": {
      "id": "1780000000620001",
      "title": "题目勘误：重入攻击题目补充说明",
      "content": "## 补充说明\n\n...",
      "announcement_type": 2,
      "announcement_type_text": "题目勘误",
      "challenge_id": "1780000000510001",
      "challenge_title": "重入攻击：银行合约",
      "published_by_name": "平台管理员"
    }
  },
  "timestamp": "2026-04-20T11:00:00Z"
}
```

#### 3.3.3 攻防赛回合状态频道 — ctf:ws:round:{competition_id}:{group_id}

**推送时机：** 回合阶段切换时

**推送数据：**

```json
{
  "type": "message",
  "channel": "round",
  "data": {
    "event": "phase_change",
    "group_id": "1780000000570001",
    "round_number": 3,
    "total_rounds": 10,
    "phase": 2,
    "phase_text": "防守阶段",
    "phase_start_at": "2026-04-20T10:40:00Z",
    "phase_end_at": "2026-04-20T10:50:00Z",
    "previous_phase_summary": {
      "phase": 1,
      "phase_text": "攻击阶段",
      "attacks_total": 12,
      "attacks_successful": 5
    }
  },
  "timestamp": "2026-04-20T10:40:00Z"
}
```

#### 3.3.4 攻防赛攻击事件频道 — ctf:ws:attacks:{competition_id}:{group_id}

**推送时机：** 有攻击提交时（成功和失败都推送）

**推送数据：**

```json
{
  "type": "message",
  "channel": "attacks",
  "data": {
    "event": "attack_result",
    "round_number": 3,
    "attacker_team": {
      "id": "1780000000540001",
      "name": "BlockSec"
    },
    "target_team": {
      "id": "1780000000540002",
      "name": "ChainGuard"
    },
    "challenge_title": "VulnerableBank",
    "is_successful": true,
    "is_first_blood": true,
    "token_reward": 400,
    "attacker_balance": 10900,
    "target_balance": 9600
  },
  "timestamp": "2026-04-20T10:35:00Z"
}
```

### 3.4 心跳机制

```json
// 客户端每30秒发送
{"type": "ping"}

// 服务端响应
{"type": "pong"}
```

> 60秒未收到客户端心跳，服务端主动断开连接。客户端断线后自动重连（指数退避，最大间隔30秒）。

---

*文档版本：v1.1*
*创建日期：2026-04-08*
*更新日期：2026-04-22*
*更新说明：v1.1 — 扩展创建题目、题目详情、模板/SWC导入和预验证接口示例，新增 runtime_mode、setup_transactions 与 Fork 模式预验证语义*
```
