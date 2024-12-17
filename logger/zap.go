package logger

import (
	"os"
	"path"
	"time"

	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"github.com/niudaii/goutil/constants"
	v1 "github.com/niudaii/goutil/constants/v1"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func getEncoder(format string) zapcore.Encoder {
	encodeConfig := zap.NewProductionEncoderConfig()
	encodeConfig.EncodeTime = formatEncodeTime
	encodeConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	encodeConfig.EncodeCaller = zapcore.ShortCallerEncoder
	encodeConfig.TimeKey = v1.TimeKey
	encodeConfig.ConsoleSeparator = constants.SingleSpace
	if format == constants.JsonFormat {
		return zapcore.NewJSONEncoder(encodeConfig)
	}
	return zapcore.NewConsoleEncoder(encodeConfig)
}

func formatEncodeTime(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(t.Format(time.DateTime))
}

func getLogWriter(director string, maxAge int, logInConsole bool) (zapcore.WriteSyncer, error) {
	fileWriter, err := rotatelogs.New(
		path.Join(director, constants.Ymd+constants.LogExt),
		rotatelogs.WithClock(rotatelogs.Local),
		rotatelogs.WithMaxAge(time.Duration(maxAge)*24*time.Hour),
		rotatelogs.WithRotationTime(time.Hour),
	)
	if logInConsole {
		return zapcore.NewMultiWriteSyncer(zapcore.AddSync(os.Stdout), zapcore.AddSync(fileWriter)), err
	}
	return zapcore.AddSync(fileWriter), err
}
