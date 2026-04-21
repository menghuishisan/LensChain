// semester_repo.go
// 模块06 — 评测与成绩：学期数据访问层。
// 负责 semesters 表的创建、查询、列表、当前学期切换和软删除恢复支撑。

package graderepo

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/pkg/database"
	"github.com/lenschain/backend/internal/pkg/pagination"
	"github.com/lenschain/backend/internal/pkg/snowflake"
)

// SemesterRepository 学期数据访问接口。
type SemesterRepository interface {
	Create(ctx context.Context, semester *entity.Semester) error
	GetByID(ctx context.Context, id int64) (*entity.Semester, error)
	GetByCode(ctx context.Context, schoolID int64, code string) (*entity.Semester, error)
	GetCurrent(ctx context.Context, schoolID int64) (*entity.Semester, error)
	List(ctx context.Context, params *SemesterListParams) ([]*entity.Semester, int64, error)
	ListByIDs(ctx context.Context, schoolID int64, ids []int64) ([]*entity.Semester, error)
	Update(ctx context.Context, id int64, values map[string]interface{}) error
	ClearCurrent(ctx context.Context, schoolID int64) error
	SetCurrent(ctx context.Context, id int64) error
	Delete(ctx context.Context, id int64) error
}

// SemesterListParams 学期列表查询参数。
type SemesterListParams struct {
	SchoolID  int64
	IsCurrent *bool
	Keyword   string
	Date      *time.Time
	SortBy    string
	SortOrder string
	Page      int
	PageSize  int
}

type semesterRepository struct {
	db *gorm.DB
}

// NewSemesterRepository 创建学期数据访问实例。
func NewSemesterRepository(db *gorm.DB) SemesterRepository {
	return &semesterRepository{db: db}
}

// Create 创建学期。
func (r *semesterRepository) Create(ctx context.Context, semester *entity.Semester) error {
	if semester.ID == 0 {
		semester.ID = snowflake.Generate()
	}
	return r.db.WithContext(ctx).Create(semester).Error
}

// GetByID 根据 ID 获取学期。
func (r *semesterRepository) GetByID(ctx context.Context, id int64) (*entity.Semester, error) {
	var semester entity.Semester
	err := r.db.WithContext(ctx).First(&semester, id).Error
	if err != nil {
		return nil, err
	}
	return &semester, nil
}

// GetByCode 获取学校内指定编码的学期。
func (r *semesterRepository) GetByCode(ctx context.Context, schoolID int64, code string) (*entity.Semester, error) {
	var semester entity.Semester
	err := r.db.WithContext(ctx).
		Scopes(database.WithSchoolID(schoolID)).
		Where("code = ?", code).
		First(&semester).Error
	if err != nil {
		return nil, err
	}
	return &semester, nil
}

// GetCurrent 获取学校当前学期。
func (r *semesterRepository) GetCurrent(ctx context.Context, schoolID int64) (*entity.Semester, error) {
	var semester entity.Semester
	err := r.db.WithContext(ctx).
		Scopes(database.WithSchoolID(schoolID)).
		Where("is_current = ?", true).
		First(&semester).Error
	if err != nil {
		return nil, err
	}
	return &semester, nil
}

// List 查询学期列表。
func (r *semesterRepository) List(ctx context.Context, params *SemesterListParams) ([]*entity.Semester, int64, error) {
	query := r.db.WithContext(ctx).Model(&entity.Semester{}).
		Scopes(database.WithSchoolID(params.SchoolID), database.WithKeywordSearch(params.Keyword, "name", "code"))
	if params.IsCurrent != nil {
		query = query.Where("is_current = ?", *params.IsCurrent)
	}
	if params.Date != nil {
		query = query.Where("start_date <= ? AND end_date >= ?", *params.Date, *params.Date)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	sortBy := params.SortBy
	if sortBy == "" {
		sortBy = "start_date"
	}
	query = (&pagination.Query{
		Page:      params.Page,
		PageSize:  params.PageSize,
		SortBy:    sortBy,
		SortOrder: params.SortOrder,
	}).ApplyToGORM(query, map[string]string{
		"start_date": "start_date",
		"end_date":   "end_date",
		"created_at": "created_at",
	})

	var semesters []*entity.Semester
	if err := query.Find(&semesters).Error; err != nil {
		return nil, 0, err
	}
	return semesters, total, nil
}

// ListByIDs 批量获取学校内指定学期。
func (r *semesterRepository) ListByIDs(ctx context.Context, schoolID int64, ids []int64) ([]*entity.Semester, error) {
	if len(ids) == 0 {
		return []*entity.Semester{}, nil
	}
	var semesters []*entity.Semester
	err := r.db.WithContext(ctx).
		Scopes(database.WithSchoolID(schoolID)).
		Where("id IN ?", ids).
		Order("start_date asc").
		Find(&semesters).Error
	return semesters, err
}

// Update 更新学期字段。
func (r *semesterRepository) Update(ctx context.Context, id int64, values map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&entity.Semester{}).
		Where("id = ?", id).
		Updates(values).Error
}

// ClearCurrent 清除学校当前学期标记。
func (r *semesterRepository) ClearCurrent(ctx context.Context, schoolID int64) error {
	return r.db.WithContext(ctx).Model(&entity.Semester{}).
		Scopes(database.WithSchoolID(schoolID)).
		Where("is_current = ?", true).
		Update("is_current", false).Error
}

// SetCurrent 将指定学期设为当前学期。
// 该方法只做同表原子更新，不承载“是否允许切换”的业务判断。
func (r *semesterRepository) SetCurrent(ctx context.Context, id int64) error {
	return database.TransactionWithDB(ctx, r.db, func(tx *gorm.DB) error {
		var semester entity.Semester
		if err := tx.First(&semester, id).Error; err != nil {
			return err
		}
		if err := tx.Model(&entity.Semester{}).
			Where("school_id = ? AND is_current = ?", semester.SchoolID, true).
			Update("is_current", false).Error; err != nil {
			return err
		}
		return tx.Model(&entity.Semester{}).
			Where("id = ?", id).
			Update("is_current", true).Error
	})
}

// Delete 软删除学期。
func (r *semesterRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&entity.Semester{}, id).Error
}
