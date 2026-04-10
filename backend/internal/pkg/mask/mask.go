// mask.go
// 数据脱敏工具包
// 提供手机号、邮箱、身份证等敏感信息的脱敏处理
// 供所有模块 service 层调用，避免各模块重复实现脱敏逻辑

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
