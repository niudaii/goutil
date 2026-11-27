// Package ui provides shared SecOps Slate-Blue design tokens, browser shell
// helpers, configurable suite navigation, and a unified AuthGate login page.
package ui

import (
	"embed"
	"net/http"
)

const (
	// TokensCSSPath is the HTTP path for the shared SecOps CSS variables and shell classes.
	TokensCSSPath = "/secops/tokens.css"
	// ShellJSPath is the HTTP path for the shared theme, toast, clipboard, shortcut, and nav script.
	ShellJSPath = "/secops/shell.js"
	// NavJSONPath is the HTTP path serving suite navigation links and active app key.
	NavJSONPath = "/secops/nav.json"
)

//go:embed secops-tokens.css secops-shell.js auth.html
var assets embed.FS

// Config configures shared SecOps UI assets and suite navigation.
type Config struct {
	Current  string
	Defaults []NavItem
}

// RegisterRoutes mounts /secops/tokens.css, /secops/shell.js, and /secops/nav.json on mux.
func RegisterRoutes(mux *http.ServeMux, cfg Config) {
	if mux == nil {
		return
	}
	nav := NavHandler(cfg)
	for _, route := range []struct {
		path    string
		handler http.HandlerFunc
	}{
		{TokensCSSPath, ServeTokensCSS},
		{ShellJSPath, ServeShellJS},
		{NavJSONPath, nav},
	} {
		mux.HandleFunc("GET "+route.path, route.handler)
		mux.HandleFunc("HEAD "+route.path, route.handler)
	}
}

// ServeTokensCSS serves the shared Slate-Blue design system CSS with nosniff and no-store headers.
func ServeTokensCSS(w http.ResponseWriter, r *http.Request) {
	serveEmbeddedAsset(w, r, "secops-tokens.css", "text/css; charset=utf-8")
}

// ServeShellJS serves the shared SecOps browser helper script with nosniff and no-store headers.
func ServeShellJS(w http.ResponseWriter, r *http.Request) {
	serveEmbeddedAsset(w, r, "secops-shell.js", "application/javascript; charset=utf-8")
}

func serveEmbeddedAsset(w http.ResponseWriter, r *http.Request, name, contentType string) {
	data, err := assets.ReadFile(name)
	if err != nil {
		http.Error(w, "asset unavailable", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	if r != nil && r.Method == http.MethodHead {
		return
	}
	_, _ = w.Write(data)
}
