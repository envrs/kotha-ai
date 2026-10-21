package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// shareRequest is the body for POST /v1/sessions/{session_id}/share.
type shareRequest struct {
	ExpiresInMinutes int `json:"expires_in_minutes,omitempty"` // 0 = never expires
}

type shareResponse struct {
	ID        string    `json:"id"`
	SessionID string    `json:"session_id"`
	URL       string    `json:"url"`
	ExpiresAt *string   `json:"expires_at,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// handleShare creates a shareable link for a session.
func (s *Server) handleShare(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("session_id")
	if sessionID == "" {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing session_id"})
		return
	}
	if _, err := s.app.Sessions.Get(r.Context(), sessionID); err != nil {
		s.writeJSON(w, http.StatusNotFound, map[string]string{"error": "session not found"})
		return
	}
	var req shareRequest
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}
	share := newShare(sessionID, req.ExpiresInMinutes)
	s.sharesMu.Lock()
	s.shares[share.ID] = share
	s.sharesMu.Unlock()
	s.writeJSON(w, http.StatusOK, shareResponse{
		ID:        share.ID,
		SessionID: share.SessionID,
		URL:       s.shareURL(share.ID),
		ExpiresAt: share.expiresAt(),
		CreatedAt: share.CreatedAt,
	})
}

// handleGetSharedSession returns the session referenced by a share link.
func (s *Server) handleGetSharedSession(w http.ResponseWriter, r *http.Request) {
	shareID := r.PathValue("share_id")
	share, ok := s.getShare(shareID)
	if !ok {
		s.writeJSON(w, http.StatusNotFound, map[string]string{"error": "share link not found or expired"})
		return
	}
	sess, err := s.app.Sessions.Get(r.Context(), share.SessionID)
	if err != nil {
		s.writeJSON(w, http.StatusNotFound, map[string]string{"error": "session not found"})
		return
	}
	s.writeJSON(w, http.StatusOK, sess)
}

func (s *Server) shareURL(shareID string) string {
	return fmt.Sprintf("/v1/share/%s", shareID)
}

func (s *Server) getShare(shareID string) (*share, bool) {
	s.sharesMu.RLock()
	defer s.sharesMu.RUnlock()
	share, ok := s.shares[shareID]
	if !ok {
		return nil, false
	}
	if share.ExpiresInMinutes > 0 && time.Now().UTC().After(share.ExpiresAt) {
		delete(s.shares, shareID)
		return nil, false
	}
	return share, true
}

// share is an in-memory record of a shareable session link.
type share struct {
	ID                string
	SessionID         string
	CreatedAt         time.Time
	ExpiresAt         time.Time
	ExpiresInMinutes  int
}

func newShare(sessionID string, expiresInMinutes int) *share {
	s := &share{
		ID:               uuid.New().String(),
		SessionID:        sessionID,
		CreatedAt:        time.Now().UTC(),
		ExpiresInMinutes: expiresInMinutes,
	}
	if expiresInMinutes > 0 {
		s.ExpiresAt = s.CreatedAt.Add(time.Duration(expiresInMinutes) * time.Minute)
	}
	return s
}

func (sh *share) expiresAt() *string {
	if sh.ExpiresInMinutes <= 0 {
		return nil
	}
	t := sh.ExpiresAt.Format(time.RFC3339)
	return &t
}