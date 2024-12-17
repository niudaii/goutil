package db

import (
	"strings"

	v1 "github.com/niudaii/goutil/constants/v1"
	"go.uber.org/zap"
	"gorm.io/gorm/logger"
)

type Writer struct {
	logger.Writer
	logZap bool
}

func NewWriter(w logger.Writer, logZap bool) *Writer {
	return &Writer{
		Writer: w,
		logZap: logZap,
	}
}

func (w *Writer) Printf(message string, data ...interface{}) {
	if !strings.Contains(message, "Error 1062") {
		return
	}
	if w.logZap {
		zap.L().Sugar().Named(v1.GormLogger).Infof(message, data...)
	} else {
		w.Writer.Printf(message, data...)
	}
}
