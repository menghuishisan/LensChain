// instance_service.go
// 模块04 — 实验环境：实验实例生命周期业务逻辑
// 负责实例启动、暂停、恢复、重启、提交、销毁、心跳
// 对照 docs/modules/04-实验环境/03-API接口设计.md

package experiment

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"gorm.io/gorm"

	"github.com/lenschain/backend/internal/model/dto"
	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/model/enum"
	"github.com/lenschain/backend/internal/pkg/cache"
	svcctx "github.com/lenschain/backend/internal/pkg/context"
	cronpkg "github.com/lenschain/backend/internal/pkg/cron"
	"github.com/lenschain/backend/internal/pkg/errcode"
	"github.com/lenschain/backend/internal/pkg/snowflake"
	"github.com/lenschain/backend/internal/pkg/storage"
	experimentrepo "github.com/lenschain/backend/internal/repository/experiment"
)

// InstanceService 实验实例服务接口
type InstanceService interface {
	// 实例生命周期
	Create(ctx context.Context, sc *svcctx.ServiceContext, req *dto.CreateInstanceReq) (*dto.CreateInstanceResp, error)
	GetByID(ctx context.Context, sc *svcctx.ServiceContext, id int64) (*dto.InstanceDetailResp, error)
	ExecuteTerminalCommand(ctx context.Context, sc *svcctx.ServiceContext, id int64, containerName, command string) (*TerminalCommandOutput, error)
	GetStudentTerminalOutput(ctx context.Context, sc *svcctx.ServiceContext, id int64, containerName string, tailLines int) (*TerminalOutput, error)
	GetTerminalOutput(ctx context.Context, sc *svcctx.ServiceContext, id int64, tailLines int) (*TerminalOutput, error)
	GetGroupMemberTerminalOutput(ctx context.Context, sc *svcctx.ServiceContext, groupID, studentID int64, tailLines int) (*TerminalOutput, error)
	GetSimEngineProxyTarget(ctx context.Context, sc *svcctx.ServiceContext, sessionID string) (*SimEngineProxyTarget, error)
	RecordSimEngineOperation(ctx context.Context, sc *svcctx.ServiceContext, instanceID int64, payload []byte)
	TouchActivity(ctx context.Context, id int64)
	List(ctx context.Context, sc *svcctx.ServiceContext, req *dto.InstanceListReq) ([]*dto.InstanceListItem, int64, error)
	ListAdmin(ctx context.Context, sc *svcctx.ServiceContext, req *dto.AdminInstanceListReq) ([]*dto.InstanceListItem, int64, error)

	// 暂停 / 恢复 / 重启 / 提交 / 销毁
	Pause(ctx context.Context, sc *svcctx.ServiceContext, id int64) (*dto.PauseInstanceResp, error)
	Resume(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.ResumeInstanceReq) (*dto.ResumeInstanceResp, error)
	Restart(ctx context.Context, sc *svcctx.ServiceContext, id int64) (*dto.CreateInstanceResp, error)
	Submit(ctx context.Context, sc *svcctx.ServiceContext, id int64) (*dto.SubmitInstanceResp, error)
	Destroy(ctx context.Context, sc *svcctx.ServiceContext, id int64) error
	ForceDestroy(ctx context.Context, sc *svcctx.ServiceContext, id int64) error

	// 心跳
	Heartbeat(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.HeartbeatReq) (*dto.HeartbeatResp, error)

	// 检查点 / 快照 / 日志 / 报告 / 教师指导
	VerifyCheckpoints(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.VerifyCheckpointReq) (*dto.VerifyCheckpointResp, error)
	ListCheckpointResults(ctx context.Context, sc *svcctx.ServiceContext, id int64) ([]dto.InstanceCheckpointItem, error)
	GradeCheckpoint(ctx context.Context, sc *svcctx.ServiceContext, resultID int64, req *dto.GradeCheckpointReq) error
	ManualGrade(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.ManualGradeReq) (*dto.ManualGradeResp, error)
	ListSnapshots(ctx context.Context, sc *svcctx.ServiceContext, id int64) ([]dto.SnapshotResp, error)
	CreateSnapshot(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.CreateSnapshotReq) (*dto.SnapshotResp, error)
	RestoreSnapshot(ctx context.Context, sc *svcctx.ServiceContext, id, snapshotID int64) error
	DeleteSnapshot(ctx context.Context, sc *svcctx.ServiceContext, id, snapshotID int64) error
	ListOperationLogs(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.InstanceOpLogListReq) ([]dto.InstanceOpLogItem, int64, error)
	CreateReport(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.CreateReportReq) (*dto.ReportResp, error)
	GetReport(ctx context.Context, sc *svcctx.ServiceContext, id int64) (*dto.ReportResp, error)
	UpdateReport(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.UpdateReportReq) (*dto.ReportResp, error)
	SendGuidance(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.SendGuidanceReq) error
	UploadExperimentFile(ctx context.Context, sc *svcctx.ServiceContext, fileName string, reader io.Reader, fileSize int64, contentType string, purpose string) (*dto.UploadExperimentFileResp, error)
}

// instanceService 实验实例服务实现
type instanceService struct {
	db                    *gorm.DB
	instanceRepo          experimentrepo.InstanceRepository
	containerRepo         experimentrepo.InstanceContainerRepository
	templateRepo          experimentrepo.TemplateRepository
	templateContainerRepo experimentrepo.ContainerRepository
	imageRepo             experimentrepo.ImageRepository
	imageVersionRepo      experimentrepo.ImageVersionRepository
	checkpointRepo        experimentrepo.CheckpointRepository
	checkResultRepo       experimentrepo.CheckpointResultRepository
	groupRepo             experimentrepo.GroupRepository
	groupMemberRepo       experimentrepo.GroupMemberRepository
	snapshotRepo          experimentrepo.SnapshotRepository
	opLogRepo             experimentrepo.OperationLogRepository
	reportRepo            experimentrepo.ReportRepository
	quotaRepo             experimentrepo.QuotaRepository
	initScriptRepo        experimentrepo.InitScriptRepository
	simSceneRepo          experimentrepo.SimSceneRepository
	scenarioRepo          experimentrepo.ScenarioRepository
	linkGroupRepo         experimentrepo.LinkGroupRepository
	linkGroupSceneRepo    experimentrepo.LinkGroupSceneRepository
	k8sSvc                K8sService
	simEngineSvc          SimEngineService
	userNameQuerier       UserNameQuerier
	userSummaryQuerier    UserSummaryQuerier
	schoolNameQuerier     SchoolNameQuerier
	courseQuerier         CourseQuerier
	courseGradeSyncer     CourseGradeSyncer
	enrollmentChecker     EnrollmentChecker
	eventDispatcher       NotificationEventDispatcher
}

const (
	experimentFilePurposeReport       = "experiment_report"
	experimentFilePurposeScenarioPack = "scenario_package"
	experimentFilePurposeImageDoc     = "image_document"
	maxExperimentReportFileSize       = 50 << 20
	maxExperimentPackageFileSize      = 100 << 20
)

// UploadExperimentFile 上传实验文件到对象存储，返回对象键和短期下载地址。
func (s *instanceService) UploadExperimentFile(ctx context.Context, sc *svcctx.ServiceContext, fileName string, reader io.Reader, fileSize int64, contentType string, purpose string) (*dto.UploadExperimentFileResp, error) {
	if err := validateExperimentFileAccess(sc, purpose); err != nil {
		return nil, err
	}
	if err := validateExperimentFile(fileName, contentType, fileSize, purpose); err != nil {
		return nil, err
	}
	extension := strings.ToLower(filepath.Ext(fileName))
	objectName := fmt.Sprintf("experiment/%s/%d/%d%s", purpose, sc.UserID, snowflake.Generate(), extension)
	payload := new(bytes.Buffer)
	if _, err := io.Copy(payload, reader); err != nil {
		return nil, errcode.ErrInvalidParams.WithMessage("读取上传文件失败")
	}
	uploadedObject, err := storage.UploadFile(ctx, objectName, bytes.NewReader(payload.Bytes()), int64(payload.Len()), contentType)
	if err != nil {
		return nil, errcode.ErrMinIO.WithMessage("上传实验文件失败")
	}
	downloadURL, err := storage.GetFileURL(ctx, uploadedObject, time.Hour)
	if err != nil {
		return nil, errcode.ErrMinIO.WithMessage("生成实验文件下载地址失败")
	}
	return &dto.UploadExperimentFileResp{FileName: fileName, FileURL: uploadedObject, DownloadURL: downloadURL, FileSize: fileSize, FileType: contentType}, nil
}

func validateExperimentFileAccess(sc *svcctx.ServiceContext, purpose string) error {
	switch purpose {
	case experimentFilePurposeReport:
		if sc.IsStudent() {
			return nil
		}
	case experimentFilePurposeScenarioPack, experimentFilePurposeImageDoc:
		if sc.IsTeacher() || sc.IsSchoolAdmin() || sc.IsSuperAdmin() {
			return nil
		}
	default:
		return errcode.ErrInvalidParams.WithMessage("不支持的实验文件用途")
	}
	return errcode.ErrForbidden
}

func validateExperimentFile(fileName string, contentType string, fileSize int64, purpose string) error {
	if strings.TrimSpace(fileName) == "" || fileSize <= 0 {
		return errcode.ErrInvalidParams.WithMessage("文件不能为空")
	}
	switch purpose {
	case experimentFilePurposeReport, experimentFilePurposeImageDoc:
		if !isExperimentDocumentType(contentType) {
			return errcode.ErrInvalidParams.WithMessage("仅支持PDF/Word/PPT文档")
		}
		if fileSize > maxExperimentReportFileSize {
			return errcode.ErrInvalidParams.WithMessage("报告文件不能超过50MB")
		}
	case experimentFilePurposeScenarioPack:
		if fileSize > maxExperimentPackageFileSize {
			return errcode.ErrInvalidParams.WithMessage("场景包不能超过100MB")
		}
	}
	return nil
}

func isExperimentDocumentType(contentType string) bool {
	switch contentType {
	case "application/pdf",
		"application/msword",
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		"application/vnd.ms-powerpoint",
		"application/vnd.openxmlformats-officedocument.presentationml.presentation":
		return true
	default:
		return false
	}
}

// NewInstanceService 创建实验实例服务实例
func NewInstanceService(
	db *gorm.DB,
	instanceRepo experimentrepo.InstanceRepository,
	containerRepo experimentrepo.InstanceContainerRepository,
	templateRepo experimentrepo.TemplateRepository,
	templateContainerRepo experimentrepo.ContainerRepository,
	imageRepo experimentrepo.ImageRepository,
	imageVersionRepo experimentrepo.ImageVersionRepository,
	checkpointRepo experimentrepo.CheckpointRepository,
	checkResultRepo experimentrepo.CheckpointResultRepository,
	groupRepo experimentrepo.GroupRepository,
	groupMemberRepo experimentrepo.GroupMemberRepository,
	snapshotRepo experimentrepo.SnapshotRepository,
	opLogRepo experimentrepo.OperationLogRepository,
	reportRepo experimentrepo.ReportRepository,
	quotaRepo experimentrepo.QuotaRepository,
	initScriptRepo experimentrepo.InitScriptRepository,
	simSceneRepo experimentrepo.SimSceneRepository,
	scenarioRepo experimentrepo.ScenarioRepository,
	linkGroupRepo experimentrepo.LinkGroupRepository,
	linkGroupSceneRepo experimentrepo.LinkGroupSceneRepository,
	k8sSvc K8sService,
	simEngineSvc SimEngineService,
	userNameQuerier UserNameQuerier,
	userSummaryQuerier UserSummaryQuerier,
	schoolNameQuerier SchoolNameQuerier,
	courseQuerier CourseQuerier,
	courseGradeSyncer CourseGradeSyncer,
	enrollmentChecker EnrollmentChecker,
	eventDispatcher NotificationEventDispatcher,
) InstanceService {
	return &instanceService{
		db:                    db,
		instanceRepo:          instanceRepo,
		containerRepo:         containerRepo,
		templateRepo:          templateRepo,
		templateContainerRepo: templateContainerRepo,
		imageRepo:             imageRepo,
		imageVersionRepo:      imageVersionRepo,
		checkpointRepo:        checkpointRepo,
		checkResultRepo:       checkResultRepo,
		groupRepo:             groupRepo,
		groupMemberRepo:       groupMemberRepo,
		snapshotRepo:          snapshotRepo,
		opLogRepo:             opLogRepo,
		reportRepo:            reportRepo,
		quotaRepo:             quotaRepo,
		initScriptRepo:        initScriptRepo,
		simSceneRepo:          simSceneRepo,
		scenarioRepo:          scenarioRepo,
		linkGroupRepo:         linkGroupRepo,
		linkGroupSceneRepo:    linkGroupSceneRepo,
		k8sSvc:                k8sSvc,
		simEngineSvc:          simEngineSvc,
		userNameQuerier:       userNameQuerier,
		userSummaryQuerier:    userSummaryQuerier,
		schoolNameQuerier:     schoolNameQuerier,
		courseQuerier:         courseQuerier,
		courseGradeSyncer:     courseGradeSyncer,
		enrollmentChecker:     enrollmentChecker,
		eventDispatcher:       eventDispatcher,
	}
}

// ---------------------------------------------------------------------------
// 启动实验环境
// ---------------------------------------------------------------------------

// Create 启动实验环境
// POST /api/v1/experiment-instances
func (s *instanceService) Create(ctx context.Context, sc *svcctx.ServiceContext, req *dto.CreateInstanceReq) (*dto.CreateInstanceResp, error) {
	// 解析模板ID
	templateID, err := snowflake.ParseString(req.TemplateID)
	if err != nil {
		return nil, errcode.ErrTemplateNotFound
	}
	if err := s.normalizeSnapshotCreateRequest(ctx, sc, templateID, req); err != nil {
		return nil, err
	}
	if err := s.normalizeGroupCreateRequest(ctx, sc, templateID, req); err != nil {
		return nil, err
	}

	// 获取模板（含完整配置）
	templateAggregate, err := loadTemplateAggregate(
		ctx,
		s.templateRepo,
		s.templateContainerRepo,
		s.checkpointRepo,
		s.initScriptRepo,
		s.simSceneRepo,
		nil,
		nil,
		nil,
		templateID,
	)
	if err != nil {
		return nil, err
	}
	template := templateAggregate.Template

	// 模板必须已发布
	if template.Status != enum.TemplateStatusPublished {
		return nil, errcode.ErrTemplateNotPublished
	}
	if template.TopologyMode != nil && *template.TopologyMode == enum.TopologyModeShared && req.LessonID == nil {
		return nil, errcode.ErrInvalidParams.WithMessage("共享基础设施实验启动时必须提供课时ID")
	}

	// 同一模板已有活动实例时复用现有实例，避免重复创建。
	activeInstance, err := s.findReusableInstance(ctx, templateID, sc.UserID)
	if err != nil {
		return nil, err
	}
	if activeInstance != nil {
		idStr := strconv.FormatInt(activeInstance.ID, 10)
		return &dto.CreateInstanceResp{
			InstanceID:   &idStr,
			SimSessionID: activeInstance.SimSessionID,
			Status:       activeInstance.Status,
			StatusText:   enum.GetInstanceStatusText(activeInstance.Status),
			AttemptNo:    activeInstance.AttemptNo,
		}, nil
	}

	// 课程选课校验
	if req.CourseID != nil {
		courseID, _ := snowflake.ParseString(*req.CourseID)
		if courseID > 0 {
			enrolled, err := s.enrollmentChecker.IsEnrolled(ctx, courseID, sc.UserID)
			if err != nil {
				return nil, err
			}
			if !enrolled {
				return nil, errcode.ErrNotCourseStudent
			}
		}
	}

	// 个人并发限制
	runningCount, err := s.instanceRepo.CountRunningByStudent(ctx, sc.UserID)
	if err != nil {
		return nil, err
	}
	maxPerStudent := 2 // 默认
	schoolQuota, _ := s.quotaRepo.GetBySchoolID(ctx, sc.SchoolID)
	if schoolQuota != nil && schoolQuota.MaxPerStudent > 0 {
		maxPerStudent = schoolQuota.MaxPerStudent
	}
	if req.CourseID != nil {
		courseID, _ := snowflake.ParseString(*req.CourseID)
		if courseID > 0 {
			courseQuota, _ := s.quotaRepo.GetByCourseID(ctx, courseID)
			if courseQuota != nil && courseQuota.MaxPerStudent > 0 {
				maxPerStudent = courseQuota.MaxPerStudent
			}
		}
	}
	if runningCount >= int64(maxPerStudent) {
		return nil, errcode.ErrConcurrencyExceeded.WithMessage("已达个人并发实验上限")
	}

	// 获取最大尝试次数
	maxAttempt, err := s.instanceRepo.GetMaxAttemptNo(ctx, templateID, sc.UserID)
	if err != nil {
		return nil, err
	}

	// 学校并发限制
	if schoolQuota != nil && schoolQuota.MaxConcurrency != nil && *schoolQuota.MaxConcurrency > 0 {
		schoolRunning, _ := s.instanceRepo.CountRunningBySchool(ctx, sc.SchoolID)
		if schoolRunning >= int64(*schoolQuota.MaxConcurrency) {
			return nil, errcode.ErrResourceQuotaExceeded
		}
	}

	// 课程并发限制
	if req.CourseID != nil {
		courseID, _ := snowflake.ParseString(*req.CourseID)
		if courseID > 0 {
			courseQuota, _ := s.quotaRepo.GetByCourseID(ctx, courseID)
			if courseQuota != nil && courseQuota.MaxConcurrency != nil && *courseQuota.MaxConcurrency > 0 {
				courseRunning, _ := s.instanceRepo.CountRunningByCourse(ctx, courseID)
				if courseRunning >= int64(*courseQuota.MaxConcurrency) {
					return s.createQueuedInstance(ctx, sc, template, req, maxAttempt+1)
				}
			}
		}
	}

	// 创建实例记录
	now := time.Now()
	instance := &entity.ExperimentInstance{
		ID:             snowflake.Generate(),
		TemplateID:     templateID,
		StudentID:      sc.UserID,
		SchoolID:       sc.SchoolID,
		ExperimentType: template.ExperimentType,
		Status:         enum.InstanceStatusCreating,
		AttemptNo:      maxAttempt + 1,
		StartedAt:      &now,
		LastActiveAt:   &now,
	}

	// 可选字段
	if req.CourseID != nil {
		courseID, _ := snowflake.ParseString(*req.CourseID)
		instance.CourseID = &courseID
	}
	if req.LessonID != nil {
		lessonID, _ := snowflake.ParseString(*req.LessonID)
		instance.LessonID = &lessonID
	}
	if req.AssignmentID != nil {
		assignID, _ := snowflake.ParseString(*req.AssignmentID)
		instance.AssignmentID = &assignID
	}
	if req.GroupID != nil {
		groupID, _ := snowflake.ParseString(*req.GroupID)
		instance.GroupID = &groupID
	}

	if err := s.instanceRepo.Create(ctx, instance); err != nil {
		return nil, err
	}
	if instance.GroupID != nil {
		member, err := s.groupMemberRepo.GetByGroupAndStudent(ctx, *instance.GroupID, sc.UserID)
		if err != nil {
			return nil, err
		}
		if err := s.groupMemberRepo.UpdateFields(ctx, member.ID, map[string]interface{}{
			"instance_id": instance.ID,
		}); err != nil {
			return nil, err
		}
	}

	// 异步创建环境（根据实验类型）
	snapshotID := ""
	if req.SnapshotID != nil {
		snapshotID = *req.SnapshotID
	}
	cronpkg.RunAsync("模块04实验环境创建", func(asyncCtx context.Context) {
		s.provisionEnvironment(detachContext(ctx), instance, templateAggregate, snapshotID, true)
	})

	// 记录操作日志
	s.recordOpLog(ctx, instance.ID, sc.UserID, enum.ActionStart, nil, nil, nil, nil, nil)
	s.pushCourseMonitorStatusChange(instance, 0, int(instance.Status))

	// 更新配额已用并发
	_ = s.quotaRepo.IncrUsedConcurrency(ctx, sc.SchoolID, 1)

	idStr := strconv.FormatInt(instance.ID, 10)
	resp := &dto.CreateInstanceResp{
		InstanceID: &idStr,
		Status:     instance.Status,
		StatusText: enum.GetInstanceStatusText(instance.Status),
		AttemptNo:  instance.AttemptNo,
	}
	readySec := 30
	resp.EstimatedReadySeconds = &readySec
	return resp, nil
}

// normalizeGroupCreateRequest 规范化多人实验启动请求，统一校验分组归属、课程归属与成员身份。
func (s *instanceService) normalizeGroupCreateRequest(ctx context.Context, sc *svcctx.ServiceContext, templateID int64, req *dto.CreateInstanceReq) error {
	template, err := s.templateRepo.GetByID(ctx, templateID)
	if err != nil {
		return errcode.ErrTemplateNotFound
	}
	isCollaborative := template.TopologyMode != nil && *template.TopologyMode == enum.TopologyModeCollaborate

	if req.GroupID == nil || strings.TrimSpace(*req.GroupID) == "" {
		if isCollaborative {
			return errcode.ErrInvalidParams.WithMessage("多人协作实验必须指定分组ID")
		}
		return nil
	}
	if !isCollaborative {
		return errcode.ErrInvalidParams.WithMessage("仅多人协作实验允许指定分组ID")
	}

	groupID, err := snowflake.ParseString(*req.GroupID)
	if err != nil {
		return errcode.ErrGroupNotFound
	}
	group, err := s.groupRepo.GetByID(ctx, groupID)
	if err != nil {
		return errcode.ErrGroupNotFound
	}
	if group.TemplateID != templateID {
		return errcode.ErrInvalidParams.WithMessage("分组不属于当前实验模板")
	}

	member, err := s.groupMemberRepo.GetByGroupAndStudent(ctx, groupID, sc.UserID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errcode.ErrForbidden.WithMessage("您不在该实验分组中")
		}
		return err
	}
	if member.RoleID == nil {
		return errcode.ErrInvalidParams.WithMessage("分组成员尚未分配角色")
	}

	if group.Status == enum.GroupStatusForming {
		return errcode.ErrGroupNotJoinable.WithMessage("分组尚未组建完成")
	}

	groupCourseID := strconv.FormatInt(group.CourseID, 10)
	if req.CourseID != nil && strings.TrimSpace(*req.CourseID) != "" {
		courseID, err := snowflake.ParseString(*req.CourseID)
		if err != nil {
			return errcode.ErrInvalidParams.WithMessage("课程ID无效")
		}
		if courseID != group.CourseID {
			return errcode.ErrInvalidParams.WithMessage("分组不属于当前课程")
		}
	} else {
		req.CourseID = &groupCourseID
	}

	return nil
}

// normalizeSnapshotCreateRequest 规范化从快照恢复启动的请求，并回填历史实例上下文。
func (s *instanceService) normalizeSnapshotCreateRequest(ctx context.Context, sc *svcctx.ServiceContext, templateID int64, req *dto.CreateInstanceReq) error {
	if req.SnapshotID == nil || strings.TrimSpace(*req.SnapshotID) == "" {
		return nil
	}

	snapshotID, err := snowflake.ParseString(*req.SnapshotID)
	if err != nil {
		return errcode.ErrSnapshotNotFound
	}
	snapshot, err := s.snapshotRepo.GetByID(ctx, snapshotID)
	if err != nil {
		return errcode.ErrSnapshotNotFound
	}

	sourceInstance, err := s.instanceRepo.GetByID(ctx, snapshot.InstanceID)
	if err != nil {
		return err
	}
	if sourceInstance.StudentID != sc.UserID {
		return errcode.ErrForbidden.WithMessage("快照不属于当前学生")
	}
	if sourceInstance.TemplateID != templateID {
		return errcode.ErrInvalidParams.WithMessage("快照不属于当前实验模板")
	}

	if req.CourseID == nil && sourceInstance.CourseID != nil {
		value := strconv.FormatInt(*sourceInstance.CourseID, 10)
		req.CourseID = &value
	}
	if req.LessonID == nil && sourceInstance.LessonID != nil {
		value := strconv.FormatInt(*sourceInstance.LessonID, 10)
		req.LessonID = &value
	}
	if req.AssignmentID == nil && sourceInstance.AssignmentID != nil {
		value := strconv.FormatInt(*sourceInstance.AssignmentID, 10)
		req.AssignmentID = &value
	}
	if req.GroupID == nil && sourceInstance.GroupID != nil {
		value := strconv.FormatInt(*sourceInstance.GroupID, 10)
		req.GroupID = &value
	}

	return nil
}

// provisionEnvironment 异步创建实验环境（K8s 容器 + SimEngine 会话）
func (s *instanceService) provisionEnvironment(ctx context.Context, instance *entity.ExperimentInstance, template *TemplateAggregate, snapshotID string, releaseConcurrencyOnError bool) {
	var accessURL string
	var errMsg string
	oldStatus := instance.Status
	var restoreSnapshot *entity.InstanceSnapshot
	var simSessionID string
	var collectorWebSocketURL string

	defer func() {
		fields := map[string]interface{}{"updated_at": time.Now()}
		if errMsg != "" {
			fields["status"] = enum.InstanceStatusError
			fields["error_message"] = errMsg
		} else {
			fields["status"] = enum.InstanceStatusRunning
			if accessURL != "" {
				fields["access_url"] = accessURL
			}
		}
		_ = s.instanceRepo.UpdateFields(ctx, instance.ID, fields)
		// 缓存实例状态到 Redis
		_ = cache.Set(ctx, fmt.Sprintf("%s:%d", cache.KeyExpInstanceStatus, instance.ID),
			fmt.Sprintf("%d", fields["status"]), 24*time.Hour)
		newStatus := int(enum.InstanceStatusRunning)
		if statusValue, ok := fields["status"].(int16); ok {
			newStatus = int(statusValue)
		}
		s.pushCourseMonitorStatusChange(instance, int(oldStatus), newStatus)
		if errMsg != "" && releaseConcurrencyOnError {
			_ = s.quotaRepo.DecrUsedConcurrency(ctx, instance.SchoolID, 1)
			if instance.CourseID != nil {
				s.activateNextQueuedInstance(ctx, *instance.CourseID)
			}
			s.pushCourseMonitorInstanceError(instance, errMsg)
		}
		if instance.GroupID != nil {
			s.refreshGroupStatus(ctx, *instance.GroupID)
		}
	}()

	if strings.TrimSpace(snapshotID) != "" {
		snapID, parseErr := snowflake.ParseString(snapshotID)
		if parseErr != nil {
			errMsg = "恢复快照无效"
			return
		}
		restoreSnapshot, _ = s.snapshotRepo.GetByID(ctx, snapID)
		if restoreSnapshot == nil {
			errMsg = "恢复快照不存在"
			return
		}
	}

	// 1. 纯仿真 / 混合实验 → 创建 SimEngine 会话
	if template.Template.ExperimentType == enum.ExperimentTypeSimulation || template.Template.ExperimentType == enum.ExperimentTypeMixed {
		simReq, err := s.buildSimSessionRequest(ctx, instance, template)
		if err != nil {
			errMsg = fmt.Sprintf("构建仿真会话请求失败: %v", err)
			return
		}

		session, err := s.simEngineSvc.CreateSession(ctx, simReq)
		if err != nil {
			errMsg = fmt.Sprintf("创建仿真会话失败: %v", err)
			return
		}

		simSessionID = session.SessionID
		collectorWebSocketURL = buildCollectorWebSocketURL(session.WebSocketURL)
		_ = s.instanceRepo.UpdateFields(ctx, instance.ID, map[string]interface{}{
			"sim_session_id":    session.SessionID,
			"sim_websocket_url": session.WebSocketURL,
		})
		instance.SimSessionID = &session.SessionID
		instance.SimWebSocketURL = &session.WebSocketURL
	}

	// 2. 真实环境 / 混合实验 → 部署 K8s 容器
	if template.Template.ExperimentType == enum.ExperimentTypeReal || template.Template.ExperimentType == enum.ExperimentTypeMixed {
		containerPlan, err := s.resolveRuntimeContainerPlan(ctx, instance, template)
		if err != nil {
			errMsg = err.Error()
			return
		}
		nsName := fmt.Sprintf("exp-%d", instance.ID)
		ns := nsName
		_ = s.instanceRepo.UpdateFields(ctx, instance.ID, map[string]interface{}{"namespace": ns})
		networkPolicy := buildInstanceNetworkPolicy(instance)
		serviceDiscoveryEnv := containerPlan.ServiceDiscoveryEnvs

		sharedNamespace := ""
		if template.Template.TopologyMode != nil && *template.Template.TopologyMode == enum.TopologyModeShared {
			sharedNamespace = buildSharedNamespaceName(instance.TemplateID, *instance.LessonID)
			if err := s.k8sSvc.CreateNamespace(ctx, sharedNamespace, buildSharedNamespaceLabels(instance), buildNamespaceResourceSpec(template.Template)); err != nil {
				errMsg = fmt.Sprintf("创建共享命名空间失败: %v", err)
				return
			}
			for _, tc := range containerPlan.SharedContainers {
				sharedPodName := fmt.Sprintf("%s-%s", sharedNamespace, tc.ContainerName)
				existingStatus, statusErr := s.k8sSvc.GetPodStatus(ctx, sharedNamespace, sharedPodName)
				if statusErr == nil {
					if existingStatus != nil && (existingStatus.Status == "Running" || existingStatus.Status == "Pending") {
						continue
					}
					errMsg = fmt.Sprintf("共享容器 %s 已存在但状态不可复用: %s", tc.ContainerName, existingStatus.Status)
					return
				}
				if !apierrors.IsNotFound(statusErr) {
					errMsg = fmt.Sprintf("查询共享容器 %s 状态失败: %v", tc.ContainerName, statusErr)
					return
				}

				containerSpec, collectorSpec, err := s.buildContainerSpecWithServiceDiscovery(
					ctx,
					tc,
					template,
					containerPlan.SharedServiceDiscoveryEnvs,
				)
				if err != nil {
					errMsg = fmt.Sprintf("构建共享容器 %s 规格失败: %v", tc.ContainerName, err)
					return
				}
				if collectorSpec != nil {
					collectorSpec.SessionID = simSessionID
					collectorSpec.CoreWebSocketURL = collectorWebSocketURL
				}
				deployReq := &DeployPodRequest{
					Namespace:     sharedNamespace,
					PodName:       sharedPodName,
					Containers:    []ContainerSpec{containerSpec},
					Labels: map[string]string{
						"app":            "lenschain-experiment",
						"template-id":    strconv.FormatInt(instance.TemplateID, 10),
						"container-name": tc.ContainerName,
						"scope":          "shared",
					},
					NetworkPolicy: buildSharedNamespaceNetworkPolicy(instance),
					Collector:     collectorSpec,
				}
				if _, err := s.k8sSvc.DeployPod(ctx, deployReq); err != nil {
					errMsg = fmt.Sprintf("部署共享容器 %s 失败: %v", tc.ContainerName, err)
					return
				}
			}
			networkPolicy = buildSharedStudentNetworkPolicy(instance)
			serviceDiscoveryEnv = mergeStringMap(serviceDiscoveryEnv, buildSharedRuntimeServiceDiscoveryEnvVars(sharedNamespace, containerPlan.SharedContainers))
			labels := buildStudentNamespaceLabels(instance, sharedNamespace)
			if err := s.k8sSvc.CreateNamespace(ctx, nsName, labels, buildNamespaceResourceSpec(template.Template)); err != nil {
				errMsg = fmt.Sprintf("创建学生命名空间失败: %v", err)
				return
			}
		} else {
			labels := buildInstanceNamespaceLabels(instance)
			if err := s.k8sSvc.CreateNamespace(ctx, nsName, labels, buildNamespaceResourceSpec(template.Template)); err != nil {
				errMsg = fmt.Sprintf("创建命名空间失败: %v", err)
				return
			}
		}

		// 部署容器
		for _, tc := range containerPlan.Containers {
			containerSpec, collectorSpec, err := s.buildContainerSpecWithServiceDiscovery(
				ctx,
				tc,
				template,
				serviceDiscoveryEnv,
			)
			if err != nil {
				errMsg = fmt.Sprintf("构建容器 %s 规格失败: %v", tc.ContainerName, err)
				return
			}
			if collectorSpec != nil {
				collectorSpec.SessionID = simSessionID
				collectorSpec.CoreWebSocketURL = collectorWebSocketURL
			}
			podLabels := map[string]string{
				"app":            "lenschain-experiment",
				"instance-id":    strconv.FormatInt(instance.ID, 10),
				"container-name": tc.ContainerName,
			}
			deployReq := &DeployPodRequest{
				Namespace:     nsName,
				PodName:       fmt.Sprintf("%s-%s", nsName, tc.ContainerName),
				Containers:    []ContainerSpec{containerSpec},
				Labels:        podLabels,
				NetworkPolicy: networkPolicy,
				Collector:     collectorSpec,
			}

			deployResp, err := s.k8sSvc.DeployPod(ctx, deployReq)
			if err != nil {
				errMsg = fmt.Sprintf("部署容器 %s 失败: %v", tc.ContainerName, err)
				return
			}

			// 记录实例容器
			ic := &entity.InstanceContainer{
				ID:                  snowflake.Generate(),
				InstanceID:          instance.ID,
				TemplateContainerID: tc.ID,
				ContainerName:       tc.ContainerName,
				PodName:             &deployResp.PodName,
				InternalIP:          &deployResp.InternalIP,
				Status:              enum.ContainerStatusRunning,
			}
			_ = s.containerRepo.Create(ctx, ic)

			if tc.IsPrimary && deployResp.InternalIP != "" {
				accessURL = fmt.Sprintf("http://%s", deployResp.InternalIP)
			}
		}

		// 新建环境执行初始化脚本；从快照恢复时跳过，避免覆盖恢复态数据。
		if restoreSnapshot == nil {
			for _, script := range template.InitScripts {
				targetNamespace := nsName
				if _, ok := containerPlan.ContainerNameSet[script.TargetContainer]; !ok {
					if sharedNamespace == "" {
						continue
					}
					if _, sharedOK := containerPlan.SharedContainerNameSet[script.TargetContainer]; !sharedOK {
						continue
					}
					targetNamespace = sharedNamespace
				}
				podName := fmt.Sprintf("%s-%s", targetNamespace, script.TargetContainer)
				scriptCtx := ctx
				cancel := func() {}
				if script.Timeout > 0 {
					scriptCtx, cancel = context.WithTimeout(ctx, time.Duration(script.Timeout)*time.Second)
				}
				_, _ = s.k8sSvc.ExecInPod(scriptCtx, targetNamespace, podName, script.TargetContainer, script.ScriptContent)
				cancel()
			}
		} else if err := s.restoreInstanceRuntimeState(ctx, instance, restoreSnapshot); err != nil {
			errMsg = fmt.Sprintf("恢复容器状态失败: %v", err)
			return
		}
	}

	// 3. 从快照恢复
	if restoreSnapshot != nil && restoreSnapshot.SimEngineState != nil {
		if simSessionID == "" && instance.SimSessionID != nil {
			simSessionID = *instance.SimSessionID
		}
		if simSessionID != "" {
			if err := s.simEngineSvc.RestoreSnapshot(ctx, simSessionID, strconv.FormatInt(restoreSnapshot.ID, 10)); err != nil {
				errMsg = fmt.Sprintf("恢复仿真快照失败: %v", err)
				return
			}
		}
	}

	// 4. 混合实验 → 启动数据采集
	if template.Template.ExperimentType == enum.ExperimentTypeMixed {
		if simSessionID == "" && instance.SimSessionID != nil {
			simSessionID = *instance.SimSessionID
		}
		if simSessionID != "" {
			_ = s.simEngineSvc.StartDataCollection(ctx, simSessionID, buildMixedDataCollectionConfig(template))
		}
	}

	if restoreSnapshot != nil {
		_ = s.syncRestoredContainerState(ctx, instance.ID)
	}
}

// refreshGroupStatus 刷新多人实验分组的聚合状态。
func (s *instanceService) refreshGroupStatus(ctx context.Context, groupID int64) {
	group, err := s.groupRepo.GetByID(ctx, groupID)
	if err != nil {
		return
	}
	members, err := s.groupMemberRepo.ListByGroupID(ctx, groupID)
	if err != nil {
		return
	}
	instances, err := s.instanceRepo.ListByGroupID(ctx, groupID)
	if err != nil {
		return
	}
	status := deriveGroupAggregateStatus(group, members, buildLatestInstanceByStudent(instances))
	if status == group.Status {
		return
	}
	_ = s.groupRepo.UpdateFields(ctx, groupID, map[string]interface{}{"status": status})
}

// buildContainerSpec 根据模板容器配置构建 K8s ContainerSpec
func (s *instanceService) buildContainerSpec(ctx context.Context, tc entity.TemplateContainer, template *TemplateAggregate) (ContainerSpec, *CollectorSidecarSpec, error) {
	return s.buildContainerSpecWithServiceDiscovery(ctx, tc, template, nil)
}

// buildContainerSpecWithServiceDiscovery 根据模板容器配置和运行时服务发现信息构建 K8s ContainerSpec。
func (s *instanceService) buildContainerSpecWithServiceDiscovery(
	ctx context.Context,
	tc entity.TemplateContainer,
	template *TemplateAggregate,
	serviceDiscoveryEnvs map[string]string,
) (ContainerSpec, *CollectorSidecarSpec, error) {
	version, err := s.imageVersionRepo.GetByID(ctx, tc.ImageVersionID)
	if err != nil {
		return ContainerSpec{}, nil, errcode.ErrImageVersionNotFound
	}
	image, err := s.imageRepo.GetByID(ctx, version.ImageID)
	if err != nil {
		return ContainerSpec{}, nil, errcode.ErrImageNotFound
	}
	allContainers := make([]entity.TemplateContainer, 0, len(template.Containers))
	for _, container := range template.Containers {
		allContainers = append(allContainers, *container)
	}
	if serviceDiscoveryEnvs == nil {
		serviceDiscoveryEnvs = buildServiceDiscoveryEnvVars(allContainers)
	}
	spec, err := mergeContainerConfigWithServiceDiscovery(tc, image, version, allContainers, serviceDiscoveryEnvs)
	if err != nil {
		return ContainerSpec{}, nil, err
	}
	return spec, buildCollectorInjectionPlan(template, &tc, image), nil
}

// buildCollectorWebSocketURL 将 SimEngine 前端会话地址转换为 Collector sidecar 使用的采集地址。
func buildCollectorWebSocketURL(simWebSocketURL string) string {
	if simWebSocketURL == "" {
		return ""
	}
	return strings.Replace(simWebSocketURL, "/api/v1/ws/sim-engine/", "/api/v1/ws/collector/", 1)
}

// buildSimSessionRequest 根据模板仿真场景构建 SimEngine 会话请求
func (s *instanceService) buildSimSessionRequest(ctx context.Context, instance *entity.ExperimentInstance, template *TemplateAggregate) (*CreateSimSessionRequest, error) {
	req := &CreateSimSessionRequest{
		InstanceID: instance.ID,
		StudentID:  instance.StudentID,
	}

	scenes := make([]SimSceneConfig, 0, len(template.SimScenes))
	hasLinkGroup := false

	for _, ts := range template.SimScenes {
		scenario, err := s.scenarioRepo.GetByID(ctx, ts.ScenarioID)
		if err != nil {
			continue
		}

		sceneCfg := decodeSimSceneConfig(json.RawMessage(ts.Config))

		sc := SimSceneConfig{
			SceneCode:        scenario.Code,
			ScenarioID:       strconv.FormatInt(ts.ScenarioID, 10),
			Params:           sceneCfg.SceneParams,
			InitialState:     sceneCfg.InitialState,
			LayoutPosition:   json.RawMessage(ts.LayoutPosition),
			DataSourceConfig: json.RawMessage(ts.DataSourceConfig),
			DataSourceMode:   toSimSceneDataSourceMode(sceneCfg.DataSourceMode, scenario.DataSourceMode),
		}

		// 联动组
		if ts.LinkGroupID != nil {
			sc.LinkGroupID = strconv.FormatInt(*ts.LinkGroupID, 10)
			linkGroup, err := s.linkGroupRepo.GetByID(ctx, *ts.LinkGroupID)
			if err == nil {
				sc.LinkGroupCode = linkGroup.Code
				sc.SharedState = json.RawMessage(linkGroup.SharedStateSchema)
				hasLinkGroup = true
			}
		}

		scenes = append(scenes, sc)
	}

	req.Scenes = scenes
	req.LinkageEnabled = hasLinkGroup

	return req, nil
}

// toSimSceneDataSourceMode 将模板场景配置中的数据源模式映射为 SimEngine 请求使用的字符串编码。
func toSimSceneDataSourceMode(templateMode int16, scenarioDefault int16) string {
	mode := templateMode
	if !enum.IsValidDataSourceMode(mode) {
		mode = scenarioDefault
	}
	switch mode {
	case enum.DataSourceModeCollect:
		return "collection"
	case enum.DataSourceModeDual:
		return "dual"
	default:
		return "simulation"
	}
}

// findReusableInstance 查找同一学生在同一模板下可直接复用的活动实例。
func (s *instanceService) findReusableInstance(ctx context.Context, templateID, studentID int64) (*entity.ExperimentInstance, error) {
	instances, err := s.instanceRepo.ListByTemplateAndStudent(ctx, templateID, studentID)
	if err != nil {
		return nil, err
	}
	for _, instance := range instances {
		switch instance.Status {
		case enum.InstanceStatusCreating, enum.InstanceStatusInitializing, enum.InstanceStatusRunning, enum.InstanceStatusQueued:
			return instance, nil
		}
	}
	return nil, nil
}

// createQueuedInstance 在课程并发已满时创建排队中的实例记录。
func (s *instanceService) createQueuedInstance(
	ctx context.Context,
	sc *svcctx.ServiceContext,
	template *entity.ExperimentTemplate,
	req *dto.CreateInstanceReq,
	attemptNo int,
) (*dto.CreateInstanceResp, error) {
	now := time.Now()
	instance := &entity.ExperimentInstance{
		ID:             snowflake.Generate(),
		TemplateID:     template.ID,
		StudentID:      sc.UserID,
		SchoolID:       sc.SchoolID,
		ExperimentType: template.ExperimentType,
		Status:         enum.InstanceStatusQueued,
		AttemptNo:      attemptNo,
		StartedAt:      &now,
		LastActiveAt:   &now,
	}
	if req.CourseID != nil {
		courseID, _ := snowflake.ParseString(*req.CourseID)
		instance.CourseID = &courseID
	}
	if req.LessonID != nil {
		lessonID, _ := snowflake.ParseString(*req.LessonID)
		instance.LessonID = &lessonID
	}
	if req.AssignmentID != nil {
		assignmentID, _ := snowflake.ParseString(*req.AssignmentID)
		instance.AssignmentID = &assignmentID
	}
	if req.GroupID != nil {
		groupID, _ := snowflake.ParseString(*req.GroupID)
		instance.GroupID = &groupID
	}
	if err := s.instanceRepo.Create(ctx, instance); err != nil {
		return nil, err
	}

	queuePosition := 1
	if instance.CourseID != nil && cache.Get() != nil {
		queueKey := fmt.Sprintf("%s:%d", cache.KeyExpQueue, *instance.CourseID)
		if length, err := cache.Get().RPush(ctx, queueKey, strconv.FormatInt(instance.ID, 10)).Result(); err == nil {
			queuePosition = int(length)
		}
		if req.SnapshotID != nil && strings.TrimSpace(*req.SnapshotID) != "" {
			snapshotKey := fmt.Sprintf("%s%d", cache.KeyExpQueueSnapshot, instance.ID)
			_ = cache.Set(ctx, snapshotKey, *req.SnapshotID, 24*time.Hour)
		}
	}

	idStr := strconv.FormatInt(instance.ID, 10)
	waitSeconds := queuePosition * 60
	return &dto.CreateInstanceResp{
		InstanceID:           &idStr,
		Status:               enum.InstanceStatusQueued,
		StatusText:           enum.GetInstanceStatusText(enum.InstanceStatusQueued),
		AttemptNo:            instance.AttemptNo,
		QueuePosition:        &queuePosition,
		EstimatedWaitSeconds: &waitSeconds,
	}, nil
}

// buildNamespaceResourceSpec 根据实验模板生成命名空间资源隔离规格。
func buildNamespaceResourceSpec(template *entity.ExperimentTemplate) *NamespaceResourceSpec {
	if template == nil {
		return nil
	}
	spec := &NamespaceResourceSpec{}
	if template.CPULimit != nil {
		spec.HardCPU = *template.CPULimit
		spec.DefaultContainerCPU = *template.CPULimit
	}
	if template.MemoryLimit != nil {
		spec.HardMemory = *template.MemoryLimit
		spec.DefaultContainerMemory = *template.MemoryLimit
	}
	if template.DiskLimit != nil {
		spec.HardStorage = *template.DiskLimit
		spec.DefaultContainerStorage = *template.DiskLimit
	}
	if spec.HardCPU == "" && spec.HardMemory == "" && spec.HardStorage == "" &&
		spec.DefaultContainerCPU == "" && spec.DefaultContainerMemory == "" && spec.DefaultContainerStorage == "" {
		return nil
	}
	return spec
}

// ---------------------------------------------------------------------------
// 获取实例详情
// ---------------------------------------------------------------------------

// GetByID 获取实验实例详情
func (s *instanceService) GetByID(ctx context.Context, sc *svcctx.ServiceContext, id int64) (*dto.InstanceDetailResp, error) {
	instance, err := s.getAccessibleInstance(ctx, sc, id)
	if err != nil {
		return nil, err
	}
	instanceAggregate, err := loadInstanceAggregate(ctx, s.instanceRepo, s.containerRepo, s.checkResultRepo, instance.ID)
	if err != nil {
		return nil, err
	}
	instance = instanceAggregate.Instance

	// 获取模板信息
	templateAggregate, _ := loadTemplateAggregate(
		ctx,
		s.templateRepo,
		s.templateContainerRepo,
		s.checkpointRepo,
		nil,
		nil,
		nil,
		nil,
		nil,
		instance.TemplateID,
	)

	resp := &dto.InstanceDetailResp{
		ID:           strconv.FormatInt(instance.ID, 10),
		Status:       instance.Status,
		StatusText:   enum.GetInstanceStatusText(instance.Status),
		AttemptNo:    instance.AttemptNo,
		SimSessionID: instance.SimSessionID,
		CreatedAt:    instance.CreatedAt.Format(time.RFC3339),
	}

	if instance.StartedAt != nil {
		t := instance.StartedAt.Format(time.RFC3339)
		resp.StartedAt = &t
	}
	if instance.LastActiveAt != nil {
		t := instance.LastActiveAt.Format(time.RFC3339)
		resp.LastActiveAt = &t
	}

	// 模板摘要
	if templateAggregate != nil && templateAggregate.Template != nil {
		template := templateAggregate.Template
		resp.Template = dto.InstanceTemplateBrief{
			ID:          strconv.FormatInt(template.ID, 10),
			Title:       template.Title,
			JudgeMode:   template.JudgeMode,
			TotalScore:  template.TotalScore,
			IdleTimeout: template.IdleTimeout,
		}
		if template.TopologyMode != nil {
			resp.Template.TopologyMode = *template.TopologyMode
		}
		if template.Instructions != nil {
			resp.Template.Instructions = template.Instructions
		}
		if template.MaxDuration != nil {
			resp.Template.MaxDuration = *template.MaxDuration
		}
	}

	// 学生摘要
	studentSummary := s.getInstanceUserSummary(ctx, instance.StudentID)
	resp.Student = dto.InstanceStudentBrief{
		ID:        strconv.FormatInt(instance.StudentID, 10),
		Name:      studentSummary.Name,
		StudentNo: studentSummary.StudentNo,
	}

	// 容器列表
	templateContainers := []*entity.TemplateContainer(nil)
	if templateAggregate != nil {
		templateContainers = templateAggregate.Containers
	}
	resp.Containers = s.buildInstanceContainerItems(ctx, instanceAggregate.Containers, templateContainers)

	// 工具列表（从容器中筛选 tool_kind 非空的）
	resp.Tools = s.buildInstanceToolItems(ctx, instanceAggregate.Containers)

	// 检查点列表
	if templateAggregate != nil {
		checkpoints := make([]dto.InstanceCheckpointItem, 0, len(templateAggregate.Checkpoints))
		for _, cp := range templateAggregate.Checkpoints {
			item := dto.InstanceCheckpointItem{
				CheckpointID: strconv.FormatInt(cp.ID, 10),
				Title:        cp.Title,
				CheckType:    cp.CheckType,
				Score:        cp.Score,
			}
			// 查找结果
			for _, cr := range instanceAggregate.CheckpointResults {
				if cr.CheckpointID == cp.ID {
					score := float64(0)
					if cr.Score != nil {
						score = *cr.Score
					}
					passed := false
					if cr.IsPassed != nil {
						passed = *cr.IsPassed
					}
					item.Result = &dto.InstanceCheckpointResult{
						IsPassed:  passed,
						Score:     score,
						CheckedAt: cr.CheckedAt.UTC().Format(time.RFC3339),
					}
					break
				}
			}
			checkpoints = append(checkpoints, item)
		}
		resp.Checkpoints = checkpoints
	}

	// 成绩
	resp.Scores = dto.InstanceScoresInfo{
		AutoScore:   instance.AutoScore,
		ManualScore: instance.ManualScore,
		TotalScore:  instance.TotalScore,
	}

	return resp, nil
}

// ---------------------------------------------------------------------------
// 实例列表
// ---------------------------------------------------------------------------

// List 我的实验实例列表
func (s *instanceService) List(ctx context.Context, sc *svcctx.ServiceContext, req *dto.InstanceListReq) ([]*dto.InstanceListItem, int64, error) {
	params := &experimentrepo.StudentInstanceListParams{
		Page:     req.Page,
		PageSize: req.PageSize,
		Status:   req.Status,
	}
	if req.TemplateID != "" {
		tid, _ := snowflake.ParseString(req.TemplateID)
		params.TemplateID = tid
	}
	if req.CourseID != "" {
		cid, _ := snowflake.ParseString(req.CourseID)
		params.CourseID = cid
	}

	instances, total, err := s.instanceRepo.ListByStudentID(ctx, sc.UserID, params)
	if err != nil {
		return nil, 0, err
	}

	items := make([]*dto.InstanceListItem, 0, len(instances))
	for _, inst := range instances {
		item := &dto.InstanceListItem{
			ID:         strconv.FormatInt(inst.ID, 10),
			TemplateID: strconv.FormatInt(inst.TemplateID, 10),
			Status:     inst.Status,
			StatusText: enum.GetInstanceStatusText(inst.Status),
			AttemptNo:  inst.AttemptNo,
			TotalScore: inst.TotalScore,
			CreatedAt:  inst.CreatedAt.Format(time.RFC3339),
		}
		if inst.StartedAt != nil {
			t := inst.StartedAt.Format(time.RFC3339)
			item.StartedAt = &t
		}
		if inst.SubmittedAt != nil {
			t := inst.SubmittedAt.Format(time.RFC3339)
			item.SubmittedAt = &t
		}
		// 模板标题
		tmpl, _ := s.templateRepo.GetByID(ctx, inst.TemplateID)
		if tmpl != nil {
			item.TemplateTitle = tmpl.Title
		}
		// 课程标题
		if inst.CourseID != nil {
			cidStr := strconv.FormatInt(*inst.CourseID, 10)
			item.CourseID = &cidStr
			title := s.courseQuerier.GetCourseTitle(ctx, *inst.CourseID)
			if title != "" {
				item.CourseTitle = &title
			}
		}
		items = append(items, item)
	}
	return items, total, nil
}
