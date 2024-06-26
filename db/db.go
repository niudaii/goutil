package db

import (
	"fmt"
	"github.com/zp857/goutil/constants"
	v1 "github.com/zp857/goutil/constants/v1"
	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"time"
)

var db *gorm.DB

const (
	retryCount = 10
	retrySleep = 30 * time.Second
)

func Init(config Config) (err error) {
	for i := 1; i <= retryCount; i++ {
		err = Conn(config)
		if err == nil {
			break
		}
		zap.L().Sugar().Errorf("conn to db err: %v, retry count: %v/%v", err, i, retryCount)
		time.Sleep(retrySleep)
	}
	return
}

func Conn(config Config) (err error) {
	if config.DBName == "" {
		err = fmt.Errorf(v1.EmptyDBNameError)
		return
	}
	switch config.DBType {
	case constants.Mysql:
		db, err = GormMysql(config)
	case constants.Pgsql:
		db, err = GormPgsql(config)
	default:
		err = fmt.Errorf(v1.UnSupportDBTypeError)
	}
	if err == nil {
		sqlDB, _ := db.DB()
		sqlDB.SetMaxIdleConns(config.MaxIdleConns)
		sqlDB.SetMaxOpenConns(config.MaxOpenConns)
	}
	return
}

func GormMysql(config Config) (db *gorm.DB, err error) {
	mysqlConfig := mysql.Config{
		DSN:                       config.DSN(),
		DefaultStringSize:         191,
		SkipInitializeWithVersion: false,
	}
	db, err = gorm.Open(mysql.New(mysqlConfig), config.GormConfig(config.Prefix, config.Singular))
	return
}

func GormPgsql(config Config) (db *gorm.DB, err error) {
	pgsqlConfig := postgres.Config{
		DSN:                  config.DSN(),
		PreferSimpleProtocol: false,
	}
	db, err = gorm.Open(postgres.New(pgsqlConfig), config.GormConfig(config.Prefix, config.Singular))
	return
}

func GetDB() *gorm.DB {
	return db
}
