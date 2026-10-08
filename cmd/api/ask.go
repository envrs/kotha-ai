package api

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

type askRequest struct {
	Prompt        string   `json:"prompt"`
	Attachments   []string `json:"attachments,omitempty"`
	AutoApprove   bool     `json:"auto_approve,omitempty"`
}

type askResult struct {
	Message struct {
		ID   string `json:"id"`
		Role string `json:"role"`
	} `json:"message"`
}

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
	_ = os.Stdout
}