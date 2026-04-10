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

### 1.5 作业提交与批改

| 方法 | 路径 | 说明 | 权限 |
|------|------|------|------|
| POST | /api/v1/assignments/:id/submit | 学生提交作业 | 学生 |
| GET | /api/v1/assignments/:id/my-submissions | 我的提交记录 | 学生 |
| GET | /api/v1/assignments/:id/submissions | 所有学生提交列表 | 课程教师 |
| GET | /api/v1/submissions/:id | 提交详情 | 教师/提交者 |
| POST | /api/v1/submissions/:id/grade | 批改提交 | 课程教师 |

### 1.6 学习进度

| 方法 | 路径 | 说明 | 权限 |
|------|------|------|------|
| POST | /api/v1/lessons/:id/progress | 更新学习进度 | 学生 |
| GET | /api/v1/courses/:id/my-progress | 我的课程学习进度 | 学生 |
| GET | /api/v1/courses/:id/students-progress | 所有学生学习进度 | 课程教师 |

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
| DELETE | /api/v1/announcements/:id | 删除公告 | 课程教师 |

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

### 1.10 课程评价

| 方法 | 路径 | 说明 | 权限 |
|------|------|------|------|
| POST | /api/v1/courses/:id/evaluations | 提交评价 | 学生 |
| GET | /api/v1/courses/:id/evaluations | 评价列表 | 教师/学生 |
| PUT | /api/v1/course-evaluations/:id | 修改评价 | 评价人 |

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

### 1.13 课程统计

| 方法 | 路径 | 说明 | 权限 |
|------|------|------|------|
| GET | /api/v1/courses/:id/statistics/overview | 课程整体统计 | 课程教师 |
| GET | /api/v1/courses/:id/statistics/assignments | 作业统计 | 课程教师 |
| GET | /api/v1/courses/:id/statistics/export | 导出统计报告 | 课程教师 |

### 1.14 学生视角

| 方法 | 路径 | 说明 | 权限 |
|------|------|------|------|
| GET | /api/v1/my-courses | 我的课程列表 | 学生 |

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

### 2.3 POST /api/v1/assignments/:id/submit — 学生提交作业

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
| 40001 | 作业已截止且不允许迟交 | 超过截止时间 |
| 40002 | 已达最大提交次数 | 超过max_submissions |
| 40003 | 作业未发布 | 作业未发布 |

---

### 2.4 POST /api/v1/lessons/:id/progress — 更新学习进度

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

### 2.5 GET /api/v1/courses/:id/grades — 成绩汇总表

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

---

### 2.6 GET /api/v1/my-schedule — 我的周课程表

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

*文档版本：v1.0*
*创建日期：2026-04-07*
