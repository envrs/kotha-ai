package shell

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/kothagpt/kotha/internal/config"
	"github.com/kothagpt/kotha/internal/logging"
)

// BackgroundCommand represents a long-running command executed in the
// background. Callers can poll the live output file and eventually
// cancel the command.
type BackgroundCommand struct {
	ID          string
	Command     string
	StartTime   time.Time
	OutputFile  string
	ErrorFile   string
	StatusFile  string
	CwdFile     string
	PID         int
	done        chan struct{}
	cancelOnce sync.Once
	result      commandResult
	resultMu    sync.Mutex
	shell       *PersistentShell
}

// IsDone reports whether the background command has finished.
func (bc *BackgroundCommand) IsDone() bool {
	select {
	case <-bc.done:
		return true
	default:
		return false
	}
}

// Wait blocks until the command finishes and returns its result.
func (bc *BackgroundCommand) Wait() commandResult {
	<-bc.done
	bc.resultMu.Lock()
	defer bc.resultMu.Unlock()
	return bc.result
}

// Result returns the command result if the command has finished.
// Returns nil if the command is still running.
func (bc *BackgroundCommand) Result() *commandResult {
	bc.resultMu.Lock()
	defer bc.resultMu.Unlock()
	if bc.IsDone() {
		return &bc.result
	}
	return nil
}

// Cancel kills the background command and its children.
func (bc *BackgroundCommand) Cancel() {
	if bc.shell != nil {
		bc.shell.killChildren()
	}
	bc.cancelOnce.Do(func() {
		close(bc.done)
	})
}

type PersistentShell struct {
	cmd          *exec.Cmd
	stdin        *os.File
	isAlive      bool
	cwd          string
	mu           sync.Mutex
	commandQueue chan *commandExecution
	closeOnce    sync.Once

	// Background commands keyed by ID.
	backgrounds   map[string]*BackgroundCommand
	backgroundsMu sync.RWMutex
}

type commandExecution struct {
	command    string
	timeout    time.Duration
	resultChan chan commandResult
	ctx        context.Context
}

type commandResult struct {
	Stdout      string
	Stderr      string
	ExitCode    int
	Interrupted bool
	Err         error
}

var (
	shellInstance     *PersistentShell
	shellInstanceOnce sync.Once
)

// ResetPersistentShell resets the singleton shell. Only useful for tests.
func ResetPersistentShell() {
	shellInstanceOnce = sync.Once{}
	shellInstance = nil
}

// GetPersistentShell returns the singleton shell for the given working
// directory. If the shell dies, a new one is created.
func GetPersistentShell(workingDir string) *PersistentShell {
	shellInstanceOnce.Do(func() {
		shellInstance = newPersistentShell(workingDir)
	})

	if shellInstance == nil {
		shellInstance = newPersistentShell(workingDir)
	} else if !shellInstance.isAlive {
		shellInstance = newPersistentShell(workingDir)
	}
	if shellInstance == nil {
		// Give up cleanly instead of returning a nil shell that callers
		// would dereference later.
		logging.Error("Failed to start persistent shell")
	}
	return shellInstance
}

// StartBackground runs a command in the background, returning a handle the
// caller can use to poll the live output file or cancel the command.
// This is the explicit backgrounding path.
func (s *PersistentShell) StartBackground(ctx context.Context, command string, timeoutMs int) (*BackgroundCommand, error) {
	if !s.isAlive {
		return nil, errors.New("shell is not alive")
	}
	bc := &BackgroundCommand{
		ID:         uuid.New().String(),
		Command:    command,
		StartTime:  time.Now(),
		done:       make(chan struct{}),
		shell:      s,
	}
	tempDir := os.TempDir()
	bc.OutputFile = filepath.Join(tempDir, fmt.Sprintf("kotha-bg-out-%s", bc.ID))
	bc.ErrorFile = filepath.Join(tempDir, fmt.Sprintf("kotha-bg-err-%s", bc.ID))
	bc.StatusFile = filepath.Join(tempDir, fmt.Sprintf("kotha-bg-status-%s", bc.ID))
	bc.CwdFile = filepath.Join(tempDir, fmt.Sprintf("kotha-bg-cwd-%s", bc.ID))

	fullCommand := fmt.Sprintf(`
eval %s < /dev/null > %s 2> %s
EXEC_EXIT_CODE=$?
pwd > %s
echo $EXEC_EXIT_CODE > %s
`,
		shellQuote(command),
		shellQuote(bc.OutputFile),
		shellQuote(bc.ErrorFile),
		shellQuote(bc.CwdFile),
		shellQuote(bc.StatusFile),
	)

	s.mu.Lock()
	if !s.isAlive {
		s.mu.Unlock()
		return nil, errors.New("shell is not alive")
	}
	_, err := s.stdin.Write([]byte(fullCommand + "\n"))
	s.mu.Unlock()
	if err != nil {
		return nil, fmt.Errorf("failed to write command to shell: %w", err)
	}

	s.backgroundsMu.Lock()
	s.backgrounds[bc.ID] = bc
	s.backgroundsMu.Unlock()

	timeout := time.Duration(timeoutMs) * time.Millisecond
	if timeoutMs <= 0 {
		timeout = 30 * time.Minute
	}

	go func() {
		defer close(bc.done)
		startTime := time.Now()

		for {
			select {
			case <-ctx.Done():
				s.killChildren()
				bc.resultMu.Lock()
				bc.result.Interrupted = true
				bc.result.Err = ctx.Err()
				bc.resultMu.Unlock()
				return
			case <-time.After(10 * time.Millisecond):
				if fileExists(bc.StatusFile) && fileSize(bc.StatusFile) > 0 {
					stdout := readFileOrEmpty(bc.OutputFile)
					stderr := readFileOrEmpty(bc.ErrorFile)
					exitCodeStr := readFileOrEmpty(bc.StatusFile)
					newCwd := readFileOrEmpty(bc.CwdFile)

					exitCode := 0
					if exitCodeStr != "" {
						_, _ = fmt.Sscanf(exitCodeStr, "%d", &exitCode)
					}
					if newCwd != "" {
						s.cwd = strings.TrimSpace(newCwd)
					}
					bc.resultMu.Lock()
				bc.result = commandResult{
					Stdout:   stdout,
					Stderr:   stderr,
					ExitCode: exitCode,
				}
					bc.resultMu.Unlock()
					return
				}
				if timeout > 0 && time.Since(startTime) > timeout {
					s.killChildren()
					bc.resultMu.Lock()
					bc.result.Interrupted = true
					bc.result.Err = errors.New("command timed out")
					bc.resultMu.Unlock()
					return
				}
			}
		}
	}()

	return bc, nil
}

// GetBackground returns a background command by ID.
func (s *PersistentShell) GetBackground(id string) *BackgroundCommand {
	s.backgroundsMu.RLock()
	defer s.backgroundsMu.RUnlock()
	return s.backgrounds[id]
}

// ListBackgrounds returns all background commands.
func (s *PersistentShell) ListBackgrounds() []*BackgroundCommand {
	s.backgroundsMu.RLock()
	defer s.backgroundsMu.RUnlock()
	out := make([]*BackgroundCommand, 0, len(s.backgrounds))
	for _, bc := range s.backgrounds {
		out = append(out, bc)
	}
	return out
}

// CancelBackground cancels a background command by ID.
func (s *PersistentShell) CancelBackground(id string) bool {
	s.backgroundsMu.RLock()
	bc, ok := s.backgrounds[id]
	s.backgroundsMu.RUnlock()
	if !ok {
		return false
	}
	if bc.IsDone() {
		return false
	}
	bc.Cancel()
	return true
}

func newPersistentShell(cwd string) *PersistentShell {
	// Get shell configuration from config
	cfg := config.Get()

	// Default to environment variable if config is not set or nil
	var shellPath string
	var shellArgs []string

	if cfg != nil {
		shellPath = cfg.Shell.Path
		shellArgs = cfg.Shell.Args
	}
	if shellPath == "" {
		shellPath = os.Getenv("SHELL")
	}
	if shellPath == "" {
		shellPath = "/bin/bash"
	}

	// Default shell args
	if len(shellArgs) == 0 {
		shellArgs = []string{"-l"}
	}

	spawn := func(dir string) (*exec.Cmd, io.WriteCloser, bool) {
		c := exec.Command(shellPath, shellArgs...)
		c.Dir = dir
		c.Env = append(os.Environ(), "GIT_EDITOR=true")
		sp, err := c.StdinPipe()
		if err != nil {
			return nil, nil, false
		}
		if err := c.Start(); err != nil {
			return nil, nil, false
		}
		return c, sp, true
	}

	cmd, stdinPipe, ok := spawn(cwd)
	started := ok
	var spawnDir string
	if !started && cwd != "" {
		// The working directory may no longer exist (e.g. a deleted temp
		// dir). Fall back to the current process working directory.
		spawnDir = ""
		cmd, stdinPipe, started = spawn(spawnDir)
	}
	if !started {
		return nil
	}

	shell := &PersistentShell{
		cmd:          cmd,
		stdin:        stdinPipe.(*os.File),
		isAlive:      true,
		cwd:          cwd,
		commandQueue: make(chan *commandExecution, 10),
		backgrounds:  make(map[string]*BackgroundCommand),
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				logging.ErrorPersist(fmt.Sprintf("Panic in shell command processor: %v", r))
				shell.markDead()
			}
		}()
		shell.processCommands()
	}()

	go func() {
		if err := cmd.Wait(); err != nil {
			logging.Debug("Persistent shell process exited", "error", err)
		}
		shell.markDead()
	}()

	return shell
}

func (s *PersistentShell) markDead() {
	s.mu.Lock()
	s.isAlive = false
	s.mu.Unlock()
	// Either the processor panic path or cmd.Wait can fire first;
	// Once guarantees exactly one close (double-close panics).
	s.closeOnce.Do(func() { close(s.commandQueue) })
}

func (s *PersistentShell) processCommands() {
	for cmd := range s.commandQueue {
		result := s.execCommand(cmd.command, cmd.timeout, cmd.ctx)
		cmd.resultChan <- result
	}
}

func (s *PersistentShell) execCommand(command string, timeout time.Duration, ctx context.Context) commandResult {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isAlive {
		return commandResult{
			Stderr:   "Shell is not alive",
			ExitCode: 1,
			Err:      errors.New("shell is not alive"),
		}
	}

	tempDir := os.TempDir()
	stdoutFile := filepath.Join(tempDir, fmt.Sprintf("kotha-stdout-%d", time.Now().UnixNano()))
	stderrFile := filepath.Join(tempDir, fmt.Sprintf("kotha-stderr-%d", time.Now().UnixNano()))
	statusFile := filepath.Join(tempDir, fmt.Sprintf("kotha-status-%d", time.Now().UnixNano()))
	cwdFile := filepath.Join(tempDir, fmt.Sprintf("kotha-cwd-%d", time.Now().UnixNano()))

	defer func() {
		_ = os.Remove(stdoutFile)
		_ = os.Remove(stderrFile)
		_ = os.Remove(statusFile)
		_ = os.Remove(cwdFile)
	}()

	fullCommand := fmt.Sprintf(`
eval %s < /dev/null > %s 2> %s
EXEC_EXIT_CODE=$?
pwd > %s
echo $EXEC_EXIT_CODE > %s
`,
		shellQuote(command),
		shellQuote(stdoutFile),
		shellQuote(stderrFile),
		shellQuote(cwdFile),
		shellQuote(statusFile),
	)

	_, err := s.stdin.Write([]byte(fullCommand + "\n"))
	if err != nil {
		return commandResult{
			Stderr:   fmt.Sprintf("Failed to write command to shell: %v", err),
			ExitCode: 1,
			Err:      err,
		}
	}

	interrupted := false

	startTime := time.Now()

	done := make(chan bool)
	go func() {
		for {
			select {
			case <-ctx.Done():
				s.killChildren()
				interrupted = true
				done <- true
				return

			case <-time.After(10 * time.Millisecond):
				if fileExists(statusFile) && fileSize(statusFile) > 0 {
					done <- true
					return
				}

				if timeout > 0 {
					elapsed := time.Since(startTime)
					if elapsed > timeout {
						s.killChildren()
						interrupted = true
						done <- true
						return
					}
				}
			}
		}
	}()

	<-done

	stdout := readFileOrEmpty(stdoutFile)
	stderr := readFileOrEmpty(stderrFile)
	exitCodeStr := readFileOrEmpty(statusFile)
	newCwd := readFileOrEmpty(cwdFile)

	exitCode := 0
	if exitCodeStr != "" {
		_, _ = fmt.Sscanf(exitCodeStr, "%d", &exitCode)
	} else if interrupted {
		exitCode = 143
		stderr += "\nCommand execution timed out or was interrupted"
	}

	if newCwd != "" {
		s.cwd = strings.TrimSpace(newCwd)
	}

	return commandResult{
		Stdout:      stdout,
		Stderr:      stderr,
		ExitCode:    exitCode,
		Interrupted: interrupted,
	}
}

func (s *PersistentShell) killChildren() {
	if s.cmd == nil || s.cmd.Process == nil {
		return
	}

	pgrepCmd := exec.Command("pgrep", "-P", fmt.Sprintf("%d", s.cmd.Process.Pid))
	output, err := pgrepCmd.Output()
	if err != nil {
		return
	}

	for pidStr := range strings.SplitSeq(string(output), "\n") {
		if pidStr = strings.TrimSpace(pidStr); pidStr != "" {
			var pid int
			_, _ = fmt.Sscanf(pidStr, "%d", &pid)
			if pid > 0 {
				proc, err := os.FindProcess(pid)
				if err == nil {
					_ = proc.Signal(syscall.SIGTERM)
				}
			}
		}
	}
}

func (s *PersistentShell) Exec(ctx context.Context, command string, timeoutMs int) (string, string, int, bool, error) {
	if !s.isAlive {
		return "", "Shell is not alive", 1, false, errors.New("shell is not alive")
	}

	timeout := time.Duration(timeoutMs) * time.Millisecond

	resultChan := make(chan commandResult)
	s.commandQueue <- &commandExecution{
		command:    command,
		timeout:    timeout,
		resultChan: resultChan,
		ctx:        ctx,
	}

	result := <-resultChan
	return result.Stdout, result.Stderr, result.ExitCode, result.Interrupted, result.Err
}

func (s *PersistentShell) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isAlive {
		return
	}

	_, _ = s.stdin.Write([]byte("exit\n"))

	_ = s.cmd.Process.Kill()
	s.isAlive = false
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

func readFileOrEmpty(path string) string {
	content, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(content)
}

// ReadFileOrEmpty reads a file and returns its contents, or "" on error.
// Exported for use by other tools that need to read shell output files.
func ReadFileOrEmpty(path string) string {
	return readFileOrEmpty(path)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func fileSize(path string) int64 {
	info, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return info.Size()
}
