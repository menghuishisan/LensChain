// experiment_instance.go
// 模块04 — 实验环境：请求/响应 DTO 定义（实例 + 分组 + 监控 + 配额 + 报告 + 管理）
// 对照 docs/modules/04-实验环境/03-API接口设计.md

package dto

import "encoding/json"

// ========== 实验实例 DTO ==========

// CreateInstanceReq 启动实验环境请求
// POST /api/v1/experiment-instances
type CreateInstanceReq struct {
	TemplateID   string  `json:"template_id" binding:"required"`
	CourseID     *string `json:"course_id"`
	LessonID     *string `json:"lesson_id"`
	AssignmentID *string `json:"assignment_id"`
	SnapshotID   *string `json:"snapshot_id"`
	GroupID      *string `json:"group_id"`
}

// CreateInstanceResp 启动实验环境响应
type CreateInstanceResp struct {
	InstanceID            *string `json:"instance_id"`
	SimSessionID          *string `json:"sim_session_id"`
	Status                int16   `json:"status"`
	StatusText            string  `json:"status_text"`
	AttemptNo             int     `json:"attempt_no,omitempty"`
	EstimatedReadySeconds *int    `json:"estimated_ready_seconds,omitempty"`
	QueuePosition         *int    `json:"queue_position,omitempty"`
	EstimatedWaitSeconds  *int    `json:"estimated_wait_seconds,omitempty"`
}

// InstanceDetailResp 实验实例详情响应
// GET /api/v1/experiment-instances/:id
type InstanceDetailResp struct {
	ID           string                   `json:"id"`
	Template     InstanceTemplateBrief    `json:"template"`
	Student      InstanceStudentBrief     `json:"student"`
	Status       int16                    `json:"status"`
	StatusText   string                   `json:"status_text"`
	AttemptNo    int                      `json:"attempt_no"`
	SimSessionID *string                  `json:"sim_session_id"`
	Tools        []InstanceToolItem       `json:"tools"`
	Containers   []InstanceContainerItem  `json:"containers"`
	Checkpoints  []InstanceCheckpointItem `json:"checkpoints"`
	Scores       InstanceScoresInfo       `json:"scores"`
	StartedAt    *string                  `json:"started_at"`
	LastActiveAt *string                  `json:"last_active_at"`
	CreatedAt    string                   `json:"created_at"`
}

// InstanceTemplateBrief 实例详情中的模板摘要
type InstanceTemplateBrief struct {
	ID           string  `json:"id"`
	Title        string  `json:"title"`
	TopologyMode int16   `json:"topology_mode"`
	JudgeMode    int16   `json:"judge_mode"`
	Instructions *string `json:"instructions"`
	MaxDuration  int     `json:"max_duration"`
	IdleTimeout  int     `json:"idle_timeout"`
	TotalScore   float64 `json:"total_score"`
}

// InstanceStudentBrief 实例详情中的学生摘要
type InstanceStudentBrief struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	StudentNo string `json:"student_no"`
}

// InstanceToolItem 实例详情中的工具条目
type InstanceToolItem struct {
	Kind          string  `json:"kind"`
	ContainerID   string  `json:"container_id"`
	ContainerName string  `json:"container_name"`
	ProxyURL      string  `json:"proxy_url"`
	Status        int16   `json:"status"`
	StatusText    string  `json:"status_text"`
}

// InstanceContainerItem 实例详情中的容器条目
type InstanceContainerItem struct {
	ID            string  `json:"id"`
	ContainerName string  `json:"container_name"`
	ImageName     string  `json:"image_name"`
	ImageVersion  string  `json:"image_version"`
	Status        int16   `json:"status"`
	StatusText    string  `json:"status_text"`
	InternalIP    *string `json:"internal_ip"`
	CPUUsage      *string `json:"cpu_usage"`
	MemoryUsage   *string `json:"memory_usage"`
	ToolKind      *string `json:"tool_kind"`
}

// InstanceCheckpointItem 实例详情中的检查点条目
type InstanceCheckpointItem struct {
	CheckpointID string                    `json:"checkpoint_id"`
	Title        string                    `json:"title"`
	CheckType    int16                     `json:"check_type"`
	Score        float64                   `json:"score"`
	Result       *InstanceCheckpointResult `json:"result"`
}

// InstanceCheckpointResult 检查点结果
type InstanceCheckpointResult struct {
	ID        string  `json:"id"`
	IsPassed  bool    `json:"is_passed"`
	Score     float64 `json:"score"`
	CheckedAt string  `json:"checked_at"`
}

// InstanceScoresInfo 实例详情中的成绩信息
type InstanceScoresInfo struct {
	AutoScore   *float64 `json:"auto_score"`
	ManualScore *float64 `json:"manual_score"`
	TotalScore  *float64 `json:"total_score"`
}

// InstanceListReq 我的实验实例列表查询参数
// GET /api/v1/experiment-instances
type InstanceListReq struct {
	Page       int    `form:"page" binding:"omitempty,min=1"`
	PageSize   int    `form:"page_size" binding:"omitempty,min=1,max=100"`
	TemplateID string `form:"template_id"`
	CourseID   string `form:"course_id"`
	Status     int16  `form:"status" binding:"omitempty,oneof=1 2 3 4 5 6 7 8 9 10"`
}

// InstanceListItem 实验实例列表条目
type InstanceListItem struct {
	ID            string   `json:"id"`
	TemplateID    string   `json:"template_id"`
	TemplateTitle string   `json:"template_title"`
	StudentID     *string  `json:"student_id,omitempty"`
	StudentName   *string  `json:"student_name,omitempty"`
	SchoolID      *string  `json:"school_id,omitempty"`
	SchoolName    *string  `json:"school_name,omitempty"`
	CourseID      *string  `json:"course_id"`
	CourseTitle   *string  `json:"course_title"`
	Status        int16    `json:"status"`
	StatusText    string   `json:"status_text"`
	AttemptNo     int      `json:"attempt_no"`
	TotalScore    *float64 `json:"total_score"`
	ErrorMessage  *string  `json:"error_message,omitempty"`
	StartedAt     *string  `json:"started_at"`
	SubmittedAt   *string  `json:"submitted_at"`
	CreatedAt     string   `json:"created_at"`
	UpdatedAt     *string  `json:"updated_at,omitempty"`
}

// InstanceListResp 我的实验实例列表响应
// GET /api/v1/experiment-instances
type InstanceListResp struct {
	List       []InstanceListItem `json:"list"`
	Pagination PaginationResp     `json:"pagination"`
}

// ========== 暂停 / 恢复 / 提交 / 销毁 / 重启 DTO ==========

// PauseInstanceResp 暂停实验响应
// POST /api/v1/experiment-instances/:id/pause
type PauseInstanceResp struct {
	InstanceID string `json:"instance_id"`
	Status     int16  `json:"status"`
	StatusText string `json:"status_text"`
	SnapshotID string `json:"snapshot_id"`
	PausedAt   string `json:"paused_at"`
}

// ResumeInstanceReq 恢复实验请求
// POST /api/v1/experiment-instances/:id/resume
type ResumeInstanceReq struct {
	SnapshotID *string `json:"snapshot_id"`
}

// ResumeInstanceResp 恢复实验响应
type ResumeInstanceResp struct {
	InstanceID            string `json:"instance_id"`
	Status                int16  `json:"status"`
	StatusText            string `json:"status_text"`
	EstimatedReadySeconds int    `json:"estimated_ready_seconds"`
}

// SubmitInstanceResp 提交实验响应
// POST /api/v1/experiment-instances/:id/submit
type SubmitInstanceResp struct {
	InstanceID  string           `json:"instance_id"`
	Status      int16            `json:"status"`
	StatusText  string           `json:"status_text"`
	Scores      SubmitScoresInfo `json:"scores"`
	CompletedAt string           `json:"completed_at"`
}

// SubmitScoresInfo 提交实验时的成绩详情
type SubmitScoresInfo struct {
	AutoScore   float64             `json:"auto_score"`
	AutoTotal   float64             `json:"auto_total"`
	ManualScore *float64            `json:"manual_score"`
	ManualTotal float64             `json:"manual_total"`
	TotalScore  *float64            `json:"total_score"`
	Details     []SubmitScoreDetail `json:"details"`
}

// SubmitScoreDetail 提交实验时的单个检查点成绩
type SubmitScoreDetail struct {
	CheckpointID string   `json:"checkpoint_id"`
	Title        string   `json:"title"`
	CheckType    int16    `json:"check_type"`
	IsPassed     *bool    `json:"is_passed,omitempty"`
	Score        *float64 `json:"score,omitempty"`
	MaxScore     float64  `json:"max_score"`
	Status       *string  `json:"status,omitempty"`
}

// ========== 心跳 DTO ==========

// HeartbeatReq 心跳上报请求
// POST /api/v1/experiment-instances/:id/heartbeat
type HeartbeatReq struct {
	ActiveContainer string `json:"active_container"`
}

// HeartbeatResp 心跳上报响应
type HeartbeatResp struct {
	Status           int16 `json:"status"`
	RemainingMinutes int   `json:"remaining_minutes"`
	IdleWarning      bool  `json:"idle_warning"`
}

// ========== 检查点验证 / 评分 DTO ==========

// VerifyCheckpointReq 触发检查点验证请求
// POST /api/v1/experiment-instances/:id/checkpoints/verify
type VerifyCheckpointReq struct {
	CheckpointID *string `json:"checkpoint_id"`
}

// CheckpointVerifyResultItem 检查点验证结果条目
type CheckpointVerifyResultItem struct {
	CheckpointID string  `json:"checkpoint_id"`
	Title        string  `json:"title"`
	IsPassed     bool    `json:"is_passed"`
	Score        float64 `json:"score"`
	CheckOutput  string  `json:"check_output"`
	CheckedAt    string  `json:"checked_at"`
}

// VerifyCheckpointResp 检查点验证响应
type VerifyCheckpointResp struct {
	Results []CheckpointVerifyResultItem `json:"results"`
}

// GradeCheckpointReq 手动评分检查点请求
// POST /api/v1/checkpoint-results/:id/grade
type GradeCheckpointReq struct {
	Score   float64 `json:"score" binding:"required,min=0"`
	Comment *string `json:"comment" binding:"omitempty,max=500"`
}

// ManualGradeReq 教师手动评分（整体）请求
// POST /api/v1/experiment-instances/:id/manual-grade
type ManualGradeReq struct {
	CheckpointGrades []CheckpointGradeItem `json:"checkpoint_grades" binding:"required,min=1,dive"`
	OverallComment   *string               `json:"overall_comment" binding:"omitempty,max=1000"`
}

// CheckpointGradeItem 手动评分中的单个检查点评分
type CheckpointGradeItem struct {
	CheckpointID string  `json:"checkpoint_id" binding:"required"`
	Score        float64 `json:"score" binding:"min=0"`
	Comment      *string `json:"comment" binding:"omitempty,max=500"`
}

// ManualGradeResp 教师手动评分响应
type ManualGradeResp struct {
	InstanceID  string  `json:"instance_id"`
	AutoScore   float64 `json:"auto_score"`
	ManualScore float64 `json:"manual_score"`
	TotalScore  float64 `json:"total_score"`
	ScoreDetail string  `json:"score_detail"`
}

// ========== 快照 DTO ==========

// CreateSnapshotReq 手动创建快照请求
// POST /api/v1/experiment-instances/:id/snapshots
type CreateSnapshotReq struct {
	Description *string `json:"description" binding:"omitempty,max=200"`
}

// SnapshotResp 快照响应
type SnapshotResp struct {
	ID               string          `json:"id"`
	InstanceID       string          `json:"instance_id"`
	SnapshotType     int16           `json:"snapshot_type"`
	SnapshotTypeText string          `json:"snapshot_type_text"`
	SnapshotDataURL  string          `json:"snapshot_data_url"`
	ContainerStates  json.RawMessage `json:"container_states"`
	SimEngineState   json.RawMessage `json:"sim_engine_state"`
	Description      *string         `json:"description"`
	CreatedAt        string          `json:"created_at"`
}

// ========== 操作日志 DTO ==========

// InstanceOpLogListReq 实例操作日志列表查询参数
// GET /api/v1/experiment-instances/:id/operation-logs
type InstanceOpLogListReq struct {
	Page            int    `form:"page" binding:"omitempty,min=1"`
	PageSize        int    `form:"page_size" binding:"omitempty,min=1,max=100"`
	Action          string `form:"action"`
	TargetContainer string `form:"target_container"`
}

// InstanceOpLogItem 实例操作日志条目
type InstanceOpLogItem struct {
	ID              string          `json:"id"`
	Action          string          `json:"action"`
	TargetContainer *string         `json:"target_container"`
	Command         *string         `json:"command"`
	Detail          json.RawMessage `json:"detail"`
	CreatedAt       string          `json:"created_at"`
}

// InstanceOpLogListResp 实例操作日志列表响应
// GET /api/v1/experiment-instances/:id/operation-logs
type InstanceOpLogListResp struct {
	List       []InstanceOpLogItem `json:"list"`
	Pagination PaginationResp      `json:"pagination"`
}

// ========== 实验报告 DTO ==========

// CreateReportReq 提交实验报告请求
// POST /api/v1/experiment-instances/:id/report
type CreateReportReq struct {
	Content  *string `json:"content"`
	FileURL  *string `json:"file_url" binding:"omitempty,max=500"`
	FileName *string `json:"file_name" binding:"omitempty,max=200"`
	FileSize *int64  `json:"file_size,omitempty" binding:"omitempty,min=1,max=52428800"`
}

// UpdateReportReq 更新实验报告请求
// PUT /api/v1/experiment-instances/:id/report
type UpdateReportReq struct {
	Content  *string `json:"content"`
	FileURL  *string `json:"file_url" binding:"omitempty,max=500"`
	FileName *string `json:"file_name" binding:"omitempty,max=200"`
	FileSize *int64  `json:"file_size,omitempty" binding:"omitempty,min=1,max=52428800"`
}

// ReportResp 实验报告响应
type ReportResp struct {
	ID         string  `json:"id"`
	InstanceID string  `json:"instance_id"`
	Content    *string `json:"content"`
	FileURL    *string `json:"file_url"`
	FileName   *string `json:"file_name"`
	FileSize   *int64  `json:"file_size,omitempty"`
	CreatedAt  string  `json:"created_at"`
	UpdatedAt  string  `json:"updated_at"`
}

// UploadExperimentFileResp 实验文件上传响应
// POST /api/v1/experiment-files/upload
type UploadExperimentFileResp struct {
	FileName    string `json:"file_name"`
	FileURL     string `json:"file_url"`
	DownloadURL string `json:"download_url"`
	FileSize    int64  `json:"file_size"`
	FileType    string `json:"file_type"`
}

// ========== 实验分组 DTO ==========

// CreateGroupReq 创建实验分组请求
// POST /api/v1/experiment-groups
type CreateGroupReq struct {
	TemplateID  string         `json:"template_id" binding:"required"`
	CourseID    string         `json:"course_id" binding:"required"`
	GroupMethod int16          `json:"group_method" binding:"required,oneof=1 2 3"`
	Groups      []GroupItemReq `json:"groups" binding:"required,min=1,dive"`
}

// GroupItemReq 创建分组时的单个分组信息
type GroupItemReq struct {
	GroupName  string          `json:"group_name" binding:"required,max=100"`
	MaxMembers int             `json:"max_members" binding:"required,min=1"`
	Members    []MemberItemReq `json:"members"`
}

// MemberItemReq 创建分组时的成员信息
type MemberItemReq struct {
	StudentID string  `json:"student_id" binding:"required"`
	RoleID    *string `json:"role_id"`
}

// UpdateGroupReq 编辑分组请求
// PUT /api/v1/experiment-groups/:id
type UpdateGroupReq struct {
	GroupName  *string `json:"group_name" binding:"omitempty,max=100"`
	MaxMembers *int    `json:"max_members" binding:"omitempty,min=1"`
	Status     *int16  `json:"status" binding:"omitempty,oneof=1 2 3 4"`
}

// GroupListReq 分组列表查询参数
// GET /api/v1/experiment-groups
type GroupListReq struct {
	Page       int    `form:"page" binding:"omitempty,min=1"`
	PageSize   int    `form:"page_size" binding:"omitempty,min=1,max=100"`
	TemplateID string `form:"template_id"`
	CourseID   string `form:"course_id"`
	Status     int16  `form:"status" binding:"omitempty,oneof=1 2 3 4"`
}

// GroupResp 分组详情响应
type GroupResp struct {
	ID          string            `json:"id"`
	TemplateID  string            `json:"template_id"`
	CourseID    string            `json:"course_id"`
	GroupName   string            `json:"group_name"`
	GroupMethod int16             `json:"group_method"`
	MaxMembers  int               `json:"max_members"`
	Status      int16             `json:"status"`
	StatusText  string            `json:"status_text"`
	Members     []GroupMemberResp `json:"members,omitempty"`
	CreatedAt   string            `json:"created_at"`
	UpdatedAt   string            `json:"updated_at"`
}

// CreateGroupResp 创建实验分组响应
// POST /api/v1/experiment-groups
type CreateGroupResp struct {
	Groups []GroupListItem `json:"groups"`
}

// GroupListItem 分组列表条目
type GroupListItem struct {
	ID          string `json:"id"`
	GroupName   string `json:"group_name"`
	MemberCount int    `json:"member_count"`
	MaxMembers  int    `json:"max_members"`
	Status      int16  `json:"status"`
	StatusText  string `json:"status_text"`
}

// GroupListResp 分组列表响应
// GET /api/v1/experiment-groups
type GroupListResp struct {
	List       []GroupListItem `json:"list"`
	Pagination PaginationResp  `json:"pagination"`
}

// GroupMemberResp 分组成员响应
type GroupMemberResp struct {
	ID          string  `json:"id"`
	StudentID   string  `json:"student_id"`
	StudentName string  `json:"student_name"`
	StudentNo   string  `json:"student_no"`
	RoleID      *string `json:"role_id"`
	RoleName    *string `json:"role_name"`
	InstanceID  *string `json:"instance_id"`
	JoinedAt    string  `json:"joined_at"`
}

// GroupMemberListResp 组员列表响应
// GET /api/v1/experiment-groups/:id/members
type GroupMemberListResp struct {
	Members []GroupMemberResp `json:"members"`
}

// JoinGroupReq 学生加入分组请求
// POST /api/v1/experiment-groups/:id/join
type JoinGroupReq struct {
	RoleID *string `json:"role_id"`
}

// AutoAssignReq 系统随机分组请求
// POST /api/v1/experiment-groups/auto-assign
type AutoAssignReq struct {
	TemplateID      string `json:"template_id" binding:"required"`
	CourseID        string `json:"course_id" binding:"required"`
	GroupSize       int    `json:"group_size" binding:"required,min=2"`
	GroupNamePrefix string `json:"group_name_prefix" binding:"required,max=20"`
}

// AutoAssignResp 系统随机分组响应
type AutoAssignResp struct {
	TotalGroups   int                   `json:"total_groups"`
	TotalStudents int                   `json:"total_students"`
	Groups        []AutoAssignGroupItem `json:"groups"`
}

// AutoAssignGroupItem 随机分组结果中的单个分组
type AutoAssignGroupItem struct {
	ID        string                 `json:"id"`
	GroupName string                 `json:"group_name"`
	Members   []AutoAssignMemberItem `json:"members"`
}

// AutoAssignMemberItem 随机分组结果中的成员
type AutoAssignMemberItem struct {
	StudentID   string `json:"student_id"`
	StudentName string `json:"student_name"`
	RoleName    string `json:"role_name"`
}

// ========== 组内通信 DTO ==========

// SendGroupMessageReq 发送组内消息请求
// POST /api/v1/experiment-groups/:id/messages
type SendGroupMessageReq struct {
	Content string `json:"content" binding:"required,max=2000"`
}

// GroupMessageListReq 组内消息历史查询参数
// GET /api/v1/experiment-groups/:id/messages
type GroupMessageListReq struct {
	Page     int `form:"page" binding:"omitempty,min=1"`
	PageSize int `form:"page_size" binding:"omitempty,min=1,max=100"`
}

// GroupMessageItem 组内消息条目
type GroupMessageItem struct {
	ID          string `json:"id"`
	SenderID    string `json:"sender_id"`
	SenderName  string `json:"sender_name"`
	Content     string `json:"content"`
	MessageType int16  `json:"message_type"`
	CreatedAt   string `json:"created_at"`
}

// GroupMessageListResp 组内消息历史响应
// GET /api/v1/experiment-groups/:id/messages
type GroupMessageListResp struct {
	List       []GroupMessageItem `json:"list"`
	Pagination PaginationResp     `json:"pagination"`
}

// ========== 组内进度同步 DTO ==========

// GroupProgressResp 组内进度同步响应
// GET /api/v1/experiment-groups/:id/progress
type GroupProgressResp struct {
	GroupID          string                    `json:"group_id"`
	GroupName        string                    `json:"group_name"`
	GroupStatus      int16                     `json:"group_status"`
	GroupStatusText  string                    `json:"group_status_text"`
	Members          []GroupMemberProgressItem `json:"members"`
	GroupCheckpoints []GroupCheckpointItem     `json:"group_checkpoints"`
}

// GroupMemberProgressItem 组内成员进度条目
type GroupMemberProgressItem struct {
	StudentID          string   `json:"student_id"`
	StudentName        string   `json:"student_name"`
	RoleName           string   `json:"role_name"`
	InstanceID         *string  `json:"instance_id"`
	InstanceStatus     *int16   `json:"instance_status"`
	InstanceStatusText *string  `json:"instance_status_text"`
	CheckpointsPassed  int      `json:"checkpoints_passed"`
	CheckpointsTotal   int      `json:"checkpoints_total"`
	PersonalScore      *float64 `json:"personal_score"`
}

// GroupCheckpointItem 组级检查点条目
type GroupCheckpointItem struct {
	CheckpointID string  `json:"checkpoint_id"`
	Title        string  `json:"title"`
	Scope        int16   `json:"scope"`
	IsPassed     bool    `json:"is_passed"`
	CheckedAt    *string `json:"checked_at"`
}

// ========== 教师监控 DTO ==========

// MonitorPanelReq 课程实验监控面板查询参数
// GET /api/v1/courses/:id/experiment-monitor
type MonitorPanelReq struct {
	TemplateID string `form:"template_id"`
	Status     int16  `form:"status" binding:"omitempty,oneof=1 2 3 4 5 6 7 8 9 10"`
}

// MonitorPanelResp 课程实验监控面板响应
type MonitorPanelResp struct {
	Summary  MonitorSummary       `json:"summary"`
	Students []MonitorStudentItem `json:"students"`
}

// MonitorSummary 监控面板汇总信息
type MonitorSummary struct {
	TotalStudents int                  `json:"total_students"`
	Running       int                  `json:"running"`
	Paused        int                  `json:"paused"`
	Completed     int                  `json:"completed"`
	NotStarted    int                  `json:"not_started"`
	AvgProgress   float64              `json:"avg_progress"`
	ResourceUsage MonitorResourceUsage `json:"resource_usage"`
}

// MonitorResourceUsage 监控面板资源使用信息
type MonitorResourceUsage struct {
	CPUUsed     string `json:"cpu_used"`
	CPUTotal    string `json:"cpu_total"`
	MemoryUsed  string `json:"memory_used"`
	MemoryTotal string `json:"memory_total"`
}

// MonitorStudentItem 监控面板中的学生条目
type MonitorStudentItem struct {
	StudentID         string  `json:"student_id"`
	StudentName       string  `json:"student_name"`
	StudentNo         string  `json:"student_no"`
	InstanceID        *string `json:"instance_id"`
	Status            *int16  `json:"status"`
	StatusText        *string `json:"status_text"`
	CheckpointsPassed int     `json:"checkpoints_passed"`
	CheckpointsTotal  int     `json:"checkpoints_total"`
	ProgressPercent   float64 `json:"progress_percent"`
	CPUUsage          *string `json:"cpu_usage"`
	MemoryUsage       *string `json:"memory_usage"`
	StartedAt         *string `json:"started_at"`
	LastActiveAt      *string `json:"last_active_at"`
}

// SendGuidanceReq 向学生发送指导消息请求
// POST /api/v1/experiment-instances/:id/message
type SendGuidanceReq struct {
	Content string `json:"content" binding:"required,max=2000"`
}

// ========== 实验统计 DTO ==========

// ExperimentStatisticsReq 实验统计数据查询参数
// GET /api/v1/courses/:id/experiment-statistics
type ExperimentStatisticsReq struct {
	TemplateID string `form:"template_id"`
}

// ExperimentStatisticsResp 实验统计数据响应
type ExperimentStatisticsResp struct {
	Templates []TemplateStatisticsItem `json:"templates"`
}

// TemplateStatisticsItem 单个模板的统计数据
type TemplateStatisticsItem struct {
	TemplateID    string                 `json:"template_id"`
	TemplateTitle string                 `json:"template_title"`
	Statistics    TemplateStatisticsData `json:"statistics"`
}

// TemplateStatisticsData 模板统计详情
type TemplateStatisticsData struct {
	TotalStudents       int                      `json:"total_students"`
	StartedCount        int                      `json:"started_count"`
	CompletedCount      int                      `json:"completed_count"`
	CompletionRate      float64                  `json:"completion_rate"`
	AvgScore            float64                  `json:"avg_score"`
	MaxScore            float64                  `json:"max_score"`
	MinScore            float64                  `json:"min_score"`
	AvgDurationMinutes  int                      `json:"avg_duration_minutes"`
	AvgAttempts         float64                  `json:"avg_attempts"`
	CheckpointPassRates []CheckpointPassRateItem `json:"checkpoint_pass_rates"`
	ScoreDistribution   ScoreDistribution        `json:"score_distribution"`
}

// CheckpointPassRateItem 检查点通过率条目
type CheckpointPassRateItem struct {
	CheckpointID string   `json:"checkpoint_id"`
	Title        string   `json:"title"`
	PassRate     *float64 `json:"pass_rate,omitempty"`
	AvgScore     *float64 `json:"avg_score,omitempty"`
	MaxScore     *float64 `json:"max_score,omitempty"`
}

// ScoreDistribution 成绩分布
type ScoreDistribution struct {
	Range90To100 int `json:"90_100"`
	Range80To89  int `json:"80_89"`
	Range70To79  int `json:"70_79"`
	Range60To69  int `json:"60_69"`
	Below60      int `json:"below_60"`
}

// ========== 资源配额 DTO ==========

// CreateQuotaReq 创建资源配额请求
// POST /api/v1/resource-quotas
type CreateQuotaReq struct {
	QuotaLevel     int16   `json:"quota_level" binding:"required,oneof=1 2"`
	SchoolID       string  `json:"school_id" binding:"required"`
	CourseID       *string `json:"course_id"`
	MaxCPU         *string `json:"max_cpu" binding:"omitempty,max=20"`
	MaxMemory      *string `json:"max_memory" binding:"omitempty,max=20"`
	MaxStorage     *string `json:"max_storage" binding:"omitempty,max=20"`
	MaxConcurrency *int    `json:"max_concurrency" binding:"omitempty,min=0"`
	MaxPerStudent  *int    `json:"max_per_student" binding:"omitempty,min=1"`
}

// UpdateQuotaReq 编辑资源配额请求
// PUT /api/v1/resource-quotas/:id
type UpdateQuotaReq struct {
	MaxCPU         *string `json:"max_cpu" binding:"omitempty,max=20"`
	MaxMemory      *string `json:"max_memory" binding:"omitempty,max=20"`
	MaxStorage     *string `json:"max_storage" binding:"omitempty,max=20"`
	MaxConcurrency *int    `json:"max_concurrency" binding:"omitempty,min=0"`
	MaxPerStudent  *int    `json:"max_per_student" binding:"omitempty,min=1"`
}

// QuotaListReq 资源配额列表查询参数
// GET /api/v1/resource-quotas
type QuotaListReq struct {
	Page       int    `form:"page" binding:"omitempty,min=1"`
	PageSize   int    `form:"page_size" binding:"omitempty,min=1,max=100"`
	QuotaLevel int16  `form:"quota_level" binding:"omitempty,oneof=1 2"`
	SchoolID   string `form:"school_id"`
}

// QuotaResp 资源配额响应
type QuotaResp struct {
	ID              string  `json:"id"`
	QuotaLevel      int16   `json:"quota_level"`
	QuotaLevelText  string  `json:"quota_level_text"`
	SchoolID        string  `json:"school_id"`
	SchoolName      string  `json:"school_name"`
	CourseID        *string `json:"course_id"`
	CourseTitle     *string `json:"course_title"`
	MaxCPU          string  `json:"max_cpu"`
	MaxMemory       string  `json:"max_memory"`
	MaxStorage      string  `json:"max_storage"`
	MaxConcurrency  int     `json:"max_concurrency"`
	MaxPerStudent   int     `json:"max_per_student"`
	UsedCPU         string  `json:"used_cpu"`
	UsedMemory      string  `json:"used_memory"`
	UsedStorage     string  `json:"used_storage"`
	UsedConcurrency int     `json:"used_concurrency"`
	CreatedAt       string  `json:"created_at"`
	UpdatedAt       string  `json:"updated_at"`
}

// QuotaListResp 资源配额列表响应
// GET /api/v1/resource-quotas
type QuotaListResp struct {
	List       []QuotaResp    `json:"list"`
	Pagination PaginationResp `json:"pagination"`
}

// ========== 学校资源使用 DTO ==========

// ResourceUsageResp 学校资源使用情况响应
// GET /api/v1/schools/:id/resource-usage
type ResourceUsageResp struct {
	SchoolID        string                `json:"school_id"`
	SchoolName      string                `json:"school_name"`
	Quota           ResourceQuotaInfo     `json:"quota"`
	Usage           ResourceUsageInfo     `json:"usage"`
	CourseBreakdown []CourseBreakdownItem `json:"course_breakdown"`
}

// ResourceQuotaInfo 资源配额信息
type ResourceQuotaInfo struct {
	MaxCPU         string `json:"max_cpu"`
	MaxMemory      string `json:"max_memory"`
	MaxStorage     string `json:"max_storage"`
	MaxConcurrency int    `json:"max_concurrency"`
}

// ResourceUsageInfo 资源使用信息
type ResourceUsageInfo struct {
	UsedCPU                 string  `json:"used_cpu"`
	UsedMemory              string  `json:"used_memory"`
	UsedStorage             string  `json:"used_storage"`
	CurrentConcurrency      int     `json:"current_concurrency"`
	CPUUsagePercent         float64 `json:"cpu_usage_percent"`
	MemoryUsagePercent      float64 `json:"memory_usage_percent"`
	StorageUsagePercent     float64 `json:"storage_usage_percent"`
	ConcurrencyUsagePercent float64 `json:"concurrency_usage_percent"`
}

// CourseBreakdownItem 课程资源使用明细
type CourseBreakdownItem struct {
	CourseID           string `json:"course_id"`
	CourseTitle        string `json:"course_title"`
	CurrentConcurrency int    `json:"current_concurrency"`
	MaxConcurrency     int    `json:"max_concurrency"`
	CPUUsed            string `json:"cpu_used"`
	MemoryUsed         string `json:"memory_used"`
}

// ========== 全局管理（超管） DTO ==========

// ExperimentOverviewResp 全平台实验概览响应
// GET /api/v1/admin/experiment-overview
type ExperimentOverviewResp struct {
	TotalInstances   int                 `json:"total_instances"`
	RunningInstances int                 `json:"running_instances"`
	TotalTemplates   int                 `json:"total_templates"`
	TotalImages      int                 `json:"total_images"`
	PendingReviews   int                 `json:"pending_reviews"`
	ClusterStatus    ClusterStatusInfo   `json:"cluster_status"`
	SchoolUsage      []SchoolUsageItem   `json:"school_usage"`
	AlertInstances   []OverviewAlertItem `json:"alert_instances"`
}

// OverviewAlertItem 全局监控页异常实例告警条目
// GET /api/v1/admin/experiment-overview
type OverviewAlertItem struct {
	InstanceID   string `json:"instance_id"`
	StudentID    string `json:"student_id"`
	StudentName  string `json:"student_name"`
	SchoolID     string `json:"school_id"`
	SchoolName   string `json:"school_name"`
	ErrorMessage string `json:"error_message"`
	UpdatedAt    string `json:"updated_at"`
}

// ClusterStatusInfo K8s集群状态信息
type ClusterStatusInfo struct {
	Nodes        int    `json:"nodes"`
	HealthyNodes int    `json:"healthy_nodes"`
	TotalCPU     string `json:"total_cpu"`
	UsedCPU      string `json:"used_cpu"`
	TotalMemory  string `json:"total_memory"`
	UsedMemory   string `json:"used_memory"`
}

// SchoolUsageItem 学校资源使用条目
type SchoolUsageItem struct {
	SchoolID          string  `json:"school_id"`
	SchoolName        string  `json:"school_name"`
	RunningInstances  int     `json:"running_instances"`
	CPUUsed           string  `json:"cpu_used"`
	MemoryUsed        string  `json:"memory_used"`
	QuotaCPU          string  `json:"quota_cpu"`
	QuotaMemory       string  `json:"quota_memory"`
	QuotaUsagePercent float64 `json:"quota_usage_percent"`
}

// AdminInstanceListReq 全平台实验实例列表查询参数
// GET /api/v1/admin/experiment-instances
type AdminInstanceListReq struct {
	Page       int    `form:"page" binding:"omitempty,min=1"`
	PageSize   int    `form:"page_size" binding:"omitempty,min=1,max=100"`
	SchoolID   string `form:"school_id"`
	TemplateID string `form:"template_id"`
	Status     int16  `form:"status" binding:"omitempty,oneof=1 2 3 4 5 6 7 8 9 10"`
	StudentID  string `form:"student_id"`
	SortBy     string `form:"sort_by"`
	SortOrder  string `form:"sort_order" binding:"omitempty,oneof=asc desc"`
}

// AdminInstanceListResp 全平台实验实例列表响应
// GET /api/v1/admin/experiment-instances
type AdminInstanceListResp struct {
	List       []InstanceListItem `json:"list"`
	Pagination PaginationResp     `json:"pagination"`
}

// ContainerResourceResp 全平台容器资源监控响应
// GET /api/v1/admin/container-resources
type ContainerResourceResp struct {
	TotalContainers   int                     `json:"total_containers"`
	RunningContainers int                     `json:"running_containers"`
	TotalCPU          string                  `json:"total_cpu"`
	UsedCPU           string                  `json:"used_cpu"`
	TotalMemory       string                  `json:"total_memory"`
	UsedMemory        string                  `json:"used_memory"`
	Nodes             []ContainerResourceNode `json:"nodes"`
}

// ContainerResourceNode 容器资源节点信息
type ContainerResourceNode struct {
	NodeName       string `json:"node_name"`
	Status         string `json:"status"`
	ContainerCount int    `json:"container_count"`
	CPUCapacity    string `json:"cpu_capacity"`
	CPUUsed        string `json:"cpu_used"`
	MemoryCapacity string `json:"memory_capacity"`
	MemoryUsed     string `json:"memory_used"`
}

// K8sClusterStatusResp K8s集群状态响应
// GET /api/v1/admin/k8s-cluster-status
type K8sClusterStatusResp struct {
	Nodes           []K8sNodeStatus `json:"nodes"`
	TotalPods       int             `json:"total_pods"`
	RunningPods     int             `json:"running_pods"`
	PendingPods     int             `json:"pending_pods"`
	FailedPods      int             `json:"failed_pods"`
	TotalNamespaces int             `json:"total_namespaces"`
}

// K8sNodeStatus K8s节点状态
type K8sNodeStatus struct {
	Name           string `json:"name"`
	Status         string `json:"status"`
	KubeletVersion string `json:"kubelet_version"`
	CPUCapacity    string `json:"cpu_capacity"`
	CPUAllocatable string `json:"cpu_allocatable"`
	MemCapacity    string `json:"mem_capacity"`
	MemAllocatable string `json:"mem_allocatable"`
	PodCount       int    `json:"pod_count"`
	PodCapacity    int    `json:"pod_capacity"`
}

// ========== 共享实验库 DTO ==========

// SharedTemplateListReq 共享实验模板列表查询参数
// GET /api/v1/shared-experiment-templates
type SharedTemplateListReq struct {
	Page      int    `form:"page" binding:"omitempty,min=1"`
	PageSize  int    `form:"page_size" binding:"omitempty,min=1,max=100"`
	Keyword   string `form:"keyword"`
	Ecosystem string `form:"ecosystem"`
	TagID     string `form:"tag_id"`
	SortBy    string `form:"sort_by"`
	SortOrder string `form:"sort_order" binding:"omitempty,oneof=asc desc"`
}

// SharedTemplateListResp 共享实验模板列表响应
// GET /api/v1/shared-experiment-templates
type SharedTemplateListResp struct {
	List       []TemplateListItem `json:"list"`
	Pagination PaginationResp     `json:"pagination"`
}

// ========== 学校管理员视角 DTO ==========

// SchoolImageListReq 本校镜像列表查询参数
// GET /api/v1/school/images
type SchoolImageListReq struct {
	Page       int    `form:"page" binding:"omitempty,min=1"`
	PageSize   int    `form:"page_size" binding:"omitempty,min=1,max=100"`
	Keyword    string `form:"keyword"`
	CategoryID string `form:"category_id"`
	Status     int16  `form:"status" binding:"omitempty,oneof=1 2 3 4"`
}

// SchoolImageListResp 本校镜像列表响应
// GET /api/v1/school/images
type SchoolImageListResp struct {
	List       []ImageListItem `json:"list"`
	Pagination PaginationResp  `json:"pagination"`
}

// SchoolMonitorResp 本校实验监控响应
// GET /api/v1/school/experiment-monitor
type SchoolMonitorResp struct {
	TotalInstances   int                       `json:"total_instances"`
	RunningInstances int                       `json:"running_instances"`
	TotalStudents    int                       `json:"total_students"`
	ActiveStudents   int                       `json:"active_students"`
	ResourceUsage    MonitorResourceUsage      `json:"resource_usage"`
	Courses          []SchoolMonitorCourseItem `json:"courses"`
}

// SchoolMonitorCourseItem 本校监控中的课程条目
type SchoolMonitorCourseItem struct {
	CourseID         string `json:"course_id"`
	CourseTitle      string `json:"course_title"`
	TeacherName      string `json:"teacher_name"`
	RunningInstances int    `json:"running_instances"`
	TotalStudents    int    `json:"total_students"`
}

// CourseQuotaReq 课程资源配额分配请求
// PUT /api/v1/school/course-quotas/:course_id
type CourseQuotaReq struct {
	MaxConcurrency int `json:"max_concurrency" binding:"required,min=0"`
	MaxPerStudent  int `json:"max_per_student" binding:"required,min=1"`
}
