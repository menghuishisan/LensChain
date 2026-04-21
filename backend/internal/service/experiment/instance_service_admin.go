// instance_service_admin.go
// 模块04 — 实验环境：管理员实例管理业务逻辑
// 负责全平台实例列表查询等管理员视角能力

package experiment

import (
	"context"
	"strconv"
	"time"

	"github.com/lenschain/backend/internal/model/dto"
	"github.com/lenschain/backend/internal/model/enum"
	svcctx "github.com/lenschain/backend/internal/pkg/context"
	"github.com/lenschain/backend/internal/pkg/errcode"
	"github.com/lenschain/backend/internal/pkg/snowflake"
	experimentrepo "github.com/lenschain/backend/internal/repository/experiment"
)

// ListAdmin 获取管理员视角的实验实例列表。
func (s *instanceService) ListAdmin(ctx context.Context, sc *svcctx.ServiceContext, req *dto.AdminInstanceListReq) ([]*dto.InstanceListItem, int64, error) {
	if !sc.IsSuperAdmin() {
		return nil, 0, errcode.ErrForbidden
	}

	schoolID, _ := snowflake.ParseString(req.SchoolID)
	templateID, _ := snowflake.ParseString(req.TemplateID)
	studentID, _ := snowflake.ParseString(req.StudentID)
	instances, total, err := s.instanceRepo.ListAdmin(ctx, &experimentrepo.AdminInstanceListParams{
		SchoolID:   schoolID,
		TemplateID: templateID,
		StudentID:  studentID,
		Status:     req.Status,
		SortBy:     req.SortBy,
		SortOrder:  req.SortOrder,
		Page:       req.Page,
		PageSize:   req.PageSize,
	})
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
			CreatedAt:  inst.CreatedAt.UTC().Format(time.RFC3339),
		}
		if inst.StartedAt != nil {
			value := inst.StartedAt.UTC().Format(time.RFC3339)
			item.StartedAt = &value
		}
		if inst.SubmittedAt != nil {
			value := inst.SubmittedAt.UTC().Format(time.RFC3339)
			item.SubmittedAt = &value
		}
		if template, templateErr := s.templateRepo.GetByID(ctx, inst.TemplateID); templateErr == nil && template != nil {
			item.TemplateTitle = template.Title
		}
		studentIDValue := strconv.FormatInt(inst.StudentID, 10)
		item.StudentID = &studentIDValue
		if s.userSummaryQuerier != nil {
			if summary := s.userSummaryQuerier.GetUserSummary(ctx, inst.StudentID); summary != nil && summary.Name != "" {
				studentName := summary.Name
				item.StudentName = &studentName
			}
		}
		schoolIDValue := strconv.FormatInt(inst.SchoolID, 10)
		item.SchoolID = &schoolIDValue
		if s.schoolNameQuerier != nil {
			if schoolName := s.schoolNameQuerier.GetSchoolName(ctx, inst.SchoolID); schoolName != "" {
				schoolNameCopy := schoolName
				item.SchoolName = &schoolNameCopy
			}
		}
		if inst.CourseID != nil {
			courseID := strconv.FormatInt(*inst.CourseID, 10)
			item.CourseID = &courseID
			if title := s.courseQuerier.GetCourseTitle(ctx, *inst.CourseID); title != "" {
				item.CourseTitle = &title
			}
		}
		if inst.ErrorMessage != nil && *inst.ErrorMessage != "" {
			item.ErrorMessage = inst.ErrorMessage
		}
		updatedAt := inst.UpdatedAt.UTC().Format(time.RFC3339)
		item.UpdatedAt = &updatedAt
		items = append(items, item)
	}
	return items, total, nil
}
