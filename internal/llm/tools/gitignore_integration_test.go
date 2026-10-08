package tools

import (
	"os"
	"path/filepath"
	"testing"
)

func TestListDirectoryRespectsGitIgnore(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, ".gitignore"), "*.log\nignored/\n")
	writeFile(t, filepath.Join(dir, "keep.txt"), "keep")
	writeFile(t, filepath.Join(dir, "app.log"), "log")
	if err := os.MkdirAll(filepath.Join(dir, "ignored"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "ignored", "x.txt"), "x")
	if err := os.MkdirAll(filepath.Join(dir, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "sub", "nested.log"), "log")

	files, _, err := listDirectory(dir, []string{}, 1000)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]bool{}
	for _, f := range files {
		rel, rerr := filepath.Rel(dir, filepath.Clean(f))
		if rerr == nil {
			got[filepath.ToSlash(rel)] = true
		}
	}
	if !got["keep.txt"] {
		t.Errorf("keep.txt should be listed, got %v", got)
	}
	for _, banned := range []string{"app.log", "ignored", "ignored/x.txt", "sub/nested.log"} {
		if got[banned] {
			t.Errorf("%s is gitignored but was listed", banned)
		}
	}
}

func TestSearchFilesWithRegexRespectsGitIgnore(t *testing.T) {
	// SkipHidden treats path components named tmp/temp as hidden, so
	// avoid t.TempDir() (which lives under /tmp) for this walk-based test.
	dir, err := os.MkdirTemp(".", "gitignore-grep-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	dir, err = filepath.Abs(dir)
	if err != nil {
		t.Fatal(err)
	}

	writeFile(t, filepath.Join(dir, ".gitignore"), "*.log\n")
	writeFile(t, filepath.Join(dir, "keep.txt"), "needle here")
	writeFile(t, filepath.Join(dir, "app.log"), "needle here")

	matches, err := searchFilesWithRegex("needle", dir, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("matches = %d, want 1 (only keep.txt)", len(matches))
	}
	rel, _ := filepath.Rel(dir, matches[0].path)
	if filepath.ToSlash(rel) != "keep.txt" {
		t.Fatalf("matched %q, want keep.txt", rel)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
