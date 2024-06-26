package db

import (
	v1 "github.com/zp857/goutil/constants/v1"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
	"log"
	"os"
	"strings"
	"time"
)

func (c *Config) GormConfig(prefix string, singular bool) *gorm.Config {
	config := &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			TablePrefix:   prefix,
			SingularTable: singular,
		},
		DisableForeignKeyConstraintWhenMigrating: true,
	}
	_default := logger.New(NewWriter(log.New(os.Stdout, "\r\n", log.LstdFlags), c.LogZap), logger.Config{
		SlowThreshold: 200 * time.Millisecond,
		LogLevel:      logger.Info,
		Colorful:      false,
	})
	switch strings.ToLower(c.LogLevel) {
	case v1.SlientLevel:
		config.Logger = _default.LogMode(logger.Silent)
	case v1.ErrorLevel:
		config.Logger = _default.LogMode(logger.Error)
	case v1.WarnLevel:
		config.Logger = _default.LogMode(logger.Warn)
	case v1.InfoLevel:
		config.Logger = _default.LogMode(logger.Info)
	default:
		config.Logger = _default.LogMode(logger.Error)
	}
	return config
}
