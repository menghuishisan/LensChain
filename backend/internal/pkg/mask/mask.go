// mask.go
// 该文件提供统一的数据脱敏工具，用于把手机号、邮箱等敏感信息转换成适合展示或导出的
// 脱敏形式。需要在日志、导出、通知或页面返回中隐藏敏感值时，应优先复用这里的规则。

package mask

// Phone 手机号脱敏（138****8000）
// 对于长度不足7位的号码，原样返回
func Phone(phone string) string {
	if len(phone) < 7 {
		return phone
	}
	return phone[:3] + "****" + phone[len(phone)-4:]
}

// Email 邮箱脱敏（u***@example.com）
// 保留首字母和 @ 后的域名部分
func Email(email string) string {
	at := -1
	for i, c := range email {
		if c == '@' {
			at = i
			break
		}
	}
	if at <= 0 {
		return email
	}
	return string(email[0]) + "***" + email[at:]
}
