package tools

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kothagpt/kotha/internal/lsp/protocol"
	"github.com/kothagpt/kotha/internal/permission"
	"github.com/stretchr/testify/require"
)

func TestFetchValidation(t *testing.T) {
	bootstrapConfig(t, t.TempDir())
	svc := permission.NewPermissionService()
	svc.AutoApproveSession("s1")
	f := NewFetchTool(svc)
	ctx := testCtx("s1", "m1")

	resp, err := f.Run(ctx, ToolCall{Input: "{"})
	require.NoError(t, err)
	require.True(t, resp.IsError)

	in, _ := json.Marshal(FetchParams{})
	resp, err = f.Run(ctx, ToolCall{Input: string(in)})
	require.NoError(t, err)
	require.True(t, resp.IsError)

	in, _ = json.Marshal(FetchParams{URL: "http://x", Format: "yaml"})
	resp, err = f.Run(ctx, ToolCall{Input: string(in)})
	require.NoError(t, err)
	require.True(t, resp.IsError)

	in, _ = json.Marshal(FetchParams{URL: "ftp://x", Format: "text"})
	resp, err = f.Run(ctx, ToolCall{Input: string(in)})
	require.NoError(t, err)
	require.True(t, resp.IsError)

	in, _ = json.Marshal(FetchParams{URL: "http://x", Format: "text"})
	_, err = f.Run(testCtx("", ""), ToolCall{Input: string(in)})
	require.Error(t, err)

	// denied
	f2 := NewFetchTool(denyPermission{svc})
	_, err = f2.Run(ctx, ToolCall{Input: string(in)})
	require.ErrorIs(t, err, permission.ErrorPermissionDenied)
}

func TestFetchSuccess(t *testing.T) {
	bootstrapConfig(t, t.TempDir())
	svc := permission.NewPermissionService()
	svc.AutoApproveSession("s1")
	f := NewFetchTool(svc)
	ctx := testCtx("s1", "m1")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/html":
			w.Header().Set("Content-Type", "text/html")
			_, _ = w.Write([]byte("<html><body><h1>Hi</h1><p>there</p></body></html>"))
		case "/json":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"a":1}`))
		case "/err":
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer srv.Close()

	// text from html
	in, _ := json.Marshal(FetchParams{URL: srv.URL + "/html", Format: "text"})
	resp, err := f.Run(ctx, ToolCall{Input: string(in)})
	require.NoError(t, err)
	require.False(t, resp.IsError)
	require.Contains(t, resp.Content, "Hi")

	// markdown from html
	in, _ = json.Marshal(FetchParams{URL: srv.URL + "/html", Format: "markdown", Timeout: 500})
	resp, err = f.Run(ctx, ToolCall{Input: string(in)})
	require.NoError(t, err)
	require.False(t, resp.IsError)
	require.Contains(t, resp.Content, "Hi")

	// html raw
	in, _ = json.Marshal(FetchParams{URL: srv.URL + "/html", Format: "html"})
	resp, err = f.Run(ctx, ToolCall{Input: string(in)})
	require.NoError(t, err)
	require.Contains(t, resp.Content, "<h1>")

	// markdown non-html wraps in code fence
	in, _ = json.Marshal(FetchParams{URL: srv.URL + "/json", Format: "markdown"})
	resp, err = f.Run(ctx, ToolCall{Input: string(in)})
	require.NoError(t, err)
	require.Contains(t, resp.Content, "```")

	// text non-html passthrough
	in, _ = json.Marshal(FetchParams{URL: srv.URL + "/json", Format: "text"})
	resp, err = f.Run(ctx, ToolCall{Input: string(in)})
	require.NoError(t, err)
	require.Contains(t, resp.Content, `"a":1`)

	// non-200
	in, _ = json.Marshal(FetchParams{URL: srv.URL + "/err", Format: "text"})
	resp, err = f.Run(ctx, ToolCall{Input: string(in)})
	require.NoError(t, err)
	require.True(t, resp.IsError)
}

func TestFetchHelpers(t *testing.T) {
	text, err := extractTextFromHTML("<html><body><p>a  b</p></body></html>")
	require.NoError(t, err)
	require.Equal(t, "a b", text)

	mkd, err := convertHTMLToMarkdown("<h1>T</h1>")
	require.NoError(t, err)
	require.Contains(t, mkd, "T")
}

func TestDiagnosticsTool(t *testing.T) {
	d := NewDiagnosticsTool(nil)
	resp, err := d.Run(context.Background(), ToolCall{Input: "{"})
	require.NoError(t, err)
	require.True(t, resp.IsError)

	in, _ := json.Marshal(DiagnosticsParams{})
	resp, err = d.Run(context.Background(), ToolCall{Input: string(in)})
	require.NoError(t, err)
	require.True(t, resp.IsError)
	require.Contains(t, resp.Content, "no LSP clients")

	require.Equal(t, "", getDiagnostics("f", nil))
	require.Equal(t, 2, countSeverity([]string{"Error a", "Error b", "Warn c"}, "Error"))
	require.Equal(t, 1, countSeverity([]string{"Error a", "Warn c"}, "Warn"))

	uri := protocol.DocumentUri("file:///f")
	cur := map[protocol.DocumentUri][]protocol.Diagnostic{uri: {{Message: "x"}}}
	require.True(t, hasDiagnosticsChanged(cur, map[protocol.DocumentUri][]protocol.Diagnostic{}))
	require.False(t, hasDiagnosticsChanged(cur, cur))
}

func TestSourcegraphValidation(t *testing.T) {
	s := NewSourcegraphTool()
	resp, err := s.Run(context.Background(), ToolCall{Input: "{"})
	require.NoError(t, err)
	require.True(t, resp.IsError)

	in, _ := json.Marshal(SourcegraphParams{})
	resp, err = s.Run(context.Background(), ToolCall{Input: string(in)})
	require.NoError(t, err)
	require.True(t, resp.IsError)
}
