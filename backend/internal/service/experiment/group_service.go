// group_service.go
// 模块04 — 实验环境：多人实验分组与组内通信业务逻辑
// 负责实验分组、随机分组、学生加入分组、组内消息、组内进度同步
// 对照 docs/modules/04-实验环境/03-API接口设计.md

package experiment

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"sort"
	"strconv"
	"time"

	"gorm.io/gorm"

	"github.com/lenschain/backend/internal/model/dto"
	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/model/enum"
	svcctx "github.com/lenschain/backend/internal/pkg/context"
	"github.com/lenschain/backend/internal/pkg/errcode"
	"github.com/lenschain/backend/internal/pkg/pagination"
	"github.com/lenschain/backend/internal/pkg/snowflake"
	"github.com/lenschain/backend/internal/pkg/ws"
	experimentrepo "github.com/lenschain/backend/internal/repository/experiment"
)

// GroupService 分组协作服务接口
type GroupService interface {
	Create(ctx context.Context, sc *svcctx.ServiceContext, req *dto.CreateGroupReq) ([]*dto.GroupListItem, error)
	List(ctx context.Context, sc *svcctx.ServiceContext, req *dto.GroupListReq) ([]*dto.GroupListItem, int64, error)
	GetByID(ctx context.Context, sc *svcctx.ServiceContext, id int64) (*dto.GroupResp, error)
	Update(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.UpdateGroupReq) error
	Delete(ctx context.Context, sc *svcctx.ServiceContext, id int64) error
	AutoAssign(ctx context.Context, sc *svcctx.ServiceContext, req *dto.AutoAssignReq) (*dto.AutoAssignResp, error)
	Join(ctx context.Context, sc *svcctx.ServiceContext, groupID int64, req *dto.JoinGroupReq) error
	RemoveMember(ctx context.Context, sc *svcctx.ServiceContext, groupID, studentID int64) error
	ListMembers(ctx context.Context, sc *svcctx.ServiceContext, groupID int64) ([]dto.GroupMemberResp, error)
	GetProgress(ctx context.Context, sc *svcctx.ServiceContext, groupID int64) (*dto.GroupProgressResp, error)
	SendMessage(ctx context.Context, sc *svcctx.ServiceContext, groupID int64, req *dto.SendGroupMessageReq) error
	ListMessages(ctx context.Context, sc *svcctx.ServiceContext, groupID int64, req *dto.GroupMessageListReq) ([]*dto.GroupMessageItem, int64, error)
}

// groupService 分组协作服务实现。
// 统一承载多人实验分组、组员同步、组内消息与进度汇总逻辑。
type groupService struct {
	db                 *gorm.DB
	groupRepo          experimentrepo.GroupRepository
	memberRepo         experimentrepo.GroupMemberRepository
	messageRepo        experimentrepo.GroupMessageRepository
	roleRepo           experimentrepo.RoleRepository
	templateRepo       experimentrepo.TemplateRepository
	checkpointRepo     experimentrepo.CheckpointRepository
	checkResultRepo    experimentrepo.CheckpointResultRepository
	instanceRepo       experimentrepo.InstanceRepository
	userSummaryQuerier UserSummaryQuerier
	courseQuerier      CourseQuerier
	courseRoster       CourseRosterQuerier
}

// NewGroupService 创建多人实验分组服务实例
func NewGroupService(
	db *gorm.DB,
	groupRepo experimentrepo.GroupRepository,
	memberRepo experimentrepo.GroupMemberRepository,
	messageRepo experimentrepo.GroupMessageRepository,
	roleRepo experimentrepo.RoleRepository,
	templateRepo experimentrepo.TemplateRepository,
	checkpointRepo experimentrepo.CheckpointRepository,
	checkResultRepo experimentrepo.CheckpointResultRepository,
	instanceRepo experimentrepo.InstanceRepository,
	userSummaryQuerier UserSummaryQuerier,
	courseQuerier CourseQuerier,
	courseRoster CourseRosterQuerier,
) GroupService {
	return &groupService{
		db:                 db,
		groupRepo:          groupRepo,
		memberRepo:         memberRepo,
		messageRepo:        messageRepo,
		roleRepo:           roleRepo,
		templateRepo:       templateRepo,
		checkpointRepo:     checkpointRepo,
		checkResultRepo:    checkResultRepo,
		instanceRepo:       instanceRepo,
		userSummaryQuerier: userSummaryQuerier,
		courseQuerier:      courseQuerier,
		courseRoster:       courseRoster,
	}
}

// Create 创建实验分组
func (s *groupService) Create(ctx context.Context, sc *svcctx.ServiceContext, req *dto.CreateGroupReq) ([]*dto.GroupListItem, error) {
	templateID, err := snowflake.ParseString(req.TemplateID)
	if err != nil {
		return nil, errcode.ErrTemplateNotFound
	}
	courseID, err := snowflake.ParseString(req.CourseID)
	if err != nil {
		return nil, errcode.ErrInvalidParams.WithMessage("课程ID无效")
	}

	template, err := s.templateRepo.GetByID(ctx, templateID)
	if err != nil {
		return nil, errcode.ErrTemplateNotFound
	}
	if template.SchoolID != sc.SchoolID && !sc.IsSuperAdmin() {
		return nil, errcode.ErrForbidden
	}
	allowed, err := s.canManageCourseGroups(ctx, sc, courseID)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, errcode.ErrForbidden
	}
	courseStudents, err := s.courseRoster.ListCourseStudents(ctx, courseID)
	if err != nil {
		return nil, err
	}
	courseStudentSet := make(map[int64]struct{}, len(courseStudents))
	for _, student := range courseStudents {
		courseStudentSet[student.StudentID] = struct{}{}
	}

	seenStudents := make(map[int64]struct{})
	result := make([]*dto.GroupListItem, 0, len(req.Groups))

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		groupRepo := experimentrepo.NewGroupRepository(tx)
		memberRepo := experimentrepo.NewGroupMemberRepository(tx)

		for _, item := range req.Groups {
			group := &entity.ExperimentGroup{
				ID:          snowflake.Generate(),
				TemplateID:  templateID,
				CourseID:    courseID,
				SchoolID:    sc.SchoolID,
				GroupName:   item.GroupName,
				GroupMethod: req.GroupMethod,
				MaxMembers:  item.MaxMembers,
				Status:      enum.GroupStatusForming,
			}
			if len(item.Members) > 0 && len(item.Members) >= item.MaxMembers {
				group.Status = enum.GroupStatusReady
			}
			if err := groupRepo.Create(ctx, group); err != nil {
				return err
			}

			members := make([]*entity.GroupMember, 0, len(item.Members))
			for _, memberItem := range item.Members {
				studentID, parseErr := snowflake.ParseString(memberItem.StudentID)
				if parseErr != nil {
					return errcode.ErrInvalidParams.WithMessage("学生ID无效")
				}
				if _, exists := seenStudents[studentID]; exists {
					return errcode.ErrInvalidParams.WithMessage("同一请求中存在重复学生")
				}
				if _, enrolled := courseStudentSet[studentID]; !enrolled {
					return errcode.ErrInvalidParams.WithMessage("存在未加入课程的学生")
				}
				seenStudents[studentID] = struct{}{}

				inGroup, inGroupErr := memberRepo.IsStudentInGroup(ctx, templateID, courseID, studentID)
				if inGroupErr != nil {
					return inGroupErr
				}
				if inGroup {
					return errcode.ErrGroupMemberExists
				}

				member := &entity.GroupMember{
					ID:        snowflake.Generate(),
					GroupID:   group.ID,
					StudentID: studentID,
					JoinedAt:  time.Now(),
				}
				if memberItem.RoleID != nil {
					roleID, roleErr := snowflake.ParseString(*memberItem.RoleID)
					if roleErr != nil {
						return errcode.ErrInvalidParams.WithMessage("角色ID无效")
					}
					member.RoleID = &roleID
				}
				members = append(members, member)
			}
			if len(members) > 0 {
				if err := memberRepo.BatchCreate(ctx, members); err != nil {
					return err
				}
			}

			result = append(result, &dto.GroupListItem{
				ID:          strconv.FormatInt(group.ID, 10),
				GroupName:   group.GroupName,
				MemberCount: len(members),
				MaxMembers:  group.MaxMembers,
				Status:      group.Status,
				StatusText:  enum.GetGroupStatusText(group.Status),
			})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// List 获取分组列表
func (s *groupService) List(ctx context.Context, sc *svcctx.ServiceContext, req *dto.GroupListReq) ([]*dto.GroupListItem, int64, error) {
	params := &experimentrepo.GroupListParams{
		SchoolID: sc.SchoolID,
		Status:   req.Status,
		Page:     1,
		PageSize: 10000,
	}
	if req.TemplateID != "" {
		templateID, err := snowflake.ParseString(req.TemplateID)
		if err != nil {
			return nil, 0, errcode.ErrInvalidParams.WithMessage("模板ID无效")
		}
		params.TemplateID = templateID
	}
	if req.CourseID != "" {
		courseID, err := snowflake.ParseString(req.CourseID)
		if err != nil {
			return nil, 0, errcode.ErrInvalidParams.WithMessage("课程ID无效")
		}
		params.CourseID = courseID
	}

	groups, total, err := s.groupRepo.List(ctx, params)
	if err != nil {
		return nil, 0, err
	}
	visibleGroups := make([]*entity.ExperimentGroup, 0, len(groups))
	for _, group := range groups {
		allowed, accessErr := s.canViewGroup(ctx, sc, group)
		if accessErr != nil {
			return nil, 0, accessErr
		}
		if allowed {
			visibleGroups = append(visibleGroups, group)
		}
	}

	total = int64(len(visibleGroups))
	page, pageSize := pagination.NormalizeValues(req.Page, req.PageSize)
	start := pagination.Offset(page, pageSize)
	if start >= len(visibleGroups) {
		return []*dto.GroupListItem{}, total, nil
	}
	end := start + pageSize
	if end > len(visibleGroups) {
		end = len(visibleGroups)
	}

	items := make([]*dto.GroupListItem, 0, end-start)
	for _, group := range visibleGroups[start:end] {
		items = append(items, &dto.GroupListItem{
			ID:          strconv.FormatInt(group.ID, 10),
			GroupName:   group.GroupName,
			MemberCount: len(group.Members),
			MaxMembers:  group.MaxMembers,
			Status:      group.Status,
			StatusText:  enum.GetGroupStatusText(group.Status),
		})
	}
	return items, total, nil
}

// GetByID 获取分组详情
func (s *groupService) GetByID(ctx context.Context, sc *svcctx.ServiceContext, id int64) (*dto.GroupResp, error) {
	group, err := s.groupRepo.GetByIDWithMembers(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errcode.ErrGroupNotFound
		}
		return nil, err
	}
	allowed, err := s.canViewGroup(ctx, sc, group)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, errcode.ErrForbidden
	}
	members, err := s.buildGroupMembers(ctx, groupMembersToPointers(group.Members))
	if err != nil {
		return nil, err
	}
	return &dto.GroupResp{
		ID:          strconv.FormatInt(group.ID, 10),
		TemplateID:  strconv.FormatInt(group.TemplateID, 10),
		CourseID:    strconv.FormatInt(group.CourseID, 10),
		GroupName:   group.GroupName,
		GroupMethod: group.GroupMethod,
		MaxMembers:  group.MaxMembers,
		Status:      group.Status,
		StatusText:  enum.GetGroupStatusText(group.Status),
		Namespace:   group.Namespace,
		Members:     members,
		CreatedAt:   group.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:   group.UpdatedAt.UTC().Format(time.RFC3339),
	}, nil
}

// Update 编辑分组
func (s *groupService) Update(ctx context.Context, sc *svcctx.ServiceContext, id int64, req *dto.UpdateGroupReq) error {
	group, err := s.groupRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errcode.ErrGroupNotFound
		}
		return err
	}
	allowed, err := s.canManageGroup(ctx, sc, group)
	if err != nil {
		return err
	}
	if !allowed {
		return errcode.ErrForbidden
	}
	fields := make(map[string]interface{})
	if req.GroupName != nil {
		fields["group_name"] = *req.GroupName
	}
	if req.MaxMembers != nil {
		fields["max_members"] = *req.MaxMembers
	}
	if req.Status != nil {
		fields["status"] = *req.Status
	}
	if len(fields) == 0 {
		return nil
	}
	return s.groupRepo.UpdateFields(ctx, id, fields)
}

// Delete 删除分组
func (s *groupService) Delete(ctx context.Context, sc *svcctx.ServiceContext, id int64) error {
	group, err := s.groupRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errcode.ErrGroupNotFound
		}
		return err
	}
	allowed, err := s.canManageGroup(ctx, sc, group)
	if err != nil {
		return err
	}
	if !allowed {
		return errcode.ErrForbidden
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		memberRepo := experimentrepo.NewGroupMemberRepository(tx)
		groupRepo := experimentrepo.NewGroupRepository(tx)
		if err := memberRepo.DeleteByGroupID(ctx, id); err != nil {
			return err
		}
		return groupRepo.Delete(ctx, id)
	})
}

// AutoAssign 系统随机分组
func (s *groupService) AutoAssign(ctx context.Context, sc *svcctx.ServiceContext, req *dto.AutoAssignReq) (*dto.AutoAssignResp, error) {
	if s.courseRoster == nil {
		return nil, errcode.ErrInternal.WithMessage("课程学生名单查询器未配置")
	}
	templateID, err := snowflake.ParseString(req.TemplateID)
	if err != nil {
		return nil, errcode.ErrInvalidParams.WithMessage("模板ID无效")
	}
	courseID, err := snowflake.ParseString(req.CourseID)
	if err != nil {
		return nil, errcode.ErrInvalidParams.WithMessage("课程ID无效")
	}
	allowed, err := s.canManageCourseGroups(ctx, sc, courseID)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, errcode.ErrForbidden
	}

	students, err := s.courseRoster.ListCourseStudents(ctx, courseID)
	if err != nil {
		return nil, err
	}
	roles, err := s.roleRepo.ListByTemplateID(ctx, templateID)
	if err != nil {
		return nil, err
	}

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	rng.Shuffle(len(students), func(i, j int) {
		students[i], students[j] = students[j], students[i]
	})

	groupItems := make([]dto.GroupItemReq, 0)
	respGroups := make([]dto.AutoAssignGroupItem, 0)
	for start, groupIndex := 0, 1; start < len(students); start, groupIndex = start+req.GroupSize, groupIndex+1 {
		end := start + req.GroupSize
		if end > len(students) {
			end = len(students)
		}
		chunk := students[start:end]
		memberReqs := make([]dto.MemberItemReq, 0, len(chunk))
		respMembers := make([]dto.AutoAssignMemberItem, 0, len(chunk))
		for i, student := range chunk {
			var roleID *string
			roleName := ""
			if len(roles) > 0 {
				role := roles[i%len(roles)]
				roleIDStr := strconv.FormatInt(role.ID, 10)
				roleID = &roleIDStr
				roleName = role.RoleName
			}
			memberReqs = append(memberReqs, dto.MemberItemReq{
				StudentID: strconv.FormatInt(student.StudentID, 10),
				RoleID:    roleID,
			})
			respMembers = append(respMembers, dto.AutoAssignMemberItem{
				StudentID:   strconv.FormatInt(student.StudentID, 10),
				StudentName: student.Name,
				RoleName:    roleName,
			})
		}
		groupName := fmt.Sprintf("%s%d组", req.GroupNamePrefix, groupIndex)
		groupItems = append(groupItems, dto.GroupItemReq{
			GroupName:  groupName,
			MaxMembers: req.GroupSize,
			Members:    memberReqs,
		})
		respGroups = append(respGroups, dto.AutoAssignGroupItem{
			GroupName: groupName,
			Members:   respMembers,
		})
	}

	created, err := s.Create(ctx, sc, &dto.CreateGroupReq{
		TemplateID:  req.TemplateID,
		CourseID:    req.CourseID,
		GroupMethod: enum.GroupMethodRandom,
		Groups:      groupItems,
	})
	if err != nil {
		return nil, err
	}
	for i := range created {
		if i < len(respGroups) {
			respGroups[i].ID = created[i].ID
		}
	}
	return &dto.AutoAssignResp{
		TotalGroups:   len(respGroups),
		TotalStudents: len(students),
		Groups:        respGroups,
	}, nil
}

// Join 学生加入分组
func (s *groupService) Join(ctx context.Context, sc *svcctx.ServiceContext, groupID int64, req *dto.JoinGroupReq) error {
	group, err := s.groupRepo.GetByIDWithMembers(ctx, groupID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errcode.ErrGroupNotFound
		}
		return err
	}
	allowed, err := s.canJoinGroup(ctx, sc, group)
	if err != nil {
		return err
	}
	if !allowed {
		return errcode.ErrForbidden
	}
	if group.Status != enum.GroupStatusForming {
		return errcode.ErrGroupNotJoinable
	}
	if len(group.Members) >= group.MaxMembers {
		return errcode.ErrGroupFull
	}

	inGroup, err := s.memberRepo.IsStudentInGroup(ctx, group.TemplateID, group.CourseID, sc.UserID)
	if err != nil {
		return err
	}
	if inGroup {
		return errcode.ErrGroupMemberExists
	}

	member := &entity.GroupMember{
		ID:        snowflake.Generate(),
		GroupID:   groupID,
		StudentID: sc.UserID,
		JoinedAt:  time.Now(),
	}
	if req.RoleID != nil {
		roleID, parseErr := snowflake.ParseString(*req.RoleID)
		if parseErr != nil {
			return errcode.ErrInvalidParams.WithMessage("角色ID无效")
		}
		role, roleErr := s.roleRepo.GetByID(ctx, roleID)
		if roleErr != nil {
			return errcode.ErrRoleNotFound
		}
		usedCount := 0
		for _, existed := range group.Members {
			if existed.RoleID != nil && *existed.RoleID == roleID {
				usedCount++
			}
		}
		if usedCount >= role.MaxMembers {
			return errcode.ErrGroupFull.WithMessage("该角色已被占满")
		}
		member.RoleID = &roleID
	}

	if err := s.memberRepo.Create(ctx, member); err != nil {
		return err
	}
	if len(group.Members)+1 >= group.MaxMembers {
		return s.groupRepo.UpdateFields(ctx, groupID, map[string]interface{}{"status": enum.GroupStatusReady})
	}
	return nil
}

// RemoveMember 移除组员
func (s *groupService) RemoveMember(ctx context.Context, sc *svcctx.ServiceContext, groupID, studentID int64) error {
	group, err := s.groupRepo.GetByID(ctx, groupID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errcode.ErrGroupNotFound
		}
		return err
	}
	allowed, err := s.canManageGroup(ctx, sc, group)
	if err != nil {
		return err
	}
	if !allowed {
		return errcode.ErrForbidden
	}
	member, err := s.memberRepo.GetByGroupAndStudent(ctx, groupID, studentID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errcode.ErrGroupNotFound.WithMessage("分组成员不存在")
		}
		return err
	}
	if err := s.memberRepo.Delete(ctx, member.ID); err != nil {
		return err
	}
	return s.groupRepo.UpdateFields(ctx, groupID, map[string]interface{}{"status": enum.GroupStatusForming})
}

// ListMembers 获取组员列表
func (s *groupService) ListMembers(ctx context.Context, sc *svcctx.ServiceContext, groupID int64) ([]dto.GroupMemberResp, error) {
	group, err := s.groupRepo.GetByID(ctx, groupID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errcode.ErrGroupNotFound
		}
		return nil, err
	}
	allowed, err := s.canViewGroup(ctx, sc, group)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, errcode.ErrForbidden
	}
	members, err := s.memberRepo.ListByGroupID(ctx, groupID)
	if err != nil {
		return nil, err
	}
	return s.buildGroupMembers(ctx, members)
}

// GetProgress 获取组内进度
func (s *groupService) GetProgress(ctx context.Context, sc *svcctx.ServiceContext, groupID int64) (*dto.GroupProgressResp, error) {
	group, err := s.groupRepo.GetByIDWithMembers(ctx, groupID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errcode.ErrGroupNotFound
		}
		return nil, err
	}
	allowed, err := s.canViewGroup(ctx, sc, group)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, errcode.ErrForbidden
	}

	templateCheckpoints, err := s.checkpointRepo.ListByTemplateID(ctx, group.TemplateID)
	if err != nil {
		return nil, err
	}
	instances, err := s.instanceRepo.ListByGroupID(ctx, groupID)
	if err != nil {
		return nil, err
	}
	instanceByStudent := make(map[int64]*entity.ExperimentInstance)
	for _, instance := range instances {
		current := instanceByStudent[instance.StudentID]
		if current == nil || instance.AttemptNo > current.AttemptNo || instance.CreatedAt.After(current.CreatedAt) {
			instanceByStudent[instance.StudentID] = instance
		}
	}

	roleNames, err := s.loadRoleNames(ctx, group.TemplateID)
	if err != nil {
		return nil, err
	}

	memberItems := make([]dto.GroupMemberProgressItem, 0, len(group.Members))
	groupCheckpointItems := make([]dto.GroupCheckpointItem, 0)
	groupCheckpointMap := make(map[int64]dto.GroupCheckpointItem)

	for _, member := range group.Members {
		summary := s.getUserSummary(ctx, member.StudentID)
		memberItem := dto.GroupMemberProgressItem{
			StudentID:   strconv.FormatInt(member.StudentID, 10),
			StudentName: summary.Name,
		}
		if member.RoleID != nil {
			memberItem.RoleName = roleNames[*member.RoleID]
		}

		instance := instanceByStudent[member.StudentID]
		if instance != nil {
			instanceID := strconv.FormatInt(instance.ID, 10)
			statusText := enum.GetInstanceStatusText(instance.Status)
			memberItem.InstanceID = &instanceID
			memberItem.InstanceStatus = &instance.Status
			memberItem.InstanceStatusText = &statusText

			results, resultErr := s.checkResultRepo.ListByInstanceID(ctx, instance.ID)
			if resultErr != nil {
				return nil, resultErr
			}
			totalScore := 0.0
			for _, cp := range templateCheckpoints {
				if cp.Scope == enum.CheckpointScopeGroup {
					item := groupCheckpointMap[cp.ID]
					item.CheckpointID = strconv.FormatInt(cp.ID, 10)
					item.Title = cp.Title
					item.Scope = cp.Scope
					for _, res := range results {
						if res.CheckpointID == cp.ID {
							item.IsPassed = item.IsPassed || res.IsPassed
							if res.CheckedAt.IsZero() {
								continue
							}
							checkedAt := res.CheckedAt.UTC().Format(time.RFC3339)
							item.CheckedAt = &checkedAt
						}
					}
					groupCheckpointMap[cp.ID] = item
					continue
				}
				memberItem.CheckpointsTotal++
				for _, res := range results {
					if res.CheckpointID == cp.ID {
						if res.IsPassed {
							memberItem.CheckpointsPassed++
						}
						if res.Score != nil {
							totalScore += *res.Score
						}
						break
					}
				}
			}
			memberItem.PersonalScore = &totalScore
		}
		memberItems = append(memberItems, memberItem)
	}

	for _, cp := range templateCheckpoints {
		if cp.Scope != enum.CheckpointScopeGroup {
			continue
		}
		if item, ok := groupCheckpointMap[cp.ID]; ok {
			groupCheckpointItems = append(groupCheckpointItems, item)
		} else {
			groupCheckpointItems = append(groupCheckpointItems, dto.GroupCheckpointItem{
				CheckpointID: strconv.FormatInt(cp.ID, 10),
				Title:        cp.Title,
				Scope:        cp.Scope,
			})
		}
	}
	sort.Slice(groupCheckpointItems, func(i, j int) bool {
		return groupCheckpointItems[i].CheckpointID < groupCheckpointItems[j].CheckpointID
	})

	return &dto.GroupProgressResp{
		GroupID:          strconv.FormatInt(group.ID, 10),
		GroupName:        group.GroupName,
		GroupStatus:      group.Status,
		GroupStatusText:  enum.GetGroupStatusText(group.Status),
		Members:          memberItems,
		GroupCheckpoints: groupCheckpointItems,
	}, nil
}

// SendMessage 发送组内消息。
// 消息会持久化保存，并通过实验分组聊天房间实时广播。
func (s *groupService) SendMessage(ctx context.Context, sc *svcctx.ServiceContext, groupID int64, req *dto.SendGroupMessageReq) error {
	group, err := s.groupRepo.GetByID(ctx, groupID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errcode.ErrGroupNotFound
		}
		return err
	}
	allowed, err := s.canSendGroupMessage(ctx, sc, group)
	if err != nil {
		return err
	}
	if !allowed {
		return errcode.ErrForbidden
	}
	message := &entity.GroupMessage{
		ID:          snowflake.Generate(),
		GroupID:     groupID,
		SenderID:    sc.UserID,
		Content:     req.Content,
		MessageType: enum.MessageTypeText,
	}
	if err := s.messageRepo.Create(ctx, message); err != nil {
		return err
	}

	manager := ws.GetManager()
	if manager != nil {
		summary := s.getUserSummary(ctx, sc.UserID)
		roleName := s.resolveMemberRoleName(ctx, groupID, sc.UserID)
		_ = manager.BroadcastToRoom(experimentGroupRoom(groupID), buildGroupWSMessage("chat_message", map[string]interface{}{
			"sender_id":   strconv.FormatInt(sc.UserID, 10),
			"sender_name": summary.Name,
			"role_name":   roleName,
			"content":     req.Content,
			"sent_at":     message.CreatedAt.UTC().Format(time.RFC3339),
		}))
	}
	return nil
}

// ListMessages 获取组内消息历史
func (s *groupService) ListMessages(ctx context.Context, sc *svcctx.ServiceContext, groupID int64, req *dto.GroupMessageListReq) ([]*dto.GroupMessageItem, int64, error) {
	group, err := s.groupRepo.GetByID(ctx, groupID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, 0, errcode.ErrGroupNotFound
		}
		return nil, 0, err
	}
	allowed, err := s.canViewGroup(ctx, sc, group)
	if err != nil {
		return nil, 0, err
	}
	if !allowed {
		return nil, 0, errcode.ErrForbidden
	}
	messages, total, err := s.messageRepo.List(ctx, &experimentrepo.GroupMessageListParams{
		GroupID:  groupID,
		Page:     req.Page,
		PageSize: req.PageSize,
	})
	if err != nil {
		return nil, 0, err
	}
	items := make([]*dto.GroupMessageItem, 0, len(messages))
	for _, message := range messages {
		summary := s.getUserSummary(ctx, message.SenderID)
		items = append(items, &dto.GroupMessageItem{
			ID:          strconv.FormatInt(message.ID, 10),
			SenderID:    strconv.FormatInt(message.SenderID, 10),
			SenderName:  summary.Name,
			Content:     message.Content,
			MessageType: message.MessageType,
			CreatedAt:   message.CreatedAt.UTC().Format(time.RFC3339),
		})
	}
	return items, total, nil
}

// buildGroupMembers 组装分组成员响应
func (s *groupService) buildGroupMembers(ctx context.Context, members []*entity.GroupMember) ([]dto.GroupMemberResp, error) {
	roleNames, err := s.loadRoleNamesFromMembers(ctx, members)
	if err != nil {
		return nil, err
	}
	items := make([]dto.GroupMemberResp, 0, len(members))
	for _, member := range members {
		if member == nil {
			continue
		}
		summary := s.getUserSummary(ctx, member.StudentID)
		item := dto.GroupMemberResp{
			ID:          strconv.FormatInt(member.ID, 10),
			StudentID:   strconv.FormatInt(member.StudentID, 10),
			StudentName: summary.Name,
			StudentNo:   summary.StudentNo,
			JoinedAt:    member.JoinedAt.UTC().Format(time.RFC3339),
		}
		if member.RoleID != nil {
			roleID := strconv.FormatInt(*member.RoleID, 10)
			item.RoleID = &roleID
			if roleName, ok := roleNames[*member.RoleID]; ok {
				item.RoleName = &roleName
			}
		}
		if member.InstanceID != nil {
			instanceID := strconv.FormatInt(*member.InstanceID, 10)
			item.InstanceID = &instanceID
		}
		items = append(items, item)
	}
	return items, nil
}

// loadRoleNames 按模板加载角色名称映射
func (s *groupService) loadRoleNames(ctx context.Context, templateID int64) (map[int64]string, error) {
	roles, err := s.roleRepo.ListByTemplateID(ctx, templateID)
	if err != nil {
		return nil, err
	}
	roleNames := make(map[int64]string, len(roles))
	for _, role := range roles {
		roleNames[role.ID] = role.RoleName
	}
	return roleNames, nil
}

// loadRoleNamesFromMembers 按成员使用到的角色加载名称映射
func (s *groupService) loadRoleNamesFromMembers(ctx context.Context, members []*entity.GroupMember) (map[int64]string, error) {
	roleNames := make(map[int64]string)
	for _, member := range members {
		if member == nil {
			continue
		}
		if member.RoleID == nil {
			continue
		}
		if _, ok := roleNames[*member.RoleID]; ok {
			continue
		}
		role, err := s.roleRepo.GetByID(ctx, *member.RoleID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				continue
			}
			return nil, err
		}
		roleNames[*member.RoleID] = role.RoleName
	}
	return roleNames, nil
}

// groupMembersToPointers 将值切片转换为指针切片，便于复用统一的成员装配逻辑。
func groupMembersToPointers(members []entity.GroupMember) []*entity.GroupMember {
	result := make([]*entity.GroupMember, 0, len(members))
	for i := range members {
		member := members[i]
		result = append(result, &member)
	}
	return result
}

// getUserSummary 获取实验模块需要的用户摘要
func (s *groupService) getUserSummary(ctx context.Context, userID int64) ExperimentUserSummary {
	if s.userSummaryQuerier == nil {
		return ExperimentUserSummary{
			UserID: userID,
			Name:   strconv.FormatInt(userID, 10),
		}
	}
	summary := s.userSummaryQuerier.GetUserSummary(ctx, userID)
	if summary == nil {
		return ExperimentUserSummary{
			UserID: userID,
			Name:   strconv.FormatInt(userID, 10),
		}
	}
	return *summary
}

// resolveMemberRoleName 获取分组成员当前角色名称。
func (s *groupService) resolveMemberRoleName(ctx context.Context, groupID, studentID int64) string {
	member, err := s.memberRepo.GetByGroupAndStudent(ctx, groupID, studentID)
	if err != nil || member.RoleID == nil {
		return ""
	}
	role, err := s.roleRepo.GetByID(ctx, *member.RoleID)
	if err != nil {
		return ""
	}
	return role.RoleName
}

// canManageCourseGroups 判断当前用户是否可管理指定课程下的实验分组。
func (s *groupService) canManageCourseGroups(ctx context.Context, sc *svcctx.ServiceContext, courseID int64) (bool, error) {
	if sc.IsSuperAdmin() {
		return true, nil
	}
	if !sc.IsTeacher() {
		return false, nil
	}
	teacherID, err := s.courseQuerier.GetCourseTeacherID(ctx, courseID)
	if err != nil {
		return false, err
	}
	return teacherID == sc.UserID, nil
}

// canViewGroup 判断当前用户是否可查看分组及其子资源。
func (s *groupService) canViewGroup(ctx context.Context, sc *svcctx.ServiceContext, group *entity.ExperimentGroup) (bool, error) {
	if group == nil {
		return false, nil
	}
	if sc.IsSuperAdmin() {
		return true, nil
	}
	if group.SchoolID != sc.SchoolID {
		return false, nil
	}
	if sc.IsTeacher() {
		return s.canManageCourseGroups(ctx, sc, group.CourseID)
	}
	if sc.IsStudent() {
		return s.isGroupMember(ctx, group.ID, sc.UserID)
	}
	return false, nil
}

// canManageGroup 判断当前用户是否可执行分组管理操作。
func (s *groupService) canManageGroup(ctx context.Context, sc *svcctx.ServiceContext, group *entity.ExperimentGroup) (bool, error) {
	if group == nil {
		return false, nil
	}
	if group.SchoolID != sc.SchoolID && !sc.IsSuperAdmin() {
		return false, nil
	}
	return s.canManageCourseGroups(ctx, sc, group.CourseID)
}

// canJoinGroup 判断当前学生是否可加入指定分组。
func (s *groupService) canJoinGroup(ctx context.Context, sc *svcctx.ServiceContext, group *entity.ExperimentGroup) (bool, error) {
	if group == nil || !sc.IsStudent() {
		return false, nil
	}
	if group.SchoolID != sc.SchoolID && !sc.IsSuperAdmin() {
		return false, nil
	}
	if s.courseRoster == nil {
		return false, errcode.ErrInternal.WithMessage("课程学生名单查询器未配置")
	}
	students, err := s.courseRoster.ListCourseStudents(ctx, group.CourseID)
	if err != nil {
		return false, err
	}
	for _, student := range students {
		if student.StudentID == sc.UserID {
			return true, nil
		}
	}
	return false, nil
}

// canSendGroupMessage 判断当前用户是否可发送组内消息。
func (s *groupService) canSendGroupMessage(ctx context.Context, sc *svcctx.ServiceContext, group *entity.ExperimentGroup) (bool, error) {
	if group == nil || !sc.IsStudent() {
		return false, nil
	}
	if group.SchoolID != sc.SchoolID && !sc.IsSuperAdmin() {
		return false, nil
	}
	return s.isGroupMember(ctx, group.ID, sc.UserID)
}

// isGroupMember 判断指定学生是否属于当前分组。
func (s *groupService) isGroupMember(ctx context.Context, groupID, studentID int64) (bool, error) {
	_, err := s.memberRepo.GetByGroupAndStudent(ctx, groupID, studentID)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	return false, err
}
