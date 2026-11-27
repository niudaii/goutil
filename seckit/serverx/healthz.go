package serverx

import (
	"context"
	"encoding/json"
	"net/http"
)

// HealthzHandler returns an HTTP handler that responds 200 {"status":"ok"} when all checks pass,
// or 503 {"status":"error"} if any check returns a non-nil error.
func HealthzHandler(checks ...func(context.Context) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		for _, check := range checks {
			if check == nil {
				continue
			}
			if err := check(r.Context()); err != nil {
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				w.WriteHeader(http.StatusServiceUnavailable)
				if r.Method != http.MethodHead {
					_ = json.NewEncoder(w).Encode(map[string]string{"status": "error"})
				}
				return
			}
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		if r.Method != http.MethodHead {
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		}
	}
}

// HealthzNoContentHandler returns a 204 No Content /healthz handler with Cache-Control: no-store.
func HealthzNoContentHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.WriteHeader(http.StatusNoContent)
	}
}

// RegisterHealthz mounts GET and HEAD /healthz on mux using HealthzHandler.
func RegisterHealthz(mux *http.ServeMux, checks ...func(context.Context) error) {
	if mux == nil {
		return
	}
	h := HealthzHandler(checks...)
	mux.HandleFunc("GET /healthz", h)
	mux.HandleFunc("HEAD /healthz", h)
}
