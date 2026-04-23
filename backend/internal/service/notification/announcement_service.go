// announcement_service.go
// 模块07 — 通知与消息：系统公告与定向通知业务。
// 该文件聚合管理员侧公告管理和用户主动发送通知能力，避免运营类流程散落。

package notification

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/lenschain/backend/internal/model/dto"
	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/model/enum"
	svcctx "github.com/lenschain/backend/internal/pkg/context"
	"github.com/lenschain/backend/internal/pkg/errcode"
	notificationrepo "github.com/lenschain/backend/internal/repository/notification"
)

// CreateAnnouncement 创建系统公告。
func (s *service) CreateAnnouncement(ctx context.Context, sc *svcctx.ServiceContext, req *dto.CreateSystemAnnouncementReq) (*dto.SystemAnnouncementDetailResp, error) {
	if sc == nil || !sc.IsSuperAdmin() {
		return nil, errcode.ErrForbidden
	}
	announcement := &entity.SystemAnnouncement{
		Title:       req.Title,
		Content:     req.Content,
		PublishedBy: sc.UserID,
		Status:      enum.SystemAnnouncementStatusDraft,
		IsPinned:    true,
	}
	if req.ScheduledAt != nil && *req.ScheduledAt != "" {
		scheduledAt, err := time.Parse(time.RFC3339, *req.ScheduledAt)
		if err != nil {
			return nil, errcode.ErrInvalidParams.WithMessage("scheduled_at 格式错误")
		}
		announcement.ScheduledAt = &scheduledAt
	}
	if err := s.announcementRepo.Create(ctx, announcement); err != nil {
		return nil, err
	}
	return buildAnnouncementDetail(announcement, true, nil), nil
}

// ListAnnouncements 获取公告列表。
func (s *service) ListAnnouncements(ctx context.Context, sc *svcctx.ServiceContext, req *dto.NotificationAnnouncementListReq) (*dto.SystemAnnouncementListResp, error) {
	if err := ensureAuthenticated(sc); err != nil {
		return nil, err
	}
	page, pageSize := normalizePagination(req.Page, req.PageSize)
	isAdmin := sc.IsSuperAdmin()

	var items []*entity.SystemAnnouncement
	var total int64
	var err error
	if isAdmin {
		items, total, err = s.announcementRepo.List(ctx, &notificationrepo.AnnouncementListParams{
			Status:   req.Status,
			Page:     page,
			PageSize: pageSize,
		})
	} else {
		items, total, err = s.announcementRepo.ListPublished(ctx, page, pageSize)
	}
	if err != nil {
		return nil, err
	}
	readMap := make(map[int64]bool, len(items))
	if !isAdmin && len(items) > 0 {
		ids := make([]int64, 0, len(items))
		for _, item := range items {
			if item == nil {
				continue
			}
			ids = append(ids, item.ID)
		}
		readIDs, readErr := s.announcementReadRepo.ListReadAnnouncementIDs(ctx, sc.UserID, ids)
		if readErr == nil {
			for _, readID := range readIDs {
				readMap[readID] = true
			}
		}
	}
	list := make([]dto.SystemAnnouncementItem, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		var isRead *bool
		if !isAdmin {
			read := readMap[item.ID]
			isRead = &read
		}
		list = append(list, buildAnnouncementItem(item, isAdmin, isRead))
	}
	return &dto.SystemAnnouncementListResp{
		List:       list,
		Pagination: buildPaginationResp(page, pageSize, total),
	}, nil
}

// GetAnnouncement 获取公告详情，普通用户查看时自动记为已读。
func (s *service) GetAnnouncement(ctx context.Context, sc *svcctx.ServiceContext, id int64) (*dto.SystemAnnouncementDetailResp, error) {
	if err := ensureAuthenticated(sc); err != nil {
		return nil, err
	}
	item, err := s.announcementRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errcode.ErrAnnouncementNotFound
		}
		return nil, err
	}
	isAdmin := sc.IsSuperAdmin()
	if !isAdmin && item.Status != enum.SystemAnnouncementStatusPublished {
		return nil, errcode.ErrAnnouncementNotFound
	}
	var isRead *bool
	if !isAdmin {
		read, _ := s.announcementReadRepo.HasRead(ctx, item.ID, sc.UserID)
		if !read {
			_ = s.announcementReadRepo.Create(ctx, &entity.AnnouncementReadStatus{
				AnnouncementID: item.ID,
				UserID:         sc.UserID,
			})
			read = true
		}
		isRead = &read
	}
	return buildAnnouncementDetail(item, isAdmin, isRead), nil
}

// UpdateAnnouncement 更新系统公告。
func (s *service) UpdateAnnouncement(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.UpdateSystemAnnouncementReq) error {
	if sc == nil || !sc.IsSuperAdmin() {
		return errcode.ErrForbidden
	}
	_, err := s.announcementRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errcode.ErrAnnouncementNotFound
		}
		return err
	}
	values := make(map[string]interface{})
	if req.Title != nil {
		values["title"] = *req.Title
	}
	if req.Content != nil {
		values["content"] = *req.Content
	}
	if req.IsPinned != nil {
		values["is_pinned"] = *req.IsPinned
	}
	if req.ScheduledAt != nil {
		if *req.ScheduledAt == "" {
			values["scheduled_at"] = nil
		} else {
			scheduledAt, parseErr := time.Parse(time.RFC3339, *req.ScheduledAt)
			if parseErr != nil {
				return errcode.ErrInvalidParams.WithMessage("scheduled_at 格式错误")
			}
			values["scheduled_at"] = scheduledAt
		}
	}
	if len(values) == 0 {
		return nil
	}
	values["updated_at"] = time.Now()
	return s.announcementRepo.Update(ctx, id, values)
}

// PublishAnnouncement 发布系统公告。
func (s *service) PublishAnnouncement(ctx context.Context, sc *svcctx.ServiceContext, id int64) error {
	if sc == nil || !sc.IsSuperAdmin() {
		return errcode.ErrForbidden
	}
	item, err := s.announcementRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errcode.ErrAnnouncementNotFound
		}
		return err
	}
	if item.Status == enum.SystemAnnouncementStatusPublished {
		return nil
	}
	now := time.Now()
	if err := s.announcementRepo.Publish(ctx, id, now); err != nil {
		return err
	}
	item.Status = enum.SystemAnnouncementStatusPublished
	item.PublishedAt = &now
	s.broadcastAnnouncement(item)
	return nil
}

// UnpublishAnnouncement 下架系统公告。
func (s *service) UnpublishAnnouncement(ctx context.Context, sc *svcctx.ServiceContext, id int64) error {
	if sc == nil || !sc.IsSuperAdmin() {
		return errcode.ErrForbidden
	}
	item, err := s.announcementRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errcode.ErrAnnouncementNotFound
		}
		return err
	}
	if item.Status != enum.SystemAnnouncementStatusPublished {
		return nil
	}
	return s.announcementRepo.Unpublish(ctx, id, time.Now())
}

// DeleteAnnouncement 删除系统公告。
func (s *service) DeleteAnnouncement(ctx context.Context, sc *svcctx.ServiceContext, id int64) error {
	if sc == nil || !sc.IsSuperAdmin() {
		return errcode.ErrForbidden
	}
	_, err := s.announcementRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errcode.ErrAnnouncementNotFound
		}
		return err
	}
	return s.announcementRepo.Delete(ctx, id)
}

// SendDirectNotification 发送定向通知。
func (s *service) SendDirectNotification(ctx context.Context, sc *svcctx.ServiceContext, req *dto.SendNotificationReq) error {
	if err := ensureAuthenticated(sc); err != nil {
		return err
	}
	receiverIDs, err := s.resolveDirectReceiverIDs(ctx, sc, req)
	if err != nil {
		return err
	}
	if err := validateDirectReceiverLimit(receiverIDs); err != nil {
		return err
	}
	notifications := make([]*entity.Notification, 0, len(receiverIDs))
	now := time.Now()
	for _, receiverID := range receiverIDs {
		user, userErr := s.sourceRepo.GetUser(ctx, receiverID)
		if userErr != nil || user == nil {
			continue
		}
		if !s.canReceiveCategory(ctx, receiverID, req.Category) {
			continue
		}
		schoolID := user.SchoolID
		notification := &entity.Notification{
			ReceiverID:   receiverID,
			SchoolID:     &schoolID,
			Category:     req.Category,
			EventType:    "manual.send",
			Title:        req.Title,
			Content:      req.Content,
			SourceModule: "module_07",
			SourceType:   &req.TargetType,
			SourceID:     buildSourceIDPointer(req.TargetID),
			CreatedAt:    now,
		}
		notifications = append(notifications, notification)
	}
	if err := s.notificationRepo.BatchCreate(ctx, notifications); err != nil {
		return err
	}
	s.afterNotificationsCreated(ctx, notifications)
	return nil
}

// resolveDirectReceiverIDs 解析定向通知的接收者。
func (s *service) resolveDirectReceiverIDs(ctx context.Context, sc *svcctx.ServiceContext, req *dto.SendNotificationReq) ([]int64, error) {
	switch req.TargetType {
	case "all_school":
		if !sc.IsSchoolAdmin() && !sc.IsSuperAdmin() {
			return nil, errcode.ErrForbidden
		}
		users, err := s.sourceRepo.ListUsersBySchool(ctx, sc.SchoolID)
		if err != nil {
			return nil, err
		}
		ids := make([]int64, 0, len(users))
		for _, user := range users {
			if user == nil {
				continue
			}
			ids = append(ids, user.ID)
		}
		return ids, nil
	case "course":
		courseID, err := parseSnowflakeID(req.TargetID)
		if err != nil {
			return nil, err
		}
		course, err := s.sourceRepo.GetCourse(ctx, courseID)
		if err != nil {
			return nil, errcode.ErrNotFound
		}
		if sc.IsTeacher() && !sc.IsSchoolAdmin() && !sc.IsSuperAdmin() && course.TeacherID != sc.UserID {
			return nil, errcode.ErrForbidden
		}
		if !sc.IsSuperAdmin() && course.SchoolID != sc.SchoolID {
			return nil, errcode.ErrForbidden
		}
		return s.sourceRepo.ListCourseStudentIDs(ctx, courseID)
	case "user":
		if sc.IsTeacher() && !sc.IsSchoolAdmin() && !sc.IsSuperAdmin() {
			return nil, errcode.ErrForbidden.WithMessage("教师仅支持向自己课程学生发送通知")
		}
		userID, err := parseSnowflakeID(req.TargetID)
		if err != nil {
			return nil, err
		}
		user, err := s.sourceRepo.GetUser(ctx, userID)
		if err != nil || user == nil {
			return nil, errcode.ErrNotFound
		}
		if !sc.IsSuperAdmin() && user.SchoolID != sc.SchoolID {
			return nil, errcode.ErrForbidden
		}
		return []int64{userID}, nil
	case "users":
		if sc.IsTeacher() && !sc.IsSchoolAdmin() && !sc.IsSuperAdmin() {
			return nil, errcode.ErrForbidden.WithMessage("教师仅支持向自己课程学生发送通知")
		}
		ids, err := parseSnowflakeIDs(splitTargetIDs(req.TargetID))
		if err != nil {
			return nil, err
		}
		users, err := s.sourceRepo.ListUsersByIDs(ctx, ids)
		if err != nil {
			return nil, err
		}
		result := make([]int64, 0, len(users))
		for _, user := range users {
			if user == nil {
				continue
			}
			if !sc.IsSuperAdmin() && user.SchoolID != sc.SchoolID {
				return nil, errcode.ErrForbidden
			}
			result = append(result, user.ID)
		}
		return result, nil
	default:
		return nil, errcode.ErrInvalidParams.WithMessage("target_type 不合法")
	}
}

// validateDirectReceiverLimit 校验单次定向通知的接收者数量上限。
func validateDirectReceiverLimit(receiverIDs []int64) error {
	if len(uniqueReceiverIDs(receiverIDs)) > 1000 {
		return errcode.ErrInvalidParams.WithMessage("单次定向通知最多发送给1000人")
	}
	return nil
}

// splitTargetIDs 解析逗号分隔的用户 ID 列表。
func splitTargetIDs(raw string) []string {
	parts := strings.Split(raw, ",")
	items := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		items = append(items, part)
	}
	return items
}
