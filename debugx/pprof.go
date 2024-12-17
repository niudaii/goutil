package debugx

import (
	"net/http"
	_ "net/http/pprof"

	"github.com/niudaii/goutil/threading"
	"go.uber.org/zap"
)

type Config struct {
	PProf     bool `yaml:"pprof" json:"pprof"`
	Goroutine bool `yaml:"goroutine" json:"goroutine"`
	Ticker    int  `yaml:"ticker" json:"ticker"`
	MinNum    int  `yaml:"minNum" json:"minNum"`
}

func StartPProf() {
	go func() {
		logger := zap.L().Named("[debug]").Sugar()
		addr := "0.0.0.0:6060"
		logger.Infof("start http pprof on %v", addr)
		err := http.ListenAndServe(addr, nil)
		if err != nil {
			logger.Errorf("start http pprof err: %v", err)
		}
	}()
}

// WatchGoroutines monitors the number of goroutines and logs if they exceed a threshold.
func WatchGoroutines(tickDuration, threshold int) {
	go func() {
		threading.WatchGoroutines(tickDuration, threshold)
	}()
}

// SetupDebugging initializes debugging tools based on the provided configuration.
func SetupDebugging(config Config) {
	if config.PProf {
		StartPProf()
	}
	if config.Goroutine {
		WatchGoroutines(config.Ticker, config.MinNum)
	}
}
