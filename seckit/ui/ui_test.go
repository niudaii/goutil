package ui

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAssetsAndNavRoutes(t *testing.T) {
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
		marker      string
	}{
		{http.MethodGet, TokensCSSPath, "text/css", "--primary: #3867e8"},
		{http.MethodHead, TokensCSSPath, "text/css", ""},
		{http.MethodGet, ShellJSPath, "application/javascript", "global.SecOps"},
		{http.MethodHead, ShellJSPath, "application/javascript", ""},
		{http.MethodGet, NavJSONPath, "application/json", `"current":"cves"`},
		{http.MethodHead, NavJSONPath, "application/json", ""},
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
		if tc.marker != "" && !strings.Contains(rec.Body.String(), tc.marker) {
			t.Fatalf("%s body missing %q", tc.path, tc.marker)
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
