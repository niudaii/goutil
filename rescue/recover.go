package rescue

import (
	v1 "github.com/zp857/goutil/constants/v1"
	"github.com/zp857/goutil/errorx"
	"go.uber.org/zap"
)

func Recover(cleanups ...func()) {
	for _, cleanup := range cleanups {
		cleanup()
	}

	if p := recover(); p != nil {
		zap.L().Sugar().Errorf(v1.RecoverWithStack, p, string(errorx.GetStack(1, 10)))
	}
}
