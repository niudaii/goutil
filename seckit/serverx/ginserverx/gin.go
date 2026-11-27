// Package ginserverx adapts seckit/serverx health probes and security header
// middleware to Gin routers.
package ginserverx

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/niudaii/goutil/seckit/serverx"
)

// SecurityHeaders applies serverx.ApplySecurityHeaders to every Gin request.
func SecurityHeaders(cfg serverx.SecurityHeadersConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		serverx.ApplySecurityHeaders(c.Writer, c.Request, cfg)
		c.Next()
	}
}

// RegisterHealthz mounts GET and HEAD /healthz on r using serverx.HealthzHandler.
func RegisterHealthz(r gin.IRoutes, checks ...func(context.Context) error) {
	if r == nil {
		return
	}
	h := gin.WrapF(serverx.HealthzHandler(checks...))
	r.GET("/healthz", h)
	r.HEAD("/healthz", h)
}

// RegisterHealthzNoContent mounts GET and HEAD /healthz returning 204 No Content.
func RegisterHealthzNoContent(r gin.IRoutes) {
	if r == nil {
		return
	}
	h := gin.WrapF(serverx.HealthzNoContentHandler())
	r.GET("/healthz", h)
	r.HEAD("/healthz", h)
}
