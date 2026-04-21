// source_repo.go
// 模块07 — 通知与消息：通知来源与接收者解析只读数据访问层。
// 负责读取用户、课程选课、竞赛报名等外部模块数据，为定向通知和定时提醒提供接收者集合。

package notificationrepo

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/model/enum"
)

// NotificationSourceRepository 通知来源只读数据访问接口。
type NotificationSourceRepository interface {
	GetUser(ctx context.Context, id int64) (*entity.User, error)
	ListUsersBySchool(ctx context.Context, schoolID int64) ([]*entity.User, error)
	ListUsersByIDs(ctx context.Context, ids []int64) ([]*entity.User, error)
	ListCourseStudentIDs(ctx context.Context, courseID int64) ([]int64, error)
	ListCompetitionRegisteredStudentIDs(ctx context.Context, competitionID int64) ([]int64, error)
	ListCompetitionUnregisteredStudentIDs(ctx context.Context, competitionID int64, schoolID int64) ([]int64, error)
	ListCompetitionStartingCandidates(ctx context.Context, now, deadline time.Time) ([]*CompetitionReminderCandidate, error)
	ListCompetitionRegistrationDeadlineCandidates(ctx context.Context, now, deadline time.Time) ([]*CompetitionReminderCandidate, error)
}

// CompetitionReminderCandidate 竞赛提醒候选项。
type CompetitionReminderCandidate struct {
	CompetitionID int64  `gorm:"column:competition_id"`
	Title         string `gorm:"column:title"`
	Scope         int16  `gorm:"column:scope"`
	SchoolID      *int64 `gorm:"column:school_id"`
}

type notificationSourceRepository struct {
	db *gorm.DB
}

// NewNotificationSourceRepository 创建通知来源只读数据访问实例。
func NewNotificationSourceRepository(db *gorm.DB) NotificationSourceRepository {
	return &notificationSourceRepository{db: db}
}

// GetUser 获取用户信息。
func (r *notificationSourceRepository) GetUser(ctx context.Context, id int64) (*entity.User, error) {
	var user entity.User
	err := r.db.WithContext(ctx).First(&user, id).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// ListUsersBySchool 查询学校用户。
func (r *notificationSourceRepository) ListUsersBySchool(ctx context.Context, schoolID int64) ([]*entity.User, error) {
	var users []*entity.User
	err := r.db.WithContext(ctx).
		Where("school_id = ?", schoolID).
		Order("id asc").
		Find(&users).Error
	return users, err
}

// ListUsersByIDs 批量查询用户。
func (r *notificationSourceRepository) ListUsersByIDs(ctx context.Context, ids []int64) ([]*entity.User, error) {
	if len(ids) == 0 {
		return []*entity.User{}, nil
	}
	var users []*entity.User
	err := r.db.WithContext(ctx).
		Where("id IN ?", ids).
		Order("id asc").
		Find(&users).Error
	return users, err
}

// ListCourseStudentIDs 查询课程选课学生 ID。
func (r *notificationSourceRepository) ListCourseStudentIDs(ctx context.Context, courseID int64) ([]int64, error) {
	var studentIDs []int64
	err := r.db.WithContext(ctx).Model(&entity.CourseEnrollment{}).
		Where("course_id = ? AND removed_at IS NULL", courseID).
		Order("student_id asc").
		Pluck("student_id", &studentIDs).Error
	return studentIDs, err
}

// ListCompetitionRegisteredStudentIDs 查询竞赛已报名学生 ID。
func (r *notificationSourceRepository) ListCompetitionRegisteredStudentIDs(ctx context.Context, competitionID int64) ([]int64, error) {
	var studentIDs []int64
	err := r.db.WithContext(ctx).Table("competition_registrations AS r").
		Select("DISTINCT tm.student_id").
		Joins("JOIN team_members tm ON tm.team_id = r.team_id").
		Where("r.competition_id = ? AND r.status = ?", competitionID, enum.RegistrationStatusRegistered).
		Order("tm.student_id asc").
		Pluck("tm.student_id", &studentIDs).Error
	return studentIDs, err
}

// ListCompetitionUnregisteredStudentIDs 查询竞赛未报名学生 ID。
func (r *notificationSourceRepository) ListCompetitionUnregisteredStudentIDs(ctx context.Context, competitionID int64, schoolID int64) ([]int64, error) {
	sub := r.db.WithContext(ctx).Table("competition_registrations AS r").
		Select("tm.student_id").
		Joins("JOIN team_members tm ON tm.team_id = r.team_id").
		Where("r.competition_id = ? AND r.status = ?", competitionID, enum.RegistrationStatusRegistered)

	query := r.db.WithContext(ctx).Model(&entity.User{}).
		Select("users.id").
		Joins("JOIN user_roles ur ON ur.user_id = users.id").
		Joins("JOIN roles r ON r.id = ur.role_id").
		Where("users.school_id = ?", schoolID).
		Where("users.status = ?", enum.UserStatusActive).
		Where("r.code = ?", enum.RoleStudent).
		Where("users.id NOT IN (?)", sub).
		Distinct()

	var studentIDs []int64
	err := query.Order("users.id asc").Pluck("users.id", &studentIDs).Error
	return studentIDs, err
}

// ListCompetitionStartingCandidates 查询即将开始的竞赛。
func (r *notificationSourceRepository) ListCompetitionStartingCandidates(ctx context.Context, now, deadline time.Time) ([]*CompetitionReminderCandidate, error) {
	var items []*CompetitionReminderCandidate
	err := r.db.WithContext(ctx).Model(&entity.Competition{}).
		Select("id AS competition_id, title, scope, school_id").
		Where("status = ? AND start_at IS NOT NULL AND start_at > ? AND start_at <= ?", enum.CompetitionStatusRegistration, now, deadline).
		Order("start_at asc").
		Find(&items).Error
	return items, err
}

// ListCompetitionRegistrationDeadlineCandidates 查询即将截止报名的竞赛。
func (r *notificationSourceRepository) ListCompetitionRegistrationDeadlineCandidates(ctx context.Context, now, deadline time.Time) ([]*CompetitionReminderCandidate, error) {
	var items []*CompetitionReminderCandidate
	err := r.db.WithContext(ctx).Model(&entity.Competition{}).
		Select("id AS competition_id, title, scope, school_id").
		Where("status = ? AND registration_end_at IS NOT NULL AND registration_end_at > ? AND registration_end_at <= ?", enum.CompetitionStatusRegistration, now, deadline).
		Order("registration_end_at asc").
		Find(&items).Error
	return items, err
}
