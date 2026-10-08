package server

import (
	"crypto/subtle"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/kothagpt/kotha/internal/logging"
	"github.com/kothagpt/kotha/internal/ratelimit"
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

// RequestLog logs method, path, status and duration for every request.
func RequestLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(sw, r)
		logging.Info("http request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", sw.status,
			"duration", time.Since(start).String(),
		)
	})
}

// RateLimit applies a per-client-IP token bucket to /v1 requests.
// Exceeding the budget returns 429 with a JSON error body.
func RateLimit(lims *ratelimit.Registry, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/v1") && !lims.Allow(clientIP(r)) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"error":"rate limited"}`))
			return
		}
		next.ServeHTTP(w, r)
	})
}

// clientIP extracts the request's client address (host part of
// RemoteAddr). Forwarded headers are deliberately not trusted.
func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// statusWriter records the response status and forwards Flush/Unwrap
// so streaming and ResponseController keep working.
type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusWriter) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (w *statusWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }
