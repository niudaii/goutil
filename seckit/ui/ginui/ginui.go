// Package ginui adapts seckit/ui shared design assets and suite navigation
// endpoints to Gin routers.
package ginui

import (
	"github.com/gin-gonic/gin"
	"github.com/niudaii/goutil/seckit/ui"
)

// RegisterRoutes mounts /secops/tokens.css, /secops/shell.js, and /secops/nav.json on r.
func RegisterRoutes(r gin.IRoutes, cfg ui.Config) {
	if r == nil {
		return
	}
	tokens := gin.WrapF(ui.ServeTokensCSS)
	shell := gin.WrapF(ui.ServeShellJS)
	nav := gin.WrapF(ui.NavHandler(cfg))

	for _, route := range []struct {
		path    string
		handler gin.HandlerFunc
	}{
		{ui.TokensCSSPath, tokens},
		{ui.ShellJSPath, shell},
		{ui.NavJSONPath, nav},
	} {
		r.GET(route.path, route.handler)
		r.HEAD(route.path, route.handler)
	}
}
