package ui

import (
	"bytes"
	"html/template"
	"net/http"
)

var authTemplate = template.Must(template.ParseFS(assets, "auth.html"))

// DefaultAuthCSP enforces strict same-origin CSS/JS/img loading with no inline scripts or styles.
const DefaultAuthCSP = "default-src 'none'; style-src 'self'; script-src 'self'; img-src 'self'; base-uri 'none'; form-action 'self'; frame-ancestors 'none'"

// AuthPageData fills the unified AuthGate login page template.
type AuthPageData struct {
	Title        string
	Heading      string
	Subtitle     string
	Message      string
	Error        string
	LoginURL     string
	LoginLabel   string
	LogoURL      string
	TokensCSSURL string
	ShellJSURL   string
}

// RenderAuthPage writes the unified AuthGate login page with strict security headers.
func RenderAuthPage(w http.ResponseWriter, status int, data AuthPageData) {
	if status <= 0 {
		status = http.StatusOK
	}
	if data.TokensCSSURL == "" {
		data.TokensCSSURL = TokensCSSPath
	}
	if data.ShellJSURL == "" {
		data.ShellJSURL = ShellJSPath
	}
	if data.Heading == "" {
		data.Heading = "安全运营平台"
	}
	if data.Title == "" {
		data.Title = data.Heading + " · 登录"
	}
	if data.LoginURL == "" {
		data.LoginURL = "/api/auth/oidc/login"
	}
	if data.LoginLabel == "" {
		data.LoginLabel = "登录"
	}

	var buf bytes.Buffer
	if err := authTemplate.Execute(&buf, data); err != nil {
		http.Error(w, "login page unavailable", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("Referrer-Policy", "no-referrer")
	if w.Header().Get("Content-Security-Policy") == "" {
		w.Header().Set("Content-Security-Policy", DefaultAuthCSP)
	}
	w.WriteHeader(status)
	_, _ = buf.WriteTo(w)
}
