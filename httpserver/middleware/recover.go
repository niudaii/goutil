package middleware

import (
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	v1 "github.com/niudaii/goutil/constants/v1"
	"github.com/niudaii/goutil/errorx"
	"go.uber.org/zap"
)

func GinRecovery(logger *zap.SugaredLogger, debug bool, skip, depth int) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// Check for a broken connection, as it is not really a
				// condition that warrants a panic stack trace.
				var brokenPipe bool
				if ne, ok := err.(*net.OpError); ok {
					if se, ok := ne.Err.(*os.SyscallError); ok {
						if strings.Contains(strings.ToLower(se.Error()), "broken pipe") || strings.Contains(strings.ToLower(se.Error()), "connection reset by peer") {
							brokenPipe = true
						}
					}
				}

				httpRequest, _ := httputil.DumpRequest(c.Request, false)
				if brokenPipe {
					logger.Error(c.Request.URL.Path,
						zap.Any(v1.ErrorKey, err),
						zap.String(v1.RequestKey, string(httpRequest)),
					)
					// If the connection is dead, we can't write a status to it.
					c.Error(err.(error)) // nolint: errcheck
					c.Abort()
					return
				}
				if debug {
					stackBytes := errorx.GetStack(skip, depth)
					logger.Errorf(v1.RecoverWithAll, err, string(httpRequest), string(stackBytes))
				} else {
					logger.Errorf(v1.RecoverWithRequest, err, string(httpRequest))
				}
				c.AbortWithStatus(http.StatusInternalServerError)
			}
		}()
		c.Next()
	}
}
