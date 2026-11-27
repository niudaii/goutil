package ginoidc

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/niudaii/goutil/seckit/oidc"
)

func TestGinAdapterAuthAndAPIKey(t *testing.T) {
	gin.SetMode(gin.TestMode)
	client, err := oidc.New(oidc.Config{
		Issuer:           "https://idp.example.test/connect",
		ClientID:         "gin-test",
		ClientSecret:     "secret-at-least-32-bytes-long-123456",
		RedirectURI:      "https://gin.example.test/api/auth/oidc/callback",
		AllowedSubjects:  []string{"alice"},
		RequireWhitelist: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	r := gin.New()
	Register(r, client)
	r.GET("/api/auth/status", Status(client))
	r.GET("/", StartHomeLogin(client), func(c *gin.Context) { c.Status(http.StatusOK) })
	r.POST("/api/items", RequireAPIKeyOrSession(client, "my-secret-api-key"), func(c *gin.Context) {
		c.String(http.StatusOK, SubjectFromContext(c))
	})

	// Unauthenticated POST rejected
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/items", nil))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}

	// Valid API key allowed
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodPost, "/api/items", nil)
	req2.Header.Set(oidc.APIKeyHeader, "my-secret-api-key")
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("expected 200 for valid API key, got %d", w2.Code)
	}

	// Valid session + matching origin allowed, wrong origin blocked
	sessVal, err := client.SealSessionForTest("alice", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	ck := &http.Cookie{Name: client.SessionCookieName(), Value: sessVal}

	wBadOrigin := httptest.NewRecorder()
	reqBad := httptest.NewRequest(http.MethodPost, "/api/items", nil)
	reqBad.AddCookie(ck)
	reqBad.Header.Set("Origin", "https://evil.example")
	r.ServeHTTP(wBadOrigin, reqBad)
	if wBadOrigin.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for cross-origin POST, got %d", wBadOrigin.Code)
	}

	wGood := httptest.NewRecorder()
	reqGood := httptest.NewRequest(http.MethodPost, "/api/items", nil)
	reqGood.AddCookie(ck)
	reqGood.Header.Set("Origin", "https://gin.example.test")
	r.ServeHTTP(wGood, reqGood)
	if wGood.Code != http.StatusOK || wGood.Body.String() != "alice" {
		t.Fatalf("expected 200 + alice, got %d %q", wGood.Code, wGood.Body.String())
	}
}
