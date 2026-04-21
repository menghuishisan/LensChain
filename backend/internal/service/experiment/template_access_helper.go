// template_access_helper.go
// 模块04 — 实验环境：模板访问控制辅助
// 负责统一收口模板创建教师、共享模板读取与克隆权限校验，避免模板主服务和子资源服务各自散落权限判断

package experiment

import (
	"context"

	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/model/enum"
	svcctx "github.com/lenschain/backend/internal/pkg/context"
	"github.com/lenschain/backend/internal/pkg/errcode"
	experimentrepo "github.com/lenschain/backend/internal/repository/experiment"
)

// ensureTemplateOwnerAccess 校验当前用户是否为模板创建教师。
// 文档规定模板编辑、发布、结构性子资源管理等操作仅允许模板创建教师执行。
func ensureTemplateOwnerAccess(ctx context.Context, repo experimentrepo.TemplateRepository, sc *svcctx.ServiceContext, templateID int64) (*entity.ExperimentTemplate, error) {
	template, err := repo.GetByID(ctx, templateID)
	if err != nil {
		return nil, errcode.ErrTemplateNotFound
	}
	if sc == nil {
		return nil, errcode.ErrForbidden
	}
	if sc.IsSuperAdmin() {
		return template, nil
	}
	if template.TeacherID != sc.UserID {
		return nil, errcode.ErrForbidden
	}
	if template.SchoolID != sc.SchoolID {
		return nil, errcode.ErrForbidden
	}
	return template, nil
}

// ensureTemplateReadAccess 校验模板详情读取权限。
// 文档规定教师可查看自己创建的模板，或查看已发布且已共享到平台实验库的模板。
func ensureTemplateReadAccess(ctx context.Context, repo experimentrepo.TemplateRepository, sc *svcctx.ServiceContext, templateID int64) (*entity.ExperimentTemplate, error) {
	template, err := repo.GetByID(ctx, templateID)
	if err != nil {
		return nil, errcode.ErrTemplateNotFound
	}
	if sc == nil {
		return nil, errcode.ErrForbidden
	}
	if sc.IsSuperAdmin() {
		return template, nil
	}
	if template.TeacherID == sc.UserID && template.SchoolID == sc.SchoolID {
		return template, nil
	}
	if template.IsShared && template.Status == enum.TemplateStatusPublished {
		return template, nil
	}
	return nil, errcode.ErrForbidden
}

// ensureTemplateCloneAccess 校验模板克隆权限。
// 克隆允许来源于“自己的模板”或“共享实验库模板”，与详情读取规则一致。
func ensureTemplateCloneAccess(ctx context.Context, repo experimentrepo.TemplateRepository, sc *svcctx.ServiceContext, templateID int64) (*entity.ExperimentTemplate, error) {
	return ensureTemplateReadAccess(ctx, repo, sc, templateID)
}
