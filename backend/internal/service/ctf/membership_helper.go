// membership_helper.go
// 模块05 — CTF竞赛：参赛成员关系辅助函数。
// 该文件提供多个 service 共享的竞赛成员定位能力，避免在团队、环境、攻防赛等服务中重复实现。

package ctf

import (
	"context"

	"github.com/lenschain/backend/internal/model/entity"
	"github.com/lenschain/backend/internal/pkg/errcode"
	ctfrepo "github.com/lenschain/backend/internal/repository/ctf"
)

// getCompetitionMemberTeam 获取学生在指定竞赛中的团队成员关系和团队实体。
func getCompetitionMemberTeam(
	ctx context.Context,
	teamMemberRepo ctfrepo.TeamMemberRepository,
	teamRepo ctfrepo.TeamRepository,
	competitionID, studentID int64,
) (*entity.TeamMember, *entity.Team, error) {
	member, err := teamMemberRepo.GetByCompetitionAndStudent(ctx, competitionID, studentID)
	if err != nil {
		return nil, nil, errcode.ErrSubmissionTeamMissing
	}
	team, err := teamRepo.GetByID(ctx, member.TeamID)
	if err != nil {
		return nil, nil, err
	}
	return member, team, nil
}
