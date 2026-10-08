package api

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

var summarizeCmd = &cobra.Command{
	Use:   "summarize",
	Short: "Trigger session summarization",
	RunE: func(cmd *cobra.Command, args []string) error {
		c := NewClient(serverFlag(cmd))
		if err := c.POST(context.Background(), "/v1/sessions/"+apiSession+"/summarize", nil, nil); err != nil {
			return err
		}
		fmt.Println("summarization started:", apiSession)
		return nil
	},
}

func init() {
	summarizeCmd.Flags().StringVar(&apiSession, "session", "", "session ID (required)")
}