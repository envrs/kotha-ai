package api

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

type session struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

var sessionsCmd = &cobra.Command{
	Use:   "sessions",
	Short: "List sessions",
	RunE: func(cmd *cobra.Command, args []string) error {
		c := NewClient(serverFlag(cmd))
		var out []session
		if err := c.GET(context.Background(), "/v1/sessions", &out); err != nil {
			return err
		}
		b, _ := json.MarshalIndent(out, "", "  ")
		fmt.Println(string(b))
		return nil
	},
}

var sessionCmd = &cobra.Command{
	Use:   "session",
	Short: "Get a single session",
	RunE: func(cmd *cobra.Command, args []string) error {
		c := NewClient(serverFlag(cmd))
		var out session
		if err := c.GET(context.Background(), "/v1/sessions/"+apiSession, &out); err != nil {
			return err
		}
		b, _ := json.MarshalIndent(out, "", "  ")
		fmt.Println(string(b))
		return nil
	},
}

func init() {
	sessionCmd.Flags().StringVar(&apiSession, "session", "", "session ID (required)")
}
