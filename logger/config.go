package logger

import (
	"github.com/niudaii/goutil/constants"
	v1 "github.com/niudaii/goutil/constants/v1"
)

type Config struct {
	Director     string `json:"director" yaml:"director"`
	Format       string `json:"format" yaml:"format"`
	Level        string `json:"level" yaml:"level"`
	MaxAge       int    `json:"maxAge" yaml:"maxAge"`
	Compress     bool   `json:"compress" yaml:"compress"`
	LogInConsole bool   `json:"logInConsole" yaml:"logInConsole"`
}

var DefaultConfig = Config{
	Director:     "logs",
	Format:       constants.ConsoleFormat,
	Level:        v1.DebugLevel,
	MaxAge:       constants.MonthDays,
	Compress:     true,
	LogInConsole: true,
}
