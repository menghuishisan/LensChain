# API 设计规范

> 本规范适用于链镜平台所有后端 API 接口设计，所有模块必须严格遵守。

---

## 一、基础约定

| 项目 | 约定 |
|------|------|
| 基础路径 | `/api/v1` |
| 认证方式 | Bearer Token（JWT） |
| 内容类型 | `application/json`（文件上传除外） |
| 字符编码 | UTF-8 |
| 时间格式 | ISO 8601（`2026-04-07T12:00:00Z`） |
| 时区 | 统一 UTC 存储，前端按用户时区展示 |

---

## 二、HTTP 方法语义

| 方法 | 语义 | 示例 |
|------|------|------|
| GET | 查询资源（无副作用） | `GET /api/v1/users` |
| POST | 创建资源 | `POST /api/v1/users` |
| PUT | 全量更新资源 | `PUT /api/v1/users/:id` |
| PATCH | 部分更新资源 | `PATCH /api/v1/users/:id` |
| DELETE | 删除资源（软删除） | `DELETE /api/v1/users/:id` |

---

## 三、URL 命名规范

- 使用 **小写字母 + 短横线** 分隔：`/api/v1/user-imports`
- 资源名使用 **复数形式**：`/users`、`/courses`、`/schools`
- 嵌套资源最多两层：`/schools/:id/students`
- 非 CRUD 操作使用动词子路径：`/users/:id/reset-password`

---

## 四、通用响应格式

### 4.1 成功响应

```json
{
  "code": 200,
  "message": "success",
  "data": { ... },
  "timestamp": 1712505600
}
```

### 4.2 分页响应

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "list": [ ... ],
    "pagination": {
      "page": 1,
      "page_size": 20,
      "total": 100,
      "total_pages": 5
    }
  },
  "timestamp": 1712505600
}
```

### 4.3 错误响应

```json
{
  "code": 40001,
  "message": "手机号格式不正确",
  "data": null,
  "timestamp": 1712505600
}
```

---

## 五、业务状态码规范

| 范围 | 含义 | 示例 |
|------|------|------|
| 200 | 成功 | 200 |
| 400xx | 请求参数错误 | 40001 手机号格式错误 |
| 401xx | 认证失败 | 40101 Token过期 |
| 403xx | 权限不足 | 40301 无权访问该资源 |
| 404xx | 资源不存在 | 40401 用户不存在 |
| 409xx | 资源冲突 | 40901 手机号已存在 |
| 500xx | 服务器内部错误 | 50001 数据库异常 |

---

## 六、分页与排序

### 请求参数

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `page` | int | 1 | 页码，从1开始 |
| `page_size` | int | 20 | 每页条数，最大100 |
| `sort_by` | string | `created_at` | 排序字段 |
| `sort_order` | string | `desc` | 排序方向：`asc` / `desc` |

### 过滤参数

- 精确匹配：`?status=1`
- 模糊搜索：`?keyword=张三`（后端决定搜索哪些字段）
- 范围过滤：`?created_from=2026-01-01&created_to=2026-12-31`

---

## 七、认证与权限

### 7.1 请求头

```
Authorization: Bearer <access_token>
```

### 7.2 无需认证的接口

- `POST /api/v1/auth/login` — 登录
- `POST /api/v1/auth/sso/callback` — SSO回调
- `POST /api/v1/auth/token/refresh` — 刷新Token

### 7.3 权限标注

每个接口文档必须标注所需角色：

```
权限要求：超级管理员 | 学校管理员
```

---

## 八、文件上传

- Content-Type：`multipart/form-data`
- 单文件大小限制：50MB（可配置）
- 支持格式在各接口中单独定义

---

*文档版本：v1.0*
*创建日期：2026-04-07*
