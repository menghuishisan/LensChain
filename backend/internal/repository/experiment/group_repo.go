// group_repo.go
// 模块04 — 实验环境：分组协作数据访问层
// 负责实验分组、分组成员、组内消息的 CRUD 操作
// 对照 docs/modules/04-实验环境/02-数据库设计.md

package experimentrepo

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/pkg/database"
	"github.com/lenschain/backend/internal/pkg/pagination"
	"github.com/lenschain/backend/internal/pkg/snowflake"
)

// ---------------------------------------------------------------------------
// 实验分组 Repository
// ---------------------------------------------------------------------------

// GroupRepository 实验分组数据访问接口
type GroupRepository interface {
	Create(ctx context.Context, group *entity.ExperimentGroup) error
	GetByID(ctx context.Context, id int64) (*entity.ExperimentGroup, error)
	GetByIDWithMembers(ctx context.Context, id int64) (*entity.ExperimentGroup, error)
	UpdateFields(ctx context.Context, id int64, fields map[string]interface{}) error
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, params *GroupListParams) ([]*entity.ExperimentGroup, int64, error)
	ListByTemplateAndCourse(ctx context.Context, templateID, courseID int64) ([]*entity.ExperimentGroup, error)
	CountMembersByGroupID(ctx context.Context, groupID int64) (int64, error)
}

// GroupListParams 分组列表查询参数
type GroupListParams struct {
	SchoolID   int64
	TemplateID int64
	CourseID   int64
	Status     int
	Keyword    string
	SortBy     string
	SortOrder  string
	Page       int
	PageSize   int
}

// groupRepository 实验分组数据访问实现
type groupRepository struct {
	db *gorm.DB
}

// NewGroupRepository 创建实验分组数据访问实例
func NewGroupRepository(db *gorm.DB) GroupRepository {
	return &groupRepository{db: db}
}

// Create 创建实验分组
func (r *groupRepository) Create(ctx context.Context, group *entity.ExperimentGroup) error {
	if group.ID == 0 {
		group.ID = snowflake.Generate()
	}
	return r.db.WithContext(ctx).Create(group).Error
}

// GetByID 根据ID获取实验分组
func (r *groupRepository) GetByID(ctx context.Context, id int64) (*entity.ExperimentGroup, error) {
	var group entity.ExperimentGroup
	err := r.db.WithContext(ctx).First(&group, id).Error
	if err != nil {
		return nil, err
	}
	return &group, nil
}

// GetByIDWithMembers 根据ID获取实验分组（含成员列表）
func (r *groupRepository) GetByIDWithMembers(ctx context.Context, id int64) (*entity.ExperimentGroup, error) {
	var group entity.ExperimentGroup
	err := r.db.WithContext(ctx).
		Preload("Members").
		First(&group, id).Error
	if err != nil {
		return nil, err
	}
	return &group, nil
}

// UpdateFields 更新实验分组指定字段
func (r *groupRepository) UpdateFields(ctx context.Context, id int64, fields map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&entity.ExperimentGroup{}).Where("id = ?", id).Updates(fields).Error
}

// Delete 删除实验分组
func (r *groupRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&entity.ExperimentGroup{}, id).Error
}

// List 分组列表查询
func (r *groupRepository) List(ctx context.Context, params *GroupListParams) ([]*entity.ExperimentGroup, int64, error) {
	query := r.db.WithContext(ctx).Model(&entity.ExperimentGroup{})

	if params.SchoolID > 0 {
		query = query.Scopes(database.WithSchoolID(params.SchoolID))
	}
	if params.TemplateID > 0 {
		query = query.Where("template_id = ?", params.TemplateID)
	}
	if params.CourseID > 0 {
		query = query.Where("course_id = ?", params.CourseID)
	}
	if params.Status > 0 {
		query = query.Scopes(database.WithStatus(params.Status))
	}
	if params.Keyword != "" {
		query = query.Scopes(database.WithKeywordSearch(params.Keyword, "group_name"))
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 排序
	sortField := "created_at"
	sortOrder := "desc"
	allowedSortFields := map[string]string{
		"created_at": "created_at",
		"group_name": "group_name",
		"status":     "status",
	}
	if field, ok := allowedSortFields[params.SortBy]; ok {
		sortField = field
	}
	if params.SortOrder == "asc" {
		sortOrder = "asc"
	}
	query = query.Order(fmt.Sprintf("%s %s", sortField, sortOrder))

	page, pageSize := pagination.NormalizeValues(params.Page, params.PageSize)
	query = query.Offset(pagination.Offset(page, pageSize)).Limit(pageSize)

	var groups []*entity.ExperimentGroup
	if err := query.Preload("Members").Find(&groups).Error; err != nil {
		return nil, 0, err
	}
	return groups, total, nil
}

// ListByTemplateAndCourse 获取模板+课程下的所有分组
func (r *groupRepository) ListByTemplateAndCourse(ctx context.Context, templateID, courseID int64) ([]*entity.ExperimentGroup, error) {
	var groups []*entity.ExperimentGroup
	err := r.db.WithContext(ctx).
		Where("template_id = ? AND course_id = ?", templateID, courseID).
		Preload("Members").
		Order("created_at asc").
		Find(&groups).Error
	return groups, err
}

// CountMembersByGroupID 统计分组成员数
func (r *groupRepository) CountMembersByGroupID(ctx context.Context, groupID int64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&entity.GroupMember{}).
		Where("group_id = ?", groupID).
		Count(&count).Error
	return count, err
}

// ---------------------------------------------------------------------------
// 分组成员 Repository
// ---------------------------------------------------------------------------

// GroupMemberRepository 分组成员数据访问接口
type GroupMemberRepository interface {
	Create(ctx context.Context, member *entity.GroupMember) error
	GetByID(ctx context.Context, id int64) (*entity.GroupMember, error)
	Delete(ctx context.Context, id int64) error
	ListByGroupID(ctx context.Context, groupID int64) ([]*entity.GroupMember, error)
	GetByGroupAndStudent(ctx context.Context, groupID, studentID int64) (*entity.GroupMember, error)
	IsStudentInGroup(ctx context.Context, templateID, courseID, studentID int64) (bool, error)
	DeleteByGroupID(ctx context.Context, groupID int64) error
	BatchCreate(ctx context.Context, members []*entity.GroupMember) error
	UpdateFields(ctx context.Context, id int64, fields map[string]interface{}) error
}

// groupMemberRepository 分组成员数据访问实现
type groupMemberRepository struct {
	db *gorm.DB
}

// NewGroupMemberRepository 创建分组成员数据访问实例
func NewGroupMemberRepository(db *gorm.DB) GroupMemberRepository {
	return &groupMemberRepository{db: db}
}

// Create 创建分组成员
func (r *groupMemberRepository) Create(ctx context.Context, member *entity.GroupMember) error {
	if member.ID == 0 {
		member.ID = snowflake.Generate()
	}
	return r.db.WithContext(ctx).Create(member).Error
}

// GetByID 根据ID获取分组成员
func (r *groupMemberRepository) GetByID(ctx context.Context, id int64) (*entity.GroupMember, error) {
	var member entity.GroupMember
	err := r.db.WithContext(ctx).First(&member, id).Error
	if err != nil {
		return nil, err
	}
	return &member, nil
}

// Delete 删除分组成员
func (r *groupMemberRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&entity.GroupMember{}, id).Error
}

// ListByGroupID 获取分组的所有成员
func (r *groupMemberRepository) ListByGroupID(ctx context.Context, groupID int64) ([]*entity.GroupMember, error) {
	var members []*entity.GroupMember
	err := r.db.WithContext(ctx).
		Where("group_id = ?", groupID).
		Order("joined_at asc").
		Find(&members).Error
	return members, err
}

// GetByGroupAndStudent 获取指定分组中的指定学生
func (r *groupMemberRepository) GetByGroupAndStudent(ctx context.Context, groupID, studentID int64) (*entity.GroupMember, error) {
	var member entity.GroupMember
	err := r.db.WithContext(ctx).
		Where("group_id = ? AND student_id = ?", groupID, studentID).
		First(&member).Error
	if err != nil {
		return nil, err
	}
	return &member, nil
}

// IsStudentInGroup 检查学生是否已在某模板+课程的分组中
func (r *groupMemberRepository) IsStudentInGroup(ctx context.Context, templateID, courseID, studentID int64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&entity.GroupMember{}).
		Where("student_id = ? AND group_id IN (?)",
			studentID,
			r.db.Model(&entity.ExperimentGroup{}).
				Select("id").
				Where("template_id = ? AND course_id = ?", templateID, courseID),
		).
		Count(&count).Error
	return count > 0, err
}

// DeleteByGroupID 删除分组的所有成员
func (r *groupMemberRepository) DeleteByGroupID(ctx context.Context, groupID int64) error {
	return r.db.WithContext(ctx).Where("group_id = ?", groupID).Delete(&entity.GroupMember{}).Error
}

// BatchCreate 批量创建分组成员
func (r *groupMemberRepository) BatchCreate(ctx context.Context, members []*entity.GroupMember) error {
	for i := range members {
		if members[i].ID == 0 {
			members[i].ID = snowflake.Generate()
		}
	}
	return r.db.WithContext(ctx).CreateInBatches(members, 50).Error
}

// UpdateFields 更新分组成员指定字段
func (r *groupMemberRepository) UpdateFields(ctx context.Context, id int64, fields map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&entity.GroupMember{}).Where("id = ?", id).Updates(fields).Error
}

// ---------------------------------------------------------------------------
// 组内消息 Repository
// ---------------------------------------------------------------------------

// GroupMessageRepository 组内消息数据访问接口
type GroupMessageRepository interface {
	Create(ctx context.Context, message *entity.GroupMessage) error
	List(ctx context.Context, params *GroupMessageListParams) ([]*entity.GroupMessage, int64, error)
	ListByGroupID(ctx context.Context, groupID int64, limit int) ([]*entity.GroupMessage, error)
}

// GroupMessageListParams 组内消息列表查询参数
type GroupMessageListParams struct {
	GroupID  int64
	SenderID int64
	Page     int
	PageSize int
}

// groupMessageRepository 组内消息数据访问实现
type groupMessageRepository struct {
	db *gorm.DB
}

// NewGroupMessageRepository 创建组内消息数据访问实例
func NewGroupMessageRepository(db *gorm.DB) GroupMessageRepository {
	return &groupMessageRepository{db: db}
}

// Create 创建组内消息
func (r *groupMessageRepository) Create(ctx context.Context, message *entity.GroupMessage) error {
	if message.ID == 0 {
		message.ID = snowflake.Generate()
	}
	return r.db.WithContext(ctx).Create(message).Error
}

// List 组内消息列表查询
func (r *groupMessageRepository) List(ctx context.Context, params *GroupMessageListParams) ([]*entity.GroupMessage, int64, error) {
	query := r.db.WithContext(ctx).Model(&entity.GroupMessage{}).
		Where("group_id = ?", params.GroupID)

	if params.SenderID > 0 {
		query = query.Where("sender_id = ?", params.SenderID)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	page, pageSize := pagination.NormalizeValues(params.Page, params.PageSize)
	query = query.Order("created_at desc").Offset(pagination.Offset(page, pageSize)).Limit(pageSize)

	var messages []*entity.GroupMessage
	if err := query.Find(&messages).Error; err != nil {
		return nil, 0, err
	}
	return messages, total, nil
}

// ListByGroupID 获取分组的最新消息
func (r *groupMessageRepository) ListByGroupID(ctx context.Context, groupID int64, limit int) ([]*entity.GroupMessage, error) {
	var messages []*entity.GroupMessage
	query := r.db.WithContext(ctx).
		Where("group_id = ?", groupID).
		Order("created_at desc")
	if limit > 0 {
		query = query.Limit(limit)
	}
	err := query.Find(&messages).Error
	return messages, err
}
