package api

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

type askRequest struct {
	Prompt      string   `json:"prompt"`
	Attachments []string `json:"attachments,omitempty"`
	AutoApprove bool     `json:"auto_approve,omitempty"`
}

type askResult struct {
	Message struct {
		ID   string `json:"id"`
		Role string `json:"role"`
	} `json:"message"`
}

var askStream bool

var askCmd = &cobra.Command{
	Use:   "ask",
	Short: "Send a prompt to a session and stream the response",
	RunE: func(cmd *cobra.Command, args []string) error {
		c := NewClient(serverFlag(cmd))
		ctx := context.Background()
		if apiTimeout > 0 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, apiTimeout)
			defer cancel()
		}
		req := askRequest{Prompt: apiPrompt, Attachments: apiAttachments, AutoApprove: apiAutoApprove}
		if askStream {
			return c.AskStream(ctx, apiSession, req, os.Stdout)
		}
		var out askResult
		if err := c.POST(ctx, "/v1/sessions/"+apiSession+"/ask", req, &out); err != nil {
			return err
		}
		b, _ := json.MarshalIndent(out, "", "  ")
		fmt.Println(string(b))
		return nil
	},
}

func init() {
	askCmd.Flags().StringVar(&apiSession, "session", "", "session ID (required)")
	askCmd.Flags().StringVar(&apiPrompt, "prompt", "", "prompt text (required)")
	askCmd.Flags().StringSliceVar(&apiAttachments, "attachment", nil, "file attachments")
	askCmd.Flags().BoolVar(&apiAutoApprove, "auto-approve", false, "auto-approve permissions")
	askCmd.Flags().DurationVar(&apiTimeout, "timeout", 0, "request timeout")
	askCmd.Flags().BoolVar(&askStream, "stream", false, "stream agent events as Server-Sent Events")
}

// AskStream posts to /v1/sessions/{session_id}/ask/stream and writes the
// decoded Server-Sent Events to the provided writer.
func (c *Client) AskStream(ctx context.Context, sessionID string, req askRequest, out io.Writer) error {
	body, err := json.Marshal(req)
	if err != nil {
		return err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/sessions/"+sessionID+"/ask/stream", strings.NewReader(string(body)))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")
	if apiAPIKey != "" {
		httpReq.Header.Set("X-API-Key", apiAPIKey)
	}
	resp, err := c.http.Do(httpReq)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 400 {
		data, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("server %s: %s", resp.Status, strings.TrimSpace(string(data)))
	}
	w := bufio.NewWriter(out)
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		_, _ = w.WriteString(line + "\n")
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return w.Flush()
}
