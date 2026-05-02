# 超级管理员API改进需求

> 问题：超管在管理全平台数据时，无法在列表中看到数据所属的学校，导致UX严重不足

---

## 一、需要修改的API

### 1.1 用户列表 API
**接口**：`GET /api/v1/users`

**当前问题**：
- `UserListItem` 没有 `school_id` 和 `school_name` 字段
- 超管无法在列表中看到用户属于哪个学校

**修改方案**：
```go
type UserListItem struct {
    ID                 string   `json:"id"`
    Phone              string   `json:"phone"`
    Name               string   `json:"name"`
    StudentNo          *string  `json:"student_no"`
    Status             int16    `json:"status"`
    StatusText         string   `json:"status_text"`
    Roles              []string `json:"roles"`
    College            *string  `json:"college"`
    Major              *string  `json:"major"`
    ClassName          *string  `json:"class_name"`
    EducationLevel     *int16   `json:"education_level"`
    EducationLevelText *string  `json:"education_level_text"`
    LastLoginAt        *string  `json:"last_login_at"`
    CreatedAt          string   `json:"created_at"`
    
    // 新增字段
    SchoolID           *string  `json:"school_id"`   // 学校ID，超管为"0"
    SchoolName         *string  `json:"school_name"` // 学校名称，超管为null
}
```

**注意事项**：
- 学校管理员调用时，这些字段可以为空或返回本校信息
- 超级管理员调用时，必须返回学校信息
- 超级管理员账号的 `school_id` 为 `"0"`，`school_name` 为 `null`

---

### 1.2 登录日志 API
**接口**：`GET /api/v1/logs/login`

**当前问题**：
- `LoginLogItem` 没有学校字段
- 超管无法区分不同学校的登录日志

**修改方案**：
```go
type LoginLogItem struct {
    ID              string  `json:"id"`
    UserID          string  `json:"user_id"`
    UserName        string  `json:"user_name"`
    Action          int16   `json:"action"`
    ActionText      string  `json:"action_text"`
    LoginMethod     *int16  `json:"login_method"`
    LoginMethodText *string `json:"login_method_text"`
    IP              string  `json:"ip"`
    UserAgent       *string `json:"user_agent"`
    FailReason      *string `json:"fail_reason"`
    CreatedAt       string  `json:"created_at"`
    
    // 新增字段
    SchoolID        *string `json:"school_id"`   // 用户所属学校ID
    SchoolName      *string `json:"school_name"` // 用户所属学校名称
}
```

---

### 1.3 操作日志 API
**接口**：`GET /api/v1/logs/operation`

**当前问题**：
- `OperationLogItem` 没有学校字段
- 超管无法区分不同学校的操作日志

**修改方案**：
```go
type OperationLogItem struct {
    ID           string  `json:"id"`
    OperatorID   string  `json:"operator_id"`
    OperatorName string  `json:"operator_name"`
    Action       string  `json:"action"`
    TargetType   string  `json:"target_type"`
    TargetID     *string `json:"target_id"`
    Detail       *string `json:"detail"`
    IP           string  `json:"ip"`
    CreatedAt    string  `json:"created_at"`
    
    // 新增字段
    SchoolID     *string `json:"school_id"`   // 操作者所属学校ID
    SchoolName   *string `json:"school_name"` // 操作者所属学校名称
}
```

---

## 二、数据库查询优化

### 2.1 用户列表查询
需要 JOIN `schools` 表获取学校名称：

```sql
SELECT 
    u.*,
    s.id as school_id,
    s.name as school_name
FROM users u
LEFT JOIN schools s ON u.school_id = s.id
WHERE ...
```

### 2.2 日志查询
需要 JOIN `users` 和 `schools` 表：

```sql
SELECT 
    l.*,
    u.school_id,
    s.name as school_name
FROM login_logs l
JOIN users u ON l.user_id = u.id
LEFT JOIN schools s ON u.school_id = s.id
WHERE ...
```

---

## 三、权限控制

### 3.1 学校管理员
- 只能查询本校数据（后端已有过滤）
- 返回的 `school_id` 和 `school_name` 可以省略或返回本校信息

### 3.2 超级管理员
- 可以查询全平台数据
- **必须**返回 `school_id` 和 `school_name`
- 用于在前端列表中显示学校列

---

## 四、前端对应修改

后端修改完成后，前端需要：

1. 更新类型定义（`types/auth.ts`）
2. 创建超管专用组件，显示学校列：
   - `SuperAdminUserListPanel`
   - `SuperAdminLoginLogPanel`
   - `SuperAdminOperationLogPanel`
3. 更新超管页面使用新组件

---

## 五、验收标准

- [ ] 后端API返回学校字段
- [ ] 超管调用API能看到学校信息
- [ ] 学校管理员调用API不受影响
- [ ] 前端超管列表显示学校列
- [ ] 数据库查询性能无明显下降

---

*文档版本：v1.0*  
*创建日期：2025-01-XX*
