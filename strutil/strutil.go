package strutil

import "strings"

func SplitByComma(itemString string) []string {
	items := strings.Split(itemString, ",")
	for i, item := range items {
		items[i] = strings.TrimSpace(item)
	}
	return items
}
