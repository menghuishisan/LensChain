// template_service_shared.go
// 模块04 — 实验环境：共享实验库业务逻辑
// 负责共享模板列表和共享模板详情查询

package experiment

import (
	"context"

	"github.com/lenschain/backend/internal/model/dto"
	"github.com/lenschain/backend/internal/model/enum"
	svcctx "github.com/lenschain/backend/internal/pkg/context"
	"github.com/lenschain/backend/internal/pkg/errcode"
	"github.com/lenschain/backend/internal/pkg/snowflake"
	experimentrepo "github.com/lenschain/backend/internal/repository/experiment"
)

// ListShared 获取共享实验模板列表。
func (s *templateService) ListShared(ctx context.Context, sc *svcctx.ServiceContext, req *dto.SharedTemplateListReq) ([]*dto.TemplateListItem, int64, error) {
	if !sc.IsTeacher() && !sc.IsSchoolAdmin() && !sc.IsSuperAdmin() {
		return nil, 0, errcode.ErrForbidden
	}

	tagID, _ := snowflake.ParseString(req.TagID)
	templates, total, err := s.templateRepo.ListShared(ctx, &experimentrepo.SharedTemplateListParams{
		Keyword:   req.Keyword,
		Ecosystem: req.Ecosystem,
		TagID:     tagID,
		SortBy:    req.SortBy,
		SortOrder: req.SortOrder,
		Page:      req.Page,
		PageSize:  req.PageSize,
	})
	if err != nil {
		return nil, 0, err
	}

	items, err := s.buildTemplateListItems(ctx, templates)
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// GetSharedByID 获取共享实验模板详情。
func (s *templateService) GetSharedByID(ctx context.Context, sc *svcctx.ServiceContext, id int64) (*dto.TemplateResp, error) {
	if !sc.IsTeacher() && !sc.IsSchoolAdmin() && !sc.IsSuperAdmin() {
		return nil, errcode.ErrForbidden
	}

	template, err := loadTemplateAggregate(
		ctx,
		s.templateRepo,
		s.containerRepo,
		s.checkpointRepo,
		s.initScriptRepo,
		s.simSceneRepo,
		s.templateTagRepo,
		s.tagRepo,
		s.roleRepo,
		id,
	)
	if err != nil {
		return nil, errcode.ErrTemplateNotFound
	}
	if !template.Template.IsShared || template.Template.Status != enum.TemplateStatusPublished {
		return nil, errcode.ErrTemplateNotFound
	}
	return s.toTemplateResp(ctx, template), nil
}
