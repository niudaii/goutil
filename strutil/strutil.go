package strutil

import "strings"

func SplitByComma(s string) []string {
	items := strings.Split(s, ",")
	for i, item := range items {
		items[i] = strings.TrimSpace(item)
	}
	return items
}

func FirstUpper(s string) string {
	if s == "" {
		return ""
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func RemoveDigits(s string) string {
	re := regexp.MustCompile("[0-9]+")
	return re.ReplaceAllString(s, "")
}

func IsChinese(s string) bool {
	for _, r := range s {
		if !unicode.Is(unicode.Han, r) {
			return false
		}
	}
	return true
}