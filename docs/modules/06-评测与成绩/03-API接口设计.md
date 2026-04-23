# 评测与成绩模块 — API接口设计

> 模块状态：✅ 已确认
> 文档版本：v1.0

---

## 一、接口总览

### API 前缀：`/api/v1/grades`

| 分类 | 方法 | 路径 | 说明 | 权限 |
|------|------|------|------|------|
| **学期管理** | POST | /semesters | 创建学期 | 学校管理员 |
| | GET | /semesters | 学期列表 | 登录用户 |
| | PUT | /semesters/:id | 更新学期 | 学校管理员 |
| | DELETE | /semesters/:id | 删除学期 | 学校管理员 |
| | PATCH | /semesters/:id/set-current | 设为当前学期 | 学校管理员 |
| **等级映射** | GET | /level-configs | 获取等级映射配置 | 学校管理员 |
| | PUT | /level-configs | 更新等级映射配置 | 学校管理员 |
| | POST | /level-configs/reset-default | 重置为默认配置 | 学校管理员 |
| **成绩审核** | POST | /reviews | 提交成绩审核 | 教师 |
| | GET | /reviews | 审核列表 | 教师/学校管理员 |
| | GET | /reviews/:id | 审核详情 | 教师/学校管理员 |
| | POST | /reviews/:id/approve | 审核通过 | 学校管理员 |
| | POST | /reviews/:id/reject | 审核驳回 | 学校管理员 |
| | POST | /reviews/:id/unlock | 解锁成绩 | 学校管理员 |
| **成绩查询** | GET | /students/:id/semester-grades | 学生学期成绩 | 教师/管理员/学生本人 |
| | GET | /students/:id/gpa | 学生GPA | 教师/管理员/学生本人 |
| | GET | /my/semester-grades | 我的学期成绩 | 学生 |
| | GET | /my/gpa | 我的GPA | 学生 |
| | GET | /my/learning-overview | 我的学习概览 | 学生 |
| **成绩申诉** | POST | /appeals | 提交申诉 | 学生 |
| | GET | /appeals | 申诉列表 | 教师/学生 |
| | GET | /appeals/:id | 申诉详情 | 教师/学生 |
| | POST | /appeals/:id/approve | 同意申诉 | 教师 |
| | POST | /appeals/:id/reject | 驳回申诉 | 教师 |
| **学业预警** | GET | /warnings | 预警列表 | 学校管理员 |
| | GET | /warnings/:id | 预警详情 | 学校管理员 |
| | POST | /warnings/:id/handle | 处理预警 | 学校管理员 |
| | GET | /warning-configs | 获取预警配置 | 学校管理员 |
| | PUT | /warning-configs | 更新预警配置 | 学校管理员 |
| **成绩单** | POST | /transcripts/generate | 生成成绩单 | 学生/教师/管理员 |
| | GET | /transcripts | 成绩单列表 | 学生/教师/管理员 |
| | GET | /transcripts/:id/download | 下载成绩单 | 学生/教师/管理员 |
| **成绩分析** | GET | /analytics/course/:id | 课程成绩分析 | 教师 |
| | GET | /analytics/school | 全校成绩分析 | 学校管理员 |
| | GET | /analytics/platform | 平台成绩总览 | 超级管理员 |

---

## 二、接口详细定义

### 2.1 学期管理

#### POST /api/v1/grades/semesters — 创建学期

**权限：** 学校管理员

**请求体：**
```json
{
  "name": "2025-2026学年第一学期",
  "code": "2025-2026-1",
  "start_date": "2025-09-01",
  "end_date": "2026-01-15"
}
```

**成功响应：**
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "id": "1880000000001",
    "school_id": "1780000000001",
    "name": "2025-2026学年第一学期",
    "code": "2025-2026-1",
    "start_date": "2025-09-01",
    "end_date": "2026-01-15",
    "is_current": false,
    "created_at": "2026-04-09T10:00:00Z"
  }
}
```

**错误码：**
| code | 说明 |
|------|------|
| 40001 | 参数校验失败（名称/编码/日期缺失或格式错误） |
| 40901 | 学期编码已存在 |
| 40002 | 开始日期必须早于结束日期 |

#### GET /api/v1/grades/semesters — 学期列表

**权限：** 登录用户（自动按 school_id 过滤）

**查询参数：** `page`, `page_size`, `sort_by`(默认`start_date`), `sort_order`(默认`desc`)

**成功响应：**
```json
{
  "code": 200,
  "data": {
    "list": [
      {
        "id": "1880000000001",
        "name": "2025-2026学年第一学期",
        "code": "2025-2026-1",
        "start_date": "2025-09-01",
        "end_date": "2026-01-15",
        "is_current": true,
        "course_count": 15,
        "review_status_summary": { "not_submitted": 3, "pending": 5, "approved": 7, "rejected": 0 }
      }
    ],
    "pagination": { "page": 1, "page_size": 20, "total": 3, "total_pages": 1 }
  }
}
```

#### PATCH /api/v1/grades/semesters/:id/set-current — 设为当前学期

**权限：** 学校管理员

**后端逻辑：** 将该学校所有学期的 `is_current` 设为 false，再将目标学期设为 true。

---

### 2.2 等级映射配置

#### GET /api/v1/grades/level-configs — 获取等级映射配置

**权限：** 学校管理员

**成功响应：**
```json
{
  "code": 200,
  "data": {
    "school_id": "1780000000001",
    "levels": [
      { "id": "1880000000010", "level_name": "A", "min_score": 90.00, "max_score": 100.00, "gpa_point": 4.00, "sort_order": 1 },
      { "id": "1880000000011", "level_name": "B", "min_score": 80.00, "max_score": 89.99, "gpa_point": 3.00, "sort_order": 2 },
      { "id": "1880000000012", "level_name": "C", "min_score": 70.00, "max_score": 79.99, "gpa_point": 2.00, "sort_order": 3 },
      { "id": "1880000000013", "level_name": "D", "min_score": 60.00, "max_score": 69.99, "gpa_point": 1.00, "sort_order": 4 },
      { "id": "1880000000014", "level_name": "F", "min_score": 0.00, "max_score": 59.99, "gpa_point": 0.00, "sort_order": 5 }
    ]
  }
}
```

#### PUT /api/v1/grades/level-configs — 更新等级映射配置

**权限：** 学校管理员

**请求体：**
```json
{
  "levels": [
    { "level_name": "A+", "min_score": 95.00, "max_score": 100.00, "gpa_point": 4.00 },
    { "level_name": "A", "min_score": 90.00, "max_score": 94.99, "gpa_point": 3.70 },
    { "level_name": "B+", "min_score": 85.00, "max_score": 89.99, "gpa_point": 3.30 },
    { "level_name": "B", "min_score": 80.00, "max_score": 84.99, "gpa_point": 3.00 },
    { "level_name": "C", "min_score": 70.00, "max_score": 79.99, "gpa_point": 2.00 },
    { "level_name": "D", "min_score": 60.00, "max_score": 69.99, "gpa_point": 1.00 },
    { "level_name": "F", "min_score": 0.00, "max_score": 59.99, "gpa_point": 0.00 }
  ]
}
```

**后端校验：**
- 分数区间不可重叠
- 必须覆盖 0-100 全范围
- 绩点值范围 0.00-4.00
- 至少2个等级

**错误码：**
| code | 说明 |
|------|------|
| 40001 | 参数校验失败 |
| 40003 | 分数区间有重叠 |
| 40004 | 分数区间未覆盖0-100全范围 |
| 40005 | 绩点值超出0-4范围 |

---

### 2.3 成绩审核

#### POST /api/v1/grades/reviews — 提交成绩审核

**权限：** 教师（课程负责人）

**请求体：**
```json
{
  "course_id": "1780000000100",
  "semester_id": "1880000000001",
  "submit_note": "本学期共45名学生，成绩已全部计算完成。"
}
```

**后端逻辑：**
1. 校验教师是否为该课程的负责人（`courses.teacher_id`）
2. 校验该课程该学期是否已有审核记录（不可重复提交）
3. 校验课程成绩是否已全部计算（所有选课学生都有加权总成绩）
4. 创建审核记录，状态设为"待审核"
5. 通过模块07发送站内信通知学校管理员

**成功响应：**
```json
{
  "code": 200,
  "data": {
    "id": "1880000000100",
    "course_id": "1780000000100",
    "semester_id": "1880000000001",
    "status": 2,
    "status_text": "待审核",
    "submitted_at": "2026-04-09T10:00:00Z"
  }
}
```

**错误码：**
| code | 说明 |
|------|------|
| 40301 | 非课程负责人，无权提交 |
| 40901 | 该课程该学期已有审核记录 |
| 40006 | 课程成绩未全部计算完成 |

#### POST /api/v1/grades/reviews/:id/approve — 审核通过

**权限：** 学校管理员

**请求体：**
```json
{
  "review_comment": "成绩分布合理，审核通过。"
}
```

**后端逻辑：**
1. 校验审核记录状态为"待审核"
2. 更新状态为"已通过"，设置 `is_locked = true`
3. 从模块03读取该课程所有学生的加权总成绩
4. 根据学校等级映射配置，计算每个学生的等级和绩点
5. 批量写入 `student_semester_grades` 表
6. 更新所有相关学生的GPA缓存
7. 触发学业预警检测
8. 通过模块07发送站内信通知教师"审核已通过"

> **跨模块数据流：** 此接口是模块06与模块03的核心数据衔接点。从模块03的 `course_grade_configs` 读取权重配置，从 `assignment_submissions` 读取各项成绩，计算加权总成绩后存入本模块的 `student_semester_grades`。

#### POST /api/v1/grades/reviews/:id/reject — 审核驳回

**权限：** 学校管理员

**请求体：**
```json
{
  "review_comment": "部分学生成绩异常，请核实后重新提交。"
}
```

**后端逻辑：**
1. 更新状态为"已驳回"
2. 通过模块07发送站内信通知教师"审核已驳回"及驳回原因

#### POST /api/v1/grades/reviews/:id/unlock — 解锁成绩

**权限：** 学校管理员

**请求体：**
```json
{
  "unlock_reason": "学生张三成绩录入有误，需要教师修改。"
}
```

**后端逻辑：**
1. 校验审核记录状态为"已通过"且 `is_locked = true`
2. 设置 `is_locked = false`，记录解锁信息
3. 状态回退为"未提交"，教师可修改后重新提交
4. 记录操作日志到 `operation_logs`

---

### 2.4 成绩查询

#### GET /api/v1/grades/my/semester-grades — 我的学期成绩

**权限：** 学生

**查询参数：** `semester_id`（可选，默认当前学期）

**成功响应：**
```json
{
  "code": 200,
  "data": {
    "semester": { "id": "1880000000001", "name": "2025-2026学年第一学期", "code": "2025-2026-1" },
    "grades": [
      {
        "course_id": "1780000000100",
        "course_name": "区块链原理与技术",
        "teacher_name": "李教授",
        "credits": 3.0,
        "final_score": 88.50,
        "grade_level": "B",
        "gpa_point": 3.00,
        "is_adjusted": false,
        "review_status": "approved",
        "review_status_text": "已审核"
      },
      {
        "course_id": "1780000000101",
        "course_name": "智能合约开发",
        "teacher_name": "王教授",
        "credits": 2.0,
        "final_score": 92.00,
        "grade_level": "A",
        "gpa_point": 4.00,
        "is_adjusted": false,
        "review_status": "approved",
        "review_status_text": "已审核"
      }
    ],
    "summary": {
      "total_credits": 5.0,
      "semester_gpa": 3.40,
      "course_count": 2,
      "passed_count": 2,
      "failed_count": 0
    }
  }
}
```

#### GET /api/v1/grades/my/gpa — 我的GPA

**权限：** 学生

**成功响应：**
```json
{
  "code": 200,
  "data": {
    "cumulative_gpa": 3.25,
    "cumulative_credits": 45.0,
    "semester_list": [
      { "semester_id": "...", "semester_name": "2024-2025学年第一学期", "gpa": 3.10, "credits": 15.0 },
      { "semester_id": "...", "semester_name": "2024-2025学年第二学期", "gpa": 3.30, "credits": 15.0 },
      { "semester_id": "...", "semester_name": "2025-2026学年第一学期", "gpa": 3.40, "credits": 15.0 }
    ],
    "gpa_trend": [3.10, 3.30, 3.40]
  }
}
```

#### GET /api/v1/grades/my/learning-overview — 我的学习概览

**权限：** 学生

**说明：** 此接口为个人中心页面提供聚合学习数据。模块06作为聚合层，只读模块03课程选课与学习进度、模块04实验实例、模块05竞赛团队成员数据；模块01 `/profile` 不再返回学习概览。

**成功响应：**
```json
{
  "code": 200,
  "data": {
    "course_count": 3,
    "experiment_count": 12,
    "competition_count": 2,
    "total_study_hours": 48.5
  }
}
```

#### GET /api/v1/grades/students/:id/semester-grades — 查看学生学期成绩

**权限：** 教师（该学生选课的教师）/ 学校管理员 / 超级管理员

**查询参数：** `semester_id`（可选）

**响应格式：** 同"我的学期成绩"。

---

### 2.5 成绩申诉

#### POST /api/v1/grades/appeals — 提交申诉

**权限：** 学生

**请求体：**
```json
{
  "grade_id": "1880000000200",
  "appeal_reason": "我认为期末测验第3题的评分有误，我的解答思路是正确的，但被判为错误。附件中有我的解题过程截图。"
}
```

**后端校验：**
1. 校验 `grade_id` 对应的成绩记录存在且属于该学生
2. 校验成绩审核状态为"已通过"
3. 校验申诉时效（审核通过后30天内）
4. 校验该课程该学期未有其他申诉记录
5. 校验申诉理由至少20字

**后端逻辑：**
1. 创建申诉记录
2. 通过模块07发送站内信通知课程教师

**错误码：**
| code | 说明 |
|------|------|
| 40007 | 成绩尚未审核通过，不可申诉 |
| 40008 | 已超过申诉时效（30天） |
| 40902 | 该课程该学期已有申诉记录 |
| 40009 | 申诉理由不足20字 |

#### POST /api/v1/grades/appeals/:id/approve — 同意申诉

**权限：** 教师（课程负责人）

**请求体：**
```json
{
  "new_score": 92.00,
  "handle_comment": "经核实，第3题评分确有误差，已修正。"
}
```

**后端逻辑：**
1. 更新申诉状态为"已同意"
2. 更新 `student_semester_grades` 中的 `final_score`
3. 根据等级映射重新计算 `grade_level` 和 `gpa_point`
4. 重新计算该学生的学期GPA和累计GPA
5. 刷新GPA缓存
6. 检查学业预警是否需要解除
7. 通过模块07发送站内信通知学生"申诉已处理"

> **跨模块联动：** 申诉修改成绩后会触发GPA重算和预警检测，并通过模块07通知学生。

#### POST /api/v1/grades/appeals/:id/reject — 驳回申诉

**权限：** 教师（课程负责人）

**请求体：**
```json
{
  "handle_comment": "经核实，原评分正确，第3题的解答缺少关键步骤。"
}
```

---

### 2.6 学业预警

#### GET /api/v1/grades/warnings — 预警列表

**权限：** 学校管理员

**查询参数：** `semester_id`, `warning_type`, `status`, `keyword`(学生姓名/学号), `page`, `page_size`

**成功响应：**
```json
{
  "code": 200,
  "data": {
    "list": [
      {
        "id": "1880000000300",
        "student_id": "1780000000500",
        "student_name": "张三",
        "student_no": "2024001",
        "semester_name": "2025-2026学年第一学期",
        "warning_type": 1,
        "warning_type_text": "低GPA",
        "detail": { "current_gpa": 1.85, "threshold": 2.0 },
        "status": 1,
        "status_text": "待处理",
        "created_at": "2026-01-20T10:00:00Z"
      }
    ],
    "pagination": { "page": 1, "page_size": 20, "total": 5, "total_pages": 1 }
  }
}
```

#### POST /api/v1/grades/warnings/:id/handle — 处理预警

**权限：** 学校管理员

**请求体：**
```json
{
  "handle_note": "已约谈学生，制定学业改进计划。"
}
```

#### PUT /api/v1/grades/warning-configs — 更新预警配置

**权限：** 学校管理员

**请求体：**
```json
{
  "gpa_threshold": 2.00,
  "fail_count_threshold": 2,
  "is_enabled": true
}
```

---

### 2.7 成绩单

#### POST /api/v1/grades/transcripts/generate — 生成成绩单

**权限：** 学生（仅自己）/ 教师 / 学校管理员

**请求体：**
```json
{
  "student_id": "1780000000500",
  "semester_ids": ["1880000000001", "1880000000002"]
}
```

> 学生调用时 `student_id` 自动设为当前用户ID。

**后端逻辑：**
1. 查询指定学期的已审核成绩
2. 读取学校信息（名称、Logo）
3. 生成PDF文件，上传到MinIO
4. 创建 `transcript_records` 记录

**成功响应：**
```json
{
  "code": 200,
  "data": {
    "id": "1880000000400",
    "file_url": "/api/v1/grades/transcripts/1880000000400/download",
    "generated_at": "2026-04-09T10:00:00Z"
  }
}
```

---

### 2.8 成绩分析

#### GET /api/v1/grades/analytics/course/:id — 课程成绩分析

**权限：** 教师（课程负责人）

**查询参数：** `semester_id`

**成功响应：**
```json
{
  "code": 200,
  "data": {
    "course_id": "1780000000100",
    "course_name": "区块链原理与技术",
    "semester_name": "2025-2026学年第一学期",
    "student_count": 45,
    "average_score": 78.5,
    "median_score": 80.0,
    "max_score": 98.0,
    "min_score": 35.0,
    "pass_rate": 0.889,
    "grade_distribution": {
      "A": 8, "B": 15, "C": 12, "D": 5, "F": 5
    },
    "score_distribution": [
      { "range": "90-100", "count": 8 },
      { "range": "80-89", "count": 15 },
      { "range": "70-79", "count": 12 },
      { "range": "60-69", "count": 5 },
      { "range": "0-59", "count": 5 }
    ]
  }
}
```

#### GET /api/v1/grades/analytics/school — 全校成绩分析

**权限：** 学校管理员

**查询参数：** `semester_id`

**成功响应：**
```json
{
  "code": 200,
  "data": {
    "semester_name": "2025-2026学年第一学期",
    "total_students": 500,
    "total_courses": 30,
    "average_gpa": 2.85,
    "gpa_distribution": [
      { "range": "3.5-4.0", "count": 80 },
      { "range": "3.0-3.49", "count": 150 },
      { "range": "2.5-2.99", "count": 120 },
      { "range": "2.0-2.49", "count": 100 },
      { "range": "0-1.99", "count": 50 }
    ],
    "fail_rate": 0.10,
    "warning_count": 25,
    "top_courses": [
      { "course_name": "区块链原理", "average_score": 82.5, "pass_rate": 0.95 }
    ],
    "bottom_courses": [
      { "course_name": "密码学基础", "average_score": 65.3, "pass_rate": 0.72 }
    ]
  }
}
```

#### GET /api/v1/grades/analytics/platform — 平台成绩总览

**权限：** 超级管理员

**成功响应：**
```json
{
  "code": 200,
  "data": {
    "total_schools": 10,
    "total_students": 5000,
    "platform_average_gpa": 2.90,
    "school_comparison": [
      { "school_name": "XX大学", "student_count": 500, "average_gpa": 3.10 },
      { "school_name": "YY大学", "student_count": 800, "average_gpa": 2.75 }
    ]
  }
}
```

---

## 三、跨模块接口调用说明

### 3.1 本模块调用外部模块

| 调用目标 | 接口 | 场景 |
|---------|------|------|
| 模块03 | `GET /api/v1/courses/:id/grades` | 审核通过时读取课程成绩汇总数据 |
| 模块03 | `GET /api/v1/courses/:id/grade-config` | 读取成绩权重配置 |
| 模块07 | 通知发送接口（内部调用） | 审核通过/驳回通知、申诉处理通知、学业预警通知 |

### 3.2 外部模块调用本模块

| 调用来源 | 接口 | 场景 |
|---------|------|------|
| 模块08 | `GET /api/v1/grades/analytics/*` | 运维仪表盘展示成绩统计概览 |

---

*文档版本：v1.0*
*创建日期：2026-04-09*
*更新日期：2026-04-09*
