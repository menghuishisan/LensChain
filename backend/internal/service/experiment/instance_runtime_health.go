package experiment

import "strings"

// evaluatePodHealth 判断 Pod 当前是否已进入需要置为异常并触发恢复的状态。
func evaluatePodHealth(status *PodStatus, queryErr error) (bool, string) {
	if queryErr != nil {
		return true, "运行容器状态查询失败"
	}
	if status == nil {
		return true, "运行容器不存在"
	}

	reason := strings.TrimSpace(status.Reason)
	message := strings.TrimSpace(status.Message)
	switch {
	case status.Status == "Failed":
		return true, buildPodHealthMessage("容器运行失败", reason, message)
	case status.Status == "Unknown":
		return true, buildPodHealthMessage("容器节点状态未知", reason, message)
	case reason == "OOMKilled":
		return true, buildPodHealthMessage("容器发生 OOMKilled", reason, message)
	case reason == "CrashLoopBackOff":
		return true, buildPodHealthMessage("容器进入 CrashLoopBackOff", reason, message)
	case reason == "ImagePullBackOff" || reason == "ErrImagePull":
		return true, buildPodHealthMessage("容器镜像拉取失败", reason, message)
	default:
		return false, ""
	}
}

// buildPodHealthMessage 统一拼接运行时异常消息。
func buildPodHealthMessage(summary, reason, message string) string {
	detail := strings.TrimSpace(strings.Join([]string{reason, message}, " "))
	detail = strings.TrimSpace(detail)
	if detail == "" {
		return summary
	}
	return summary + "：" + detail
}
