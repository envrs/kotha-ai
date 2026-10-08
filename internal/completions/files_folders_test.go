package completions

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProcessNullTerminatedOutput(t *testing.T) {
	require.Empty(t, processNullTerminatedOutput(nil))
	require.Empty(t, processNullTerminatedOutput([]byte{0}))
	require.Equal(t, []string{"a", "b"}, processNullTerminatedOutput([]byte("a\x00b\x00")))
	require.Equal(t, []string{"a"}, processNullTerminatedOutput([]byte("a\x00\x00")))
	// hidden files filtered
	out := processNullTerminatedOutput([]byte(".hidden\x00visible.txt\x00"))
	require.Equal(t, []string{"visible.txt"}, out)
}

func TestContextGroupMeta(t *testing.T) {
	g := NewFileAndFolderContextGroup()
	require.Equal(t, "file", g.GetId())
	require.NotNil(t, g.GetEntry())
}

func TestGetFilesFallback(t *testing.T) {
	// rg/fzf absent in CI env -> doublestar+fuzzy path
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(dir+"/foo.go", []byte("x"), 0o644))
	old, _ := os.Getwd()
	require.NoError(t, os.Chdir(dir))
	defer func() { _ = os.Chdir(old) }()

	g := &filesAndFoldersContextGroup{prefix: "file"}
	matches, err := g.getFiles("foo")
	require.NoError(t, err)
	require.Contains(t, matches, "foo.go")

	items, err := g.GetChildEntries("foo")
	require.NoError(t, err)
	require.NotEmpty(t, items)
}
