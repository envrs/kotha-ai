package api

import (
	"context"
	"fmt"
	"net/url"

	"github.com/spf13/cobra"
)

var (
	recallSession string
	recallLimit   int
)

var recallCmd = &cobra.Command{
	Use:   "recall",
	Short: "Recall context from recent sessions",
	RunE: func(cmd *cobra.Command, args []string) error {
		c := NewClient(serverFlag(cmd))
		var out struct {
			Summaries []struct {
				ID    string `json:"id"`
				Title string `json:"title"`
			} `json:"summaries"`
			Files []struct {
				ID        string `json:"id"`
				SessionID string `json:"session_id"`
				Path      string `json:"path"`
				Content   string `json:"content"`
				Version   string `json:"version"`
			} `json:"files"`
		}
		path := fmt.Sprintf("/v1/recall?session_id=%s&limit=%d", url.QueryEscape(recallSession), recallLimit)
		if err := c.GET(context.Background(), path, &out); err != nil {
			return err
		}
		if len(out.Summaries) > 0 {
			for _, s := range out.Summaries {
				fmt.Printf("  %s: %s\n", s.ID, s.Title)
			}
			fmt.Println()
		}
		if len(out.Files) > 0 {
			for _, f := range out.Files {
				fmt.Printf("  %s (%s)\n", f.Path, f.Version)
			}
		}
		return nil
	},
}

func init() {
	recallCmd.Flags().StringVar(&recallSession, "session", "", "exclude this session from recall")
	recallCmd.Flags().IntVar(&recallLimit, "limit", 5, "max number of recent sessions to recall")
}
