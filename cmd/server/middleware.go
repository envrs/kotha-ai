package server

import (
	"crypto/subtle"
	"net/http"
	"strings"
)

// APIKeyAuth wraps next with API-key authentication. When key is empty
// the wrapper is a passthrough (local development default). Otherwise
// every request must present the key as "Authorization: Bearer <key>"
// or "X-API-Key: <key>". /healthz stays open so probes work without
// credentials. Comparison is constant-time.
func APIKeyAuth(key string, next http.Handler) http.Handler {
	if key == "" {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/healthz" {
			next.ServeHTTP(w, r)
			return
		}
		if !authorized(r, key) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":"unauthorized"}`))
			return
		}
		next.ServeHTTP(w, r)
	})
}

// authorized reports whether r carries the expected key.
func authorized(r *http.Request, key string) bool {
	if k := r.Header.Get("X-API-Key"); k != "" {
		return subtle.ConstantTimeCompare([]byte(k), []byte(key)) == 1
	}
	const prefix = "Bearer "
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, prefix) {
		return subtle.ConstantTimeCompare([]byte(strings.TrimPrefix(auth, prefix)), []byte(key)) == 1
	}
	return false
}
