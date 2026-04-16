// instance_service_grade_sync.go
// 模块04 — 实验环境：实验成绩回传课程成绩体系
// 统一根据模板成绩策略选择最终成绩，并通过跨模块接口回写模块03作业提交记录

package experiment

import (
	"context"
	"time"

	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/model/enum"
)

// syncCourseGradeIfNeeded 在实验最终成绩可确定时回传课程成绩体系。
func (s *instanceService) syncCourseGradeIfNeeded(
	ctx context.Context,
	instance *entity.ExperimentInstance,
	template *entity.ExperimentTemplate,
	teacherComment *string,
) error {
	if s.courseGradeSyncer == nil || instance == nil || template == nil {
		return nil
	}
	if instance.CourseID == nil || *instance.CourseID == 0 || instance.AssignmentID == nil || *instance.AssignmentID == 0 {
		return nil
	}

	score, submittedAt, ok, err := s.resolveCourseSyncScore(ctx, instance, template)
	if err != nil || !ok {
		return err
	}

	return s.courseGradeSyncer.SyncExperimentScore(
		ctx,
		*instance.CourseID,
		*instance.AssignmentID,
		instance.StudentID,
		template.Title,
		score,
		submittedAt,
		teacherComment,
	)
}

// resolveCourseSyncScore 根据模板成绩策略解析应该回传课程系统的最终成绩。
func (s *instanceService) resolveCourseSyncScore(
	ctx context.Context,
	instance *entity.ExperimentInstance,
	template *entity.ExperimentTemplate,
) (float64, time.Time, bool, error) {
	instances, err := s.instanceRepo.ListByTemplateAndStudent(ctx, template.ID, instance.StudentID)
	if err != nil {
		return 0, time.Time{}, false, err
	}

	selected, ok := selectSyncedInstance(instances, template.ScoreStrategy)
	if !ok || selected.TotalScore == nil {
		return 0, time.Time{}, false, nil
	}

	submittedAt := time.Now()
	if selected.SubmittedAt != nil {
		submittedAt = *selected.SubmittedAt
	}
	return *selected.TotalScore, submittedAt, true, nil
}

// selectSyncedInstance 根据成绩策略从同模板多次实验中选择需要回传的实例。
func selectSyncedInstance(instances []*entity.ExperimentInstance, scoreStrategy int) (*entity.ExperimentInstance, bool) {
	var selected *entity.ExperimentInstance
	for _, instance := range instances {
		if instance == nil || instance.TotalScore == nil {
			continue
		}
		if instance.Status != enum.InstanceStatusSubmitted && instance.Status != enum.InstanceStatusDestroyed {
			continue
		}
		if selected == nil {
			selected = instance
			continue
		}
		switch scoreStrategy {
		case enum.ScoreStrategyHighest:
			if *instance.TotalScore > *selected.TotalScore {
				selected = instance
				continue
			}
			if *instance.TotalScore == *selected.TotalScore && instance.AttemptNo > selected.AttemptNo {
				selected = instance
			}
		default:
			if instance.AttemptNo > selected.AttemptNo {
				selected = instance
				continue
			}
			if instance.AttemptNo == selected.AttemptNo && instance.UpdatedAt.After(selected.UpdatedAt) {
				selected = instance
			}
		}
	}
	return selected, selected != nil
}
