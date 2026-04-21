# 通知与消息模块 — API接口设计

> 模块状态：✅ 已确认
> 文档版本：v1.0

---

## 一、接口总览

### API 前缀：`/api/v1/notifications`

| 分类 | 方法 | 路径 | 说明 | 权限 |
|------|------|------|------|------|
| **站内信** | GET | /inbox | 收件箱列表 | 登录用户 |
| | GET | /inbox/:id | 消息详情 | 登录用户 |
| | PATCH | /inbox/:id/read | 标记已读 | 登录用户 |
| | POST | /inbox/batch-read | 批量标记已读 | 登录用户 |
| | POST | /inbox/read-all | 全部标记已读 | 登录用户 |
| | DELETE | /inbox/:id | 删除消息 | 登录用户 |
| | GET | /inbox/unread-count | 未读消息计数 | 登录用户 |
| **系统公告** | POST | /announcements | 创建公告 | 超级管理员 |
| | GET | /announcements | 公告列表 | 登录用户 |
| | GET | /announcements/:id | 公告详情 | 登录用户 |
| | PUT | /announcements/:id | 编辑公告 | 超级管理员 |
| | POST | /announcements/:id/publish | 发布公告 | 超级管理员 |
| | POST | /announcements/:id/unpublish | 下架公告 | 超级管理员 |
| | DELETE | /announcements/:id | 删除公告 | 超级管理员 |
| **定向通知** | POST | /send | 发送定向通知 | 管理员/教师 |
| **通知偏好** | GET | /preferences | 获取通知偏好 | 登录用户 |
| | PUT | /preferences | 更新通知偏好 | 登录用户 |
| **消息模板** | GET | /templates | 模板列表 | 超级管理员 |
| | GET | /templates/:id | 模板详情 | 超级管理员 |
| | PUT | /templates/:id | 更新模板 | 超级管理员 |
| | POST | /templates/:id/preview | 预览模板 | 超级管理员 |
| **统计** | GET | /statistics | 消息统计 | 超级管理员 |

### 内部接口（模块间调用，不对外暴露）

| 方法 | 路径 | 说明 | 调用方 |
|------|------|------|--------|
| POST | /internal/send-event | 发送通知事件 | 所有模块 |

---

## 二、接口详细定义

### 2.1 站内信

#### GET /api/v1/notifications/inbox — 收件箱列表

**权限：** 登录用户

**查询参数：**
| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page | int | 否 | 页码，默认1 |
| page_size | int | 否 | 每页条数，默认20 |
| category | int | 否 | 分类筛选：1系统 2课程 3实验 4竞赛 5成绩 |
| is_read | bool | 否 | 已读状态筛选 |
| keyword | string | 否 | 标题关键词搜索 |

**成功响应：**
```json
{
  "code": 200,
  "data": {
    "list": [
      {
        "id": "1900000000001",
        "category": 2,
        "category_text": "课程通知",
        "title": "新作业发布",
        "content": "课程《区块链原理》发布了新作业《作业1》，截止时间：2026-04-15 23:59。",
        "source_module": "module_03",
        "source_type": "assignment",
        "source_id": "1780000000200",
        "is_read": false,
        "created_at": "2026-04-09T10:00:00Z"
      },
      {
        "id": "1900000000002",
        "category": 5,
        "category_text": "成绩通知",
        "title": "成绩已发布",
        "content": "课程《智能合约开发》2025-2026学年第一学期成绩已发布，请查看。",
        "source_module": "module_06",
        "source_type": "grade_review",
        "source_id": "1880000000100",
        "is_read": true,
        "read_at": "2026-04-09T11:00:00Z",
        "created_at": "2026-04-09T08:00:00Z"
      }
    ],
    "pagination": { "page": 1, "page_size": 20, "total": 25, "total_pages": 2 },
    "unread_count": 8
  }
}
```

> **说明：** 列表接口同时返回 `unread_count`，减少前端额外请求。系统公告不在此列表中，前端需要单独请求公告接口并合并展示。

#### GET /api/v1/notifications/inbox/:id — 消息详情

**权限：** 登录用户（仅自己的消息）

**后端逻辑：** 返回消息详情，同时自动标记为已读（更新 `is_read=true`, `read_at=NOW()`），更新Redis未读计数。

#### POST /api/v1/notifications/inbox/batch-read — 批量标记已读

**权限：** 登录用户

**请求体：**
```json
{
  "ids": ["1900000000001", "1900000000003", "1900000000005"]
}
```

#### POST /api/v1/notifications/inbox/read-all — 全部标记已读

**权限：** 登录用户

**后端逻辑：** 将该用户所有未读消息标记为已读，Redis未读计数归零。

#### GET /api/v1/notifications/inbox/unread-count — 未读消息计数

**权限：** 登录用户

**成功响应：**
```json
{
  "code": 200,
  "data": {
    "total": 8,
    "by_category": {
      "system": 1,
      "course": 3,
      "experiment": 2,
      "competition": 1,
      "grade": 1
    }
  }
}
```

> **说明：** 此接口优先从Redis缓存读取，性能要求 ≤ 50ms。

---

### 2.2 系统公告

#### POST /api/v1/notifications/announcements — 创建公告

**权限：** 超级管理员

**请求体：**
```json
{
  "title": "系统维护通知",
  "content": "<p>平台将于2026年4月15日凌晨2:00-4:00进行系统维护...</p>",
  "scheduled_at": "2026-04-14T18:00:00Z"
}
```

> `scheduled_at` 为空则保存为草稿，需手动发布。

#### POST /api/v1/notifications/announcements/:id/publish — 发布公告

**权限：** 超级管理员

**后端逻辑：**
1. 状态变为"已发布"
2. 通过NATS异步通知所有在线用户（WebSocket推送新公告提醒）
3. 不写入 `notifications` 表（公告通过 `system_announcements` + `announcement_read_status` 管理）

#### GET /api/v1/notifications/announcements — 公告列表

**权限：** 登录用户

**查询参数：** `status`（管理员可筛选草稿/已发布/已下架），`page`, `page_size`

**成功响应（用户视角）：**
```json
{
  "code": 200,
  "data": {
    "list": [
      {
        "id": "1900000000100",
        "title": "系统维护通知",
        "content": "<p>平台将于2026年4月15日...</p>",
        "is_pinned": true,
        "is_read": false,
        "published_at": "2026-04-14T18:00:00Z"
      }
    ],
    "pagination": { "page": 1, "page_size": 20, "total": 3, "total_pages": 1 }
  }
}
```

> 用户视角只返回已发布的公告，`is_read` 通过 `announcement_read_status` 表判断。

---

### 2.3 定向通知

#### POST /api/v1/notifications/send — 发送定向通知

**权限：** 学校管理员 / 教师

**请求体：**
```json
{
  "title": "期中考试安排通知",
  "content": "请各位同学注意，期中考试将于4月20日进行...",
  "target_type": "course",
  "target_id": "1780000000100",
  "category": 2
}
```

**target_type 说明：**
| target_type | 说明 | target_id |
|-------------|------|-----------|
| `all_school` | 全校用户 | school_id（自动取当前用户学校） |
| `course` | 课程学生 | course_id |
| `user` | 指定用户 | user_id |
| `users` | 多个用户 | 逗号分隔的user_id列表 |

**后端逻辑：**
1. 根据 target_type 解析接收者列表
2. 教师只能向自己课程的学生发送
3. 学校管理员只能向本校用户发送
4. 通过NATS异步批量创建通知记录

---

### 2.4 通知偏好

#### GET /api/v1/notifications/preferences — 获取通知偏好

**权限：** 登录用户

**成功响应：**
```json
{
  "code": 200,
  "data": {
    "preferences": [
      { "category": 1, "category_text": "系统通知", "is_enabled": true, "is_forced": true },
      { "category": 2, "category_text": "课程通知", "is_enabled": true, "is_forced": false },
      { "category": 3, "category_text": "实验通知", "is_enabled": true, "is_forced": false },
      { "category": 4, "category_text": "竞赛通知", "is_enabled": false, "is_forced": false },
      { "category": 5, "category_text": "成绩通知", "is_enabled": true, "is_forced": true }
    ]
  }
}
```

> `is_forced=true` 的分类不可关闭，前端禁用开关。

#### PUT /api/v1/notifications/preferences — 更新通知偏好

**权限：** 登录用户

**请求体：**
```json
{
  "preferences": [
    { "category": 2, "is_enabled": true },
    { "category": 3, "is_enabled": false },
    { "category": 4, "is_enabled": false }
  ]
}
```

**后端校验：** 忽略对强制分类（1系统、5成绩）的关闭请求。

---

### 2.5 消息模板

#### GET /api/v1/notifications/templates — 模板列表

**权限：** 超级管理员

**成功响应：**
```json
{
  "code": 200,
  "data": {
    "list": [
      {
        "id": "1900000000200",
        "event_type": "assignment.published",
        "category": 2,
        "category_text": "课程通知",
        "title_template": "新作业发布",
        "content_template": "课程《{course_name}》发布了新作业《{assignment_name}》，截止时间：{deadline}。",
        "variables": [
          { "name": "course_name", "description": "课程名称", "required": true },
          { "name": "assignment_name", "description": "作业名称", "required": true },
          { "name": "deadline", "description": "截止时间", "required": false }
        ],
        "is_enabled": true
      }
    ]
  }
}
```

#### PUT /api/v1/notifications/templates/:id — 更新模板

**权限：** 超级管理员

**请求体：**
```json
{
  "title_template": "📝 新作业发布",
  "content_template": "课程《{course_name}》发布了新作业《{assignment_name}》，请在{deadline}前完成提交。",
  "is_enabled": true
}
```

> 不可修改 `event_type` 和 `variables`（系统预定义）。

#### POST /api/v1/notifications/templates/:id/preview — 预览模板

**权限：** 超级管理员

**请求体：**
```json
{
  "params": {
    "course_name": "区块链原理",
    "assignment_name": "作业1",
    "deadline": "2026-04-15 23:59"
  }
}
```

**成功响应：**
```json
{
  "code": 200,
  "data": {
    "title": "📝 新作业发布",
    "content": "课程《区块链原理》发布了新作业《作业1》，请在2026-04-15 23:59前完成提交。"
  }
}
```

---

### 2.6 内部接口

#### POST /api/v1/notifications/internal/send-event — 发送通知事件

> **此接口仅供内部模块调用，不对外暴露。** 通过内部网络调用或NATS消息触发。

**请求体：**
```json
{
  "event_type": "assignment.published",
  "receiver_ids": ["1780000000500", "1780000000501", "1780000000502"],
  "params": {
    "course_name": "区块链原理",
    "assignment_name": "作业1",
    "deadline": "2026-04-15 23:59"
  },
  "source_module": "module_03",
  "source_type": "assignment",
  "source_id": "1780000000200"
}
```

**后端逻辑：**
1. 查询消息模板（`notification_templates`），如模板未启用则跳过
2. 渲染标题和内容（替换变量占位符）
3. 遍历 `receiver_ids`，检查每个用户的通知偏好
4. 对启用该分类通知的用户，批量写入 `notifications` 表
5. 更新每个接收者的Redis未读计数
6. 通过WebSocket推送未读数更新到在线用户

> **跨模块调用示例：** 模块03发布作业后，调用此接口通知课程学生。模块06成绩审核通过后，调用此接口通知教师和学生。

---

### 2.7 WebSocket

#### 通知推送通道

**连接地址：** `ws://host/api/v1/ws/notifications`

**认证：** 连接时携带JWT Token（同模块04/05的WebSocket认证方式）

**服务端推送消息类型：**

| type | 说明 | payload |
|------|------|---------|
| `unread_count_update` | 未读计数更新 | `{ "total": 9, "by_category": {...} }` |
| `new_notification` | 新消息到达 | `{ "id": "...", "title": "...", "category": 2, "created_at": "..." }` |
| `new_announcement` | 新系统公告 | `{ "id": "...", "title": "...", "published_at": "..." }` |

> **与其他模块WebSocket的关系：** 模块04（实验状态推送）和模块05（竞赛排行榜/回合推送）各自维护独立的WebSocket连接。模块07的WebSocket仅负责通知相关的推送。前端可同时维护多个WebSocket连接。

---

### 2.8 统计

#### GET /api/v1/notifications/statistics — 消息统计

**权限：** 超级管理员

**查询参数：** `date_from`, `date_to`

**成功响应：**
```json
{
  "code": 200,
  "data": {
    "total_sent": 15000,
    "total_read": 12000,
    "read_rate": 0.80,
    "by_category": [
      { "category": "系统通知", "sent": 500, "read": 480, "read_rate": 0.96 },
      { "category": "课程通知", "sent": 8000, "read": 6500, "read_rate": 0.81 },
      { "category": "实验通知", "sent": 3000, "read": 2200, "read_rate": 0.73 },
      { "category": "竞赛通知", "sent": 1500, "read": 1200, "read_rate": 0.80 },
      { "category": "成绩通知", "sent": 2000, "read": 1620, "read_rate": 0.81 }
    ],
    "daily_trend": [
      { "date": "2026-04-08", "sent": 500, "read": 400 },
      { "date": "2026-04-09", "sent": 600, "read": 450 }
    ]
  }
}
```

---

## 三、跨模块接口调用说明

### 3.1 本模块调用外部模块

| 调用目标 | 接口/数据 | 场景 |
|---------|----------|------|
| 模块01 | 读取 `users` 表 | 获取接收者信息 |
| 模块03 | 读取 `course_enrollments` | 定时通知需查询课程学生列表 |
| 模块05 | 读取竞赛报名数据 | 定时通知需查询已报名/未报名学生 |

### 3.2 外部模块调用本模块

| 调用来源 | 接口 | 场景 |
|---------|------|------|
| 模块01 | `POST /internal/send-event` | 账号创建、密码重置通知 |
| 模块02 | `POST /internal/send-event` | 学校授权到期等面向内部用户的站内信通知 |
| 模块03 | `POST /internal/send-event` | 作业发布、批改完成通知 |
| 模块04 | `POST /internal/send-event` | 实验发布、超时提醒、评分通知 |
| 模块05 | `POST /internal/send-event` | 竞赛发布、开始提醒通知 |
| 模块06 | `POST /internal/send-event` | 成绩审核、申诉处理、学业预警通知 |
| 模块08 | `POST /internal/send-event` | 系统维护、告警通知 |

---

*文档版本：v1.0*
*创建日期：2026-04-09*
*更新日期：2026-04-09*
