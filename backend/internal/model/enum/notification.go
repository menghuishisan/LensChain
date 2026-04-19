// notification.go
// 模块07 — 通知与消息：统一维护站内信、公告、模板等领域枚举。
// 该文件用于支撑收件箱、公告、偏好设置和内部通知事件的统一语义。

package enum

const (
	NotificationCategorySystem      = 1 // 系统通知
	NotificationCategoryCourse      = 2 // 课程通知
	NotificationCategoryExperiment  = 3 // 实验通知
	NotificationCategoryCompetition = 4 // 竞赛通知
	NotificationCategoryGrade       = 5 // 成绩通知
)

// NotificationCategoryText 通知分类文本映射。
var NotificationCategoryText = map[int16]string{
	NotificationCategorySystem:      "系统通知",
	NotificationCategoryCourse:      "课程通知",
	NotificationCategoryExperiment:  "实验通知",
	NotificationCategoryCompetition: "竞赛通知",
	NotificationCategoryGrade:       "成绩通知",
}

// GetNotificationCategoryText 获取通知分类文本。
func GetNotificationCategoryText(value int16) string {
	if text, ok := NotificationCategoryText[value]; ok {
		return text
	}
	return "未知"
}

// IsValidNotificationCategory 校验通知分类是否合法。
func IsValidNotificationCategory(value int16) bool {
	_, ok := NotificationCategoryText[value]
	return ok
}

const (
	SystemAnnouncementStatusDraft       = 1 // 草稿
	SystemAnnouncementStatusPublished   = 2 // 已发布
	SystemAnnouncementStatusUnpublished = 3 // 已下架
)

// SystemAnnouncementStatusText 公告状态文本映射。
var SystemAnnouncementStatusText = map[int16]string{
	SystemAnnouncementStatusDraft:       "草稿",
	SystemAnnouncementStatusPublished:   "已发布",
	SystemAnnouncementStatusUnpublished: "已下架",
}

// GetSystemAnnouncementStatusText 获取公告状态文本。
func GetSystemAnnouncementStatusText(value int16) string {
	if text, ok := SystemAnnouncementStatusText[value]; ok {
		return text
	}
	return "未知"
}

// IsValidSystemAnnouncementStatus 校验公告状态是否合法。
func IsValidSystemAnnouncementStatus(value int16) bool {
	_, ok := SystemAnnouncementStatusText[value]
	return ok
}
