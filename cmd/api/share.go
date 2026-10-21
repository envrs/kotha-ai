package api

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

type shareResponse struct {
	ID        string `json:"id"`
	SessionID string `json:"session_id"`
	URL       string `json:"url"`
}

var shareCmd = &cobra.Command{
	Use:   "share",
	Short: "Create a shareable link for a session",
	RunE: func(cmd *cobra.Command, args []string) error {
		c := NewClient(serverFlag(cmd))
		var out shareResponse
		if err := c.POST(context.Background(), "/v1/sessions/"+apiSession+"/share", nil, &out); err != nil {
			return err
		}
		b, _ := json.MarshalIndent(out, "", "  ")
		fmt.Println(string(b))
		return nil
	},
}

func init() {
	shareCmd.Flags().StringVar(&apiSession, "session", "", "session ID (required)")
}