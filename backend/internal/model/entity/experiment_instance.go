// experiment_instance.go
// 模块04 — 实验环境：数据库实体结构体（实例、分组、配额、报告部分）
// 对照 docs/modules/04-实验环境/02-数据库设计.md
// 包含 10 张表的 GORM 映射结构体：实例、容器、检查点结果、快照、操作日志、分组、配额、报告

package entity

import (
	"encoding/json"
	"time"
)

// ---------------------------------------------------------------------------
// 3.1 实验实例
// ---------------------------------------------------------------------------

// ExperimentInstance 实验实例表
// 对应 experiment_instances 表，25 个字段
// 记录学生启动实验后生成的实例，实例记录永久保留（无软删除）
type ExperimentInstance struct {
	ID              int64      `gorm:"primaryKey;autoIncrement:false" json:"id,string"`
	TemplateID      int64      `gorm:"not null;index" json:"template_id,string"`
	StudentID       int64      `gorm:"not null;index" json:"student_id,string"`
	SchoolID        int64      `gorm:"not null;index" json:"school_id,string"`
	CourseID        *int64     `gorm:"index" json:"course_id,omitempty,string"`
	LessonID        *int64     `gorm:"" json:"lesson_id,omitempty,string"`
	AssignmentID    *int64     `gorm:"" json:"assignment_id,omitempty,string"`
	GroupID         *int64     `gorm:"index" json:"group_id,omitempty,string"`
	ExperimentType  int        `gorm:"type:smallint;not null" json:"experiment_type"`
	Status          int        `gorm:"type:smallint;not null;default:1" json:"status"`
	AttemptNo       int        `gorm:"not null;default:1" json:"attempt_no"`
	Namespace       *string    `gorm:"type:varchar(100)" json:"namespace,omitempty"`
	AccessURL       *string    `gorm:"type:varchar(500)" json:"access_url,omitempty"`
	TotalScore      *float64   `gorm:"type:decimal(6,2)" json:"total_score,omitempty"`
	AutoScore       *float64   `gorm:"type:decimal(6,2)" json:"auto_score,omitempty"`
	ManualScore     *float64   `gorm:"type:decimal(6,2)" json:"manual_score,omitempty"`
	StartedAt       *time.Time `gorm:"" json:"started_at,omitempty"`
	PausedAt        *time.Time `gorm:"" json:"paused_at,omitempty"`
	SubmittedAt     *time.Time `gorm:"" json:"submitted_at,omitempty"`
	DestroyedAt     *time.Time `gorm:"" json:"destroyed_at,omitempty"`
	LastActiveAt    *time.Time `gorm:"" json:"last_active_at,omitempty"`
	ErrorMessage    *string    `gorm:"type:text" json:"error_message,omitempty"`
	SimSessionID    *string    `gorm:"type:varchar(100)" json:"sim_session_id,omitempty"`
	SimWebSocketURL *string    `gorm:"type:varchar(500)" json:"sim_websocket_url,omitempty"`
	CreatedAt       time.Time  `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt       time.Time  `gorm:"not null;default:now()" json:"updated_at"`

	// 关联（非数据库字段，用于预加载）
	Containers        []InstanceContainer `gorm:"foreignKey:InstanceID" json:"containers,omitempty"`
	CheckpointResults []CheckpointResult  `gorm:"foreignKey:InstanceID" json:"checkpoint_results,omitempty"`
	Snapshots         []InstanceSnapshot  `gorm:"foreignKey:InstanceID" json:"snapshots,omitempty"`
}

// TableName 指定表名
func (ExperimentInstance) TableName() string {
	return "experiment_instances"
}

// ---------------------------------------------------------------------------
// 3.2 实例容器
// ---------------------------------------------------------------------------

// InstanceContainer 实例容器表
// 对应 instance_containers 表，12 个字段
// 记录实例运行时每个容器的状态和资源信息
type InstanceContainer struct {
	ID                  int64     `gorm:"primaryKey;autoIncrement:false" json:"id,string"`
	InstanceID          int64     `gorm:"not null;index" json:"instance_id,string"`
	TemplateContainerID int64     `gorm:"not null" json:"template_container_id,string"`
	ContainerName       string    `gorm:"type:varchar(100);not null" json:"container_name"`
	PodName             *string   `gorm:"type:varchar(200)" json:"pod_name,omitempty"`
	InternalIP          *string   `gorm:"type:varchar(50)" json:"internal_ip,omitempty"`
	Status              int       `gorm:"type:smallint;not null;default:1" json:"status"`
	CPUUsage            *string   `gorm:"type:varchar(20)" json:"cpu_usage,omitempty"`
	MemoryUsage         *string   `gorm:"type:varchar(20)" json:"memory_usage,omitempty"`
	CreatedAt           time.Time `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt           time.Time `gorm:"not null;default:now()" json:"updated_at"`
}

// TableName 指定表名
func (InstanceContainer) TableName() string {
	return "instance_containers"
}

// ---------------------------------------------------------------------------
// 3.3 检查点结果
// ---------------------------------------------------------------------------

// CheckpointResult 检查点结果表
// 对应 checkpoint_results 表，14 个字段
// 记录学生在实验中通过自动/手动检查点的结果，checked_at 表示最近一次有效检查时间
type CheckpointResult struct {
	ID              int64           `gorm:"primaryKey;autoIncrement:false" json:"id,string"`
	InstanceID      int64           `gorm:"not null;index" json:"instance_id,string"`
	CheckpointID    int64           `gorm:"not null;index" json:"checkpoint_id,string"`
	StudentID       int64           `gorm:"not null;index" json:"student_id,string"`
	IsPassed        bool            `gorm:"not null;default:false" json:"is_passed"`
	Score           *float64        `gorm:"type:decimal(6,2)" json:"score,omitempty"`
	CheckOutput     *string         `gorm:"type:text" json:"check_output,omitempty"`
	AssertionResult json.RawMessage `gorm:"type:jsonb" json:"assertion_result,omitempty"`
	TeacherComment  *string         `gorm:"type:varchar(500)" json:"teacher_comment,omitempty"`
	GradedBy        *int64          `gorm:"" json:"graded_by,omitempty,string"`
	GradedAt        *time.Time      `gorm:"" json:"graded_at,omitempty"`
	CheckedAt       time.Time       `gorm:"not null;default:now()" json:"checked_at"`
	CreatedAt       time.Time       `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt       time.Time       `gorm:"not null;default:now()" json:"updated_at"`
}

// TableName 指定表名
func (CheckpointResult) TableName() string {
	return "checkpoint_results"
}

// ---------------------------------------------------------------------------
// 3.4 实例快照
// ---------------------------------------------------------------------------

// InstanceSnapshot 实例快照表
// 对应 instance_snapshots 表，9 个字段
// 保存实例的容器状态和仿真引擎状态快照，只有 created_at（不可修改）
type InstanceSnapshot struct {
	ID              int64           `gorm:"primaryKey;autoIncrement:false" json:"id,string"`
	InstanceID      int64           `gorm:"not null;index" json:"instance_id,string"`
	SnapshotType    int             `gorm:"type:smallint;not null" json:"snapshot_type"`
	SnapshotDataURL string          `gorm:"type:varchar(500);not null" json:"snapshot_data_url"`
	SnapshotSize    *int64          `gorm:"" json:"snapshot_size,omitempty,string"`
	ContainerStates json.RawMessage `gorm:"type:jsonb" json:"container_states,omitempty"`
	SimEngineState  json.RawMessage `gorm:"type:jsonb" json:"sim_engine_state,omitempty"`
	Description     *string         `gorm:"type:varchar(200)" json:"description,omitempty"`
	CreatedAt       time.Time       `gorm:"not null;default:now()" json:"created_at"`
}

// TableName 指定表名
func (InstanceSnapshot) TableName() string {
	return "instance_snapshots"
}

// ---------------------------------------------------------------------------
// 3.5 实例操作日志
// ---------------------------------------------------------------------------

// InstanceOperationLog 实例操作日志表
// 对应 instance_operation_logs 表，11 个字段
// 审计日志：只插入不更新不删除，只有 created_at
type InstanceOperationLog struct {
	ID              int64           `gorm:"primaryKey;autoIncrement:false" json:"id,string"`
	InstanceID      int64           `gorm:"not null;index" json:"instance_id,string"`
	StudentID       int64           `gorm:"not null;index" json:"student_id,string"`
	Action          string          `gorm:"type:varchar(50);not null" json:"action"`
	TargetContainer *string         `gorm:"type:varchar(100)" json:"target_container,omitempty"`
	TargetScene     *string         `gorm:"type:varchar(100)" json:"target_scene,omitempty"`
	Command         *string         `gorm:"type:text" json:"command,omitempty"`
	CommandOutput   *string         `gorm:"type:text" json:"command_output,omitempty"`
	Detail          json.RawMessage `gorm:"type:jsonb" json:"detail,omitempty"`
	ClientIP        *string         `gorm:"type:varchar(50)" json:"client_ip,omitempty"`
	CreatedAt       time.Time       `gorm:"not null;default:now()" json:"created_at"`
}

// TableName 指定表名
func (InstanceOperationLog) TableName() string {
	return "instance_operation_logs"
}

// ---------------------------------------------------------------------------
// 3.6 实验分组
// ---------------------------------------------------------------------------

// ExperimentGroup 实验分组表
// 对应 experiment_groups 表，10 个字段
// 多人协作实验的分组信息
type ExperimentGroup struct {
	ID          int64     `gorm:"primaryKey;autoIncrement:false" json:"id,string"`
	TemplateID  int64     `gorm:"not null;index" json:"template_id,string"`
	CourseID    int64     `gorm:"not null;index" json:"course_id,string"`
	SchoolID    int64     `gorm:"not null;index" json:"school_id,string"`
	GroupName   string    `gorm:"type:varchar(100);not null" json:"group_name"`
	GroupMethod int       `gorm:"type:smallint;not null;default:1" json:"group_method"`
	MaxMembers  int       `gorm:"not null;default:4" json:"max_members"`
	Status      int       `gorm:"type:smallint;not null;default:1" json:"status"`
	Namespace   *string   `gorm:"type:varchar(100)" json:"namespace,omitempty"`
	CreatedAt   time.Time `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt   time.Time `gorm:"not null;default:now()" json:"updated_at"`

	// 关联（非数据库字段，用于预加载）
	Members  []GroupMember  `gorm:"foreignKey:GroupID" json:"members,omitempty"`
	Messages []GroupMessage `gorm:"foreignKey:GroupID" json:"messages,omitempty"`
}

// TableName 指定表名
func (ExperimentGroup) TableName() string {
	return "experiment_groups"
}

// ---------------------------------------------------------------------------
// 3.7 分组成员
// ---------------------------------------------------------------------------

// GroupMember 分组成员表
// 对应 group_members 表，7 个字段
// 记录分组中的学生及其角色，只有 created_at
type GroupMember struct {
	ID         int64     `gorm:"primaryKey;autoIncrement:false" json:"id,string"`
	GroupID    int64     `gorm:"not null;index" json:"group_id,string"`
	StudentID  int64     `gorm:"not null;index" json:"student_id,string"`
	RoleID     *int64    `gorm:"" json:"role_id,omitempty,string"`
	InstanceID *int64    `gorm:"" json:"instance_id,omitempty,string"`
	JoinedAt   time.Time `gorm:"not null;default:now()" json:"joined_at"`
	CreatedAt  time.Time `gorm:"not null;default:now()" json:"created_at"`
}

// TableName 指定表名
func (GroupMember) TableName() string {
	return "group_members"
}

// ---------------------------------------------------------------------------
// 3.8 组内消息
// ---------------------------------------------------------------------------

// GroupMessage 组内消息表
// 对应 group_messages 表，6 个字段
// 分组内成员通信消息，只有 created_at
type GroupMessage struct {
	ID          int64     `gorm:"primaryKey;autoIncrement:false" json:"id,string"`
	GroupID     int64     `gorm:"not null;index" json:"group_id,string"`
	SenderID    int64     `gorm:"not null;index" json:"sender_id,string"`
	Content     string    `gorm:"type:text;not null" json:"content"`
	MessageType int       `gorm:"type:smallint;not null;default:1" json:"message_type"`
	CreatedAt   time.Time `gorm:"not null;default:now()" json:"created_at"`
}

// TableName 指定表名
func (GroupMessage) TableName() string {
	return "group_messages"
}

// ---------------------------------------------------------------------------
// 3.9 资源配额
// ---------------------------------------------------------------------------

// ResourceQuota 资源配额表
// 对应 resource_quotas 表，15 个字段
// 按学校/课程维度管理实验资源配额和使用量
type ResourceQuota struct {
	ID              int64     `gorm:"primaryKey;autoIncrement:false" json:"id,string"`
	QuotaLevel      int       `gorm:"type:smallint;not null" json:"quota_level"`
	SchoolID        int64     `gorm:"not null;index" json:"school_id,string"`
	CourseID        *int64    `gorm:"index" json:"course_id,omitempty,string"`
	MaxCPU          string    `gorm:"type:varchar(20);not null;default:'0'" json:"max_cpu"`
	MaxMemory       string    `gorm:"type:varchar(20);not null;default:'0'" json:"max_memory"`
	MaxStorage      string    `gorm:"type:varchar(20);not null;default:'0'" json:"max_storage"`
	MaxConcurrency  int       `gorm:"not null;default:0" json:"max_concurrency"`
	MaxPerStudent   int       `gorm:"not null;default:2" json:"max_per_student"`
	UsedCPU         string    `gorm:"type:varchar(20);not null;default:'0'" json:"used_cpu"`
	UsedMemory      string    `gorm:"type:varchar(20);not null;default:'0'" json:"used_memory"`
	UsedStorage     string    `gorm:"type:varchar(20);not null;default:'0'" json:"used_storage"`
	UsedConcurrency int       `gorm:"not null;default:0" json:"used_concurrency"`
	CreatedAt       time.Time `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt       time.Time `gorm:"not null;default:now()" json:"updated_at"`
}

// TableName 指定表名
func (ResourceQuota) TableName() string {
	return "resource_quotas"
}

// ---------------------------------------------------------------------------
// 3.10 实验报告
// ---------------------------------------------------------------------------

// ExperimentReport 实验报告表
// 对应 experiment_reports 表，10 个字段
// 学生提交的实验报告内容及附件，submitted_at 表示首次提交时间
type ExperimentReport struct {
	ID          int64     `gorm:"primaryKey;autoIncrement:false" json:"id,string"`
	InstanceID  int64     `gorm:"not null;index" json:"instance_id,string"`
	StudentID   int64     `gorm:"not null;index" json:"student_id,string"`
	Content     *string   `gorm:"type:text" json:"content,omitempty"`
	FileURL     *string   `gorm:"type:varchar(500)" json:"file_url,omitempty"`
	FileName    *string   `gorm:"type:varchar(200)" json:"file_name,omitempty"`
	FileSize    *int64    `gorm:"" json:"file_size,omitempty,string"`
	SubmittedAt time.Time `gorm:"not null;default:now()" json:"submitted_at"`
	CreatedAt   time.Time `gorm:"not null;default:now()" json:"created_at"`
	UpdatedAt   time.Time `gorm:"not null;default:now()" json:"updated_at"`
}

// TableName 指定表名
func (ExperimentReport) TableName() string {
	return "experiment_reports"
}
