package api

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

var cancelCmd = &cobra.Command{
	Use:   "cancel",
	Short: "Cancel in-flight work for a session",
	RunE: func(cmd *cobra.Command, args []string) error {
		c := NewClient(serverFlag(cmd))
		if err := c.POST(context.Background(), "/v1/sessions/"+apiSession+"/cancel", nil, nil); err != nil {
			return err
		}
		fmt.Println("cancelled:", apiSession)
		return nil
	},
}

func init() {
	cancelCmd.Flags().StringVar(&apiSession, "session", "", "session ID (required)")
}
