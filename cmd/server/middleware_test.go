package server

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kothagpt/kotha/internal/ratelimit"
)

func authHandler() (http.Handler, *int) {
	calls := 0
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "ok")
	})
	return h, &calls
}

func doReq(t *testing.T, h http.Handler, path string, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestAPIKeyAuthDisabledPassthrough(t *testing.T) {
	h, calls := authHandler()
	wrapped := APIKeyAuth("", h)
	rec := doReq(t, wrapped, "/v1/sessions", nil)
	if rec.Code != http.StatusOK || *calls != 1 {
		t.Fatalf("code=%d calls=%d", rec.Code, *calls)
	}
}

func TestAPIKeyAuthRejectsMissingKey(t *testing.T) {
	h, calls := authHandler()
	wrapped := APIKeyAuth("secret", h)
	for _, path := range []string{"/v1/sessions", "/v1/stop"} {
		rec := doReq(t, wrapped, path, nil)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("%s code=%d, want 401", path, rec.Code)
		}
	}
	if *calls != 0 {
		t.Fatalf("handler called %d times without auth", *calls)
	}
}

func TestAPIKeyAuthRejectsWrongKey(t *testing.T) {
	h, _ := authHandler()
	wrapped := APIKeyAuth("secret", h)
	cases := []map[string]string{
		{"X-API-Key": "wrong"},
		{"Authorization": "Bearer wrong"},
		{"Authorization": "Basic dXNlcjpwYXNz"},
	}
	for _, headers := range cases {
		rec := doReq(t, wrapped, "/v1/sessions", headers)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("headers=%v code=%d, want 401", headers, rec.Code)
		}
	}
}

func TestAPIKeyAuthAcceptsBothHeaderStyles(t *testing.T) {
	h, calls := authHandler()
	wrapped := APIKeyAuth("secret", h)
	cases := []map[string]string{
		{"X-API-Key": "secret"},
		{"Authorization": "Bearer secret"},
	}
	for _, headers := range cases {
		rec := doReq(t, wrapped, "/v1/sessions", headers)
		if rec.Code != http.StatusOK {
			t.Fatalf("headers=%v code=%d, want 200", headers, rec.Code)
		}
	}
	if *calls != 2 {
		t.Fatalf("calls=%d, want 2", *calls)
	}
}

func TestAPIKeyAuthHealthzOpen(t *testing.T) {
	h, calls := authHandler()
	wrapped := APIKeyAuth("secret", h)
	rec := doReq(t, wrapped, "/healthz", nil)
	if rec.Code != http.StatusOK || *calls != 1 {
		t.Fatalf("healthz code=%d calls=%d", rec.Code, *calls)
	}
}

func TestRateLimit429AfterBudget(t *testing.T) {
	h, calls := authHandler()
	// Tiny refill rate, capacity 1: exactly one request fits the bucket.
	lims := ratelimit.NewRegistry(0.0001, 1)
	wrapped := RateLimit(lims, h)

	if rec := doReq(t, wrapped, "/v1/sessions", nil); rec.Code != http.StatusOK {
		t.Fatalf("first request code=%d, want 200", rec.Code)
	}
	rec := doReq(t, wrapped, "/v1/sessions", nil)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("second request code=%d, want 429", rec.Code)
	}
	// Non-/v1 paths bypass the limiter.
	if rec := doReq(t, wrapped, "/healthz", nil); rec.Code != http.StatusOK {
		t.Fatalf("healthz code=%d, want 200", rec.Code)
	}
	if *calls != 2 {
		t.Fatalf("handler calls=%d, want 2 (first + healthz)", *calls)
	}
}

func TestRequestLogPassthrough(t *testing.T) {
	h, calls := authHandler()
	wrapped := RequestLog(h)
	rec := doReq(t, wrapped, "/v1/sessions", nil)
	if rec.Code != http.StatusOK || *calls != 1 {
		t.Fatalf("code=%d calls=%d", rec.Code, *calls)
	}
}

func TestStatusWriterCapturesStatus(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	})
	sw := &statusWriter{ResponseWriter: httptest.NewRecorder(), status: http.StatusOK}
	inner.ServeHTTP(sw, nil)
	if sw.status != http.StatusTeapot {
		t.Fatalf("status=%d, want 418", sw.status)
	}
}
