package threading

import (
	"github.com/zp857/goutil/rescue"
	"go.uber.org/zap"
	"runtime"
	"time"
)

func GoSafe(fn func()) {
	go RunSafe(fn)
}

func RunSafe(fn func()) {
	defer rescue.Recover()
	fn()
}

func WatchGoroutines(ticker, minNum int) {
	tk := time.NewTicker(time.Duration(ticker) * time.Second)
	for {
		num := runtime.NumGoroutine()
		if num > minNum {
			zap.L().Sugar().Warnf("current goroutine: %v", num)
		}
		<-tk.C
	}
}
