// warning_repo.go
// 模块06 — 评测与成绩：学业预警数据访问层。
// 负责 academic_warnings 表的创建、查询、列表、处理和自动解除状态更新。

package graderepo

import (
	"context"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"

	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/model/enum"
	"github.com/lenschain/backend/internal/pkg/database"
	"github.com/lenschain/backend/internal/pkg/pagination"
	"github.com/lenschain/backend/internal/pkg/snowflake"
)

// AcademicWarningRepository 学业预警数据访问接口。
type AcademicWarningRepository interface {
	Create(ctx context.Context, warning *entity.AcademicWarning) error
	BatchCreate(ctx context.Context, warnings []*entity.AcademicWarning) error
	GetByID(ctx context.Context, id int64) (*entity.AcademicWarning, error)
	List(ctx context.Context, params *AcademicWarningListParams) ([]*entity.AcademicWarning, int64, error)
	ListActiveBySemester(ctx context.Context, schoolID, semesterID int64) ([]*entity.AcademicWarning, error)
	ListUnresolvedByStudent(ctx context.Context, schoolID, studentID int64) ([]*entity.AcademicWarning, error)
	GetExisting(ctx context.Context, schoolID, studentID, semesterID int64, warningType int16) (*entity.AcademicWarning, error)
	Handle(ctx context.Context, id, handledBy int64, note *string, handledAt time.Time) error
	Resolve(ctx context.Context, id int64) error
	UpdateDetail(ctx context.Context, id int64, detail datatypes.JSON) error
	CountActiveBySemester(ctx context.Context, schoolID, semesterID int64) (int64, error)
}

// ListActiveBySemester 查询学期内所有未解除预警，供重算时自动解除。
func (r *academicWarningRepository) ListActiveBySemester(ctx context.Context, schoolID, semesterID int64) ([]*entity.AcademicWarning, error) {
	var warnings []*entity.AcademicWarning
	err := r.db.WithContext(ctx).Model(&entity.AcademicWarning{}).
		Scopes(database.WithSchoolID(schoolID)).
		Where("semester_id = ? AND status <> ?", semesterID, enum.AcademicWarningStatusResolved).
		Order("student_id asc, warning_type asc").
		Find(&warnings).Error
	return warnings, err
}

// AcademicWarningListParams 学业预警列表查询参数。
type AcademicWarningListParams struct {
	SchoolID    int64
	StudentID   int64
	SemesterID  int64
	WarningType int16
	Status      int16
	Keyword     string
	From        *time.Time
	To          *time.Time
	SortBy      string
	SortOrder   string
	Page        int
	PageSize    int
}

type academicWarningRepository struct {
	db *gorm.DB
}

// NewAcademicWarningRepository 创建学业预警数据访问实例。
func NewAcademicWarningRepository(db *gorm.DB) AcademicWarningRepository {
	return &academicWarningRepository{db: db}
}

// Create 创建学业预警。
func (r *academicWarningRepository) Create(ctx context.Context, warning *entity.AcademicWarning) error {
	if warning.ID == 0 {
		warning.ID = snowflake.Generate()
	}
	return r.db.WithContext(ctx).Create(warning).Error
}

// BatchCreate 批量创建学业预警。
func (r *academicWarningRepository) BatchCreate(ctx context.Context, warnings []*entity.AcademicWarning) error {
	if len(warnings) == 0 {
		return nil
	}
	for i := range warnings {
		if warnings[i].ID == 0 {
			warnings[i].ID = snowflake.Generate()
		}
	}
	return r.db.WithContext(ctx).CreateInBatches(warnings, 100).Error
}

// GetByID 根据 ID 获取学业预警。
func (r *academicWarningRepository) GetByID(ctx context.Context, id int64) (*entity.AcademicWarning, error) {
	var warning entity.AcademicWarning
	err := r.db.WithContext(ctx).First(&warning, id).Error
	if err != nil {
		return nil, err
	}
	return &warning, nil
}

// List 查询学业预警列表。
func (r *academicWarningRepository) List(ctx context.Context, params *AcademicWarningListParams) ([]*entity.AcademicWarning, int64, error) {
	query := r.db.WithContext(ctx).Model(&entity.AcademicWarning{}).
		Scopes(database.WithSchoolID(params.SchoolID), database.WithStatus(params.Status))
	if params.Keyword != "" {
		query = query.Joins("JOIN users u ON u.id = academic_warnings.student_id").
			Scopes(database.WithKeywordSearch(params.Keyword, "u.name", "u.student_no"))
	}
	if params.StudentID > 0 {
		query = query.Where("student_id = ?", params.StudentID)
	}
	if params.SemesterID > 0 {
		query = query.Where("semester_id = ?", params.SemesterID)
	}
	if params.WarningType > 0 {
		query = query.Where("warning_type = ?", params.WarningType)
	}
	if params.From != nil {
		query = query.Where("created_at >= ?", *params.From)
	}
	if params.To != nil {
		query = query.Where("created_at <= ?", *params.To)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	sortBy := params.SortBy
	if sortBy == "" {
		sortBy = "created_at"
	}
	query = (&pagination.Query{
		Page:      params.Page,
		PageSize:  params.PageSize,
		SortBy:    sortBy,
		SortOrder: params.SortOrder,
	}).ApplyToGORM(query, map[string]string{
		"created_at": "created_at",
		"handled_at": "handled_at",
	})

	var warnings []*entity.AcademicWarning
	if err := query.Find(&warnings).Error; err != nil {
		return nil, 0, err
	}
	return warnings, total, nil
}

// ListUnresolvedByStudent 查询学生未解除学业预警。
func (r *academicWarningRepository) ListUnresolvedByStudent(ctx context.Context, schoolID, studentID int64) ([]*entity.AcademicWarning, error) {
	var warnings []*entity.AcademicWarning
	err := r.db.WithContext(ctx).
		Scopes(database.WithSchoolID(schoolID)).
		Where("student_id = ? AND status <> ?", studentID, enum.AcademicWarningStatusResolved).
		Order("created_at desc").
		Find(&warnings).Error
	return warnings, err
}

// GetExisting 获取同学生同学期同类型预警。
func (r *academicWarningRepository) GetExisting(ctx context.Context, schoolID, studentID, semesterID int64, warningType int16) (*entity.AcademicWarning, error) {
	var warning entity.AcademicWarning
	err := r.db.WithContext(ctx).
		Scopes(database.WithSchoolID(schoolID)).
		Where("student_id = ? AND semester_id = ? AND warning_type = ?", studentID, semesterID, warningType).
		First(&warning).Error
	if err != nil {
		return nil, err
	}
	return &warning, nil
}

// Handle 标记学业预警为已处理。
func (r *academicWarningRepository) Handle(ctx context.Context, id, handledBy int64, note *string, handledAt time.Time) error {
	return r.db.WithContext(ctx).Model(&entity.AcademicWarning{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":      enum.AcademicWarningStatusHandled,
			"handled_by":  handledBy,
			"handled_at":  handledAt,
			"handle_note": note,
			"updated_at":  gorm.Expr("now()"),
		}).Error
}

// Resolve 标记学业预警为已解除。
func (r *academicWarningRepository) Resolve(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Model(&entity.AcademicWarning{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":     enum.AcademicWarningStatusResolved,
			"updated_at": gorm.Expr("now()"),
		}).Error
}

// UpdateDetail 更新预警明细 JSON。
func (r *academicWarningRepository) UpdateDetail(ctx context.Context, id int64, detail datatypes.JSON) error {
	return r.db.WithContext(ctx).Model(&entity.AcademicWarning{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"detail":     detail,
			"updated_at": gorm.Expr("now()"),
		}).Error
}

// CountActiveBySemester 统计学期内未解除预警数量。
func (r *academicWarningRepository) CountActiveBySemester(ctx context.Context, schoolID, semesterID int64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&entity.AcademicWarning{}).
		Scopes(database.WithSchoolID(schoolID)).
		Where("semester_id = ? AND status <> ?", semesterID, enum.AcademicWarningStatusResolved).
		Count(&count).Error
	return count, err
}
