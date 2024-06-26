package signal

import (
	v1 "github.com/zp857/goutil/constants/v1"
	"go.uber.org/zap"
	"os"
	"os/signal"
	"syscall"
)

func WaitingSignal() {
	sig := make(chan os.Signal)
	// SIGHUP: terminal closed
	// SIGINT: Ctrl+C
	// SIGTERM: program exit
	// SIGQUIT: Ctrl+/
	// SIGKILL: kill
	signal.Notify(sig,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
		syscall.SIGKILL,
	)
	for t := range sig {
		zap.L().Sugar().Errorf(v1.SignalExit, t.String())
		os.Exit(0)
	}
}
