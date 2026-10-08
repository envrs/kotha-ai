package server

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/kothagpt/kotha/internal/config"
	"github.com/kothagpt/kotha/internal/llm/agent"
	"github.com/kothagpt/kotha/internal/llm/models"
	"github.com/kothagpt/kotha/internal/session"
)

// askRequest mirrors the OpenAPI AskRequest schema.
type askRequest struct {
	Prompt        string   `json:"prompt"`
	Attachments   []string `json:"attachments,omitempty"`
	AutoApprove   bool     `json:"auto_approve,omitempty"`
}

// askResult mirrors the OpenAPI AskResult schema.
type askResult struct {
	Message struct {
		ID   string `json:"id"`
		Role string `json:"role"`
	} `json:"message"`
}

// updateModelRequest mirrors the OpenAPI UpdateModelRequest schema.
type updateModelRequest struct {
	ModelID string `json:"model_id"`
}

func (s *Server) writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if v != nil {
		_ = json.NewEncoder(w).Encode(v)
	}
}

func (s *Server) readJSON(w http.ResponseWriter, r *http.Request, v any) error {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "failed to read body"})
		return err
	}
	if len(body) == 0 {
		return nil
	}
	if err := json.Unmarshal(body, v); err != nil {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		return err
	}
	return nil
}

func (s *Server) sessionID(r *http.Request) string {
	return r.PathValue("session_id")
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	s.writeJSON(w, http.StatusOK, map[string]any{
		"status":    "ok",
		"uptime":   s.Uptime().String(),
		"addr":     s.Addr(),
	})
}

func (s *Server) handleStop(w http.ResponseWriter, r *http.Request) {
	go func() {
		time.Sleep(100 * time.Millisecond)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = s.Stop(ctx)
	}()
	s.writeJSON(w, http.StatusOK, map[string]string{"status": "stopping"})
}

func (s *Server) handleListSessions(w http.ResponseWriter, r *http.Request) {
	sessions, err := s.app.Sessions.List(r.Context())
	if err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	s.writeJSON(w, http.StatusOK, sessions)
}

func (s *Server) handleCreateSession(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Title string `json:"title"`
	}
	if err := s.readJSON(w, r, &body); err != nil {
		return
	}
	sess, err := s.app.Sessions.Create(r.Context(), body.Title)
	if err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	s.writeJSON(w, http.StatusOK, sess)
}

func (s *Server) handleGetSession(w http.ResponseWriter, r *http.Request) {
	sess, err := s.app.Sessions.Get(r.Context(), s.sessionID(r))
	if err != nil {
		s.writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}
	s.writeJSON(w, http.StatusOK, sess)
}

func (s *Server) handleDeleteSession(w http.ResponseWriter, r *http.Request) {
	if err := s.app.Sessions.Delete(r.Context(), s.sessionID(r)); err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleListMessages(w http.ResponseWriter, r *http.Request) {
	msgs, err := s.app.Messages.List(r.Context(), s.sessionID(r))
	if err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	s.writeJSON(w, http.StatusOK, msgs)
}

// handleExportSession serves GET /v1/sessions/{session_id}/export in
// JSON (default) or plain text via ?format=.
func (s *Server) handleExportSession(w http.ResponseWriter, r *http.Request) {
	id := s.sessionID(r)
	sess, err := s.app.Sessions.Get(r.Context(), id)
	if err != nil {
		s.writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}
	msgs, err := s.app.Messages.List(r.Context(), id)
	if err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	exp := session.NewExport(sess, msgs)
	switch format := r.URL.Query().Get("format"); format {
	case "", "json":
		b, err := exp.JSON()
		if err != nil {
			s.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(b)
	case "text":
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, exp.Text())
	default:
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "format must be json or text"})
	}
}

func (s *Server) handleCurrentModel(w http.ResponseWriter, r *http.Request) {
	m := s.app.CoderAgent.Model()
	s.writeJSON(w, http.StatusOK, m)
}

func (s *Server) handleUpdateModel(w http.ResponseWriter, r *http.Request) {
	var body updateModelRequest
	if err := s.readJSON(w, r, &body); err != nil {
		return
	}
	m, err := s.app.CoderAgent.Update(config.AgentCoder, models.ModelID(body.ModelID))
	if err != nil {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	s.writeJSON(w, http.StatusOK, m)
}

func (s *Server) handleCancel(w http.ResponseWriter, r *http.Request) {
	s.app.CoderAgent.Cancel(s.sessionID(r))
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleSummarize(w http.ResponseWriter, r *http.Request) {
	if err := s.app.CoderAgent.Summarize(r.Context(), s.sessionID(r)); err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAsk(w http.ResponseWriter, r *http.Request) {
	var body askRequest
	if err := s.readJSON(w, r, &body); err != nil {
		return
	}
	if strings.TrimSpace(body.Prompt) == "" {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "prompt is required"})
		return
	}
	sid := s.sessionID(r)
	if body.AutoApprove {
		s.app.Permissions.AutoApproveSession(sid)
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Minute)
	defer cancel()
	events, err := s.app.CoderAgent.Run(ctx, sid, body.Prompt)
	if err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	var final agent.AgentEvent
	for ev := range events {
		final = ev
		if ev.Error != nil {
			s.writeJSON(w, http.StatusInternalServerError, map[string]any{"error": ev.Error.Error(), "message": ev.Message})
			return
		}
	}
	result := askResult{}
	result.Message.ID = final.Message.ID
	result.Message.Role = string(final.Message.Role)
	s.writeJSON(w, http.StatusOK, result)
}
