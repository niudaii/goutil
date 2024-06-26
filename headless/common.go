package headless

import (
	"time"
)

const (
	waitTime  = 3 * time.Second
	debugTime = 20 * time.Second
)

func Wait() {
	time.Sleep(waitTime)
}

func DebugWait() {
	time.Sleep(debugTime)
}
