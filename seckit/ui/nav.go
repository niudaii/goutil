package ui

import (
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"strings"
)

// NavItem describes one system entry in the top suite navigation bar.
type NavItem struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	URL   string `json:"url"`
}

// NavResponse is the JSON payload returned by NavJSONPath.
type NavResponse struct {
	Current string    `json:"current"`
	Items   []NavItem `json:"items"`
}

var navEnvKeyReplacer = strings.NewReplacer("-", "_", " ", "_")

// ResolveNav resolves suite navigation links using defaults and optional
// SECOPS_NAV_JSON or SECOPS_NAV_<KEY>_URL environment overrides.
// Setting SECOPS_NAV_<KEY>_URL to "-" hides that item.
func ResolveNav(defaults []NavItem) []NavItem {
	return resolveNavFromEnv(defaults, os.Getenv)
}

func resolveNavFromEnv(defaults []NavItem, getenv func(string) string) []NavItem {
	if getenv == nil {
		getenv = func(string) string { return "" }
	}
	if raw := strings.TrimSpace(getenv("SECOPS_NAV_JSON")); raw != "" {
		var parsed []NavItem
		if err := json.Unmarshal([]byte(raw), &parsed); err == nil {
			out := make([]NavItem, 0, len(parsed))
			for _, item := range parsed {
				item.Key = strings.TrimSpace(item.Key)
				item.Label = strings.TrimSpace(item.Label)
				item.URL = strings.TrimSpace(item.URL)
				if item.Key != "" && item.Label != "" && isSafeNavURL(item.URL) {
					out = append(out, item)
				}
			}
			return out
		}
	}

	out := make([]NavItem, 0, len(defaults))
	for _, item := range defaults {
		item.Key = strings.TrimSpace(item.Key)
		item.Label = strings.TrimSpace(item.Label)
		item.URL = strings.TrimSpace(item.URL)
		if item.Key == "" || item.Label == "" {
			continue
		}
		envKey := "SECOPS_NAV_" + navEnvKeyReplacer.Replace(strings.ToUpper(item.Key)) + "_URL"
		if override, ok := lookupEnvValue(getenv, envKey); ok {
			if override == "-" {
				continue
			}
			if isSafeNavURL(override) {
				item.URL = override
			}
		}
		if isSafeNavURL(item.URL) {
			out = append(out, item)
		}
	}
	return out
}

func lookupEnvValue(getenv func(string) string, key string) (string, bool) {
	v := strings.TrimSpace(getenv(key))
	if v == "" {
		return "", false
	}
	return v, true
}

func isSafeNavURL(raw string) bool {
	if raw == "" {
		return false
	}
	if strings.HasPrefix(raw, "/") {
		return !strings.HasPrefix(raw, "//") && !strings.ContainsAny(raw, "\r\n\t ")
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return false
	}
	scheme := strings.ToLower(parsed.Scheme)
	return (scheme == "http" || scheme == "https") && parsed.Host != ""
}

// NavHandler returns an HTTP handler serving NavResponse JSON.
func NavHandler(cfg Config) http.HandlerFunc {
	defaults := append([]NavItem(nil), cfg.Defaults...)
	current := strings.TrimSpace(cfg.Current)
	return func(w http.ResponseWriter, r *http.Request) {
		payload := NavResponse{
			Current: current,
			Items:   ResolveNav(defaults),
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Cache-Control", "no-store")
		w.WriteHeader(http.StatusOK)
		if r != nil && r.Method == http.MethodHead {
			return
		}
		_ = json.NewEncoder(w).Encode(payload)
	}
}
