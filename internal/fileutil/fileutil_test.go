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

func TestExpandTilde(t *testing.T) {
	home, err := os.UserHomeDir()
	require.NoError(t, err)

	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"~", home},
		{"~/", home}, // filepath.Join(home, "") returns home without trailing separator
		{"~/file.txt", filepath.Join(home, "file.txt")},
		{"~/dir/file.txt", filepath.Join(home, "dir", "file.txt")},
		{"~draft.md", "~draft.md"},           // no separator after ~
		{"~user/file.txt", "~user/file.txt"}, // not expanded for security
		{"relative/path", "relative/path"},   // no tilde
		{"/absolute/path", "/absolute/path"}, // absolute path
	}

	for _, tc := range tests {
		result := ExpandTilde(tc.input)
		require.Equal(t, tc.expected, result, "input: %q", tc.input)
	}
}

func TestExpandTildeWindows(t *testing.T) {
	// Test Windows-style backslash paths on all platforms (function handles it)
	home, err := os.UserHomeDir()
	require.NoError(t, err)

	// On non-Windows, backslash is not a separator, so ~\path should not expand
	// but we test the logic doesn't panic
	result := ExpandTilde(`~\path`)
	require.Equal(t, `~\path`, result) // backslash not separator on Unix

	// Test forward slash always works
	result = ExpandTilde("~/path")
	require.Equal(t, filepath.Join(home, "path"), result)
}

func TestMustExpandTilde(t *testing.T) {
	home, err := os.UserHomeDir()
	require.NoError(t, err)

	result := MustExpandTilde("~/file.txt")
	require.Equal(t, filepath.Join(home, "file.txt"), result)

	// Should panic on unexpandable paths
	for _, tc := range []string{"", "~user/file"} {
		func() {
			defer func() {
				if r := recover(); r == nil {
					t.Fatalf("expected panic for %q, got none", tc)
				}
			}()
			MustExpandTilde(tc)
		}()
	}
}
