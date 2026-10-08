package session

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/kothagpt/kotha/internal/message"
)

// ExportedMessage is one message in a portable session export.
type ExportedMessage struct {
	Role      string `json:"role"`
	Content   string `json:"content"`
	CreatedAt int64  `json:"created_at,omitempty"`
}

// Export is a portable snapshot of a session and its conversation,
// renderable as JSON or plain text.
type Export struct {
	SessionID  string            `json:"session_id"`
	Title      string            `json:"title"`
	ExportedAt time.Time         `json:"exported_at"`
	Messages   []ExportedMessage `json:"messages"`
}

// NewExport snapshots a session and its messages.
func NewExport(sess Session, msgs []message.Message) Export {
	out := make([]ExportedMessage, len(msgs))
	for i, m := range msgs {
		out[i] = ExportedMessage{
			Role:      string(m.Role),
			Content:   messageText(m),
			CreatedAt: m.CreatedAt,
		}
	}
	return Export{
		SessionID:  sess.ID,
		Title:      sess.Title,
		ExportedAt: time.Now().UTC(),
		Messages:   out,
	}
}

// JSON renders the export as indented JSON.
func (e Export) JSON() ([]byte, error) {
	return json.MarshalIndent(e, "", "  ")
}

// Text renders the export as a human-readable transcript.
func (e Export) Text() string {
	var b strings.Builder
	title := e.Title
	if title == "" {
		title = e.SessionID
	}
	fmt.Fprintf(&b, "%s\n%s\n\n", title, strings.Repeat("=", len(title)))
	for _, m := range e.Messages {
		fmt.Fprintf(&b, "[%s]\n%s\n\n", m.Role, m.Content)
	}
	return b.String()
}

// messageText concatenates every text part of a message.
func messageText(m message.Message) string {
	var b strings.Builder
	for _, part := range m.Parts {
		if tc, ok := part.(message.TextContent); ok {
			if b.Len() > 0 {
				b.WriteString("\n")
			}
			b.WriteString(tc.Text)
		}
	}
	return b.String()
}
