package db

import (
	v1 "github.com/zp857/goutil/constants/v1"
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
	if w.logZap {
		zap.L().Sugar().Named(v1.GormLogger).Infof(message, data...)
	} else {
		w.Writer.Printf(message, data...)
	}
}
