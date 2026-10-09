package api

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check if a server is running",
	RunE: func(cmd *cobra.Command, args []string) error {
		c := NewClient(serverFlag(cmd))
		resp, err := c.http.Get(c.baseURL + "/healthz")
		if err != nil {
			fmt.Println("server not running")
			return nil
		}
		defer func() { _ = resp.Body.Close() }()
		var out map[string]any
		_ = json.NewDecoder(resp.Body).Decode(&out)
		fmt.Println("server running:", resp.Status)
		return nil
	},
}

func init() {
	statusCmd.Flags().StringVar(&apiSession, "session", "", "unused placeholder")
	_ = context.Background
}
