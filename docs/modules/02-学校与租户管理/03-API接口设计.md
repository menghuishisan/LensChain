# 学校与租户管理模块 — API 接口设计

> 模块状态：✅ 已确认
> 文档版本：v1.0
> 遵循规范：[API规范](../../standards/API规范.md)

---

## 一、接口总览

| 分组 | 方法 | 路径 | 说明 | 权限 |
|------|------|------|------|------|
| **入驻申请（公开）** | POST | /api/v1/school-applications | 提交入驻申请 | 无需认证 |
| | POST | /api/v1/school-applications/send-sms-code | 发送查询验证码 | 无需认证 |
| | GET | /api/v1/school-applications/query | 查询申请状态 | 无需认证（手机号+验证码） |
| | POST | /api/v1/school-applications/:id/reapply | 重新申请 | 无需认证（手机号+验证码） |
| **入驻审核** | GET | /api/v1/admin/school-applications | 申请列表 | 超管 |
| | GET | /api/v1/admin/school-applications/:id | 申请详情 | 超管 |
| | POST | /api/v1/admin/school-applications/:id/approve | 审核通过 | 超管 |
| | POST | /api/v1/admin/school-applications/:id/reject | 审核拒绝 | 超管 |
| **学校管理（超管）** | GET | /api/v1/admin/schools | 学校列表 | 超管 |
| | POST | /api/v1/admin/schools | 后台直接创建学校 | 超管 |
| | GET | /api/v1/admin/schools/:id | 学校详情 | 超管 |
| | PUT | /api/v1/admin/schools/:id | 编辑学校信息 | 超管 |
| | PATCH | /api/v1/admin/schools/:id/license | 设置有效期 | 超管 |
| | POST | /api/v1/admin/schools/:id/freeze | 冻结学校 | 超管 |
| | POST | /api/v1/admin/schools/:id/unfreeze | 解冻学校 | 超管 |
| | POST | /api/v1/admin/schools/:id/cancel | 注销学校 | 超管 |
| | POST | /api/v1/admin/schools/:id/restore | 恢复已注销学校 | 超管 |
| **学校配置（校管）** | GET | /api/v1/school/profile | 获取本校信息 | 校管 |
| | PUT | /api/v1/school/profile | 编辑本校信息 | 校管 |
| | GET | /api/v1/school/sso-config | 获取SSO配置 | 校管 |
| | PUT | /api/v1/school/sso-config | 更新SSO配置 | 校管 |
| | POST | /api/v1/school/sso-config/test | 测试SSO连接 | 校管 |
| | POST | /api/v1/school/sso-config/enable | 启用/禁用SSO | 校管 |
| | GET | /api/v1/school/license | 查看授权状态 | 校管 |
| **公开接口** | GET | /api/v1/schools/sso-list | 获取已配置SSO的学校列表 | 无需认证 |

---

## 二、接口详细定义

### 2.1 入驻申请接口（公开）

#### POST /api/v1/school-applications — 提交入驻申请

**请求体：**

```json
{
  "school_name": "XX大学",
  "school_code": "10001",
  "school_address": "北京市海淀区XX路1号",
  "school_website": "https://www.xxu.edu.cn",
  "school_logo_url": null,
  "contact_name": "张教授",
  "contact_phone": "13800138000",
  "contact_email": "zhang@xxu.edu.cn",
  "contact_title": "计算机学院副院长"
}
```

**成功响应：**

```json
{
  "code": 200,
  "message": "申请提交成功",
  "data": {
    "application_id": "1780000000000001",
    "status": 1,
    "status_text": "待审核",
    "tip": "请使用联系人手机号查询审核进度"
  }
}
```

#### POST /api/v1/school-applications/send-sms-code — 发送查询验证码

**请求体：**

```json
{
  "phone": "13800138000"
}
```

**响应：**

```json
{
  "code": 200,
  "message": "验证码发送成功",
  "data": null,
  "timestamp": "2026-04-09T10:00:00Z"
}
```

**业务规则：**
- 同一手机号 60 秒内仅允许发送一次
- 验证码有效期 5 分钟
- 为避免泄露申请状态，该接口对不存在申请记录的手机号也返回统一成功响应

**错误响应：**

| code | message | 场景 |
|------|---------|------|
| 40001 | 学校名称不能为空 | 必填字段缺失 |
| 40002 | 手机号格式不正确 | 手机号校验失败 |
| 40901 | 该手机号已有待审核的申请 | 重复提交 |
| 40902 | 该学校名称已存在 | 学校名称重复 |

---

#### GET /api/v1/school-applications/query — 查询申请状态

**查询参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| phone | string | 是 | 联系人手机号 |
| sms_code | string | 是 | 短信验证码 |

**响应：**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "applications": [
      {
        "application_id": "1780000000000001",
        "school_name": "XX大学",
        "status": 1,
        "status_text": "待审核",
        "created_at": "2026-04-01T10:00:00Z",
        "reviewed_at": null,
        "reject_reason": null
      }
    ]
  }
}
```

---

#### POST /api/v1/school-applications/:id/reapply — 重新申请

**请求体：** 同提交申请，额外需要 `sms_code` 验证身份

```json
{
  "sms_code": "123456",
  "school_name": "XX大学",
  "school_code": "10001",
  "contact_name": "张教授",
  "contact_phone": "13800138000",
  "...": "其他修改后的字段"
}
```

> 仅状态为"已拒绝"的申请可重新申请，生成新申请记录并关联 `previous_application_id`。

---

### 2.2 入驻审核接口

#### GET /api/v1/admin/school-applications — 申请列表

**权限：** 超级管理员

**查询参数：**

| 参数 | 类型 | 说明 |
|------|------|------|
| page / page_size | int | 分页 |
| status | int | 状态筛选：1待审核 2已通过 3已拒绝 |
| keyword | string | 搜索（学校名称、联系人姓名、手机号） |
| sort_by / sort_order | string | 排序 |

---

#### POST /api/v1/admin/school-applications/:id/approve — 审核通过

**权限：** 超级管理员

**请求体：**

```json
{
  "license_end_at": "2027-04-07T00:00:00Z"
}
```

**后端逻辑：**

1. 申请状态更新为"已通过"
2. 创建学校记录（状态：已激活）
3. 用联系人信息创建首个校管账号（角色：教师+学校管理员）
4. 系统随机生成初始密码
5. 发送短信通知联系人：审核通过 + 登录账号 + 初始密码 + 平台地址

**响应：**

```json
{
  "code": 200,
  "message": "审核通过",
  "data": {
    "school_id": "1780000000000100",
    "admin_user_id": "1780000000000200",
    "admin_phone": "13800138000",
    "sms_sent": true
  }
}
```

---

#### POST /api/v1/admin/school-applications/:id/reject — 审核拒绝

**权限：** 超级管理员

**请求体：**

```json
{
  "reject_reason": "学校信息不完整，请补充学校编码和官网地址"
}
```

**后端逻辑：**

1. 申请状态更新为"已拒绝"
2. 发送短信通知联系人：审核未通过 + 拒绝原因

---

### 2.3 学校管理接口（超管）

#### GET /api/v1/admin/schools — 学校列表

**权限：** 超级管理员

**查询参数：**

| 参数 | 类型 | 说明 |
|------|------|------|
| page / page_size | int | 分页 |
| keyword | string | 搜索（学校名称、编码） |
| status | int | 状态筛选 |
| license_expiring | bool | 筛选7天内即将到期的学校 |
| sort_by / sort_order | string | 排序 |

**响应 data.list 中每项结构：**

```json
{
  "id": "1780000000000100",
  "name": "XX大学",
  "code": "10001",
  "logo_url": "https://oss.example.com/logos/xxu.png",
  "status": 2,
  "status_text": "已激活",
  "license_start_at": "2026-04-07T00:00:00Z",
  "license_end_at": "2027-04-07T00:00:00Z",
  "license_remaining_days": 365,
  "contact_name": "张教授",
  "contact_phone": "138****8000",
  "created_at": "2026-04-07T10:00:00Z"
}
```

---

#### POST /api/v1/admin/schools — 后台直接创建学校

**权限：** 超级管理员

**请求体：**

```json
{
  "name": "YY大学",
  "code": "10002",
  "address": "上海市浦东新区XX路2号",
  "website": "https://www.yyu.edu.cn",
  "logo_url": null,
  "description": "YY大学简介",
  "license_start_at": "2026-04-07T00:00:00Z",
  "license_end_at": "2027-04-07T00:00:00Z",
  "contact_name": "李教授",
  "contact_phone": "13900139000",
  "contact_email": "li@yyu.edu.cn",
  "contact_title": "信息中心主任"
}
```

**后端逻辑：** 同审核通过，直接创建学校（已激活）+ 创建校管 + 短信通知

---

#### PATCH /api/v1/admin/schools/:id/license — 设置有效期

**权限：** 超级管理员

**请求体：**

```json
{
  "license_end_at": "2028-04-07T00:00:00Z"
}
```

---

#### POST /api/v1/admin/schools/:id/freeze — 冻结学校

**权限：** 超级管理员

**请求体：**

```json
{
  "reason": "合作协议到期未续约"
}
```

**后端逻辑：**

1. 学校状态变为"已冻结"
2. 该校所有用户Session立即失效
3. 清除该校的学校状态缓存
4. 记录操作日志

---

#### POST /api/v1/admin/schools/:id/unfreeze — 解冻学校

**权限：** 超级管理员

**后端逻辑：**

1. 学校状态恢复为"已激活"
2. 刷新学校状态缓存

---

#### POST /api/v1/admin/schools/:id/cancel — 注销学校

**权限：** 超级管理员

**后端逻辑：**

1. 学校状态变为"已注销"，设置 `deleted_at`
2. 该校所有用户同时软删除
3. 该校所有用户Session立即失效

> 需二次确认，接口需传入确认参数 `"confirm": true`

---

#### POST /api/v1/admin/schools/:id/restore — 恢复已注销学校

**权限：** 超级管理员

**后端逻辑：**

1. 清除学校的 `deleted_at`，状态恢复为"已激活"
2. 该校所有用户同时恢复（清除 `deleted_at`）

---

### 2.4 学校配置接口（校管）

#### GET /api/v1/school/profile — 获取本校信息

**权限：** 学校管理员

**响应：**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "id": "1780000000000100",
    "name": "XX大学",
    "code": "10001",
    "logo_url": "https://oss.example.com/logos/xxu.png",
    "address": "北京市海淀区XX路1号",
    "website": "https://www.xxu.edu.cn",
    "description": "XX大学简介...",
    "status": 2,
    "status_text": "已激活"
  }
}
```

---

#### PUT /api/v1/school/profile — 编辑本校信息

**权限：** 学校管理员

**请求体（仅可修改以下字段）：**

```json
{
  "logo_url": "https://oss.example.com/logos/xxu_new.png",
  "description": "更新后的学校简介",
  "address": "北京市海淀区XX路1号（新校区）",
  "website": "https://www.xxu.edu.cn"
}
```

> 学校名称和编码不可由校管修改，需联系超管。

---

#### GET /api/v1/school/sso-config — 获取SSO配置

**权限：** 学校管理员

**响应：**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "provider": "oauth2",
    "is_enabled": true,
    "is_tested": true,
    "tested_at": "2026-04-05T14:00:00Z",
    "config": {
      "authorize_url": "https://oauth.xxu.edu.cn/authorize",
      "token_url": "https://oauth.xxu.edu.cn/token",
      "userinfo_url": "https://oauth.xxu.edu.cn/userinfo",
      "client_id": "lianjing_client",
      "client_secret": "******",
      "redirect_uri": "https://lianjing.com/api/v1/auth/sso/callback",
      "scope": "openid profile",
      "user_id_attribute": "student_id"
    }
  }
}
```

> `client_secret` 脱敏显示为 `******`。

---

#### PUT /api/v1/school/sso-config — 更新SSO配置

**权限：** 学校管理员

**请求体：**

```json
{
  "provider": "oauth2",
  "config": {
    "authorize_url": "https://oauth.xxu.edu.cn/authorize",
    "token_url": "https://oauth.xxu.edu.cn/token",
    "userinfo_url": "https://oauth.xxu.edu.cn/userinfo",
    "client_id": "lianjing_client",
    "client_secret": "new_secret_value",
    "redirect_uri": "https://lianjing.com/api/v1/auth/sso/callback",
    "scope": "openid profile",
    "user_id_attribute": "student_id"
  }
}
```

**后端逻辑：**

1. 保存配置，`client_secret` 加密存储
2. 重置 `is_tested = false`（配置变更后需重新测试）
3. `is_enabled` 保持不变

---

#### POST /api/v1/school/sso-config/test — 测试SSO连接

**权限：** 学校管理员

**响应（成功）：**

```json
{
  "code": 200,
  "message": "SSO连接测试成功",
  "data": {
    "is_tested": true,
    "tested_at": "2026-04-07T15:00:00Z",
    "test_detail": "成功连接到OAuth2授权服务器"
  }
}
```

**响应（失败）：**

```json
{
  "code": 40010,
  "message": "SSO连接测试失败",
  "data": {
    "is_tested": false,
    "error_detail": "无法连接到Token端点：Connection refused"
  }
}
```

#### POST /api/v1/school/sso-config/enable — 启用/禁用SSO

**权限：** 学校管理员

**请求体：**

```json
{
  "is_enabled": true
}
```

**响应：**

```json
{
  "code": 200,
  "message": "SSO已启用",
  "data": null,
  "timestamp": "2026-04-09T10:00:00Z"
}
```

**业务规则：**
- 启用前必须已存在 SSO 配置且最近一次测试通过
- 禁用时不要求重新测试

---

#### GET /api/v1/school/license — 查看授权状态

**权限：** 学校管理员

**响应：**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "license_start_at": "2026-04-07T00:00:00Z",
    "license_end_at": "2027-04-07T00:00:00Z",
    "remaining_days": 365,
    "status": 2,
    "status_text": "已激活",
    "is_expiring_soon": false
  }
}
```

---

### 2.5 公开接口

#### GET /api/v1/schools/sso-list — 获取已配置SSO的学校列表

> 用于登录页的SSO学校选择。

**响应：**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "list": [
      {
        "id": "1780000000000100",
        "name": "XX大学",
        "logo_url": "https://oss.example.com/logos/xxu.png"
      },
      {
        "id": "1780000000000101",
        "name": "YY大学",
        "logo_url": "https://oss.example.com/logos/yyu.png"
      }
    ]
  }
}
```

> 仅返回状态为"已激活"且SSO已启用且已通过测试的学校。

---

*文档版本：v1.0*
*创建日期：2026-04-07*
