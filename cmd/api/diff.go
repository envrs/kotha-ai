package api

import (
	"context"
	"fmt"
	"net/url"

	"github.com/spf13/cobra"
)

var (
	diffPath       string
	diffOldSession string
	diffNewSession string
)

var diffCmd = &cobra.Command{
	Use:   "diff",
	Short: "Show a unified diff of a file between two sessions",
	RunE: func(cmd *cobra.Command, args []string) error {
		if diffPath == "" {
			return fmt.Errorf("--path is required")
		}
		if diffOldSession == "" || diffNewSession == "" {
			return fmt.Errorf("--old and --new session IDs are required")
		}
		c := NewClient(serverFlag(cmd))
		var out struct {
			Path       string `json:"path"`
			OldSession string `json:"old_session"`
			NewSession string `json:"new_session"`
			Diff       string `json:"diff"`
			Additions  int    `json:"additions"`
			Removals   int    `json:"removals"`
		}
		path := fmt.Sprintf("/v1/sessions/diff?path=%s&old=%s&new=%s",
			url.PathEscape(diffPath),
			url.QueryEscape(diffOldSession),
			url.QueryEscape(diffNewSession),
		)
		if err := c.GET(context.Background(), path, &out); err != nil {
			return err
		}
		fmt.Printf("--- %s\n+++ %s\n", out.OldSession, out.NewSession)
		fmt.Print(out.Diff)
		return nil
	},
}

func init() {
	diffCmd.Flags().StringVar(&diffPath, "path", "", "file path to diff (required)")
	diffCmd.Flags().StringVar(&diffOldSession, "old", "", "old session ID (required)")
	diffCmd.Flags().StringVar(&diffNewSession, "new", "", "new session ID (required)")
}
