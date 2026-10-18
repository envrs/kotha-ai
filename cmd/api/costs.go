package api

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

var costsCmd = &cobra.Command{
	Use:   "costs",
	Short: "Show aggregate cost and token usage",
	RunE: func(cmd *cobra.Command, args []string) error {
		c := NewClient(serverFlag(cmd))
		var out any
		if err := c.GET(context.Background(), "/v1/costs", &out); err != nil {
			return err
		}
		b, _ := json.MarshalIndent(out, "", "  ")
		fmt.Println(string(b))
		return nil
	},
}

func init() {
}