package db

import (
	"fmt"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/niudaii/goutil/constants"
	v1 "github.com/niudaii/goutil/constants/v1"
	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var db *gorm.DB

const (
	retryCount = 10
	retrySleep = 30 * time.Second
)

func Init(config Config) (err error) {
	for i := 1; i <= retryCount; i++ {
		if err = connectDB(config); err == nil {
			return
		}
		zap.L().Sugar().Errorf("conn to db err: %v, retry count: %v/%v", err, i, retryCount)
		time.Sleep(retrySleep)
	}
	return
}

func connectDB(config Config) (err error) {
	if config.DBName == "" {
		err = fmt.Errorf(v1.EmptyDBNameError)
		return
	}
	switch config.DBType {
	case constants.Mysql:
		db, err = GormMysql(config)
	case constants.Pgsql:
		db, err = GormPgsql(config)
	case constants.Sqlite:
		db, err = GormSqlite(config)
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

func GormSqlite(config Config) (db *gorm.DB, err error) {
	db, err = gorm.Open(sqlite.Open(config.DSN()), config.GormConfig(config.Prefix, config.Singular))
	return
}

func GetDB() *gorm.DB {
	return db
}
