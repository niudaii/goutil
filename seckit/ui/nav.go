package ui

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
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

const (
	defaultNavCacheTTL = 30 * time.Second
	minNavCacheTTL     = 1 * time.Second
	navFetchTimeout    = 2 * time.Second
	maxNavPayloadBytes = 256 << 10
)

type navURLEntry struct {
	items     []NavItem
	expiresAt time.Time
}

type navFileEntry struct {
	modTime time.Time
	size    int64
	items   []NavItem
}

var (
	navEnvKeyReplacer = strings.NewReplacer("-", "_", " ", "_")
	navHTTPClient     = &http.Client{Timeout: navFetchTimeout}
	navNow            = time.Now

	navURLMu    sync.RWMutex
	navURLCache = make(map[string]navURLEntry)

	navFileMu    sync.RWMutex
	navFileCache = make(map[string]navFileEntry)
)

func resetNavCaches() {
	navURLMu.Lock()
	navURLCache = make(map[string]navURLEntry)
	navURLMu.Unlock()

	navFileMu.Lock()
	navFileCache = make(map[string]navFileEntry)
	navFileMu.Unlock()

	navNow = time.Now
}

// ResolveNav resolves suite navigation links using dynamic sources
// (SECOPS_NAV_URL, SECOPS_NAV_FILE, SECOPS_NAV_JSON) before falling back to
// defaults and optional SECOPS_NAV_<KEY>_URL environment overrides.
// Setting SECOPS_NAV_<KEY>_URL to "-" hides that item.
func ResolveNav(defaults []NavItem) []NavItem {
	return resolveNavFromEnv(defaults, os.Getenv)
}

func resolveNavFromEnv(defaults []NavItem, getenv func(string) string) []NavItem {
	if getenv == nil {
		getenv = func(string) string { return "" }
	}
	if rawURL := strings.TrimSpace(getenv("SECOPS_NAV_URL")); isSafeRemoteNavURL(rawURL) {
		ttl := parseNavCacheTTL(getenv("SECOPS_NAV_CACHE_TTL"))
		if items, ok := resolveNavFromURL(rawURL, ttl); ok {
			return items
		}
	}
	if filePath := strings.TrimSpace(getenv("SECOPS_NAV_FILE")); filePath != "" {
		if items, ok := resolveNavFromFile(filePath); ok {
			return items
		}
	}
	if raw := strings.TrimSpace(getenv("SECOPS_NAV_JSON")); raw != "" {
		if items, ok := parseNavPayload([]byte(raw)); ok {
			return items
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

func parseNavCacheTTL(raw string) time.Duration {
	v := strings.TrimSpace(raw)
	if v == "" {
		return defaultNavCacheTTL
	}
	d, err := time.ParseDuration(v)
	if err != nil || d <= 0 {
		return defaultNavCacheTTL
	}
	if d < minNavCacheTTL {
		return minNavCacheTTL
	}
	return d
}

func cloneNavItems(items []NavItem) []NavItem {
	if len(items) == 0 {
		return nil
	}
	out := make([]NavItem, len(items))
	copy(out, items)
	return out
}

func sanitizeNavItems(parsed []NavItem) []NavItem {
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

var jsonScriptTypeRE = regexp.MustCompile(`(?i)(?:^|\s)type\s*=\s*(?:"application/json(?:\s*;[^"]*)?"|'application/json(?:\s*;[^']*)?'|application/json(?:\s|$))`)

func trimBOMAndSpace(raw []byte) []byte {
	trimmed := bytes.TrimSpace(raw)
	trimmed = bytes.TrimPrefix(trimmed, []byte{0xef, 0xbb, 0xbf})
	return bytes.TrimSpace(trimmed)
}

func parseJSONNav(trimmed []byte) ([]NavItem, bool) {
	if len(trimmed) == 0 {
		return nil, false
	}
	if trimmed[0] == '[' {
		var list []NavItem
		if err := json.Unmarshal(trimmed, &list); err != nil {
			return nil, false
		}
		return sanitizeNavItems(list), true
	}
	if trimmed[0] == '{' {
		var wrapper struct {
			Items []NavItem `json:"items"`
		}
		if err := json.Unmarshal(trimmed, &wrapper); err != nil || wrapper.Items == nil {
			return nil, false
		}
		return sanitizeNavItems(wrapper.Items), true
	}
	return nil, false
}

func extractEmbeddedNavJSON(html []byte) ([]NavItem, bool) {
	lower := bytes.ToLower(html)
	offset := 0
	for offset < len(lower) {
		relStart := bytes.Index(lower[offset:], []byte("<script"))
		if relStart < 0 {
			break
		}
		tagStart := offset + relStart
		relClose := bytes.IndexByte(lower[tagStart:], '>')
		if relClose < 0 {
			break
		}
		attrs := html[tagStart+len("<script") : tagStart+relClose]
		contentStart := tagStart + relClose + 1
		relEnd := bytes.Index(lower[contentStart:], []byte("</script"))
		if relEnd < 0 {
			break
		}
		contentEnd := contentStart + relEnd
		offset = contentEnd + len("</script")
		if jsonScriptTypeRE.Match(attrs) {
			if items, ok := parseJSONNav(trimBOMAndSpace(html[contentStart:contentEnd])); ok && len(items) > 0 {
				return items, true
			}
		}
	}
	return nil, false
}

func parseNavPayload(raw []byte) ([]NavItem, bool) {
	trimmed := trimBOMAndSpace(raw)
	if len(trimmed) == 0 {
		return nil, false
	}
	if items, ok := parseJSONNav(trimmed); ok {
		return items, true
	}
	if trimmed[0] == '<' {
		return extractEmbeddedNavJSON(trimmed)
	}
	return nil, false
}

func resolveNavFromURL(rawURL string, ttl time.Duration) ([]NavItem, bool) {
	now := navNow()
	navURLMu.RLock()
	cached, hasCached := navURLCache[rawURL]
	navURLMu.RUnlock()
	if hasCached && len(cached.items) > 0 && now.Before(cached.expiresAt) {
		return cloneNavItems(cached.items), true
	}

	items, ok := fetchNavFromURL(rawURL)
	if ok && len(items) > 0 {
		navURLMu.Lock()
		navURLCache[rawURL] = navURLEntry{
			items:     cloneNavItems(items),
			expiresAt: navNow().Add(ttl),
		}
		navURLMu.Unlock()
		return cloneNavItems(items), true
	}
	if hasCached && len(cached.items) > 0 {
		return cloneNavItems(cached.items), true
	}
	return nil, false
}

func fetchNavFromURL(rawURL string) ([]NavItem, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), navFetchTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, false
	}
	req.Header.Set("Accept", "application/json, text/plain, text/html;q=0.9, */*;q=0.8")
	resp, err := navHTTPClient.Do(req)
	if err != nil {
		return nil, false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, false
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxNavPayloadBytes+1))
	if err != nil || len(body) > maxNavPayloadBytes {
		return nil, false
	}
	items, ok := parseNavPayload(body)
	if !ok || len(items) == 0 {
		return nil, false
	}
	return items, true
}

func resolveNavFromFile(path string) ([]NavItem, bool) {
	info, err := os.Stat(path)
	navFileMu.RLock()
	cached, hasCached := navFileCache[path]
	navFileMu.RUnlock()

	if err != nil || info.IsDir() {
		if hasCached && len(cached.items) > 0 {
			return cloneNavItems(cached.items), true
		}
		return nil, false
	}
	if hasCached && len(cached.items) > 0 && cached.modTime.Equal(info.ModTime()) && cached.size == info.Size() {
		return cloneNavItems(cached.items), true
	}

	f, err := os.Open(path)
	if err != nil {
		if hasCached && len(cached.items) > 0 {
			return cloneNavItems(cached.items), true
		}
		return nil, false
	}
	body, readErr := io.ReadAll(io.LimitReader(f, maxNavPayloadBytes+1))
	_ = f.Close()
	if readErr != nil || len(body) > maxNavPayloadBytes {
		if hasCached && len(cached.items) > 0 {
			return cloneNavItems(cached.items), true
		}
		return nil, false
	}
	items, ok := parseNavPayload(body)
	if !ok || len(items) == 0 {
		if hasCached && len(cached.items) > 0 {
			return cloneNavItems(cached.items), true
		}
		return nil, false
	}

	navFileMu.Lock()
	navFileCache[path] = navFileEntry{
		modTime: info.ModTime(),
		size:    info.Size(),
		items:   cloneNavItems(items),
	}
	navFileMu.Unlock()
	return cloneNavItems(items), true
}

func lookupEnvValue(getenv func(string) string, key string) (string, bool) {
	v := strings.TrimSpace(getenv(key))
	if v == "" {
		return "", false
	}
	return v, true
}

func isSafeRemoteNavURL(raw string) bool {
	if raw == "" {
		return false
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return false
	}
	scheme := strings.ToLower(parsed.Scheme)
	return (scheme == "http" || scheme == "https") && parsed.Host != ""
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
