// markdown_sanitizer.go
// 模块公共能力：Markdown/HTML 内容安全过滤
// 对照 docs/modules/03-课程与教学/05-验收标准.md 安全要求中的 XSS 防护

package contentsafety

import (
	"regexp"
	"strings"
)

var blockedTagPattern = regexp.MustCompile(`(?is)<(?:script|style|iframe|object|embed|link|meta)\b[^>]*>[\s\S]*?</(?:script|style|iframe|object|embed|link|meta)\s*>`)
var blockedSelfClosingTagPattern = regexp.MustCompile(`(?is)<(script|style|iframe|object|embed|link|meta)\b[^>]*?/?>`)
var eventAttrPattern = regexp.MustCompile(`(?is)\s+on[a-z0-9:_-]+\s*=\s*(".*?"|'.*?'|[^\s>]+)`)
var styleAttrPattern = regexp.MustCompile(`(?is)\s+style\s*=\s*(".*?"|'.*?'|[^\s>]+)`)
var dangerousURIAttrPattern = regexp.MustCompile(`(?is)\s+(href|src|action|formaction|xlink:href)\s*=\s*("([^"]*)"|'([^']*)'|([^\s>]+))`)

// SanitizeMarkdown 过滤 Markdown 原文中的危险 HTML。
// 这里只处理文档验收要求里的危险 HTML 标签、事件属性和危险协议，保留普通 Markdown 文本。
func SanitizeMarkdown(input string) string {
	if input == "" {
		return input
	}

	output := blockedTagPattern.ReplaceAllString(input, "")
	output = blockedSelfClosingTagPattern.ReplaceAllString(output, "")
	output = eventAttrPattern.ReplaceAllString(output, "")
	output = styleAttrPattern.ReplaceAllString(output, "")
	output = dangerousURIAttrPattern.ReplaceAllStringFunc(output, sanitizeDangerousURIAttr)
	return output
}

// SanitizeOptionalMarkdown 过滤可选 Markdown 字段中的危险 HTML。
func SanitizeOptionalMarkdown(input *string) *string {
	if input == nil {
		return nil
	}
	sanitized := SanitizeMarkdown(*input)
	return &sanitized
}

// sanitizeDangerousURIAttr 过滤使用危险协议的 URL 属性。
func sanitizeDangerousURIAttr(attr string) string {
	matches := dangerousURIAttrPattern.FindStringSubmatch(attr)
	if len(matches) == 0 {
		return attr
	}

	raw := firstNonEmpty(matches[3], matches[4], matches[5])
	if isDangerousURL(raw) {
		return ""
	}
	return attr
}

// isDangerousURL 判断 URL 是否使用危险协议。
func isDangerousURL(raw string) bool {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return false
	}

	lower := strings.ToLower(trimmed)
	return strings.HasPrefix(lower, "javascript:") ||
		strings.HasPrefix(lower, "vbscript:") ||
		strings.HasPrefix(lower, "data:")
}

// firstNonEmpty 返回第一个非空字符串。
func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
