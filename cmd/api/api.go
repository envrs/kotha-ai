package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var (
	apiServer      string
	apiSession     string
	apiPrompt      string
	apiAttachments []string
	apiAutoApprove bool
	apiTimeout     time.Duration
	apiFormat      string
	apiAPIKey      string
)

// Cmd is the `kotha api` subcommand that talks to a running server.
var Cmd = &cobra.Command{
	Use:   "api",
	Short: "Make a request to a running Kotha server",
	Long: `Make a request to a running Kotha server.

Subcommands: ask, cancel, summarize, sessions, session, messages, export, models, stop, status.`,
	Example: `
  kotha api ask -s http://127.0.0.1:8080 -i my-session -p "summarize this repo"
  kotha api stop -s http://127.0.0.1:8080
  kotha api status
`,
}

func init() {
	Cmd.PersistentFlags().StringVarP(&apiServer, "server", "s", "http://127.0.0.1:8080", "server base URL")
	Cmd.PersistentFlags().StringVar(&apiAPIKey, "api-key", os.Getenv("KOTHA_API_KEY"), "API key for the server (env KOTHA_API_KEY)")
	Cmd.AddCommand(askCmd, cancelCmd, summarizeCmd, sessionsCmd, sessionCmd, messagesCmd, exportCmd, modelsCmd, stopCmd, statusCmd)
}

// Client wraps an HTTP server for the api subcommand.
type Client struct {
	baseURL string
	http    *http.Client
}

// NewClient builds an API client.
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		http:    &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *Client) doBytes(ctx context.Context, method, path string, body io.Reader) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if apiAPIKey != "" {
		req.Header.Set("X-API-Key", apiAPIKey)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("server %s: %s", resp.Status, strings.TrimSpace(string(data)))
	}
	return data, nil
}

func (c *Client) do(ctx context.Context, method, path string, body io.Reader, out any) error {
	data, err := c.doBytes(ctx, method, path, body)
	if err != nil {
		return err
	}
	if out != nil && len(data) > 0 {
		return json.Unmarshal(data, out)
	}
	return nil
}

func (c *Client) GET(ctx context.Context, path string, out any) error {
	return c.do(ctx, http.MethodGet, path, nil, out)
}

func (c *Client) POST(ctx context.Context, path string, body any, out any) error {
	var r io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		r = strings.NewReader(string(b))
	}
	return c.do(ctx, http.MethodPost, path, r, out)
}

func (c *Client) DELETE(ctx context.Context, path string) error {
	return c.do(ctx, http.MethodDelete, path, nil, nil)
}

func (c *Client) PUT(ctx context.Context, path string, body any) error {
	b, err := json.Marshal(body)
	if err != nil {
		return err
	}
	return c.do(ctx, http.MethodPut, path, strings.NewReader(string(b)), nil)
}

func serverFlag(cmd *cobra.Command) string {
	v, _ := cmd.Flags().GetString("server")
	if v == "" {
		return "http://127.0.0.1:8080"
	}
	return v
}

var _ = errors.New
