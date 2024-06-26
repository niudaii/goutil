package strutil

import (
	"encoding/json"
)

func ToMap(s string) (result map[string]any, err error) {
	result = make(map[string]any)
	err = json.Unmarshal([]byte(s), &result)
	return
}

func MustToMap(s string) (result map[string]any) {
	var err error
	result, err = ToMap(s)
	if err != nil {
		panic(err)
	}
	return
}
