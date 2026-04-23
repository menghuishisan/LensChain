# 用户与认证模块 — API 接口设计

> 模块状态：✅ 已确认
> 文档版本：v1.0
> 遵循规范：[API规范](../../standards/API规范.md)

---

## 一、接口总览

| 分组 | 方法 | 路径 | 说明 | 权限 |
|------|------|------|------|------|
| **认证** | POST | /api/v1/auth/login | 手机号+密码登录 | 无需认证 |
| | POST | /api/v1/auth/logout | 登出 | 已登录 |
| | POST | /api/v1/auth/token/refresh | 刷新Token | 无需认证（需Refresh Token） |
| | GET | /api/v1/auth/sso-schools | 获取已配置SSO学校列表 | 无需认证 |
| | GET | /api/v1/auth/sso/:school_id/login | SSO登录跳转 | 无需认证 |
| | GET | /api/v1/auth/sso/callback | SSO回调 | 无需认证 |
| **密码** | POST | /api/v1/auth/change-password | 修改密码 | 已登录 |
| | POST | /api/v1/auth/force-change-password | 首次登录强制改密 | 特殊Token |
| **用户管理** | GET | /api/v1/users | 用户列表 | 超管/校管 |
| | GET | /api/v1/users/:id | 用户详情 | 超管/校管 |
| | POST | /api/v1/users | 手动创建用户 | 超管/校管 |
| | POST | /api/v1/users/super-admins | 创建超级管理员 | 超管 |
| | PUT | /api/v1/users/:id | 更新用户信息 | 超管/校管 |
| | DELETE | /api/v1/users/:id | 删除用户（软删除） | 超管/校管 |
| | PATCH | /api/v1/users/:id/status | 变更账号状态 | 超管/校管 |
| | POST | /api/v1/users/:id/reset-password | 重置用户密码 | 超管/校管 |
| | POST | /api/v1/users/:id/unlock | 解锁账号 | 超管/校管 |
| | POST | /api/v1/users/batch-delete | 批量删除 | 超管/校管 |
| **用户导入** | GET | /api/v1/user-imports/template | 下载导入模板 | 校管 |
| | POST | /api/v1/user-imports/preview | 上传文件预览 | 校管 |
| | POST | /api/v1/user-imports/execute | 确认执行导入 | 校管 |
| | GET | /api/v1/user-imports/:id/failures | 下载失败明细 | 校管 |
| **个人中心** | GET | /api/v1/profile | 获取个人信息 | 已登录 |
| | PUT | /api/v1/profile | 更新个人信息 | 已登录 |
| **安全策略** | GET | /api/v1/security-policies | 获取安全策略配置 | 超管 |
| | PUT | /api/v1/security-policies | 更新安全策略配置 | 超管 |
| **日志** | GET | /api/v1/login-logs | 登录日志列表 | 超管/校管 |
| | GET | /api/v1/operation-logs | 操作日志列表 | 超管/校管 |

---

## 二、接口详细定义

### 2.1 认证接口

#### POST /api/v1/auth/login — 手机号+密码登录

**请求体：**

```json
{
  "phone": "13800138000",
  "password": "MyPass123"
}
```

**成功响应（正常登录）：**

```json
{
  "code": 200,
  "message": "登录成功",
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIs...",
    "refresh_token": "eyJhbGciOiJIUzI1NiIs...",
    "expires_in": 1800,
    "token_type": "Bearer",
    "user": {
      "id": "1780000000000001",
      "name": "张三",
      "phone": "13800138000",
      "roles": ["student"],
      "school_id": "1780000000000100",
      "school_name": "XX大学",
      "is_first_login": false
    }
  }
}
```

**成功响应（首次登录，需强制改密）：**

```json
{
  "code": 200,
  "message": "首次登录，请修改密码",
  "data": {
    "force_change_password": true,
    "temp_token": "eyJhbGciOiJIUzI1NiIs...",
    "temp_token_expires_in": 300
  }
}
```

> `temp_token` 是一个5分钟有效的临时Token，仅可用于调用强制改密接口。

**错误响应：**

| code | message | 场景 |
|------|---------|------|
| 40101 | 手机号或密码错误 | 手机号不存在或密码错误 |
| 40102 | 账号已被禁用，请联系管理员 | 账号状态为禁用 |
| 40103 | 账号已归档，请联系管理员 | 账号状态为归档 |
| 40104 | 账号已锁定，请{N}分钟后重试 | 登录失败次数达到阈值 |
| 40105 | 密码错误，还剩{N}次机会 | 密码错误但未达锁定阈值 |

---

#### POST /api/v1/auth/logout — 登出

**请求头：** `Authorization: Bearer <access_token>`

**成功响应：**

```json
{
  "code": 200,
  "message": "登出成功",
  "data": null
}
```

**后端逻辑：**
1. 将当前Access Token加入黑名单（Redis）
2. 删除该用户的Session（Redis）
3. 记录登出日志

---

#### POST /api/v1/auth/token/refresh — 刷新Token

**请求体：**

```json
{
  "refresh_token": "eyJhbGciOiJIUzI1NiIs..."
}
```

**成功响应：**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIs...",
    "refresh_token": "eyJhbGciOiJIUzI1NiIs...",
    "expires_in": 1800,
    "token_type": "Bearer"
  }
}
```

**错误响应：**

| code | message | 场景 |
|------|---------|------|
| 40106 | Refresh Token已过期，请重新登录 | Token过期 |
| 40107 | Refresh Token无效 | Token被篡改或已被替换（其他设备登录） |

---

#### GET /api/v1/auth/sso/:school_id/login — SSO登录跳转

**路径参数：**

| 参数 | 类型 | 说明 |
|------|------|------|
| school_id | string | 学校ID |

**响应：** HTTP 302 重定向到学校SSO登录页面

---

#### GET /api/v1/auth/sso-schools — 获取已配置SSO学校列表

**权限：** 无需认证

**查询参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| keyword | string | 否 | 学校名称搜索关键词 |

**成功响应：**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "list": [
      {
        "id": "1780000000000100",
        "name": "XX大学",
        "logo_url": "https://oss.example.com/schools/logo.png"
      }
    ]
  }
}
```

---

#### GET /api/v1/auth/sso/callback — SSO回调

**查询参数：** 由SSO系统回传（CAS的ticket / OAuth2的code等）

**成功响应：** 同登录接口的成功响应格式

**错误响应：**

| code | message | 场景 |
|------|---------|------|
| 40108 | SSO认证失败 | SSO验证ticket/code失败 |
| 40109 | 账号未开通，请联系管理员 | SSO返回的学号在该学校下未找到匹配用户 |

---

### 2.2 密码接口

#### POST /api/v1/auth/change-password — 修改密码

**请求头：** `Authorization: Bearer <access_token>`

**请求体：**

```json
{
  "old_password": "OldPass123",
  "new_password": "NewPass456"
}
```

**错误响应：**

| code | message | 场景 |
|------|---------|------|
| 40011 | 旧密码不正确 | 旧密码验证失败 |
| 40012 | 新密码不符合复杂度要求 | 不满足8位+大小写+数字 |
| 40013 | 新密码不能与当前密码相同 | 新旧密码一致 |

---

#### POST /api/v1/auth/force-change-password — 首次登录强制改密

**请求头：** `Authorization: Bearer <temp_token>`

**请求体：**

```json
{
  "new_password": "NewPass456"
}
```

**成功响应：** 返回正式的双Token（同登录成功响应）

---

### 2.3 用户管理接口

#### GET /api/v1/users — 用户列表

**权限：** 超级管理员 | 学校管理员（仅本校数据）

**查询参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page | int | 否 | 页码，默认1 |
| page_size | int | 否 | 每页条数，默认20，最大100 |
| keyword | string | 否 | 搜索关键词（匹配姓名、手机号、学号） |
| status | int | 否 | 账号状态筛选：1正常 2禁用 3归档 |
| role | string | 否 | 角色筛选：teacher / student |
| college | string | 否 | 学院筛选 |
| education_level | int | 否 | 学业层次：1专科 2本科 3硕士 4博士 |
| sort_by | string | 否 | 排序字段，默认created_at |
| sort_order | string | 否 | 排序方向，默认desc |

**响应 data.list 中每项结构：**

```json
{
  "id": "1780000000000001",
  "phone": "13800138000",
  "name": "张三",
  "student_no": "2024001",
  "status": 1,
  "status_text": "正常",
  "roles": ["student"],
  "college": "计算机学院",
  "major": "软件工程",
  "class_name": "软工2401",
  "education_level": 2,
  "education_level_text": "本科",
  "last_login_at": "2026-04-07T10:00:00Z",
  "created_at": "2026-03-01T08:00:00Z"
}
```

---

#### POST /api/v1/users — 手动创建用户

**权限：** 超级管理员 | 学校管理员

**请求体：**

```json
{
  "phone": "13800138000",
  "name": "李老师",
  "password": "InitPass123",
  "role": "teacher",
  "student_no": "T2024001",
  "college": "计算机学院",
  "major": null,
  "class_name": null,
  "education_level": null,
  "email": "li@example.com",
  "remark": "新入职教师"
}
```

> 学校管理员创建时 `school_id` 自动取当前用户所属学校。
> 当前后端创建用户接口 `role` 仅支持 `teacher` / `student`；超级管理员账号创建需走初始化或后续专用管理能力，不通过该接口传 `super_admin`。

**成功响应：**

```json
{
  "code": 200,
  "message": "用户创建成功",
  "data": {
    "id": "1780000000000001"
  }
}
```

---

#### POST /api/v1/users/super-admins — 创建超级管理员

**权限：** 超级管理员

**请求体：**

```json
{
  "phone": "13900139000",
  "name": "平台管理员",
  "password": "InitPass123",
  "school_id": "1780000000000100",
  "email": "platform-admin@example.com",
  "remark": "新增平台超级管理员"
}
```

**成功响应：**

```json
{
  "code": 201,
  "message": "created",
  "data": {
    "id": "1780000000000300"
  }
}
```

**业务规则：**
1. 仅超级管理员可创建超级管理员
2. 手机号全局唯一
3. 密码必须满足当前安全策略
4. 新账号角色固定为 `super_admin`
5. 由于当前用户表 `school_id` 为必填且存在外键约束，新超管账号必须有 `school_id`；未传时后端回退创建者的 `school_id`，若创建者 Token 中也没有学校归属则返回参数错误
6. 超级管理员权限边界由 `super_admin` 角色决定，不依赖 `school_id` 做数据范围收缩

---

#### PUT /api/v1/users/:id — 更新用户信息

**权限：** 超级管理员 | 学校管理员

**请求体（后端当前支持字段）：**

```json
{
  "name": "张三",
  "student_no": "2024001",
  "college": "计算机学院",
  "major": "软件工程",
  "class_name": "软工2401",
  "enrollment_year": 2024,
  "education_level": 2,
  "grade": 2024,
  "email": "zhangsan@example.com",
  "remark": "学籍信息更新"
}
```

> 后端当前更新接口不支持修改手机号和角色；前端编辑页应将手机号和角色作为只读/不可改字段处理。

---

#### PATCH /api/v1/users/:id/status — 变更账号状态

**权限：** 超级管理员 | 学校管理员

**请求体：**

```json
{
  "status": 2,
  "reason": "违规操作"
}
```

| status | 含义 | 说明 |
|--------|------|------|
| 1 | 正常 | 启用账号 |
| 2 | 禁用 | 禁用后立即踢下线 |
| 3 | 归档 | 归档后立即踢下线 |

---

#### POST /api/v1/users/:id/reset-password — 重置密码

**权限：** 超级管理员 | 学校管理员

**请求体：**

```json
{
  "new_password": "ResetPass123"
}
```

**后端逻辑：**
1. 更新密码哈希
2. 设置 `is_first_login = true`（重置后需再次强制改密）
3. 踢掉该用户当前会话
4. 记录操作日志

---

### 2.4 用户导入接口

#### GET /api/v1/user-imports/template — 下载导入模板

**权限：** 学校管理员

**查询参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| type | string | 是 | 模板类型：student / teacher |

**响应：** 文件下载（Excel .xlsx）

**模板字段（学生）：**

| 列 | 字段 | 必填 | 说明 |
|----|------|------|------|
| A | 姓名 | 是 | |
| B | 手机号 | 是 | 11位手机号 |
| C | 学号 | 是 | 校内唯一 |
| D | 初始密码 | 是 | 每条记录单独填写，且需满足复杂度要求 |
| E | 学院 | 否 | |
| F | 专业 | 否 | |
| G | 班级 | 否 | |
| H | 入学年份 | 否 | 如2024 |
| I | 学业层次 | 否 | 专科/本科/硕士/博士 |
| J | 年级 | 否 | |
| K | 邮箱 | 否 | |
| L | 备注 | 否 | |

---

#### POST /api/v1/user-imports/preview — 上传文件预览

**权限：** 学校管理员

**请求：** `multipart/form-data`

| 字段 | 类型 | 说明 |
|------|------|------|
| file | File | Excel/CSV文件 |
| type | string | 导入类型：student / teacher |

**响应：**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "import_id": "imp_1780000000000001",
    "total": 150,
    "valid": 140,
    "invalid": 5,
    "conflict": 5,
    "preview_list": [
      {
        "row": 2,
        "name": "张三",
        "phone": "13800138000",
        "student_no": "2024001",
        "status": "valid",
        "message": null
      },
      {
        "row": 3,
        "name": "李四",
        "phone": "13800138001",
        "student_no": "2024002",
        "status": "conflict",
        "message": "手机号已存在，当前用户：王五（学号2023005）"
      },
      {
        "row": 4,
        "name": "",
        "phone": "138001",
        "student_no": "2024003",
        "status": "invalid",
        "message": "姓名不能为空；手机号格式不正确"
      }
    ]
  }
}
```

> `preview_list` 仅返回前100条预览 + 所有冲突和无效记录。

---

#### POST /api/v1/user-imports/execute — 确认执行导入

**权限：** 学校管理员

**请求体：**

```json
{
  "import_id": "imp_1780000000000001",
  "conflict_strategy": "skip",
  "conflict_overrides": ["13800138001"]
}
```

| 字段 | 说明 |
|------|------|
| import_id | 预览时返回的导入批次ID |
| conflict_strategy | 冲突默认策略：skip（跳过）/ overwrite（覆盖） |
| conflict_overrides | 需要单独覆盖处理的手机号列表（覆盖默认策略） |

**响应：**

```json
{
  "code": 200,
  "message": "导入完成",
  "data": {
    "import_id": "imp_1780000000000001",
    "success_count": 138,
    "fail_count": 5,
    "skip_count": 7,
    "overwrite_count": 2
  }
}
```

---

#### GET /api/v1/user-imports/:id/failures — 下载失败明细

**权限：** 学校管理员

**响应：** 文件下载（Excel .xlsx），包含失败行数据和失败原因列

---

### 2.5 个人中心接口

#### GET /api/v1/profile — 获取个人信息

**权限：** 已登录

**响应：**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "id": "1780000000000001",
    "phone": "138****8000",
    "name": "张三",
    "nickname": "小张",
    "avatar_url": "https://oss.example.com/avatars/xxx.jpg",
    "email": "zhangsan@example.com",
    "student_no": "2024001",
    "school_name": "XX大学",
    "college": "计算机学院",
    "major": "软件工程",
    "class_name": "软工2401",
    "education_level": 2,
    "education_level_text": "本科",
    "roles": ["student"]
  }
}
```

> 手机号脱敏展示，仅显示前3位和后4位。
> 学习概览不属于模块01接口职责，由模块06 `GET /api/v1/grades/my/learning-overview` 提供，前端个人中心页面组合调用。

---

#### PUT /api/v1/profile — 更新个人信息

**权限：** 已登录

**请求体（仅可修改以下字段）：**

```json
{
  "nickname": "新昵称",
  "avatar_url": "https://oss.example.com/avatars/new.jpg",
  "email": "new@example.com"
}
```

> 姓名、手机号、学号、学籍信息等由管理员管理，学生不可自行修改。

---

### 2.6 安全策略接口

#### GET /api/v1/security-policies — 获取安全策略

**权限：** 超级管理员

**响应：**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "login_fail_max_count": 5,
    "login_lock_duration_minutes": 15,
    "password_min_length": 8,
    "password_require_uppercase": true,
    "password_require_lowercase": true,
    "password_require_digit": true,
    "password_require_special_char": false,
    "access_token_expire_minutes": 30,
    "refresh_token_expire_days": 7
  }
}
```

---

#### PUT /api/v1/security-policies — 更新安全策略

**权限：** 超级管理员

**请求体：** 同上响应中的data结构（部分更新）

---

### 2.7 日志接口

#### GET /api/v1/login-logs — 登录日志列表

**权限：** 超级管理员（全部） | 学校管理员（本校）

**查询参数：**

| 参数 | 类型 | 说明 |
|------|------|------|
| page / page_size | int | 分页 |
| user_id | string | 按用户筛选 |
| action | int | 操作类型筛选 |
| created_from / created_to | string | 时间范围 |

---

#### GET /api/v1/operation-logs — 操作日志列表

**权限：** 超级管理员（全部） | 学校管理员（本校）

**查询参数：**

| 参数 | 类型 | 说明 |
|------|------|------|
| page / page_size | int | 分页 |
| operator_id | string | 按操作人筛选 |
| action | string | 操作类型筛选 |
| target_type | string | 目标资源类型筛选 |
| created_from / created_to | string | 时间范围 |

---

*文档版本：v1.0*
*创建日期：2026-04-07*
