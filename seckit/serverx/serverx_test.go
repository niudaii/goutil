package serverx

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestDefaultIsHashedAsset(t *testing.T) {
	hashed := []string{
		"/assets/index-BDD-RXTH.js",
		"/assets/index-_3zgA4i-.js",
		"/assets/index-C57wfC55.js",
		"/assets/logo-C2ID3Aj9.svg",
		"/assets/app.a1b2c3d4.js",
	}
	for _, p := range hashed {
		if !DefaultIsHashedAsset(p) {
			t.Errorf("expected hashed asset for %q", p)
		}
	}

	notHashed := []string{
		"/",
		"/index.html",
		"/api/tasks",
		"/assets/styles.css",
		"/assets/dashboard.js",
		"/assets/logo.svg",
		"/assets/logo-20261127.svg",
		"/assets/favicon-20261127.ico",
		"/assets/my-component.js",
		"/logo-C2ID3Aj9.svg",
	}
	for _, p := range notHashed {
		if DefaultIsHashedAsset(p) {
			t.Errorf("expected non-hashed asset for %q", p)
		}
	}
}

func TestSecurityHeadersAndHealthz(t *testing.T) {
	mux := http.NewServeMux()
	RegisterHealthz(mux)
	mux.HandleFunc("GET /assets/index-BDD-RXTH.js", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("GET /api/data", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	h := Recover(nil)(SecurityHeaders(SecurityHeadersConfig{
		CSP:          "default-src 'self'",
		FrameOptions: "DENY",
	})(mux))

	// /healthz returns 200 {"status":"ok"}
	hz := httptest.NewRecorder()
	h.ServeHTTP(hz, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if hz.Code != http.StatusOK || !strings.Contains(hz.Body.String(), `"ok"`) ||
		hz.Header().Get("Cache-Control") != "no-store" ||
		hz.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("unexpected /healthz response: %d %q %v", hz.Code, hz.Body.String(), hz.Header())
	}

	// Hashed asset gets immutable cache-control
	asset := httptest.NewRecorder()
	h.ServeHTTP(asset, httptest.NewRequest(http.MethodGet, "/assets/index-BDD-RXTH.js", nil))
	if asset.Header().Get("Cache-Control") != DefaultHashedAssetCacheControl {
		t.Fatalf("expected immutable Cache-Control on hashed asset, got %q", asset.Header().Get("Cache-Control"))
	}

	// API gets no-store
	api := httptest.NewRecorder()
	h.ServeHTTP(api, httptest.NewRequest(http.MethodGet, "/api/data", nil))
	if api.Header().Get("Cache-Control") != "no-store" || api.Header().Get("X-Frame-Options") != "DENY" {
		t.Fatalf("unexpected API headers: %v", api.Header())
	}

	// Failing health check returns 503
	failMux := http.NewServeMux()
	RegisterHealthz(failMux, func(context.Context) error { return errors.New("db down") })
	failRec := httptest.NewRecorder()
	failMux.ServeHTTP(failRec, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if failRec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 on failing healthz, got %d", failRec.Code)
	}
}

func TestRunGracefulShutdown(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	listening := make(chan string, 1)
	done := make(chan error, 1)

	mux := http.NewServeMux()
	RegisterHealthz(mux)

	go func() {
		done <- Run(ctx, ServerConfig{
			Listener: ln,
			Handler:  mux,
			OnListen: func(addr net.Addr) { listening <- addr.String() },
		})
	}()

	addr := <-listening
	resp, err := http.Get("http://" + addr + "/healthz")
	if err != nil {
		t.Fatalf("GET /healthz failed: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run returned error on graceful shutdown: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Run did not shut down promptly")
	}
}
