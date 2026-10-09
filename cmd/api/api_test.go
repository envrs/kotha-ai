package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestClientRoundTrip(t *testing.T) {
	var got struct {
		ID    string `json:"id"`
		Title string `json:"title"`
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/sessions/s1" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"id": "s1", "title": "hi"})
	}))
	defer srv.Close()
	c := NewClient(srv.URL)
	if err := c.GET(context.Background(), "/v1/sessions/s1", &got); err != nil {
		t.Fatal(err)
	}
	if got.ID != "s1" || got.Title != "hi" {
		t.Fatalf("got %+v", got)
	}
}

func TestClientErrorStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":"not found"}`))
	}))
	defer srv.Close()
	c := NewClient(srv.URL)
	if err := c.GET(context.Background(), "/v1/sessions/x", nil); err == nil {
		t.Fatal("expected error")
	}
}

func TestServerFlagDefault(t *testing.T) {
	cmd := newTestCmd()
	if v := serverFlag(cmd); v != "http://127.0.0.1:8080" {
		t.Fatalf("default flag=%q", v)
	}
}

func TestClientSendsAPIKey(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-API-Key") != "k123" {
			t.Errorf("api key header = %q, want k123", r.Header.Get("X-API-Key"))
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"ok": "1"})
	}))
	defer srv.Close()

	old := apiAPIKey
	apiAPIKey = "k123"
	t.Cleanup(func() { apiAPIKey = old })

	c := NewClient(srv.URL)
	var out map[string]string
	if err := c.GET(context.Background(), "/v1/sessions", &out); err != nil {
		t.Fatal(err)
	}
}

func newTestCmd() *cobra.Command {
	c := &cobra.Command{Use: "test"}
	c.Flags().StringVar(&apiServer, "server", "", "server base URL")
	return c
}

var _ = fmt.Sprint

func TestAskStream(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/sessions/s1/ask/stream" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"type":"response","done":true}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	var buf strings.Builder
	req := askRequest{Prompt: "hi"}
	if err := c.AskStream(context.Background(), "s1", req, &buf); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), `"done":true`) {
		t.Fatalf("expected done event, got %q", buf.String())
	}
}

func TestAskStreamErrorStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"boom"}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	var buf strings.Builder
	if err := c.AskStream(context.Background(), "s1", askRequest{Prompt: "hi"}, &buf); err == nil {
		t.Fatal("expected error")
	}
}
