// helpers.go
// 模块05 — CTF竞赛：service 层公共业务规则辅助函数。
// 该文件只承载纯业务计算，不访问数据库、不依赖 Gin，用于多个 service 文件共享动态计分、
// 自动分组和攻防赛回合编排等规则。

package ctf

import (
	"math"
	"time"

	"github.com/lenschain/backend/internal/model/dto"
)

// roundSchedule 表示攻防赛单回合各阶段的时间切片。
type roundSchedule struct {
	RoundStartAt    time.Time
	AttackEndAt     time.Time
	DefenseEndAt    time.Time
	SettlementEndAt time.Time
}

// calculateJeopardyScore 计算解题赛题目当前得分。
// 规则：
// 1. 每出现一次新的正确解题，按 decay_factor 的指数幂衰减。
// 2. 题目最低分不低于 base_score * min_score_ratio。
// 3. First Blood 按当前题目基础分乘以 first_blood_bonus 比例额外加成。
// 4. 文档示例采用向下取整口径，不使用四舍五入。
func calculateJeopardyScore(baseScore int, solveCount int64, cfg dto.JeopardyScoringConfig, isFirstBlood bool) int {
	if baseScore <= 0 {
		return 0
	}

	minScore := float64(baseScore) * cfg.MinScoreRatio
	score := float64(baseScore) * math.Pow(cfg.DecayFactor, float64(solveCount))
	if score < minScore {
		score = minScore
	}
	if isFirstBlood {
		score += float64(baseScore) * cfg.FirstBloodBonus
	}
	return int(math.Floor(score))
}

// autoAssignTeamsToGroups 按固定最大容量顺序切分团队列表。
// 文档没有要求复杂的权重或蛇形编排时，先采用稳定顺序分桶，保证结果可预测、可复现。
func autoAssignTeamsToGroups(teamIDs []int64, maxTeamsPerGroup int) [][]int64 {
	if len(teamIDs) == 0 || maxTeamsPerGroup <= 0 {
		return [][]int64{}
	}

	groups := make([][]int64, 0, (len(teamIDs)+maxTeamsPerGroup-1)/maxTeamsPerGroup)
	for start := 0; start < len(teamIDs); start += maxTeamsPerGroup {
		end := start + maxTeamsPerGroup
		if end > len(teamIDs) {
			end = len(teamIDs)
		}
		bucket := make([]int64, 0, end-start)
		bucket = append(bucket, teamIDs[start:end]...)
		groups = append(groups, bucket)
	}
	return groups
}

// buildRoundSchedule 计算指定回合的开始与各阶段结束时间。
// roundNumber 从 1 开始，后续回合会在前序所有阶段总时长基础上顺延。
func buildRoundSchedule(
	baseStartAt time.Time,
	roundNumber int,
	attackDurationMinutes int,
	defenseDurationMinutes int,
	settlementDurationMinutes int,
) roundSchedule {
	if roundNumber <= 0 {
		roundNumber = 1
	}
	roundOffsetMinutes := (roundNumber - 1) * (attackDurationMinutes + defenseDurationMinutes + settlementDurationMinutes)
	roundStartAt := baseStartAt.Add(time.Duration(roundOffsetMinutes) * time.Minute)
	attackEndAt := roundStartAt.Add(time.Duration(attackDurationMinutes) * time.Minute)
	defenseEndAt := attackEndAt.Add(time.Duration(defenseDurationMinutes) * time.Minute)
	settlementEndAt := defenseEndAt.Add(time.Duration(settlementDurationMinutes) * time.Minute)

	return roundSchedule{
		RoundStartAt:    roundStartAt,
		AttackEndAt:     attackEndAt,
		DefenseEndAt:    defenseEndAt,
		SettlementEndAt: settlementEndAt,
	}
}
