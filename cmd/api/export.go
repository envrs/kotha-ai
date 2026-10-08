package api

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os"

	"github.com/spf13/cobra"
)

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export a session as JSON or text",
	Long: `Export a session's conversation as JSON (default) or a plain-text transcript.

Examples:
  kotha api export --session my-session-id
  kotha api export --session my-session-id --format text`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if apiSession == "" {
			return fmt.Errorf("--session is required")
		}
		if apiFormat != "json" && apiFormat != "text" {
			return fmt.Errorf("--format must be json or text, got %q", apiFormat)
		}
		c := NewClient(serverFlag(cmd))
		path := "/v1/sessions/" + url.PathEscape(apiSession) + "/export?format=" + url.QueryEscape(apiFormat)
		return writeExport(context.Background(), c, path, os.Stdout)
	},
}

// writeExport fetches an export and writes the raw body to out.
func writeExport(ctx context.Context, c *Client, path string, out io.Writer) error {
	data, err := c.doBytes(ctx, "GET", path, nil)
	if err != nil {
		return err
	}
	_, err = out.Write(data)
	return err
}

func init() {
	exportCmd.Flags().StringVar(&apiSession, "session", "", "session ID (required)")
	exportCmd.Flags().StringVar(&apiFormat, "format", "json", "output format: json or text")
}
