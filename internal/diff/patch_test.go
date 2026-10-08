package diff

import (
	"errors"
	"os"
	"testing"

	"github.com/kothagpt/kotha/internal/config"
	"github.com/stretchr/testify/require"
)

func TestGenerateDiffWithConfig(t *testing.T) {
	config.SetDefaultProvider(config.NewDefaultProvider(&config.Config{WorkingDir: "/tmp"}))
	d, adds, dels := GenerateDiff("a\nb\n", "a\nc\n", "f.go")
	require.NotEmpty(t, d)
	require.Equal(t, 1, adds)
	require.Equal(t, 1, dels)

	d2, a2, dl2 := GenerateDiff("same\n", "same\n", "f.go")
	require.Empty(t, d2)
	require.Equal(t, 0, a2)
	require.Equal(t, 0, dl2)
}

func TestTextToPatchUpdate(t *testing.T) {
	text := "*** Begin Patch\n*** Update File: f.txt\n@@\n line1\n-line2\n+line2new\n line3\n*** End Patch"
	orig := map[string]string{"f.txt": "line1\nline2\nline3"}
	p, fuzz, err := TextToPatch(text, orig)
	require.NoError(t, err)
	require.Equal(t, 0, fuzz)
	commit, err := PatchToCommit(p, orig)
	require.NoError(t, err)
	require.Contains(t, *commit.Changes["f.txt"].NewContent, "line2new")
}

func TestTextToPatchAddDelete(t *testing.T) {
	text := "*** Begin Patch\n*** Add File: n.txt\n+hello\n+world\n*** End Patch"
	p, _, err := TextToPatch(text, map[string]string{})
	require.NoError(t, err)
	commit, err := PatchToCommit(p, map[string]string{})
	require.NoError(t, err)
	require.Equal(t, ActionAdd, commit.Changes["n.txt"].Type)

	text = "*** Begin Patch\n*** Delete File: f.txt\n*** End Patch"
	p, _, err = TextToPatch(text, map[string]string{"f.txt": "x"})
	require.NoError(t, err)
	commit, err = PatchToCommit(p, map[string]string{"f.txt": "x"})
	require.NoError(t, err)
	require.Equal(t, ActionDelete, commit.Changes["f.txt"].Type)
}

func TestTextToPatchErrors(t *testing.T) {
	_, _, err := TextToPatch("garbage", nil)
	require.Error(t, err)
	_, _, err = TextToPatch("*** Begin Patch\n*** Update File: missing.txt\n*** End Patch", map[string]string{})
	require.Error(t, err)
	_, _, err = TextToPatch("*** Begin Patch\n*** Frobnicate\n*** End Patch", map[string]string{})
	require.Error(t, err)
	_, _, err = TextToPatch("*** Begin Patch\n*** Add File: a.txt\nno-plus\n*** End Patch", map[string]string{})
	require.Error(t, err)
}

func TestIdentifyFiles(t *testing.T) {
	text := "*** Begin Patch\n*** Update File: a.txt\n*** Delete File: b.txt\n*** Add File: c.txt\n*** End Patch"
	require.Contains(t, IdentifyFilesNeeded(text), "a.txt")
	require.Contains(t, IdentifyFilesNeeded(text), "b.txt")
	require.Contains(t, IdentifyFilesAdded(text), "c.txt")
}

func TestAssembleAndApply(t *testing.T) {
	commit := AssembleChanges(
		map[string]string{"a": "x", "b": "y"},
		map[string]string{"a": "x2", "c": "new", "b": ""},
	)
	require.Equal(t, ActionUpdate, commit.Changes["a"].Type)
	require.Equal(t, ActionAdd, commit.Changes["c"].Type)
	require.Equal(t, ActionDelete, commit.Changes["b"].Type)

	written := map[string]string{}
	removed := []string{}
	err := ApplyCommit(commit,
		func(p, c string) error { written[p] = c; return nil },
		func(p string) error { removed = append(removed, p); return nil })
	require.NoError(t, err)
	require.Equal(t, "x2", written["a"])
	require.Contains(t, removed, "b")

	// nil content errors
	bad := Commit{Changes: map[string]FileChange{"x": {Type: ActionAdd}}}
	require.Error(t, ApplyCommit(bad, func(p, c string) error { return nil }, func(p string) error { return nil }))
	bad2 := Commit{Changes: map[string]FileChange{"x": {Type: ActionUpdate}}}
	require.Error(t, ApplyCommit(bad2, func(p, c string) error { return nil }, func(p string) error { return nil }))

	// write error propagates
	require.Error(t, ApplyCommit(commit,
		func(p, c string) error { return errors.New("wfail") },
		func(p string) error { return nil }))
}

func TestProcessAndValidatePatch(t *testing.T) {
	text := "*** Begin Patch\n*** Add File: n.txt\n+hi\n*** End Patch"
	files := map[string]string{}
	written := map[string]string{}
	msg, err := ProcessPatch(text,
		func(p string) (string, error) { return files[p], nil },
		func(p, c string) error { written[p] = c; files[p] = c; return nil },
		func(p string) error { delete(files, p); return nil })
	require.NoError(t, err)
	require.Contains(t, msg, "successfully")

	ok, _, err := ValidatePatch(text, map[string]string{})
	require.NoError(t, err)
	require.True(t, ok)

	ok, reason, _ := ValidatePatch("nope", nil)
	require.False(t, ok)
	require.NotEmpty(t, reason)

	ok, _, _ = ValidatePatch("*** Begin Patch\n*** Update File: z.txt\n*** End Patch", map[string]string{})
	require.False(t, ok)
}

func TestFindContextVariants(t *testing.T) {
	lines := []string{"  a  ", "b", "c"}
	idx, _ := findContext(lines, []string{"a"}, 0, false)
	require.Equal(t, 0, idx)
	require.Equal(t, -1, func() int { i, _ := findContext(lines, []string{"zzz"}, 0, false); return i }())
	idx, _ = findContext(lines, []string{"c"}, 0, true)
	require.Equal(t, 2, idx)
}

func TestLoadOpenWriteHelpers(t *testing.T) {
	m, err := LoadFiles([]string{"a"}, func(p string) (string, error) { return "content", nil })
	require.NoError(t, err)
	require.Equal(t, "content", m["a"])
	_, err = LoadFiles([]string{"a"}, func(p string) (string, error) { return "", errors.New("nf") })
	require.Error(t, err)

	dir := t.TempDir()
	oldWd, _ := os.Getwd()
	require.NoError(t, os.Chdir(dir))
	defer func() { _ = os.Chdir(oldWd) }()
	rel := "sub/f.txt"
	require.NoError(t, WriteFile(rel, "hi"))
	got, err := OpenFile(rel)
	require.NoError(t, err)
	require.Equal(t, "hi", got)
	_, err = OpenFile("missing.txt")
	require.Error(t, err)
	require.Error(t, WriteFile("/abs/path.txt", "x"))
	require.NoError(t, RemoveFile(rel))
}
