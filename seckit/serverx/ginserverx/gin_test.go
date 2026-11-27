package ginserverx

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/niudaii/goutil/seckit/serverx"
)

func TestGinServerxMiddlewareAndHealthz(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(SecurityHeaders(serverx.SecurityHeadersConfig{CSP: "default-src 'self'"}))
	RegisterHealthz(r)
	r.GET("/assets/index-BDD-RXTH.js", func(c *gin.Context) { c.Status(http.StatusOK) })
	r.GET("/api/test", func(c *gin.Context) { c.Status(http.StatusOK) })

	hz := httptest.NewRecorder()
	r.ServeHTTP(hz, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if hz.Code != http.StatusOK || !strings.Contains(hz.Body.String(), `"ok"`) || hz.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("unexpected /healthz: %d %s", hz.Code, hz.Body.String())
	}

	asset := httptest.NewRecorder()
	r.ServeHTTP(asset, httptest.NewRequest(http.MethodGet, "/assets/index-BDD-RXTH.js", nil))
	if asset.Header().Get("Cache-Control") != serverx.DefaultHashedAssetCacheControl {
		t.Fatalf("expected immutable Cache-Control on hashed asset, got %q", asset.Header().Get("Cache-Control"))
	}

	api := httptest.NewRecorder()
	r.ServeHTTP(api, httptest.NewRequest(http.MethodGet, "/api/test", nil))
	if api.Header().Get("Cache-Control") != "no-store" || api.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("unexpected API headers: %v", api.Header())
	}
}
