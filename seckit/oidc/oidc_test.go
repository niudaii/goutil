package oidc

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const testIssuer = "https://idp.example.test/connect"

func newTestPair(t *testing.T, cfg Config) (*Client, *Client) {
	t.Helper()
	if cfg.Issuer == "" {
		cfg.Issuer = testIssuer
	}
	if cfg.ClientID == "" {
		cfg.ClientID = "test-client"
	}
	if cfg.ClientSecret == "" {
		cfg.ClientSecret = "test-secret-key-at-least-32-bytes"
	}
	if cfg.RedirectURI == "" {
		cfg.RedirectURI = "https://app.example.test/api/auth/oidc/callback"
	}
	a, err := New(cfg)
	if err != nil {
		t.Fatalf("New(a): %v", err)
	}
	b, err := New(cfg)
	if err != nil {
		t.Fatalf("New(b): %v", err)
	}
	return a, b
}

func TestStatelessMultiReplicaLoginAndTamperRejection(t *testing.T) {
	replicaA, replicaB := newTestPair(t, Config{
		AllowedSubjects:  []string{"alice"},
		RequireWhitelist: true,
	})

	var capturedNonce string
	tokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		now := time.Now()
		tok := jwt.NewWithClaims(jwt.SigningMethodHS256, idClaims{
			RegisteredClaims: jwt.RegisteredClaims{
				Issuer:    testIssuer,
				Subject:   "alice",
				Audience:  jwt.ClaimStrings{replicaA.ClientID()},
				IssuedAt:  jwt.NewNumericDate(now),
				ExpiresAt: jwt.NewNumericDate(now.Add(time.Minute)),
			},
			Nonce: capturedNonce,
		})
		signed, _ := tok.SignedString([]byte(replicaA.clientSecret))
		_ = json.NewEncoder(w).Encode(map[string]string{"id_token": signed})
	}))
	defer tokenSrv.Close()
	replicaB.SetTokenURLForTest(tokenSrv.URL)

	// Replica A starts login
	muxA := http.NewServeMux()
	replicaA.Register(muxA)
	loginRec := httptest.NewRecorder()
	muxA.ServeHTTP(loginRec, httptest.NewRequest(http.MethodGet, replicaA.LoginPath(), nil))
	if loginRec.Code != http.StatusFound {
		t.Fatalf("expected 302 login redirect, got %d", loginRec.Code)
	}
	loc, _ := url.Parse(loginRec.Header().Get("Location"))
	capturedNonce = loc.Query().Get("nonce")
	binding := loginRec.Result().Cookies()[0]

	// Replica B completes callback with encrypted cookie created by Replica A
	muxB := http.NewServeMux()
	replicaB.Register(muxB)
	cbReq := httptest.NewRequest(http.MethodGet, replicaB.CallbackPath()+"?code=c1&state="+url.QueryEscape(loc.Query().Get("state")), nil)
	cbReq.AddCookie(binding)
	cbRec := httptest.NewRecorder()
	muxB.ServeHTTP(cbRec, cbReq)
	if cbRec.Code != http.StatusFound || cbRec.Header().Get("Location") != "/" {
		t.Fatalf("replica B callback failed: %d -> %s", cbRec.Code, cbRec.Header().Get("Location"))
	}

	var sessCookie *http.Cookie
	for _, c := range cbRec.Result().Cookies() {
		if c.Name == replicaB.SessionCookieName() {
			sessCookie = c
		}
	}
	if sessCookie == nil || sessCookie.Value == "" {
		t.Fatal("missing session cookie")
	}

	// Replica A verifies the session cookie minted by Replica B
	checkReq := httptest.NewRequest(http.MethodGet, "/protected", nil)
	checkReq.AddCookie(sessCookie)
	if got := replicaA.Subject(checkReq); got != "alice" {
		t.Fatalf("replica A failed to validate stateless cookie: got %q", got)
	}

	// Tampered cookie must fail closed
	tampered := *sessCookie
	tampered.Value = strings.Repeat("A", len(sessCookie.Value))
	tamperReq := httptest.NewRequest(http.MethodGet, "/protected", nil)
	tamperReq.AddCookie(&tampered)
	if got := replicaA.Subject(tamperReq); got != "" {
		t.Fatalf("tampered cookie accepted for %q", got)
	}
}

func TestRS256JWKSVerification(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	var srv *httptest.Server
	var expectedNonce string
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/jwks":
			n := base64.RawURLEncoding.EncodeToString(priv.PublicKey.N.Bytes())
			e := base64.RawURLEncoding.EncodeToString(big.NewInt(int64(priv.PublicKey.E)).Bytes())
			_ = json.NewEncoder(w).Encode(map[string]any{
				"keys": []map[string]string{{"kty": "RSA", "use": "sig", "kid": "k1", "n": n, "e": e}},
			})
		case "/token":
			now := time.Now()
			tok := jwt.NewWithClaims(jwt.SigningMethodRS256, idClaims{
				RegisteredClaims: jwt.RegisteredClaims{
					Issuer:    srv.URL,
					Subject:   "alice",
					Audience:  jwt.ClaimStrings{"rs-client"},
					IssuedAt:  jwt.NewNumericDate(now),
					ExpiresAt: jwt.NewNumericDate(now.Add(time.Minute)),
				},
				Nonce: expectedNonce,
			})
			tok.Header["kid"] = "k1"
			signed, _ := tok.SignedString(priv)
			_ = json.NewEncoder(w).Encode(map[string]string{"id_token": signed})
		}
	}))
	defer srv.Close()

	c, err := New(Config{
		Issuer:               srv.URL,
		ClientID:             "rs-client",
		ClientSecret:         "secret-at-least-32-bytes-long-123456",
		RedirectURI:          "https://app.example.test/",
		AllowedCallbackPaths: []string{"/"},
		AllowRS256:           true,
		AllowedSubjects:      []string{"alice"},
		RequireWhitelist:     true,
		HTTPClient:           srv.Client(),
	})
	if err != nil {
		t.Fatal(err)
	}

	mux := http.NewServeMux()
	c.Register(mux)
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		c.HandleHome(w, r, func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	})

	loginRec := httptest.NewRecorder()
	mux.ServeHTTP(loginRec, httptest.NewRequest(http.MethodGet, "/", nil))
	loc, _ := url.Parse(loginRec.Header().Get("Location"))
	expectedNonce = loc.Query().Get("nonce")

	cbReq := httptest.NewRequest(http.MethodGet, "/?code=c1&state="+url.QueryEscape(loc.Query().Get("state")), nil)
	cbReq.AddCookie(loginRec.Result().Cookies()[0])
	cbRec := httptest.NewRecorder()
	mux.ServeHTTP(cbRec, cbReq)
	if cbRec.Code != http.StatusFound || cbRec.Header().Get("Location") != "/" {
		t.Fatalf("RS256 callback failed: %d -> %s", cbRec.Code, cbRec.Header().Get("Location"))
	}
	var sess *http.Cookie
	for _, ck := range cbRec.Result().Cookies() {
		if ck.Name == c.SessionCookieName() {
			sess = ck
		}
	}
	if sess == nil {
		t.Fatal("missing session cookie after RS256 callback")
	}
}

func TestCookieNameAvoidsDoubleHostPrefix(t *testing.T) {
	if _, err := New(Config{
		ClientID:     "c",
		ClientSecret: "secret-value",
		RedirectURI:  "https://app.example.test/api/auth/oidc/callback",
	}); err == nil {
		t.Fatal("expected error when issuer URL is empty")
	}
	c, err := New(Config{
		Issuer:            testIssuer,
		ClientID:          "c",
		ClientSecret:      "secret-value",
		RedirectURI:       "https://app.example.test/api/auth/oidc/callback",
		LoginCookieName:   "__Host-custom_login",
		SessionCookieName: "__Host-custom_session",
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if got := c.CookieName("__Host-foo"); got != "__Host-foo" {
		t.Fatalf("CookieName(__Host-foo) = %q, want __Host-foo", got)
	}
	if got := c.LoginCookieName(); got != "__Host-custom_login" {
		t.Fatalf("LoginCookieName() = %q, want __Host-custom_login", got)
	}
	if got := c.SessionCookieName(); got != "__Host-custom_session" {
		t.Fatalf("SessionCookieName() = %q, want __Host-custom_session", got)
	}
}
