package server

import (
	"context"
	"net/http"
	"time"

	"github.com/kothagpt/kotha/internal/session"
)

// costResponse is the body for GET /v1/costs.
type costResponse struct {
	TotalCost           float64   `json:"total_cost"`
	TotalPromptTokens   int64     `json:"total_prompt_tokens"`
	TotalCompletionTokens int64   `json:"total_completion_tokens"`
	SessionCount        int64     `json:"session_count"`
	ByDay               []dayCost `json:"by_day"`
}

type dayCost struct {
	Date string  `json:"date"`
	Cost float64 `json:"cost"`
}

// handleCost returns aggregate cost and token usage across all sessions.
func (s *Server) handleCost(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	sessions, err := s.app.Sessions.List(ctx)
	if err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	resp := costResponse{}
	byDay := map[string]*dayCost{}
	for _, sess := range sessions {
		resp.TotalCost += sess.Cost
		resp.TotalPromptTokens += sess.PromptTokens
		resp.TotalCompletionTokens += sess.CompletionTokens
		resp.SessionCount++
		if sess.CreatedAt > 0 {
			t := time.UnixMilli(sess.CreatedAt).UTC().Format("2006-01-02")
			dc := byDay[t]
			if dc == nil {
				dc = &dayCost{Date: t}
				byDay[t] = dc
			}
			dc.Cost += sess.Cost
		}
	}
	for _, v := range byDay {
		resp.ByDay = append(resp.ByDay, *v)
	}
	s.writeJSON(w, http.StatusOK, resp)
}

var _ = session.Session{}