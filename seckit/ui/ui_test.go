package ui

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestAssetsAndNavRoutes(t *testing.T) {
	resetNavCaches()
	t.Cleanup(resetNavCaches)

	defaults := []NavItem{
		{Key: "intel", Label: "Intel", URL: "https://intel.example.test"},
		{Key: "cves", Label: "CVEs", URL: "https://cves.example.test"},
		{Key: "unsafe", Label: "Bad", URL: "javascript:alert(1)"},
	}
	mux := http.NewServeMux()
	RegisterRoutes(mux, Config{Current: "cves", Defaults: defaults})

	for _, tc := range []struct {
		method      string
		path        string
		contentType string
		markers     []string
	}{
		{http.MethodGet, TokensCSSPath, "text/css", []string{"--primary: #2b59d9", "--primary-btn:", "--primary-contrast:", "--soft: #edf1f7"}},
		{http.MethodHead, TokensCSSPath, "text/css", nil},
		{http.MethodGet, ShellJSPath, "application/javascript", []string{"global.SecOps", "enhanceSearchInput", "copyIconSVG"}},
		{http.MethodHead, ShellJSPath, "application/javascript", nil},
		{http.MethodGet, NavJSONPath, "application/json", []string{`"current":"cves"`}},
		{http.MethodHead, NavJSONPath, "application/json", nil},
	} {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest(tc.method, tc.path, nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("%s %s status=%d", tc.method, tc.path, rec.Code)
		}
		if !strings.Contains(rec.Header().Get("Content-Type"), tc.contentType) {
			t.Fatalf("%s %s content-type=%q", tc.method, tc.path, rec.Header().Get("Content-Type"))
		}
		if rec.Header().Get("Cache-Control") != "no-store" || rec.Header().Get("X-Content-Type-Options") != "nosniff" {
			t.Fatalf("%s %s missing cache/nosniff headers: %+v", tc.method, tc.path, rec.Header())
		}
		if tc.method == http.MethodHead && rec.Body.Len() != 0 {
			t.Fatalf("HEAD %s returned body length %d", tc.path, rec.Body.Len())
		}
		for _, marker := range tc.markers {
			if !strings.Contains(rec.Body.String(), marker) {
				t.Fatalf("%s body missing %q", tc.path, marker)
			}
		}
	}

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, NavJSONPath, nil))
	var nav NavResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &nav); err != nil {
		t.Fatal(err)
	}
	if nav.Current != "cves" || len(nav.Items) != 2 {
		t.Fatalf("unexpected nav response (unsafe URL should be dropped): %+v", nav)
	}
}

func TestResolveNavEnvOverrides(t *testing.T) {
	resetNavCaches()
	t.Cleanup(resetNavCaches)

	defaults := []NavItem{
		{Key: "intel", Label: "Intel", URL: "https://intel.example.test"},
		{Key: "cves", Label: "CVEs", URL: "https://cves.example.test"},
	}

	t.Setenv("SECOPS_NAV_INTEL_URL", "https://custom-intel.example.test")
	t.Setenv("SECOPS_NAV_CVES_URL", "-")
	items := ResolveNav(defaults)
	if len(items) != 1 || items[0].Key != "intel" || items[0].URL != "https://custom-intel.example.test" {
		t.Fatalf("per-key override failed: %+v", items)
	}

	t.Setenv("SECOPS_NAV_JSON", `[{"key":"news","label":"News","url":"/news"},{"key":"bad","label":"Bad","url":"//evil.example.test"}]`)
	jsonItems := ResolveNav(defaults)
	if len(jsonItems) != 1 || jsonItems[0].Key != "news" || jsonItems[0].URL != "/news" {
		t.Fatalf("SECOPS_NAV_JSON override failed: %+v", jsonItems)
	}

	t.Setenv("SECOPS_NAV_JSON", `{"items":[{"key":"os","label":"OS","url":"https://os.example.test"}]}`)
	wrappedItems := ResolveNav(defaults)
	if len(wrappedItems) != 1 || wrappedItems[0].Key != "os" {
		t.Fatalf("SECOPS_NAV_JSON wrapper override failed: %+v", wrappedItems)
	}
}

func TestResolveNavFileHotReload(t *testing.T) {
	resetNavCaches()
	t.Cleanup(resetNavCaches)

	dir := t.TempDir()
	navPath := filepath.Join(dir, "nav.json")
	writeFile := func(content string, mod time.Time) {
		t.Helper()
		if err := os.WriteFile(navPath, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(navPath, mod, mod); err != nil {
			t.Fatal(err)
		}
	}

	t1 := time.Date(2026, 11, 28, 8, 0, 0, 0, time.UTC)
	writeFile(`[{"key":"intel","label":"Intel V1","url":"https://intel-v1.example.test"}]`, t1)
	t.Setenv("SECOPS_NAV_FILE", navPath)

	defaults := []NavItem{{Key: "fallback", Label: "Fallback", URL: "https://fallback.example.test"}}
	v1 := ResolveNav(defaults)
	if len(v1) != 1 || v1[0].Label != "Intel V1" {
		t.Fatalf("expected v1 from SECOPS_NAV_FILE, got %+v", v1)
	}

	// Rewrite file with updated mtime -> immediate hot reload without restart.
	t2 := t1.Add(2 * time.Second)
	writeFile(`{"items":[{"key":"intel","label":"Intel V2","url":"https://intel-v2.example.test"},{"key":"bad","label":"Bad","url":"javascript:alert(1)"}]}`, t2)
	v2 := ResolveNav(defaults)
	if len(v2) != 1 || v2[0].Label != "Intel V2" || v2[0].URL != "https://intel-v2.example.test" {
		t.Fatalf("expected hot-reloaded v2 from SECOPS_NAV_FILE, got %+v", v2)
	}

	// Corrupt file temporarily -> stale-while-error fallback keeps last known good v2.
	t3 := t2.Add(2 * time.Second)
	writeFile(`{not-json`, t3)
	stale := ResolveNav(defaults)
	if len(stale) != 1 || stale[0].Label != "Intel V2" {
		t.Fatalf("expected stale v2 fallback on corrupt SECOPS_NAV_FILE, got %+v", stale)
	}
}

func TestResolveNavURLRealTimeAndFallback(t *testing.T) {
	resetNavCaches()
	t.Cleanup(resetNavCaches)

	var version atomic.Int32
	version.Store(1)
	var failUpstream atomic.Bool
	var hits atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		if failUpstream.Load() {
			http.Error(w, "upstream down", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if version.Load() == 1 {
			_, _ = w.Write([]byte(`{"items":[{"key":"intel","label":"Live V1","url":"https://v1.example.test"},{"key":"bad","label":"Bad","url":"javascript:alert(1)"}]}`))
		} else {
			_, _ = w.Write([]byte(`{"items":[{"key":"intel","label":"Live V2","url":"https://v2.example.test"}]}`))
		}
	}))
	defer srv.Close()

	now := time.Date(2026, 11, 28, 8, 0, 0, 0, time.UTC)
	navNow = func() time.Time { return now }

	t.Setenv("SECOPS_NAV_URL", srv.URL)
	t.Setenv("SECOPS_NAV_CACHE_TTL", "10s")
	defaults := []NavItem{{Key: "fallback", Label: "Fallback", URL: "https://fallback.example.test"}}

	// Initial fetch gets V1 and drops unsafe URL.
	got1 := ResolveNav(defaults)
	if len(got1) != 1 || got1[0].Label != "Live V1" {
		t.Fatalf("expected Live V1, got %+v", got1)
	}
	if hits.Load() != 1 {
		t.Fatalf("expected 1 upstream hit, got %d", hits.Load())
	}

	// Upstream changes to V2, within TTL returns cached V1 without upstream hit.
	version.Store(2)
	now = now.Add(5 * time.Second)
	gotCached := ResolveNav(defaults)
	if len(gotCached) != 1 || gotCached[0].Label != "Live V1" || hits.Load() != 1 {
		t.Fatalf("expected cached Live V1 within TTL, got %+v (hits=%d)", gotCached, hits.Load())
	}

	// Advance past TTL -> fetches Live V2.
	now = now.Add(10 * time.Second)
	got2 := ResolveNav(defaults)
	if len(got2) != 1 || got2[0].Label != "Live V2" || hits.Load() != 2 {
		t.Fatalf("expected Live V2 after TTL expiry, got %+v (hits=%d)", got2, hits.Load())
	}

	// Expire TTL again while upstream returns 500 -> returns stale Live V2.
	failUpstream.Store(true)
	now = now.Add(15 * time.Second)
	gotStale := ResolveNav(defaults)
	if len(gotStale) != 1 || gotStale[0].Label != "Live V2" || hits.Load() != 3 {
		t.Fatalf("expected stale Live V2 fallback on 500, got %+v (hits=%d)", gotStale, hits.Load())
	}
}

func TestResolveNavURLHTMLHostedJSON(t *testing.T) {
	resetNavCaches()
	t.Cleanup(resetNavCaches)

	var mode atomic.Int32
	mode.Store(1)
	var sawAccept atomic.Value

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawAccept.Store(r.Header.Get("Accept"))
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if mode.Load() == 1 {
			// Static nav.json.html storing BOM + raw JSON array under text/html Content-Type
			_, _ = w.Write([]byte("\xef\xbb\xbf" + `[{"key":"intel","label":"HTML Raw JSON","url":"https://raw-html.example.test"}]`))
			return
		}
		// Static nav.json.html wrapping embedded <script type="application/json">
		_, _ = w.Write([]byte(`<!doctype html><html><head><script>var ignore=1;</script><script id="secops-nav" type="application/json">` + "\xef\xbb\xbf" + `{"items":[{"key":"cves","label":"Embedded JSON","url":"https://embedded.example.test"}]}</script></head><body></body></html>`))
	}))
	defer srv.Close()

	now := time.Date(2026, 11, 28, 9, 0, 0, 0, time.UTC)
	navNow = func() time.Time { return now }

	t.Setenv("SECOPS_NAV_URL", srv.URL+"/nav.json.html")
	t.Setenv("SECOPS_NAV_CACHE_TTL", "5s")
	defaults := []NavItem{{Key: "fallback", Label: "Fallback", URL: "https://fallback.example.test"}}

	gotRaw := ResolveNav(defaults)
	if len(gotRaw) != 1 || gotRaw[0].Label != "HTML Raw JSON" {
		t.Fatalf("expected raw JSON over text/html, got %+v", gotRaw)
	}
	if accept, _ := sawAccept.Load().(string); !strings.Contains(accept, "text/html") {
		t.Fatalf("expected Accept header to include text/html, got %q", accept)
	}

	mode.Store(2)
	now = now.Add(10 * time.Second)
	gotEmbedded := ResolveNav(defaults)
	if len(gotEmbedded) != 1 || gotEmbedded[0].Key != "cves" || gotEmbedded[0].Label != "Embedded JSON" {
		t.Fatalf("expected embedded <script type=application/json>, got %+v", gotEmbedded)
	}
}

func TestRenderAuthPage(t *testing.T) {
	rec := httptest.NewRecorder()
	RenderAuthPage(rec, http.StatusForbidden, AuthPageData{
		Title:      "Dashboard Login",
		Heading:    "Security Dashboard",
		Subtitle:   "SecOps · UTC+8",
		Message:    "Please sign in.",
		Error:      "Access denied for this account.",
		LoginURL:   "/auth/login",
		LoginLabel: "Sign in with SSO",
		LogoURL:    "/logo.svg",
	})
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status=%d, want 403", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{TokensCSSPath, ShellJSPath, "Security Dashboard", "Access denied for this account.", `href="/auth/login"`, "Sign in with SSO", `src="/logo.svg"`} {
		if !strings.Contains(body, want) {
			t.Fatalf("auth page missing %q: %s", want, body)
		}
	}
	if strings.Contains(body, "<style") || strings.Contains(body, "<script>") {
		t.Fatalf("auth page must not contain inline style/script tags: %s", body)
	}
	if csp := rec.Header().Get("Content-Security-Policy"); !strings.Contains(csp, "style-src 'self'") || !strings.Contains(csp, "script-src 'self'") {
		t.Fatalf("unexpected CSP header: %q", csp)
	}
}
