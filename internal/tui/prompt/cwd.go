package prompt

import (
	"os"
	"path/filepath"
	"strings"
)

// Cwd tracks the composer working directory and resolves @-mentions.
type Cwd struct {
	Dir string
}

func NewCwd(dir string) Cwd {
	if dir == "" {
		dir, _ = os.Getwd()
	}
	return Cwd{Dir: dir}
}

// Resolve turns a mention path into an absolute path. Absolute mentions
// pass through; relative ones anchor at the cwd.
func (c Cwd) Resolve(mention string) string {
	mention = strings.TrimSpace(mention)
	if mention == "" {
		return c.Dir
	}
	if filepath.IsAbs(mention) {
		return filepath.Clean(mention)
	}
	return filepath.Join(c.Dir, mention)
}

// Mention splits trailing "@partial" token from text for completion.
func MentionToken(text string) (prefix, token string, ok bool) {
	idx := strings.LastIndex(text, "@")
	if idx < 0 {
		return text, "", false
	}
	// A space after @ closes the token.
	if tail := text[idx+1:]; strings.ContainsAny(tail, " \t\n") {
		return text, "", false
	}
	return text[:idx], text[idx+1:], true
}
