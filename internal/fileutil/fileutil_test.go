package fileutil

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSkipHidden(t *testing.T) {
	require.True(t, SkipHidden(".env"))
	require.True(t, SkipHidden(".git/config"))
	require.True(t, SkipHidden("node_modules/foo.js"))
	require.True(t, SkipHidden("vendor/x"))
	require.False(t, SkipHidden("main.go"))
	require.False(t, SkipHidden("src/app.go"))
}

func TestGlobWithDoublestar(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "a.go"), []byte("x"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "b.txt"), []byte("y"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".hidden"), []byte("z"), 0o644))

	matches, truncated, err := GlobWithDoublestar("**/*.go", dir, 10)
	require.NoError(t, err)
	require.False(t, truncated)
	require.Len(t, matches, 1)

	matches, truncated, err = GlobWithDoublestar("**/*.txt", dir, 0)
	require.NoError(t, err)
	require.False(t, truncated)
	require.Len(t, matches, 1)
}

func TestGetRgFzfCmdNilSafe(t *testing.T) {
	// Must not panic regardless of whether rg/fzf installed
	_ = GetRgCmd("*.go")
	_ = GetFzfCmd("query")
}
