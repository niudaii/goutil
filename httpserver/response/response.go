package response

import (
	"github.com/gin-gonic/gin"
	"github.com/zp857/goutil/constants/v1"
	"github.com/zp857/goutil/errorx"
	"github.com/zp857/goutil/jsonutil"
	"github.com/zp857/goutil/structs"
	"go.uber.org/zap"
	"net/http"
)

func Result(code int, data interface{}, msg string, c *gin.Context) {
	response := structs.Response{
		Code:    code,
		Message: msg,
		Data:    data,
	}
	zap.L().Named(v1.GinLogger).Sugar().Infof(v1.Response, jsonutil.MustPretty(response))
	c.JSON(http.StatusOK, response)
}

func Ok(data interface{}, msg string, c *gin.Context) {
	Result(http.StatusOK, data, msg, c)
}

func OkWithMessage(msg string, c *gin.Context) {
	Result(http.StatusOK, struct{}{}, msg, c)
}

func ErrorWithMessage(msg string, err error, c *gin.Context) {
	if err != nil {
		zap.L().Named(v1.GinLogger).Sugar().Errorf("%v => %v\n%v",
			msg,
			err,
			string(errorx.GetStack(2, 10)),
		)
	}
	Result(http.StatusInternalServerError, struct{}{}, msg, c)
}

func UnAuthWithMessage(msg string, c *gin.Context) {
	Result(http.StatusUnauthorized, struct{}{}, msg, c)
}

func BadRequestWithMessage(msg string, c *gin.Context) {
	Result(http.StatusBadRequest, struct{}{}, msg, c)
}
