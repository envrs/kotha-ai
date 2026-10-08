package api

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

type model struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Provider      string `json:"provider"`
	ContextWindow int64  `json:"context_window"`
	CanReason     bool   `json:"can_reason"`
}

var modelsCmd = &cobra.Command{
	Use:   "models",
	Short: "Show the current model",
	RunE: func(cmd *cobra.Command, args []string) error {
		c := NewClient(serverFlag(cmd))
		var out model
		if err := c.GET(context.Background(), "/v1/models", &out); err != nil {
			return err
		}
		b, _ := json.MarshalIndent(out, "", "  ")
		fmt.Println(string(b))
		return nil
	},
}
