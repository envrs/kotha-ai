package api

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop a running server (sends SIGTERM)",
	RunE: func(cmd *cobra.Command, args []string) error {
		c := NewClient(serverFlag(cmd))
		if err := c.POST(context.Background(), "/v1/stop", nil, nil); err != nil {
			return err
		}
		fmt.Println("stop requested")
		return nil
	},
}
