package tools

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResponses(t *testing.T) {
	r := NewTextResponse("hi")
	require.Equal(t, ToolResponseTypeText, r.Type)
	require.Equal(t, "hi", r.Content)
	require.False(t, r.IsError)

	e := NewTextErrorResponse("bad")
	require.True(t, e.IsError)

	withMeta := WithResponseMetadata(r, map[string]string{"k": "v"})
	require.Contains(t, withMeta.Metadata, `"k":"v"`)
	require.Equal(t, r, WithResponseMetadata(r, nil))

	badCh := make(chan int)
	require.Equal(t, r, WithResponseMetadata(r, badCh))
}

func TestGetContextValues(t *testing.T) {
	ctx := context.Background()
	sid, mid := GetContextValues(ctx)
	require.Equal(t, "", sid)
	require.Equal(t, "", mid)

	ctx = context.WithValue(ctx, SessionIDContextKey, "s1")
	sid, mid = GetContextValues(ctx)
	require.Equal(t, "s1", sid)
	require.Equal(t, "", mid)

	ctx = context.WithValue(ctx, MessageIDContextKey, "m1")
	sid, mid = GetContextValues(ctx)
	require.Equal(t, "s1", sid)
	require.Equal(t, "m1", mid)
}

func TestToolInfos(t *testing.T) {
	for _, tool := range []BaseTool{NewGrepTool(), NewGlobTool(), NewViewTool(nil), NewLsTool(), NewDiagnosticsTool(nil), NewSourcegraphTool()} {
		info := tool.Info()
		require.NotEmpty(t, info.Name)
		require.NotEmpty(t, info.Description)
	}
}

func TestPureHelpers(t *testing.T) {
	require.Equal(t, "a\\.b", escapeRegexPattern("a.b"))
	require.Contains(t, globToRegex("*.go"), "go")
	require.Contains(t, addLineNumbers("a\nb", 1), "1")
	require.Contains(t, addLineNumbers("a\nb", 1), "2")

	dir := t.TempDir()
	fp := filepath.Join(dir, "f.txt")
	require.NoError(t, os.WriteFile(fp, []byte("l1\nl2\nl3\n"), 0o644))
	content, total, err := readTextFile(fp, 0, 2)
	require.NoError(t, err)
	require.GreaterOrEqual(t, total, 2)
	require.Contains(t, content, "l1")
	_, _, err = readTextFile(filepath.Join(dir, "missing.txt"), 0, 1)
	require.Error(t, err)

	mimeDir := t.TempDir()
	oldWd, _ := os.Getwd()
	require.NoError(t, os.Chdir(mimeDir))
	defer func() { _ = os.Chdir(oldWd) }()
	require.NoError(t, os.WriteFile("g.txt", []byte("l1\nl2\nl3\n"), 0o644))

	ok, mime := isImageFile("x.png")
	require.True(t, ok)
	require.NotEmpty(t, mime)
	ok, _ = isImageFile("x.txt")
	require.False(t, ok)

	matches, err := searchFilesWithRegex("l2", ".", "")
	require.NoError(t, err)
	require.NotEmpty(t, matches)
}
