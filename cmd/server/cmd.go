package server

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/spf13/cobra"
)

var (
	serverAddr    string
	serverCwd     string
	serverDebug   bool
	serverAPIKey  string
	serverRate    float64
	serverBurst   int
	serverTimeout time.Duration
	serverDaemon  bool
	serverPIDFile string
)

// Cmd is the `kotha server` subcommand that runs the background HTTP server.
var Cmd = &cobra.Command{
	Use:   "server",
	Short: "Run Kotha as a background HTTP server",
	Long: `Run Kotha as a background HTTP server exposing the SDK surface.

The server can be started in the foreground or as a daemon (with --daemon).
Use the 'kotha api' subcommand to make requests to a running server.`,
	Example: `
  # Start the server in the foreground
  kotha server --addr 127.0.0.1:8080

  # Start as a daemon writing a PID file
  kotha server --daemon --pidfile /var/run/kotha.pid

  # Make a request to the running server
  kotha api ask -s http://127.0.0.1:8080 -i my-session -p "summarize this repo"
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := DefaultServerConfig()
		cfg.Addr = serverAddr
		cfg.Cwd = serverCwd
		cfg.Debug = serverDebug
		cfg.Timeout = serverTimeout
		cfg.APIKey = serverAPIKey
		if cfg.APIKey == "" {
			cfg.APIKey = os.Getenv("KOTHA_API_KEY")
		}
		cfg.Rate = serverRate
		cfg.Burst = serverBurst
		if serverDaemon {
			return runDaemon(cfg)
		}
		return RunServer(cfg)
	},
}

func init() {
	Cmd.Flags().StringVar(&serverAddr, "addr", DefaultServerConfig().Addr, "server listen address")
	Cmd.Flags().StringVar(&serverCwd, "cwd", "", "working directory")
	Cmd.Flags().BoolVar(&serverDebug, "debug", false, "enable debug logging")
	Cmd.Flags().StringVar(&serverAPIKey, "api-key", "", "require this API key on /v1 requests (env KOTHA_API_KEY)")
	Cmd.Flags().Float64Var(&serverRate, "rate", DefaultServerConfig().Rate, "max requests per second per client IP on /v1 (0 disables)")
	Cmd.Flags().IntVar(&serverBurst, "burst", DefaultServerConfig().Burst, "burst size for the per-IP request limiter")
	Cmd.Flags().DurationVar(&serverTimeout, "timeout", DefaultServerConfig().Timeout, "request timeout")
	Cmd.Flags().BoolVar(&serverDaemon, "daemon", false, "run as a background daemon")
	Cmd.Flags().StringVar(&serverPIDFile, "pidfile", "", "write daemon PID to file")
}

// runDaemon forks the server into the background and writes a PID file.
func runDaemon(cfg ServerConfig) error {
	if serverPIDFile == "" {
		return fmt.Errorf("--pidfile is required when --daemon is set")
	}
	// Re-exec ourselves detached from the terminal.
	binary, err := os.Executable()
	if err != nil {
		return err
	}
	c := exec.Command(binary, "server", "--daemon=false",
		"--addr", cfg.Addr, "--cwd", cfg.Cwd, "--timeout", cfg.Timeout.String(),
		"--api-key", cfg.APIKey,
		"--rate", fmt.Sprintf("%g", cfg.Rate),
		"--burst", fmt.Sprintf("%d", cfg.Burst))
	if cfg.Debug {
		c.Args = append(c.Args, "--debug")
	}
	c.Stdin = nil
	c.Stdout = nil
	c.Stderr = nil
	c.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := c.Start(); err != nil {
		return err
	}
	return writePID(serverPIDFile, c.Process.Pid)
}

// writePID writes the daemon PID to a file.
func writePID(path string, pid int) error {
	return os.WriteFile(path, []byte(fmt.Sprintf("%d\n", pid)), 0o644)
}

// readPID reads a PID from a file.
func readPID(path string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	var pid int
	if _, err := fmt.Sscanf(string(data), "%d", &pid); err != nil {
		return 0, fmt.Errorf("invalid pid file: %w", err)
	}
	return pid, nil
}

// pidAlive reports whether a process with the given PID is running.
func pidAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	err := syscall.Kill(pid, syscall.Signal(0))
	return err == nil
}

// StopServer stops a running daemon via its PID file.
func StopServer(pidfile string) error {
	pid, err := readPID(pidfile)
	if err != nil {
		return err
	}
	if err := syscall.Kill(pid, syscall.SIGTERM); err != nil {
		return err
	}
	return nil
}

// ServerStatus reports whether a server is running.
type ServerStatus struct {
	Running bool
	PID     int
}

// ServerStatusFor reports the status of a server tracked by a PID file.
func ServerStatusFor(pidfile string) ServerStatus {
	pid, err := readPID(pidfile)
	if err != nil {
		return ServerStatus{}
	}
	return ServerStatus{Running: pidAlive(pid), PID: pid}
}
