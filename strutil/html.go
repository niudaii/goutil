package strutil

import (
	"regexp"
	"strings"
)

func CleanHTMLContent(content string) string {
	// 移除CDATA标签
	content = strings.ReplaceAll(content, "<![CDATA[", "")
	content = strings.ReplaceAll(content, "]]>", "")

	// 移除HTML标签
	htmlTagRe := regexp.MustCompile(`<[^>]*>`)
	content = htmlTagRe.ReplaceAllString(content, "")

	// 移除多余的空白字符
	spaceRe := regexp.MustCompile(`\s+`)
	content = spaceRe.ReplaceAllString(content, " ")

	return strings.TrimSpace(content)
}
