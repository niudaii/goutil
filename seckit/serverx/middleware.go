package serverx

import (
	"log/slog"
	"net/http"
	"path"
	"strings"
)

// DefaultHashedAssetCacheControl is applied to content-hashed static assets under /assets/.
const DefaultHashedAssetCacheControl = "public, max-age=31536000, immutable"

// SecurityHeadersConfig configures SecurityHeaders and ApplySecurityHeaders.
type SecurityHeadersConfig struct {
	CSP                     string
	ReferrerPolicy          string
	FrameOptions            string
	StrictTransportSecurity string
	HashedAssetCacheControl string
	IsHashedAsset           func(urlPath string) bool
}

var staticAssetExts = map[string]bool{
	".js":    true,
	".mjs":   true,
	".css":   true,
	".svg":   true,
	".png":   true,
	".jpg":   true,
	".jpeg":  true,
	".webp":  true,
	".gif":   true,
	".ico":   true,
	".woff":  true,
	".woff2": true,
	".ttf":   true,
	".eot":   true,
	".map":   true,
}

// DefaultIsHashedAsset reports whether urlPath points to a content-hashed asset under /assets/
// (such as Vite's index-BDD-RXTH.js or webpack's app.a1b2c3d4.js), returning false for
// unhashed assets (/assets/styles.css) and pure-digit date stamps (/assets/logo-20261127.svg).
func DefaultIsHashedAsset(urlPath string) bool {
	cleaned := path.Clean("/" + urlPath)
	if !strings.HasPrefix(cleaned, "/assets/") {
		return false
	}
	base := path.Base(cleaned)
	ext := strings.ToLower(path.Ext(base))
	if !staticAssetExts[ext] {
		return false
	}
	stem := strings.TrimSuffix(base, path.Ext(base))
	// Vite appends an 8-char base64url hash (which itself may contain '-' or '_'),
	// e.g. index-BDD-RXTH.js or index-_3zgA4i-.js.
	if len(stem) > 9 && (stem[len(stem)-9] == '-' || stem[len(stem)-9] == '.') {
		if isHashToken(stem[len(stem)-8:], false) {
			return true
		}
	}
	sepIdx := strings.LastIndexAny(stem, "-.")
	if sepIdx < 0 || sepIdx >= len(stem)-1 {
		return false
	}
	return isHashToken(stem[sepIdx+1:], true)
}

func isHashToken(hash string, hexOnly bool) bool {
	if len(hash) < 8 {
		return false
	}
	var hasLetter, hasDigit, hasUpperOrUnderscore bool
	for i := 0; i < len(hash); i++ {
		ch := hash[i]
		switch {
		case ch >= 'a' && ch <= 'f':
			hasLetter = true
		case ch >= 'a' && ch <= 'z':
			if hexOnly {
				return false
			}
			hasLetter = true
		case ch >= 'A' && ch <= 'Z':
			if hexOnly {
				return false
			}
			hasLetter = true
			hasUpperOrUnderscore = true
		case ch >= '0' && ch <= '9':
			hasDigit = true
		case ch == '_' || ch == '-':
			if hexOnly {
				return false
			}
			if ch == '_' {
				hasUpperOrUnderscore = true
			}
		default:
			return false
		}
	}
	if hexOnly {
		return hasLetter && hasDigit
	}
	return hasLetter && (hasDigit || hasUpperOrUnderscore)
}

// ApplySecurityHeaders writes standard security headers and differentiates Cache-Control
// between immutable content-hashed assets and no-store HTML/API responses.
func ApplySecurityHeaders(w http.ResponseWriter, r *http.Request, cfg SecurityHeadersConfig) {
	w.Header().Set("X-Content-Type-Options", "nosniff")

	refPolicy := cfg.ReferrerPolicy
	if refPolicy == "" {
		refPolicy = "no-referrer"
	}
	w.Header().Set("Referrer-Policy", refPolicy)

	if cfg.CSP != "" {
		w.Header().Set("Content-Security-Policy", cfg.CSP)
	}
	if cfg.FrameOptions != "" {
		w.Header().Set("X-Frame-Options", cfg.FrameOptions)
	}
	if cfg.StrictTransportSecurity != "" {
		w.Header().Set("Strict-Transport-Security", cfg.StrictTransportSecurity)
	}

	isHashed := cfg.IsHashedAsset
	if isHashed == nil {
		isHashed = DefaultIsHashedAsset
	}
	if r != nil && isHashed(r.URL.Path) {
		cc := cfg.HashedAssetCacheControl
		if cc == "" {
			cc = DefaultHashedAssetCacheControl
		}
		w.Header().Set("Cache-Control", cc)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
}

// SecurityHeaders wraps next to apply ApplySecurityHeaders on every response.
func SecurityHeaders(cfg SecurityHeadersConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ApplySecurityHeaders(w, r, cfg)
			next.ServeHTTP(w, r)
		})
	}
}

// Recover catches panics in next, logs them with logger, and returns HTTP 500 JSON.
func Recover(logger *slog.Logger) func(http.Handler) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					logger.Error("HTTP handler panic", "method", r.Method, "path", r.URL.Path, "panic", rec)
					w.Header().Set("Content-Type", "application/json; charset=utf-8")
					w.WriteHeader(http.StatusInternalServerError)
					_, _ = w.Write([]byte(`{"message":"服务器内部错误，请稍后重试"}`))
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
