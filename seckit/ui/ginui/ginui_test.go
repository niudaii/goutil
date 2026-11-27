package ginui

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/niudaii/goutil/seckit/ui"
)

func TestRegisterRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	RegisterRoutes(r, ui.Config{
		Current: "cves",
		Defaults: []ui.NavItem{
			{Key: "cves", Label: "CVEs", URL: "https://cves.example.test"},
		},
	})

	for _, tc := range []struct {
		path   string
		marker string
	}{
		{ui.TokensCSSPath, "--primary: #3867e8"},
		{ui.ShellJSPath, "SecOps"},
		{ui.NavJSONPath, `"current":"cves"`},
	} {
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tc.path, nil))
		if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), tc.marker) {
			t.Fatalf("%s got %d %q", tc.path, rec.Code, rec.Body.String())
		}
	}
}
