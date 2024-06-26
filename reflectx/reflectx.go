package reflectx

import (
	"reflect"
	"runtime"
)

func GetFuncName(f interface{}) string {
	return runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name()
}
