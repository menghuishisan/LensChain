# 课程与教学模块 — API 接口设计

> 模块状态：✅ 已确认
> 文档版本：v1.0
> 遵循规范：[API规范](../../standards/API规范.md)

---

## 一、接口总览

### 1.1 课程管理（教师）

| 方法 | 路径 | 说明 | 权限 |
|------|------|------|------|
| POST | /api/v1/courses | 创建课程 | 教师 |
| GET | /api/v1/courses | 课程列表（教师视角） | 教师 |
| GET | /api/v1/courses/:id | 课程详情 | 教师/学生 |
| PUT | /api/v1/courses/:id | 编辑课程信息 | 课程教师 |
| DELETE | /api/v1/courses/:id | 删除课程（仅草稿） | 课程教师 |
| POST | /api/v1/courses/:id/publish | 发布课程 | 课程教师 |
| POST | /api/v1/courses/:id/end | 结束课程 | 课程教师 |
| POST | /api/v1/courses/:id/archive | 归档课程 | 课程教师 |
| POST | /api/v1/courses/:id/clone | 克隆课程 | 教师 |
| PATCH | /api/v1/courses/:id/share | 设置共享状态 | 课程教师 |
| POST | /api/v1/courses/:id/invite-code/refresh | 刷新邀请码 | 课程教师 |

**查询参数补充：**

- `GET /api/v1/courses`
  - `page`：页码，从1开始
  - `page_size`：每页条数，默认20，最大100
  - `keyword`：按课程标题或主题搜索
  - `status`：课程状态筛选
    - `1`：草稿
    - `2`：已发布
    - `3`：进行中
    - `4`：已结束
    - `5`：已归档
  - `course_type`：课程类型筛选
    - `1`：理论课
    - `2`：实验课
    - `3`：混合课
    - `4`：研讨课

### 1.2 章节与课时

| 方法 | 路径 | 说明 | 权限 |
|------|------|------|------|
| GET | /api/v1/courses/:id/chapters | 获取章节列表（含课时） | 教师/学生 |
| POST | /api/v1/courses/:id/chapters | 创建章节 | 课程教师 |
| PUT | /api/v1/chapters/:id | 编辑章节 | 课程教师 |
| DELETE | /api/v1/chapters/:id | 删除章节 | 课程教师 |
| PUT | /api/v1/courses/:id/chapters/sort | 章节排序 | 课程教师 |
| POST | /api/v1/chapters/:id/lessons | 创建课时 | 课程教师 |
| GET | /api/v1/lessons/:id | 课时详情 | 教师/学生 |
| PUT | /api/v1/lessons/:id | 编辑课时 | 课程教师 |
| DELETE | /api/v1/lessons/:id | 删除课时 | 课程教师 |
| PUT | /api/v1/chapters/:id/lessons/sort | 课时排序 | 课程教师 |
| POST | /api/v1/lessons/:id/attachments | 上传课时附件 | 课程教师 |
| DELETE | /api/v1/lesson-attachments/:id | 删除附件 | 课程教师 |

### 1.3 选课管理

| 方法 | 路径 | 说明 | 权限 |
|------|------|------|------|
| POST | /api/v1/courses/join | 通过邀请码加入课程 | 学生 |
| POST | /api/v1/courses/:id/students | 教师添加学生 | 课程教师 |
| POST | /api/v1/courses/:id/students/batch | 批量添加学生 | 课程教师 |
| DELETE | /api/v1/courses/:id/students/:student_id | 移除学生 | 课程教师 |
| GET | /api/v1/courses/:id/students | 课程学生列表 | 课程教师 |

**查询参数补充：**

- `GET /api/v1/courses/:id/students`
  - `page`：页码，从1开始
  - `page_size`：每页条数，默认20，最大100
  - `keyword`：按学生姓名或学号搜索

### 1.4 作业管理

| 方法 | 路径 | 说明 | 权限 |
|------|------|------|------|
| POST | /api/v1/courses/:id/assignments | 创建作业 | 课程教师 |
| GET | /api/v1/courses/:id/assignments | 作业列表 | 教师/学生 |
| GET | /api/v1/assignments/:id | 作业详情（含题目） | 教师/学生 |
| PUT | /api/v1/assignments/:id | 编辑作业 | 课程教师 |
| DELETE | /api/v1/assignments/:id | 删除作业 | 课程教师 |
| POST | /api/v1/assignments/:id/publish | 发布作业 | 课程教师 |
| POST | /api/v1/assignments/:id/questions | 添加题目 | 课程教师 |
| PUT | /api/v1/assignment-questions/:id | 编辑题目 | 课程教师 |
| DELETE | /api/v1/assignment-questions/:id | 删除题目 | 课程教师 |

**查询参数补充：**

- `GET /api/v1/courses/:id/assignments`
  - `page`：页码，从1开始
  - `page_size`：每页条数，默认20，最大100
  - `assignment_type`：作业类型筛选
    - `1`：普通作业
    - `2`：测验

**字段可见性补充：**

- `GET /api/v1/assignments/:id`
  - 课程教师可查看题目的 `correct_answer`、`reference_answer`、`judge_config`
  - 学生访问同一接口时，不返回 `correct_answer`、`reference_answer`、`judge_config`

### 1.5 作业提交与批改

| 方法 | 路径 | 说明 | 权限 |
|------|------|------|------|
| PUT | /api/v1/assignments/:id/draft | 保存我的作答草稿 | 学生 |
| GET | /api/v1/assignments/:id/draft | 获取我的作答草稿 | 学生 |
| POST | /api/v1/assignments/:id/submit | 学生提交作业 | 学生 |
| GET | /api/v1/assignments/:id/my-submissions | 我的提交记录 | 学生 |
| GET | /api/v1/assignments/:id/submissions | 所有学生提交列表 | 课程教师 |
| GET | /api/v1/submissions/:id | 提交详情 | 教师/提交者 |
| POST | /api/v1/submissions/:id/grade | 批改提交 | 课程教师 |

**查询参数补充：**

- `GET /api/v1/assignments/:id/submissions`
  - `page`：页码，从1开始
  - `page_size`：每页条数，默认20，最大100
  - `status`：提交状态筛选
    - `1`：已提交
    - `2`：待批改
    - `3`：已批改
  - `keyword`：按学生姓名或学号搜索

### 1.6 学习进度

| 方法 | 路径 | 说明 | 权限 |
|------|------|------|------|
| POST | /api/v1/lessons/:id/progress | 更新学习进度 | 学生 |
| GET | /api/v1/courses/:id/my-progress | 我的课程学习进度 | 学生 |
| GET | /api/v1/courses/:id/students-progress | 所有学生学习进度 | 课程教师 |

**查询参数补充：**

- `GET /api/v1/courses/:id/students-progress`
  - `page`：页码，从1开始
  - `page_size`：每页条数，默认20，最大100
  - `keyword`：按学生姓名或学号搜索

### 1.7 课程表

| 方法 | 路径 | 说明 | 权限 |
|------|------|------|------|
| PUT | /api/v1/courses/:id/schedules | 设置课程表 | 课程教师 |
| GET | /api/v1/courses/:id/schedules | 获取课程表 | 教师/学生 |
| GET | /api/v1/my-schedule | 我的周课程表 | 学生/教师 |

### 1.8 公告

| 方法 | 路径 | 说明 | 权限 |
|------|------|------|------|
| POST | /api/v1/courses/:id/announcements | 发布公告 | 课程教师 |
| GET | /api/v1/courses/:id/announcements | 公告列表 | 教师/学生 |
| PUT | /api/v1/announcements/:id | 编辑公告 | 课程教师 |
| PATCH | /api/v1/announcements/:id/pin | 置顶/取消置顶公告 | 课程教师 |
| DELETE | /api/v1/announcements/:id | 删除公告 | 课程教师 |

**查询参数补充：**

- `GET /api/v1/courses/:id/announcements`
  - `page`：页码，从1开始
  - `page_size`：每页条数，默认20，最大100

### 1.9 讨论区

| 方法 | 路径 | 说明 | 权限 |
|------|------|------|------|
| POST | /api/v1/courses/:id/discussions | 发帖 | 教师/学生 |
| GET | /api/v1/courses/:id/discussions | 帖子列表 | 教师/学生 |
| GET | /api/v1/discussions/:id | 帖子详情（含回复） | 教师/学生 |
| DELETE | /api/v1/discussions/:id | 删除帖子 | 课程教师/发帖人 |
| PATCH | /api/v1/discussions/:id/pin | 置顶/取消置顶 | 课程教师 |
| POST | /api/v1/discussions/:id/replies | 回复帖子 | 教师/学生 |
| DELETE | /api/v1/discussion-replies/:id | 删除回复 | 课程教师/回复人 |
| POST | /api/v1/discussions/:id/like | 点赞 | 教师/学生 |
| DELETE | /api/v1/discussions/:id/like | 取消点赞 | 教师/学生 |

**查询参数补充：**

- `GET /api/v1/courses/:id/discussions`
  - `page`：页码，从1开始
  - `page_size`：每页条数，默认20，最大100

### 1.10 课程评价

| 方法 | 路径 | 说明 | 权限 |
|------|------|------|------|
| POST | /api/v1/courses/:id/evaluations | 提交评价 | 学生 |
| GET | /api/v1/courses/:id/evaluations | 评价列表 | 课程教师 |
| PUT | /api/v1/course-evaluations/:id | 修改评价 | 学生（评价人本人） |

**查询参数补充：**

- `GET /api/v1/courses/:id/evaluations`
  - `page`：页码，从1开始
  - `page_size`：每页条数，默认20，最大100

### 1.11 成绩管理

| 方法 | 路径 | 说明 | 权限 |
|------|------|------|------|
| PUT | /api/v1/courses/:id/grade-config | 配置成绩权重 | 课程教师 |
| GET | /api/v1/courses/:id/grade-config | 获取成绩权重配置 | 课程教师 |
| GET | /api/v1/courses/:id/grades | 成绩汇总表 | 课程教师 |
| PATCH | /api/v1/courses/:id/grades/:student_id | 手动调整成绩 | 课程教师 |
| GET | /api/v1/courses/:id/grades/export | 导出成绩单 | 课程教师 |
| GET | /api/v1/courses/:id/my-grades | 我的成绩 | 学生 |

### 1.12 共享课程库

| 方法 | 路径 | 说明 | 权限 |
|------|------|------|------|
| GET | /api/v1/shared-courses | 共享课程库列表 | 教师 |
| GET | /api/v1/shared-courses/:id | 共享课程详情 | 教师 |

**查询参数补充：**

- `GET /api/v1/shared-courses`
  - `page`：页码，从1开始
  - `page_size`：每页条数，默认20，最大100
  - `keyword`：按课程标题或主题搜索
  - `course_type`：课程类型筛选
  - `difficulty`：难度筛选
  - `topic`：主题精确筛选

### 1.13 课程统计

| 方法 | 路径 | 说明 | 权限 |
|------|------|------|------|
| GET | /api/v1/courses/:id/statistics/overview | 课程整体统计 | 课程教师 |
| GET | /api/v1/courses/:id/statistics/assignments | 作业统计 | 课程教师 |
| GET | /api/v1/courses/:id/statistics/export | 导出课程统计报告（含课程概览+作业统计） | 课程教师 |

### 1.14 学生视角

| 方法 | 路径 | 说明 | 权限 |
|------|------|------|------|
| GET | /api/v1/my-courses | 我的课程列表 | 学生 |

**查询参数补充：**

- `GET /api/v1/my-courses`
  - `page`：页码，从1开始
  - `page_size`：每页条数，默认20，最大100
  - `status`：课程状态筛选
    - `2`：已发布
    - `3`：进行中
    - `4`：已结束

---

## 二、核心接口详细定义

### 2.1 POST /api/v1/courses — 创建课程

**请求体：**

```json
{
  "title": "区块链技术与应用",
  "description": "本课程系统介绍区块链核心技术...",
  "cover_url": null,
  "course_type": 3,
  "difficulty": 2,
  "topic": "区块链基础",
  "credits": 3.0,
  "semester_id": "1880000000001",
  "start_at": "2026-09-01T00:00:00Z",
  "end_at": "2027-01-15T00:00:00Z",
  "max_students": null
}
```

**响应：**

```json
{
  "code": 200,
  "message": "创建成功",
  "data": {
    "id": "1780000000010001",
    "title": "区块链技术与应用",
    "status": 1,
    "status_text": "草稿",
    "invite_code": "A3B7K9",
    "cover_url": "https://oss.example.com/covers/auto_generated.png"
  }
}
```

> 创建时自动生成邀请码和默认封面。
> `credits` 和 `semester_id` 由模块03维护，供模块06在成绩审核和GPA计算时只读使用。

---

### 2.2 POST /api/v1/courses/join — 通过邀请码加入课程

**请求体：**

```json
{
  "invite_code": "A3B7K9"
}
```

**错误响应：**

| code | message | 场景 |
|------|---------|------|
| 40401 | 邀请码无效 | 邀请码不存在 |
| 40901 | 您已加入该课程 | 重复加入 |
| 40902 | 课程已结束，无法加入 | 课程状态为已结束/已归档 |
| 40903 | 课程人数已满 | 超过max_students |

---

### 2.3 GET /api/v1/courses/:id — 课程详情

**权限：** 课程教师

**说明：**

- 返回课程基础信息和课程目录结构
- 课程目录结构按章节排序返回，每个章节下包含课时列表
- `invite_code` 仅课程教师可见；学生访问课程详情时不返回该字段

**响应：**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "id": "1780000000010001",
    "title": "区块链技术与应用",
    "description": "本课程系统介绍区块链核心技术...",
    "cover_url": "https://oss.example.com/covers/course1.png",
    "course_type": 3,
    "course_type_text": "理论课",
    "difficulty": 2,
    "difficulty_text": "中级",
    "topic": "区块链基础",
    "status": 3,
    "status_text": "进行中",
    "is_shared": false,
    "credits": 3.0,
    "semester_id": "1880000000001",
    "start_at": "2026-09-01T00:00:00Z",
    "end_at": "2027-01-15T00:00:00Z",
    "max_students": 60,
    "student_count": 48,
    "teacher_id": "1780000000001001",
    "teacher_name": "张老师",
    "created_at": "2026-09-01T00:00:00Z",
    "updated_at": "2026-09-02T00:00:00Z",
    "chapters": [
      {
        "id": "1780000000011001",
        "title": "第一章 区块链概述",
        "description": "介绍区块链基础概念",
        "sort_order": 1,
        "lessons": [
          {
            "id": "1780000000012001",
            "title": "1.1 区块链发展史",
            "content_type": 1,
            "content_type_text": "视频",
            "video_duration": 1200,
            "experiment_id": null,
            "estimated_minutes": 25,
            "sort_order": 1
          }
        ]
      }
    ]
  }
}
```

---

### 2.4 POST /api/v1/assignments/:id/submit — 学生提交作业

**请求体：**

```json
{
  "answers": [
    {
      "question_id": "1780000000020001",
      "answer_content": "A"
    },
    {
      "question_id": "1780000000020002",
      "answer_content": "区块链是一种分布式账本技术..."
    },
    {
      "question_id": "1780000000020003",
      "answer_content": "pragma solidity ^0.8.0;\ncontract Hello { ... }",
    },
    {
      "question_id": "1780000000020004",
      "answer_file_url": "https://oss.example.com/reports/xxx.pdf"
    }
  ]
}
```

**响应（含即时反馈）：**

```json
{
  "code": 200,
  "message": "提交成功",
  "data": {
    "submission_id": "1780000000030001",
    "submission_no": 1,
    "remaining_submissions": 2,
    "is_late": false,
    "instant_feedback": {
      "auto_graded_score": 30,
      "auto_graded_total": 40,
      "details": [
        { "question_id": "1780000000020001", "is_correct": true, "score": 10 },
        { "question_id": "1780000000020002", "status": "pending_review" },
        { "question_id": "1780000000020003", "status": "judging" },
        { "question_id": "1780000000020004", "status": "pending_review" }
      ]
    }
  }
}
```

**错误响应：**

| code | message | 场景 |
|------|---------|------|
| 40300 | 课程未开始，暂不可提交作业 | 课程状态为已发布但未进入进行中 |
| 40015 | 作业已截止且不允许迟交 | 超过截止时间 |
| 40002 | 已达最大提交次数 | 超过max_submissions |
| 40003 | 作业未发布 | 作业未发布 |

---

### 2.5 PUT /api/v1/assignments/:id/draft — 保存我的作答草稿

**请求体：**

```json
{
  "answers": [
    {
      "question_id": "1780000000020001",
      "answer_content": "A"
    },
    {
      "question_id": "1780000000020002",
      "answer_content": "区块链是一种分布式账本技术..."
    },
    {
      "question_id": "1780000000020004",
      "answer_file_url": "https://oss.example.com/reports/xxx.pdf"
    }
  ]
}
```

**响应：**

```json
{
  "code": 200,
  "message": "草稿保存成功",
  "data": {
    "assignment_id": "1780000000020000",
    "saved_at": "2026-04-09T10:30:00Z",
    "answer_count": 3
  }
}
```

**说明：**

- 自动保存和手动保存共用该接口
- 接口语义为覆盖当前学生在该作业下的最新草稿
- 草稿不计入提交次数，不触发自动判分
- 仅课程处于“进行中”状态时允许写入服务端草稿；课程已发布但未开始时，前端仅可保留本地草稿
- 作业已截止且不允许迟交、课程已结束或已归档时，拒绝写入新草稿

**错误响应：**

| code | message | 场景 |
|------|---------|------|
| 40300 | 课程未开始，暂不可提交作业 | 课程状态为已发布但未进入进行中 |
| 40015 | 作业已截止且不允许迟交 | 超过截止时间 |
| 40003 | 作业未发布 | 作业未发布 |
| 40301 | 无权限访问该作业 | 非作业所属学生 |

---

### 2.6 GET /api/v1/assignments/:id/draft — 获取我的作答草稿

**响应：**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "assignment_id": "1780000000020000",
    "saved_at": "2026-04-09T10:30:00Z",
    "answers": [
      {
        "question_id": "1780000000020001",
        "answer_content": "A"
      },
      {
        "question_id": "1780000000020002",
        "answer_content": "区块链是一种分布式账本技术..."
      }
    ]
  }
}
```

**说明：**

- 当前学生在该作业下无服务端草稿时，返回 `data = null`
- 仅返回当前学生自己的草稿
- 正式提交成功后，历史草稿不再返回

---

### 2.7 POST /api/v1/lessons/:id/progress — 更新学习进度

**请求体：**

```json
{
  "status": 2,
  "video_progress": 360,
  "study_duration_increment": 120
}
```

> `study_duration_increment` 为本次增量（秒），后端累加到总时长。
> 前端每隔30秒上报一次进度。

---

### 2.8 GET /api/v1/my-schedule — 我的周课程表

**响应：**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "schedules": [
      {
        "course_id": "1780000000010001",
        "course_title": "区块链技术与应用",
        "day_of_week": 3,
        "start_time": "08:00",
        "end_time": "09:40",
        "location": "教学楼A301",
        "teacher_name": "李教授"
      },
      {
        "course_id": "1780000000010002",
        "course_title": "智能合约安全",
        "day_of_week": 5,
        "start_time": "14:00",
        "end_time": "15:40",
        "location": "实验楼B201",
        "teacher_name": "王教授"
      }
    ]
  }
}
```

---

### 2.9 POST /api/v1/courses/:id/announcements — 发布公告

**权限：** 课程教师

**请求体：**

```json
{
  "title": "期末测验安排通知",
  "content": "本周五晚上8点开放期末测验，请提前完成设备检查。"
}
```

**响应：**

```json
{
  "code": 201,
  "message": "created",
  "data": {
    "id": "1780000000080001"
  }
}
```

**错误响应：**

| code | message | 场景 |
|------|---------|------|
| 40300 | 课程已结束，不可继续互动 | 课程已结束 |
| 40300 | 课程当前状态不可互动 | 课程未发布等不可互动状态 |

---

### 2.10 PUT /api/v1/announcements/:id — 编辑公告

**权限：** 课程教师

**请求体：**

```json
{
  "title": "期末测验安排通知（更新）",
  "content": "本周五晚上8点开放期末测验，请提前30分钟完成设备检查。"
}
```

**说明：**

- `title`、`content` 至少传一个字段
- 仅更新传入字段，未传字段保持不变

**错误响应：**

| code | message | 场景 |
|------|---------|------|
| 40426 | 公告不存在 | 公告ID不存在 |
| 40300 | 课程已结束，不可继续互动 | 课程已结束 |
| 40300 | 课程当前状态不可互动 | 课程未发布等不可互动状态 |

---

### 2.11 PATCH /api/v1/announcements/:id/pin — 置顶/取消置顶公告

**权限：** 课程教师

**请求体：**

```json
{
  "is_pinned": true
}
```

**响应：**

```json
{
  "code": 200,
  "message": "操作成功",
  "data": null
}
```

---

### 2.12 DELETE /api/v1/announcements/:id — 删除公告

**权限：** 课程教师

**响应：**

```json
{
  "code": 200,
  "message": "删除成功",
  "data": null
}
```

**错误响应：**

| code | message | 场景 |
|------|---------|------|
| 40426 | 公告不存在 | 公告ID不存在 |
| 40300 | 课程已结束，不可继续互动 | 课程已结束 |
| 40300 | 课程当前状态不可互动 | 课程未发布等不可互动状态 |

---

### 2.13 GET /api/v1/courses/:id/announcements — 公告列表

**权限：** 教师/学生

**响应：**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "list": [
      {
        "id": "1780000000080001",
        "title": "期末测验安排通知",
        "content": "本周五晚上8点开放期末测验，请提前完成设备检查。",
        "is_pinned": true,
        "teacher_name": "张老师",
        "created_at": "2026-12-20T09:00:00Z",
        "updated_at": "2026-12-20T10:30:00Z"
      }
    ],
    "pagination": {
      "page": 1,
      "page_size": 20,
      "total": 1,
      "total_page": 1
    }
  }
}
```

**说明：**

- 列表按 `is_pinned DESC, created_at DESC` 排序
- 置顶公告始终排在普通公告之前

---

### 2.14 POST /api/v1/courses/:id/discussions — 发帖

**权限：** 教师/学生

**请求体：**

```json
{
  "title": "如何理解区块链中的最终一致性？",
  "content": "在PBFT和PoW场景下，最终一致性的体现有什么不同？"
}
```

**响应：**

```json
{
  "code": 201,
  "message": "created",
  "data": {
    "id": "1780000000070001"
  }
}
```

**错误响应：**

| code | message | 场景 |
|------|---------|------|
| 40300 | 课程已结束，不可继续互动 | 课程已结束 |
| 40300 | 课程当前状态不可互动 | 课程未发布等不可互动状态 |

---

### 2.15 GET /api/v1/courses/:id/discussions — 帖子列表

**权限：** 教师/学生

**响应：**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "list": [
      {
        "id": "1780000000070001",
        "title": "如何理解区块链中的最终一致性？",
        "author_id": "1780000000000001",
        "author_name": "张三",
        "is_pinned": true,
        "reply_count": 8,
        "like_count": 15,
        "is_liked": true,
        "last_replied_at": "2026-12-27T14:20:00Z",
        "created_at": "2026-12-26T09:00:00Z"
      }
    ],
    "pagination": {
      "page": 1,
      "page_size": 20,
      "total": 1,
      "total_page": 1
    }
  }
}
```

**说明：**

- 列表固定按 `is_pinned DESC, last_replied_at DESC NULLS LAST, created_at DESC` 排序
- `is_liked` 表示当前登录用户是否已点赞该帖子

---

### 2.16 GET /api/v1/discussions/:id — 帖子详情（含回复）

**权限：** 教师/学生

**响应：**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "id": "1780000000070001",
    "course_id": "1780000000010001",
    "title": "如何理解区块链中的最终一致性？",
    "content": "在PBFT和PoW场景下，最终一致性的体现有什么不同？",
    "author_id": "1780000000000001",
    "author_name": "张三",
    "is_pinned": false,
    "reply_count": 2,
    "like_count": 5,
    "is_liked": false,
    "created_at": "2026-12-26T09:00:00Z",
    "replies": [
      {
        "id": "1780000000071001",
        "author_id": "1780000000001001",
        "author_name": "李老师",
        "content": "可以先区分概率最终一致性和确定性最终一致性。",
        "reply_to_id": null,
        "reply_to_name": null,
        "created_at": "2026-12-26T10:00:00Z"
      },
      {
        "id": "1780000000071002",
        "author_id": "1780000000000002",
        "author_name": "王五",
        "content": "我理解 PoW 更依赖确认块数。",
        "reply_to_id": "1780000000071001",
        "reply_to_name": "李老师",
        "created_at": "2026-12-26T10:10:00Z"
      }
    ]
  }
}
```

---

### 2.17 DELETE /api/v1/discussions/:id — 删除帖子

**权限：** 课程教师 / 发帖人

**响应：**

```json
{
  "code": 200,
  "message": "删除成功",
  "data": null
}
```

**错误响应：**

| code | message | 场景 |
|------|---------|------|
| 40409 | 讨论帖不存在 | 讨论帖ID不存在 |
| 40300 | 无权限访问 | 非发帖人且非课程教师 |
| 40300 | 课程已结束，不可继续互动 | 课程已结束 |
| 40300 | 课程当前状态不可互动 | 课程未发布等不可互动状态 |

---

### 2.18 PATCH /api/v1/discussions/:id/pin — 置顶/取消置顶

**权限：** 课程教师

**请求体：**

```json
{
  "is_pinned": true
}
```

**响应：**

```json
{
  "code": 200,
  "message": "操作成功",
  "data": null
}
```

---

### 2.19 POST /api/v1/discussions/:id/replies — 回复帖子

**权限：** 教师/学生

**请求体：**

```json
{
  "content": "我理解 PoW 更依赖确认块数。",
  "reply_to_id": "1780000000071001"
}
```

**说明：**

- `reply_to_id` 可为空；为空时表示直接回复主帖
- 传入 `reply_to_id` 时，必须是当前帖子的有效回复ID

**响应：**

```json
{
  "code": 201,
  "message": "created",
  "data": {
    "id": "1780000000071002"
  }
}
```

**错误响应：**

| code | message | 场景 |
|------|---------|------|
| 40409 | 讨论帖不存在 | 讨论帖ID不存在 |
| 40412 | 回复不存在 | `reply_to_id` 不存在 |
| 40001 | 回复对象不属于当前讨论帖 | `reply_to_id` 不属于当前帖子 |
| 40300 | 课程已结束，不可继续互动 | 课程已结束 |
| 40300 | 课程当前状态不可互动 | 课程未发布等不可互动状态 |

---

### 2.20 DELETE /api/v1/discussion-replies/:id — 删除回复

**权限：** 课程教师 / 回复人

**响应：**

```json
{
  "code": 200,
  "message": "删除成功",
  "data": null
}
```

**错误响应：**

| code | message | 场景 |
|------|---------|------|
| 40412 | 回复不存在 | 回复ID不存在 |
| 40300 | 无权限访问 | 非回复人且非课程教师 |
| 40300 | 课程已结束，不可继续互动 | 课程已结束 |
| 40300 | 课程当前状态不可互动 | 课程未发布等不可互动状态 |

---

### 2.21 POST /api/v1/discussions/:id/like — 点赞

**权限：** 教师/学生

**响应：**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "liked": true
  }
}
```

**说明：**

- 已点赞时再次请求仍返回 `liked = true`，接口保持幂等

---

### 2.22 DELETE /api/v1/discussions/:id/like — 取消点赞

**权限：** 教师/学生

**响应：**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "liked": false
  }
}
```

**说明：**

- 未点赞时取消点赞仍返回 `liked = false`，接口保持幂等

---

### 2.23 POST /api/v1/courses/:id/evaluations — 提交课程评价

**权限：** 学生

**请求体：**

```json
{
  "rating": 5,
  "comment": "课程内容扎实，案例很多"
}
```

**说明：**

- 仅课程已结束后允许评价
- 同一学生对同一课程仅允许评价一次
- `comment` 可为空

**响应：**

```json
{
  "code": 201,
  "message": "created",
  "data": {
    "id": "1780000000090001"
  }
}
```

**错误响应：**

| code | message | 场景 |
|------|---------|------|
| 40001 | 仅已结束的课程允许评价 | 课程未结束 |
| 40916 | 已评价过该课程 | 重复评价 |

---

### 2.24 GET /api/v1/courses/:id/evaluations — 课程评价列表

**权限：** 课程教师

**响应：**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "summary": {
      "avg_rating": 4.6,
      "total_count": 18,
      "distribution": [0, 1, 2, 6, 9]
    },
    "items": [
      {
        "id": "1780000000090001",
        "student_id": "1780000000000001",
        "student_name": "张三",
        "rating": 5,
        "comment": "课程内容扎实，案例很多",
        "created_at": "2026-12-28T09:00:00Z"
      }
    ],
    "pagination": {
      "page": 1,
      "page_size": 20,
      "total": 18,
      "total_page": 1
    }
  }
}
```

---

### 2.25 PUT /api/v1/course-evaluations/:id — 修改评价

**权限：** 学生（评价人本人）

**请求体：**

```json
{
  "rating": 4,
  "comment": "课程内容扎实，希望增加更多链上实验案例。"
}
```

**说明：**

- `rating`、`comment` 至少传一个字段
- 仅评价人本人可修改
- 仅课程已结束后允许修改评价

**错误响应：**

| code | message | 场景 |
|------|---------|------|
| 40413 | 评价不存在 | 评价ID不存在 |
| 40300 | 无权限访问 | 非评价人本人 |
| 40001 | 仅已结束的课程允许评价 | 课程未结束 |

---

### 2.26 PUT /api/v1/courses/:id/grade-config — 配置成绩权重

**权限：** 课程教师

**请求体：**

```json
{
  "items": [
    { "assignment_id": "1780000000001", "name": "作业1", "weight": 20 },
    { "assignment_id": "1780000000002", "name": "实验报告", "weight": 30 },
    { "assignment_id": "1780000000003", "name": "期末测验", "weight": 50 }
  ]
}
```

**说明：**

- `items` 中每一项必须引用当前课程下的作业
- 同一作业在成绩配置中只允许出现一次
- 所有权重之和必须等于 `100`
- 若模块06已锁定该课程成绩，则模块03不得再修改成绩权重配置

**响应：**

```json
{
  "code": 200,
  "message": "设置成功",
  "data": null
}
```

**错误响应：**

| code | message | 场景 |
|------|---------|------|
| 40001 | 权重总和必须为100% | 权重总和不为100 |
| 40001 | 成绩配置中存在重复作业 | 同一作业重复出现 |
| 40001 | 成绩配置中的作业不属于当前课程 | 引用了其他课程的作业 |
| 40001 | 成绩配置中的作业不存在 | assignment_id 无效或不存在 |
| 40941 | 成绩已锁定，如需修改请联系学校管理员解锁 | 模块06已锁定课程成绩 |

---

### 2.27 GET /api/v1/courses/:id/grade-config — 获取成绩权重配置

**权限：** 课程教师

**响应：**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "items": [
      { "assignment_id": "1780000000001", "name": "作业1", "weight": 20 },
      { "assignment_id": "1780000000002", "name": "实验报告", "weight": 30 },
      { "assignment_id": "1780000000003", "name": "期末测验", "weight": 50 }
    ]
  }
}
```

**说明：**

- 若当前课程尚未配置成绩权重，返回空数组 `items: []`

---

### 2.28 GET /api/v1/courses/:id/grades — 成绩汇总表

**权限：** 课程教师

**响应：**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "grade_config": {
      "items": [
        { "assignment_id": "1780000000001", "name": "作业1", "weight": 20 },
        { "assignment_id": "1780000000002", "name": "实验报告", "weight": 30 },
        { "assignment_id": "1780000000003", "name": "期末测验", "weight": 50 }
      ]
    },
    "students": [
      {
        "student_id": "1780000000000001",
        "student_name": "张三",
        "student_no": "2024001",
        "scores": {
          "1780000000001": 85,
          "1780000000002": 90,
          "1780000000003": 78
        },
        "weighted_total": 82.9,
        "final_score": 82.9,
        "is_adjusted": false
      }
    ]
  }
}
```

**说明：**

- 成绩计算以“每名学生每个作业的最后一次提交”为准
- `weighted_total` 为系统自动计算值
- `final_score` 为最终成绩；若存在手动调分记录，则返回调整后的结果
- `is_adjusted = true` 表示当前学生成绩已被教师手动覆盖

---

### 2.29 PATCH /api/v1/courses/:id/grades/:student_id — 手动调整成绩

**权限：** 课程教师

**请求体：**

```json
{
  "final_score": 88.5,
  "reason": "课堂表现优异，教师酌情调整"
}
```

**响应：**

```json
{
  "code": 200,
  "message": "成绩调整成功",
  "data": null
}
```

**说明：**

- 调整后需保留自动计算值作为参考，不覆盖 `weighted_total`
- 若模块06已锁定该课程成绩，则模块03不得再修改

**错误响应：**

| code | message | 场景 |
|------|---------|------|
| 40941 | 成绩已锁定，如需修改请联系学校管理员解锁 | 模块06已锁定课程成绩 |
| 40304 | 未加入该课程 | 目标学生不属于当前课程 |

---

### 2.30 GET /api/v1/courses/:id/grades/export — 导出成绩单

**权限：** 课程教师

**说明：**

- 返回 Excel 文件下载，不返回 JSON 数据
- 文件包含所有学生的各项成绩、加权总成绩、最终成绩、是否已调整

**响应头：**

```http
Content-Type: application/vnd.openxmlformats-officedocument.spreadsheetml.sheet
Content-Disposition: attachment; filename*=UTF-8''课程成绩单.xlsx
```

---

### 2.31 GET /api/v1/courses/:id/my-grades — 我的成绩

**权限：** 学生

**响应：**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "grade_config": {
      "items": [
        { "assignment_id": "1780000000001", "name": "作业1", "weight": 20 },
        { "assignment_id": "1780000000002", "name": "实验报告", "weight": 30 },
        { "assignment_id": "1780000000003", "name": "期末测验", "weight": 50 }
      ]
    },
    "scores": {
      "1780000000001": 85,
      "1780000000002": 90,
      "1780000000003": 78
    },
    "weighted_total": 82.9,
    "final_score": 82.9,
    "is_adjusted": false
  }
}
```

**说明：**

- `weighted_total` 为系统按权重自动计算的成绩
- `final_score` 为最终成绩；若教师手动调整过，则该字段返回调整后的结果
- `is_adjusted = true` 表示当前最终成绩已被教师覆盖调整

---

### 2.32 GET /api/v1/shared-courses/:id — 共享课程详情

**权限：** 教师

**说明：**

- 仅返回当前仍在共享课程库中可见的课程
- 返回共享课程的基础信息、教师与学校信息、评分、课程目录结构
- 不返回课程邀请码等仅课程教师可用的数据

**响应：**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "id": "1780000000010001",
    "title": "区块链技术与应用",
    "description": "本课程系统介绍区块链核心技术...",
    "cover_url": "https://oss.example.com/covers/course1.png",
    "course_type": 3,
    "course_type_text": "理论课",
    "difficulty": 2,
    "difficulty_text": "中级",
    "topic": "区块链基础",
    "status": 3,
    "status_text": "进行中",
    "credits": 3.0,
    "start_at": "2026-09-01T00:00:00Z",
    "end_at": "2027-01-15T00:00:00Z",
    "max_students": 60,
    "student_count": 48,
    "teacher_name": "张老师",
    "school_name": "链镜实验中学",
    "rating": 4.7,
    "chapters": [
      {
        "id": "1780000000011001",
        "title": "第一章 区块链概述",
        "description": "介绍区块链基础概念",
        "sort_order": 1,
        "lessons": [
          {
            "id": "1780000000012001",
            "title": "1.1 区块链发展史",
            "content_type": 1,
            "content_type_text": "视频",
            "video_duration": 1200,
            "experiment_id": null,
            "estimated_minutes": 25,
            "sort_order": 1
          }
        ]
      }
    ]
  }
}
```

---

### 2.33 GET /api/v1/courses/:id/statistics/overview — 课程整体统计

**权限：** 课程教师

**响应：**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "student_count": 48,
    "lesson_count": 16,
    "assignment_count": 5,
    "avg_progress": 67.5,
    "avg_score": 81.2,
    "completion_rate": 43.75,
    "activity_rate": 85.42,
    "total_study_hours": 326.5,
    "progress_distribution": {
      "not_started_rate": 14.58,
      "in_progress_rate": 41.67,
      "completed_rate": 43.75
    }
  }
}
```

> `activity_rate` 定义为：课程已选学生中，至少产生过一次学习进度记录的学生占比。
> `progress_distribution` 以学生为单位统计：
> - 未开始：没有任何学习进度记录
> - 进行中：已有学习进度记录，但未完成全部课时
> - 已完成：已完成课时数等于课程总课时数

---

### 2.34 GET /api/v1/courses/:id/statistics/assignments — 作业统计

**权限：** 课程教师

**响应：**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "assignments": [
      {
        "id": "1780000000012001",
        "title": "作业1：区块链基础",
        "submit_count": 42,
        "total_students": 48,
        "submit_rate": 87.5,
        "avg_score": 83.4,
        "max_score": 98,
        "min_score": 61,
        "score_distribution": [
          { "range": "90-100", "count": 12 },
          { "range": "80-89", "count": 15 },
          { "range": "70-79", "count": 8 },
          { "range": "60-69", "count": 5 },
          { "range": "0-59", "count": 2 }
        ]
      }
    ]
  }
}
```

> `submit_count`、`submit_rate`、分数统计、分数分布均以“每名学生最后一次提交”为统计口径。
> `score_distribution` 只统计已有有效得分的最新提交，分段固定为 `90-100`、`80-89`、`70-79`、`60-69`、`0-59`。

---

### 2.35 GET /api/v1/courses/:id/statistics/export — 导出课程统计报告

**权限：** 课程教师

**说明：**

- 返回 Excel 文件下载，不返回 JSON 数据
- 文件至少包含两个工作表：
  - `课程概览`：课程整体统计、学习进度分布
  - `作业统计`：每次作业的提交率、平均分、最高分、最低分、分数分布
- 课程概览工作表中的统计口径与 `GET /api/v1/courses/:id/statistics/overview` 保持一致
- 作业统计工作表中的统计口径与 `GET /api/v1/courses/:id/statistics/assignments` 保持一致

**响应头：**

```http
Content-Type: application/vnd.openxmlformats-officedocument.spreadsheetml.sheet
Content-Disposition: attachment; filename*=UTF-8''课程统计报告.xlsx
```

---

### 2.36 GET /api/v1/courses — 课程列表（教师视角）

**权限：** 教师

**查询参数：**

- `page`：页码，从1开始
- `page_size`：每页条数，默认20，最大100
- `keyword`：按课程标题或主题搜索
- `status`：课程状态，`1` 草稿 / `2` 已发布 / `3` 进行中 / `4` 已结束 / `5` 已归档
- `course_type`：课程类型，`1` 理论课 / `2` 实验课 / `3` 混合课 / `4` 研讨课

**响应：**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "list": [
      {
        "id": "1780000000010001",
        "title": "区块链技术与应用",
        "cover_url": "https://oss.example.com/covers/course1.png",
        "course_type": 3,
        "course_type_text": "混合课",
        "difficulty": 2,
        "difficulty_text": "中级",
        "topic": "区块链基础",
        "status": 3,
        "status_text": "进行中",
        "is_shared": true,
        "student_count": 48,
        "start_at": "2026-09-01T00:00:00Z",
        "end_at": "2027-01-15T00:00:00Z",
        "created_at": "2026-08-20T09:00:00Z"
      }
    ],
    "pagination": {
      "page": 1,
      "page_size": 20,
      "total": 1,
      "total_page": 1
    }
  }
}
```

---

### 2.37 PUT /api/v1/courses/:id — 编辑课程信息

**权限：** 课程教师

**请求体：**

```json
{
  "title": "区块链技术与应用（更新）",
  "description": "补充智能合约安全与链上实验内容",
  "course_type": 3,
  "difficulty": 3,
  "topic": "区块链进阶",
  "credits": 3.5,
  "semester_id": "1880000000001",
  "start_at": "2026-09-01T00:00:00Z",
  "end_at": "2027-01-15T00:00:00Z",
  "max_students": 80
}
```

**说明：**

- 按字段增量更新，未传字段保持不变
- 草稿、已发布、进行中课程允许编辑基础信息；归档课程仅允许查看，不允许再编辑

**错误响应：**

| code | message | 场景 |
|------|---------|------|
| 40402 | 课程不存在 | 课程ID不存在 |
| 40300 | 无权限访问 | 非课程教师 |
| 40001 | 归档课程不允许修改 | 课程状态为已归档 |

---

### 2.38 DELETE /api/v1/courses/:id — 删除课程

**权限：** 课程教师

**说明：**

- 仅草稿课程允许删除
- 删除为软删除，不物理移除数据库记录

**错误响应：**

| code | message | 场景 |
|------|---------|------|
| 40402 | 课程不存在 | 课程ID不存在 |
| 40300 | 无权限访问 | 非课程教师 |
| 40904 | 仅草稿状态的课程可删除 | 课程不是草稿状态 |

---

### 2.39 POST /api/v1/courses/:id/publish — 发布课程

**权限：** 课程教师

**说明：**

- 发布前必须满足“至少有1个章节且至少有1个课时”
- 发布成功后课程状态变为“已发布”

**响应：**

```json
{
  "code": 200,
  "message": "发布成功",
  "data": null
}
```

**错误响应：**

| code | message | 场景 |
|------|---------|------|
| 40402 | 课程不存在 | 课程ID不存在 |
| 40001 | 请至少添加一个章节和一个课时 | 课程无章节或无课时 |
| 40905 | 当前课程状态不可发布 | 非草稿状态重复发布 |

---

### 2.40 POST /api/v1/courses/:id/end — 结束课程

**权限：** 课程教师

**说明：**

- 仅进行中课程允许手动结束
- 结束后学生不可再提交作业，但教师仍可批改和导出数据

**错误响应：**

| code | message | 场景 |
|------|---------|------|
| 40402 | 课程不存在 | 课程ID不存在 |
| 40906 | 当前课程状态不可结束 | 非进行中状态 |

---

### 2.41 POST /api/v1/courses/:id/archive — 归档课程

**权限：** 课程教师

**说明：**

- 仅已结束课程允许归档
- 归档后学生不可见，仅教师可查看和导出数据

**错误响应：**

| code | message | 场景 |
|------|---------|------|
| 40402 | 课程不存在 | 课程ID不存在 |
| 40907 | 当前课程状态不可归档 | 非已结束状态 |

---

### 2.42 POST /api/v1/courses/:id/clone — 克隆课程

**权限：** 教师

**说明：**

- 支持克隆自己的课程和共享课程库中的课程
- 克隆结果固定为新草稿课程
- 包含：课程基础信息、章节结构、课时内容、作业题目
- 不包含：学生名单、提交记录、成绩数据、讨论帖子、公告
- 大型课程允许异步完成；最终通知接入统一异步/通知体系

**响应：**

```json
{
  "code": 201,
  "message": "created",
  "data": {
    "id": "1780000000099001"
  }
}
```

---

### 2.43 PATCH /api/v1/courses/:id/share — 设置共享状态

**权限：** 课程教师

**请求体：**

```json
{
  "is_shared": true
}
```

**说明：**

- 仅已发布、进行中、已结束课程允许设置共享
- 草稿课程禁止共享

**错误响应：**

| code | message | 场景 |
|------|---------|------|
| 40402 | 课程不存在 | 课程ID不存在 |
| 40908 | 仅已发布/进行中/已结束的课程可共享 | 草稿课程尝试共享 |

---

### 2.44 POST /api/v1/courses/:id/invite-code/refresh — 刷新邀请码

**权限：** 课程教师

**说明：**

- 生成新的6位字母数字邀请码
- 刷新后旧邀请码立即失效

**响应：**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "invite_code": "H8M2Q7"
  }
}
```

---

### 2.45 GET /api/v1/courses/:id/chapters — 获取章节列表（含课时）

**权限：** 教师/学生

**说明：**

- 返回当前课程完整目录树
- 章节按 `sort_order ASC` 排序
- 课时按 `sort_order ASC` 排序

**响应：**

```json
{
  "code": 200,
  "message": "success",
  "data": [
    {
      "id": "1780000000011001",
      "title": "第一章 区块链概述",
      "description": "介绍区块链基础概念",
      "sort_order": 1,
      "lessons": [
        {
          "id": "1780000000012001",
          "title": "1.1 区块链发展史",
          "content_type": 1,
          "content_type_text": "视频",
          "video_duration": 1200,
          "experiment_id": null,
          "estimated_minutes": 25,
          "sort_order": 1
        }
      ]
    }
  ]
}
```

---

### 2.46 POST /api/v1/courses/:id/chapters — 创建章节

**权限：** 课程教师

**请求体：**

```json
{
  "title": "第一章 区块链概述",
  "description": "介绍区块链基础概念"
}
```

**响应：**

```json
{
  "code": 201,
  "message": "created",
  "data": {
    "id": "1780000000011001"
  }
}
```

---

### 2.47 PUT /api/v1/courses/:id/chapters/sort — 章节排序

**权限：** 课程教师

**请求体：**

```json
{
  "ids": [
    "1780000000011002",
    "1780000000011001"
  ]
}
```

**说明：**

- `ids` 必须覆盖当前课程全部章节且不得重复
- 排序结果立即持久化

---

### 2.48 PUT /api/v1/chapters/:id — 编辑章节

**权限：** 课程教师

**请求体：**

```json
{
  "title": "第一章 区块链基础概念",
  "description": "补充区块、链、共识机制基本概念"
}
```

**说明：**

- 按字段增量更新，未传字段保持不变

---

### 2.49 DELETE /api/v1/chapters/:id — 删除章节

**权限：** 课程教师

**说明：**

- 删除章节时一并删除其下课时
- 已归档课程不允许删除内容

**错误响应：**

| code | message | 场景 |
|------|---------|------|
| 40403 | 章节不存在 | 章节ID不存在 |
| 40001 | 归档课程不允许修改内容 | 所属课程已归档 |

---

### 2.50 POST /api/v1/chapters/:id/lessons — 创建课时

**权限：** 课程教师

**请求体：**

```json
{
  "title": "1.1 区块链发展史",
  "content_type": 1,
  "content": null,
  "video_url": "https://oss.example.com/videos/lesson-1.mp4",
  "video_duration": 1200,
  "experiment_id": null,
  "estimated_minutes": 25
}
```

**说明：**

- `content_type`：`1` 视频 / `2` 图文 / `3` 附件 / `4` 实验
- 不同类型按需填写 `content`、`video_url`、`experiment_id`

**响应：**

```json
{
  "code": 201,
  "message": "created",
  "data": {
    "id": "1780000000012001"
  }
}
```

---

### 2.51 PUT /api/v1/chapters/:id/lessons/sort — 课时排序

**权限：** 课程教师

**请求体：**

```json
{
  "ids": [
    "1780000000012002",
    "1780000000012001"
  ]
}
```

**说明：**

- `ids` 必须覆盖当前章节全部课时且不得重复

---

### 2.52 GET /api/v1/lessons/:id — 课时详情

**权限：** 教师/学生

**响应：**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "id": "1780000000012001",
    "chapter_id": "1780000000011001",
    "course_id": "1780000000010001",
    "title": "1.1 区块链发展史",
    "content_type": 1,
    "content_type_text": "视频",
    "content": null,
    "video_url": "https://oss.example.com/videos/lesson-1.mp4",
    "video_duration": 1200,
    "experiment_id": null,
    "estimated_minutes": 25,
    "sort_order": 1,
    "attachments": []
  }
}
```

---

### 2.53 PUT /api/v1/lessons/:id — 编辑课时

**权限：** 课程教师

**请求体：**

```json
{
  "title": "1.1 区块链发展史（更新）",
  "estimated_minutes": 30
}
```

**说明：**

- 按字段增量更新，未传字段保持不变

---

### 2.54 DELETE /api/v1/lessons/:id — 删除课时

**权限：** 课程教师

**说明：**

- 删除课时时一并删除课时附件与学习进度引用数据
- 已归档课程不允许删除内容

---

### 2.55 POST /api/v1/lessons/:id/attachments — 上传课时附件

**权限：** 课程教师

**请求体：**

```json
{
  "file_name": "课件-区块链概述.pdf",
  "file_url": "https://oss.example.com/files/lesson-1.pdf",
  "file_size": 1048576,
  "file_type": "application/pdf"
}
```

**说明：**

- 文件实际上传由对象存储完成，本接口只保存附件元数据
- 安全限制遵循验收标准：视频≤500MB，文档≤50MB

**响应：**

```json
{
  "code": 201,
  "message": "created",
  "data": {
    "id": "1780000000013001"
  }
}
```

---

### 2.56 DELETE /api/v1/lesson-attachments/:id — 删除课时附件

**权限：** 课程教师

**说明：**

- 删除附件记录，不删除关联课时主体

---

### 2.57 POST /api/v1/courses/:id/students — 教师添加学生

**权限：** 课程教师

**请求体：**

```json
{
  "student_id": "1780000000002001"
}
```

**说明：**

- 若学生已在课程中，接口保持幂等并返回成功
- 同样受课程人数上限约束

---

### 2.58 POST /api/v1/courses/:id/students/batch — 批量添加学生

**权限：** 课程教师

**请求体：**

```json
{
  "student_ids": [
    "1780000000002001",
    "1780000000002002"
  ]
}
```

**说明：**

- 已存在的学生自动忽略
- 任一学生不属于当前学校时整体拒绝

---

### 2.59 DELETE /api/v1/courses/:id/students/:student_id — 移除学生

**权限：** 课程教师

**说明：**

- 已存在历史提交、成绩、进度数据的学生仍允许移出选课关系，但历史记录保留用于审计和统计

---

### 2.60 GET /api/v1/courses/:id/students — 课程学生列表

**权限：** 课程教师

**查询参数：**

- `page`
- `page_size`
- `keyword`

**响应：**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "list": [
      {
        "id": "1780000000002001",
        "name": "张三",
        "student_no": "2024001",
        "college": "计算机学院",
        "major": "区块链工程",
        "class_name": "区块链2401",
        "join_method": 1,
        "join_method_text": "邀请码",
        "joined_at": "2026-09-03T10:00:00Z",
        "progress": 56.25
      }
    ],
    "pagination": {
      "page": 1,
      "page_size": 20,
      "total": 1,
      "total_page": 1
    }
  }
}
```

---

### 2.61 POST /api/v1/courses/:id/assignments — 创建作业

**权限：** 课程教师

**请求体：**

```json
{
  "title": "作业1：区块链基础",
  "description": "完成基础概念练习与简答题",
  "chapter_id": "1780000000011001",
  "assignment_type": 1,
  "deadline_at": "2026-10-10T23:59:59Z",
  "max_submissions": 3,
  "late_policy": 2,
  "late_deduction_per_day": 10
}
```

**说明：**

- 新建后默认为未发布状态
- `total_score` 不允许手工传入，初始值为 `0`，由题目分值自动累计
- 发布前必须至少包含一道题目

**响应：**

```json
{
  "code": 201,
  "message": "created",
  "data": {
    "id": "1780000000020000"
  }
}
```

---

### 2.62 GET /api/v1/courses/:id/assignments — 作业列表

**权限：** 教师/学生

**查询参数：**

- `page`
- `page_size`
- `assignment_type`

**响应：**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "list": [
      {
        "id": "1780000000020000",
        "title": "作业1：区块链基础",
        "assignment_type": 1,
        "assignment_type_text": "普通作业",
        "total_score": 100,
        "deadline_at": "2026-10-10T23:59:59Z",
        "is_published": true,
        "submit_count": 42,
        "total_students": 48,
        "sort_order": 1
      }
    ],
    "pagination": {
      "page": 1,
      "page_size": 20,
      "total": 1,
      "total_page": 1
    }
  }
}
```

**说明：**

- 学生仅可看到已发布的作业
- 教师可看到草稿和已发布作业

---

### 2.63 GET /api/v1/assignments/:id — 作业详情

**权限：** 教师/学生

**说明：**

- 返回作业基础信息及题目列表
- 学生视角不返回 `correct_answer`、`reference_answer`、`judge_config`
- 未发布作业对学生不可见

**响应：**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "id": "1780000000020000",
    "course_id": "1780000000010001",
    "chapter_id": "1780000000011001",
    "title": "作业1：区块链基础",
    "description": "完成基础概念练习与简答题",
    "assignment_type": 1,
    "assignment_type_text": "普通作业",
    "total_score": 100,
    "deadline_at": "2026-10-10T23:59:59Z",
    "max_submissions": 3,
    "late_policy": 2,
    "late_policy_text": "允许迟交并扣分",
    "late_deduction_per_day": 10,
    "is_published": true,
    "questions": [
      {
        "id": "1780000000021001",
        "question_type": 1,
        "question_type_text": "单选题",
        "title": "区块链最核心的特征是什么？",
        "options": "[\"A.中心化\",\"B.可追溯\",\"C.分布式记账\",\"D.单机存储\"]",
        "correct_answer": "C",
        "reference_answer": null,
        "score": 10,
        "judge_config": null,
        "sort_order": 1
      }
    ]
  }
}
```

---

### 2.64 PUT /api/v1/assignments/:id — 编辑作业

**权限：** 课程教师

**请求体：**

```json
{
  "title": "作业1：区块链基础（更新）",
  "deadline_at": "2026-10-12T23:59:59Z",
  "max_submissions": 2
}
```

**说明：**

- 按字段增量更新
- `total_score` 不允许通过本接口修改，需通过题目新增/编辑/删除自动回写
- 已有学生提交后，不允许做会破坏成绩计算的一致性修改

---

### 2.65 DELETE /api/v1/assignments/:id — 删除作业

**权限：** 课程教师

**说明：**

- 若作业已有学生提交，必须拒绝删除
- 进行中课程中，教师不可删除已有学生提交过的作业

**错误响应：**

| code | message | 场景 |
|------|---------|------|
| 40404 | 作业不存在 | 作业ID不存在 |
| 40909 | 该作业已有学生提交，不可删除 | 已存在提交记录 |

---

### 2.66 POST /api/v1/assignments/:id/publish — 发布作业

**权限：** 课程教师

**说明：**

- 至少包含一道题目后才允许发布
- 发布后学生方可看到并开始作答

**错误响应：**

| code | message | 场景 |
|------|---------|------|
| 40001 | 请至少添加一道题目后再发布 | 作业没有题目 |
| 40910 | 作业已发布 | 重复发布 |

---

### 2.67 POST /api/v1/assignments/:id/questions — 添加题目

**权限：** 课程教师

**请求体：**

```json
{
  "question_type": 5,
  "title": "请简述区块链不可篡改性的实现机制",
  "options": null,
  "correct_answer": null,
  "reference_answer": "可从哈希链、分布式共识等角度说明",
  "score": 20,
  "judge_config": null
}
```

**说明：**

- `question_type`：`1` 单选 / `2` 多选 / `3` 判断 / `4` 填空 / `5` 简答 / `6` 编程 / `7` 实验报告
- 主观题和实验报告题需人工批改

---

### 2.68 PUT /api/v1/assignment-questions/:id — 编辑题目

**权限：** 课程教师

**请求体：**

```json
{
  "title": "请解释区块链不可篡改性的主要机制",
  "score": 25,
  "reference_answer": "可从哈希链、时间戳、共识机制等方面回答"
}
```

**说明：**

- 按字段增量更新

---

### 2.69 DELETE /api/v1/assignment-questions/:id — 删除题目

**权限：** 课程教师

**说明：**

- 删除题目后应同步更新作业总分
- 已有学生提交后，不允许删除已参与计分的题目

---

### 2.70 GET /api/v1/assignments/:id/my-submissions — 我的提交记录

**权限：** 学生

**响应：**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "submissions": [
      {
        "id": "1780000000030001",
        "submission_no": 1,
        "status": 3,
        "status_text": "已批改",
        "total_score": 86,
        "is_late": false,
        "submitted_at": "2026-10-08T10:00:00Z"
      }
    ]
  }
}
```

**说明：**

- 仅返回当前学生自己的提交
- 按 `submission_no DESC` 返回

---

### 2.71 GET /api/v1/assignments/:id/submissions — 所有学生提交列表

**权限：** 课程教师

**查询参数：**

- `page`
- `page_size`
- `status`
- `keyword`

**响应：**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "list": [
      {
        "id": "1780000000030001",
        "student_id": "1780000000002001",
        "student_name": "张三",
        "student_no": "2024001",
        "submission_no": 1,
        "status": 2,
        "status_text": "待批改",
        "total_score": 30,
        "is_late": false,
        "submitted_at": "2026-10-08T10:00:00Z"
      }
    ],
    "pagination": {
      "page": 1,
      "page_size": 20,
      "total": 1,
      "total_page": 1
    }
  }
}
```

---

### 2.72 GET /api/v1/submissions/:id — 提交详情

**权限：** 课程教师 / 提交者本人

**响应：**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "id": "1780000000030001",
    "assignment_id": "1780000000020000",
    "student_id": "1780000000002001",
    "student_name": "张三",
    "submission_no": 1,
    "status": 3,
    "status_text": "已批改",
    "total_score": 86,
    "is_late": false,
    "late_days": 0,
    "score_before_deduction": 86,
    "score_after_deduction": 86,
    "teacher_comment": "主观题回答较完整",
    "submitted_at": "2026-10-08T10:00:00Z",
    "graded_at": "2026-10-09T08:30:00Z",
    "answers": [
      {
        "id": "1780000000031001",
        "question_id": "1780000000021001",
        "question_title": "区块链最核心的特征是什么？",
        "question_type": 1,
        "answer_content": "C",
        "answer_file_url": null,
        "is_correct": true,
        "score": 10,
        "teacher_comment": null,
        "auto_judge_result": null
      }
    ]
  }
}
```

---

### 2.73 POST /api/v1/submissions/:id/grade — 批改提交

**权限：** 课程教师

**请求体：**

```json
{
  "teacher_comment": "整体完成较好，主观题建议补充案例",
  "answers": [
    {
      "question_id": "1780000000021002",
      "score": 18,
      "teacher_comment": "论述完整"
    }
  ]
}
```

**说明：**

- 仅对需人工批改的答案进行评分
- 单题分值不得超过题目原始分值
- 提交完成后状态更新为“已批改”，总得分自动重算

**错误响应：**

| code | message | 场景 |
|------|---------|------|
| 40001 | 批改得分不能超过题目分值 | 超出题目分值 |
| 40911 | 提交已批改，不能重复批改 | 重复批改 |

---

### 2.74 GET /api/v1/courses/:id/my-progress — 我的课程学习进度

**权限：** 学生

**响应：**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "course_id": "1780000000010001",
    "total_lessons": 16,
    "completed_count": 9,
    "progress": 56.25,
    "total_study_hours": 12.5,
    "lessons": [
      {
        "lesson_id": "1780000000012001",
        "lesson_title": "1.1 区块链发展史",
        "chapter_title": "第一章 区块链概述",
        "status": 3,
        "status_text": "已完成",
        "video_progress": 1140,
        "video_duration": 1200,
        "study_duration": 1600,
        "completed_at": "2026-09-05T10:00:00Z",
        "last_accessed_at": "2026-09-05T09:50:00Z"
      }
    ]
  }
}
```

---

### 2.75 GET /api/v1/courses/:id/students-progress — 所有学生学习进度

**权限：** 课程教师

**查询参数：**

- `page`
- `page_size`
- `keyword`

**响应：**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "list": [
      {
        "student_id": "1780000000002001",
        "student_name": "张三",
        "student_no": "2024001",
        "completed_count": 9,
        "total_lessons": 16,
        "progress": 56.25,
        "total_study_hours": 12.5,
        "last_accessed_at": "2026-09-05T09:50:00Z"
      }
    ],
    "pagination": {
      "page": 1,
      "page_size": 20,
      "total": 1,
      "total_page": 1
    }
  }
}
```

---

### 2.76 PUT /api/v1/courses/:id/schedules — 设置课程表

**权限：** 课程教师

**请求体：**

```json
{
  "schedules": [
    {
      "day_of_week": 3,
      "start_time": "08:00",
      "end_time": "09:40",
      "location": "教学楼A301"
    },
    {
      "day_of_week": 5,
      "start_time": "14:00",
      "end_time": "15:40",
      "location": "实验楼B201"
    }
  ]
}
```

**说明：**

- 接口语义为覆盖当前课程全部课程表配置
- `day_of_week` 取值 `1-7`

---

### 2.77 GET /api/v1/courses/:id/schedules — 获取课程表

**权限：** 教师/学生

**响应：**

```json
{
  "code": 200,
  "message": "success",
  "data": [
    {
      "id": "1780000000040001",
      "day_of_week": 3,
      "start_time": "08:00",
      "end_time": "09:40",
      "location": "教学楼A301"
    }
  ]
}
```

---

### 2.78 GET /api/v1/shared-courses — 共享课程库列表

**权限：** 教师

**查询参数：**

- `page`
- `page_size`
- `keyword`
- `course_type`
- `difficulty`
- `topic`

**响应：**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "list": [
      {
        "id": "1780000000010001",
        "title": "区块链技术与应用",
        "description": "本课程系统介绍区块链核心技术...",
        "cover_url": "https://oss.example.com/covers/course1.png",
        "course_type": 3,
        "course_type_text": "混合课",
        "difficulty": 2,
        "difficulty_text": "中级",
        "topic": "区块链基础",
        "teacher_name": "张老师",
        "school_name": "链镜实验中学",
        "student_count": 48,
        "rating": 4.7
      }
    ],
    "pagination": {
      "page": 1,
      "page_size": 20,
      "total": 1,
      "total_page": 1
    }
  }
}
```

---

### 2.79 GET /api/v1/my-courses — 我的课程列表

**权限：** 学生

**查询参数：**

- `page`
- `page_size`
- `status`

**响应：**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "list": [
      {
        "id": "1780000000010001",
        "title": "区块链技术与应用",
        "cover_url": "https://oss.example.com/covers/course1.png",
        "course_type": 3,
        "course_type_text": "混合课",
        "teacher_name": "张老师",
        "status": 3,
        "status_text": "进行中",
        "progress": 56.25,
        "joined_at": "2026-09-03T10:00:00Z"
      }
    ],
    "pagination": {
      "page": 1,
      "page_size": 20,
      "total": 1,
      "total_page": 1
    }
  }
}
```

---

*文档版本：v1.0*
*创建日期：2026-04-07*
