package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kothagpt/kotha/internal/app"
	"github.com/kothagpt/kotha/internal/config"
	"github.com/kothagpt/kotha/internal/llm/agent"
	"github.com/kothagpt/kotha/internal/llm/models"
	"github.com/kothagpt/kotha/internal/message"
	"github.com/kothagpt/kotha/internal/permission"
	"github.com/kothagpt/kotha/internal/pubsub"
	"github.com/kothagpt/kotha/internal/session"
)

type stubSessionSvc struct{ session.Service }
type stubMessageSvc struct{ message.Service }
type stubPermissionSvc struct{ permission.Service }

type stubAgent struct {
	events chan agent.AgentEvent
}

func (s *stubAgent) Subscribe(ctx context.Context) <-chan pubsub.Event[agent.AgentEvent] { return nil }
func (s *stubAgent) Model() models.Model                                                 { return models.Model{ID: "stub"} }
func (s *stubAgent) Run(ctx context.Context, sessionID, content string, attachments ...message.Attachment) (<-chan agent.AgentEvent, error) {
	return s.events, nil
}
func (s *stubAgent) Cancel(sessionID string)             {}
func (s *stubAgent) IsSessionBusy(sessionID string) bool { return false }
func (s *stubAgent) IsBusy() bool                        { return false }
func (s *stubAgent) Update(_ config.AgentName, _ models.ModelID) (models.Model, error) {
	return models.Model{}, nil
}
func (s *stubAgent) Summarize(ctx context.Context, sessionID string) error { return nil }

func newTestServer(t *testing.T) *Server {
	t.Helper()
	events := make(chan agent.AgentEvent, 4)
	events <- agent.AgentEvent{Type: agent.AgentEventTypeResponse, Done: true}
	close(events)

	srv := &Server{
		cfg: DefaultServerConfig(),
		app: &app.App{
			Sessions:    stubSessionSvc{},
			Messages:    stubMessageSvc{},
			Permissions: stubPermissionSvc{},
			CoderAgent:  &stubAgent{events: events},
		},
	}
	return srv
}

func TestHandleAskStream(t *testing.T) {
	srv := newTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/v1/sessions/s1/ask/stream", strings.NewReader(`{"prompt":"hello"}`))
	req.SetPathValue("session_id", "s1")
	rec := httptest.NewRecorder()
	srv.handleAskStream(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%q", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "text/event-stream") {
		t.Fatalf("expected text/event-stream, got %q", ct)
	}
	if !strings.Contains(rec.Body.String(), `"done":true`) {
		t.Fatalf("expected done event, got %q", rec.Body.String())
	}
}

func TestHandleAskStreamEmptyPrompt(t *testing.T) {
	srv := newTestServer(t)
	req := httptest.NewRequest(http.MethodPost, "/v1/sessions/s1/ask/stream", strings.NewReader(`{"prompt":"   "}`))
	req.SetPathValue("session_id", "s1")
	rec := httptest.NewRecorder()
	srv.handleAskStream(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHandleAskStreamBadJSON(t *testing.T) {
	srv := newTestServer(t)
	req := httptest.NewRequest(http.MethodPost, "/v1/sessions/s1/ask/stream", strings.NewReader("not json"))
	req.SetPathValue("session_id", "s1")
	rec := httptest.NewRecorder()
	srv.handleAskStream(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}
