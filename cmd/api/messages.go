package api

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

type message struct {
	ID  string `json:"id"`
	Role string `json:"role"`
}

var messagesCmd = &cobra.Command{
	Use:   "messages",
	Short: "List messages for a session",
	RunE: func(cmd *cobra.Command, args []string) error {
		c := NewClient(serverFlag(cmd))
		var out []message
		if err := c.GET(context.Background(), "/v1/sessions/"+apiSession+"/messages", &out); err != nil {
			return err
		}
		b, _ := json.MarshalIndent(out, "", "  ")
		fmt.Println(string(b))
		return nil
	},
}

func init() {
	messagesCmd.Flags().StringVar(&apiSession, "session", "", "session ID (required)")
}