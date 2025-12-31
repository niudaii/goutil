package db

import (
	"log"
	"os"
	"strings"
	"time"

	v1 "github.com/niudaii/goutil/constants/v1"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

func init() {
	time.Local = cst
}

// 设置默认时区为 CST
var cst = time.FixedZone("CST", 8*3600)

func (c *Config) GormConfig(prefix string, singular bool) *gorm.Config {
	config := &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			TablePrefix:   prefix,
			SingularTable: singular,
		},
		DisableForeignKeyConstraintWhenMigrating: true,
		NowFunc: func() time.Time {
			return time.Now().In(cst)
		},
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
