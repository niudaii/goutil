package logger

import (
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"testing"
)

func TestInit(t *testing.T) {
	err := Init()
	assert.Equal(t, nil, err)
	zap.L().Sugar().Infof("hello")
	zap.L().Info("123", zap.Any("1", 1), zap.Any("456", "123"))
}
