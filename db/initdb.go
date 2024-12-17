package db

import (
	v1 "github.com/niudaii/goutil/constants/v1"
	"go.uber.org/zap"
)

type InitData interface {
	TableName() string
	Initialize() (err error)
	CheckDataExist() bool
}

func InitTableData(inits ...InitData) error {
	logger := zap.L().Sugar()
	for i := 0; i < len(inits); i++ {
		if inits[i].CheckDataExist() {
			logger.Infof(v1.InitDataExist, inits[i].TableName())
			continue
		}
		if err := inits[i].Initialize(); err != nil {
			logger.Infof(v1.InitDataError, inits[i].TableName(), err)
		}
		logger.Infof(v1.InitDataSuccess, inits[i].TableName())
	}
	return nil
}
