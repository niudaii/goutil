package structutil

import "reflect"

func IsStruct(v any) bool {
	k := reflect.ValueOf(v).Kind()
	if k == reflect.Invalid {
		return false
	}
	return k == reflect.Struct
}
