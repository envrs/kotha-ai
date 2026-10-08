package tuicontext

import "os"

// Local captures machine-local snapshot info for status lines.
type Local struct {
	Hostname string
	Cwd      string
}

func DetectLocal(cwd string) Local {
	host, _ := os.Hostname()
	if cwd == "" {
		cwd, _ = os.Getwd()
	}
	return Local{Hostname: host, Cwd: cwd}
}
