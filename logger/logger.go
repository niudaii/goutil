package logger

import (
	"os"

	"github.com/niudaii/goutil/fileutil"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func Init() (err error) {
	err = InitWithConfig(&DefaultConfig)
	return
}

func InitWithConfig(config *Config) (err error) {
	if !fileutil.IsExist(config.Director) {
		_ = os.MkdirAll(config.Director, os.ModePerm)
	}
	// 获取日志写入位置
	var writeSyncer zapcore.WriteSyncer
	writeSyncer, err = getLogWriter(config.Director, config.MaxAge, config.LogInConsole)
	if err != nil {
		return
	}
	// 获取日志编码格式
	encoder := getEncoder(config.Format)
	// 获取日志最低等级，即>=该等级，才会被写入
	var l = new(zapcore.Level)
	if err = l.UnmarshalText([]byte(config.Level)); err != nil {
		return
	}
	// 创建一个将日志写入 WriteSyncer 的核心
	core := zapcore.NewCore(encoder, writeSyncer, l)
	logger := zap.New(core, zap.AddCaller())
	// 替换 zap 包中全局的 logger 实例，后续在其他包中只需使用 zap.L() 调用即可
	zap.ReplaceGlobals(logger)
	return
}
