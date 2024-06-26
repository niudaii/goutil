package structutil

import "encoding/json"

func ToMap(v any) (m map[string]any, err error) {
	m = make(map[string]any)
	j, err := json.Marshal(v)
	if err != nil {
		return
	}
	err = json.Unmarshal(j, &m)
	return
}
