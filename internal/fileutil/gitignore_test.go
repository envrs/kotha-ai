package fileutil

import (
	"os"
	"path/filepath"
	"testing"
)

func mustLoad(t *testing.T, root string, files map[string]string) *GitIgnore {
	t.Helper()
	g := NewGitIgnore(root)
	for dir, content := range files {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, dir, ".gitignore"), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := g.LoadDir(filepath.Join(root, dir)); err != nil {
			t.Fatal(err)
		}
	}
	return g
}

func TestGitIgnoreBasics(t *testing.T) {
	root := t.TempDir()
	g := mustLoad(t, root, map[string]string{
		"": "# comment\n\n*.log\n!important.log\n/build/\nsecret\n",
	})

	cases := []struct {
		rel   string
		isDir bool
		want  bool
	}{
		{"app.log", false, true},
		{"a/b/app.log", false, true},
		{"important.log", false, false},
		{"notes.txt", false, false},
		{"build", true, true},         // anchored dir at root
		{"a/build", true, false},      // anchored: only root-relative
		{"secret", true, true},        // unanchored dir pattern
		{"a/secret", true, true},      // matches at any depth
		{"secret/x.txt", false, true}, // subtree of ignored dir
		{"a/b/c.txt", false, false},
	}
	for _, c := range cases {
		if got := g.Match(c.rel, c.isDir); got != c.want {
			t.Errorf("Match(%q, dir=%v) = %v, want %v", c.rel, c.isDir, got, c.want)
		}
	}
}

func TestGitIgnoreDirOnlyNotFile(t *testing.T) {
	root := t.TempDir()
	g := mustLoad(t, root, map[string]string{"": "cache/\n"})
	if !g.Match("cache", true) {
		t.Error("dir cache should be ignored")
	}
	if g.Match("cache", false) {
		t.Error("file named cache should not be ignored by dir-only pattern")
	}
}

func TestGitIgnoreNegationCannotRescueUnderIgnoredDir(t *testing.T) {
	root := t.TempDir()
	g := mustLoad(t, root, map[string]string{"": "logs/\n!logs/keep.txt\n"})
	if !g.Match("logs/keep.txt", false) {
		t.Error("file under ignored dir must stay ignored despite negation")
	}
}

func TestGitIgnoreNestedOverridesRoot(t *testing.T) {
	root := t.TempDir()
	g := mustLoad(t, root, map[string]string{
		"":    "*.log\n",
		"a":   "!important.log\n",
		"a/b": "nested.log\n",
	})
	if g.Match("a/important.log", false) {
		t.Error("nested negation should override root *.log")
	}
	if !g.Match("important.log", false) {
		t.Error("root-level important.log should stay ignored; negation is scoped to a/")
	}
	if !g.Match("a/b/nested.log", false) {
		t.Error("nested rule should ignore nested.log")
	}
	if !g.Match("a/other.log", false) {
		t.Error("root rule still applies where nested file has no override")
	}
}

func TestGitIgnoreDoublestar(t *testing.T) {
	root := t.TempDir()
	g := mustLoad(t, root, map[string]string{"": "docs/**/draft.md\n**/tmp\n"})
	if !g.Match("docs/a/b/draft.md", false) {
		t.Error("** pattern should match across directories")
	}
	if !g.Match("x/y/tmp", true) {
		t.Error("leading **/ should match at any depth")
	}
}

func TestGitIgnoreEscapedHashAndComment(t *testing.T) {
	root := t.TempDir()
	g := mustLoad(t, root, map[string]string{"": "# leading comment\n\\#notacomment\n"})
	if !g.Match("#notacomment", false) {
		t.Error("escaped # pattern should match a literal # filename")
	}
}

func TestGitIgnoreMatchAbsOutsideRoot(t *testing.T) {
	root := t.TempDir()
	g := mustLoad(t, root, map[string]string{"": "*.log\n"})
	outside := filepath.Join(t.TempDir(), "x.log")
	if g.MatchAbs(outside, false) {
		t.Error("paths outside root must never be ignored")
	}
	if !g.MatchAbs(filepath.Join(root, "x.log"), false) {
		t.Error("root-relative path should match")
	}
}

func TestGitIgnoreLoadMissingFileOK(t *testing.T) {
	g := NewGitIgnore(t.TempDir())
	if err := g.LoadDir(g.Root()); err != nil {
		t.Fatalf("missing .gitignore should not error: %v", err)
	}
	if g.Match("anything", false) {
		t.Error("no rules loaded")
	}
}
