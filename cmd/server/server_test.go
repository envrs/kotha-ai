package server

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"
)

func TestDefaultServerConfig(t *testing.T) {
	cfg := DefaultServerConfig()
	if cfg.Addr == "" || cfg.Timeout <= 0 {
		t.Fatal("default config must be valid")
	}
}

func TestServerUptime(t *testing.T) {
	s := &Server{start: time.Now()}
	if s.Uptime() < 0 {
		t.Fatal("uptime negative")
	}
}

func TestServerStopNoServer(t *testing.T) {
	s := &Server{}
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("stop without server panicked: %v", r)
		}
	}()
	_ = s.Stop(context.Background())
}

func TestPIDHelpers(t *testing.T) {
	dir := t.TempDir()
	pidfile := filepath.Join(dir, "kotha.pid")
	if err := writePID(pidfile, 4242); err != nil {
		t.Fatal(err)
	}
	pid, err := readPID(pidfile)
	if err != nil || pid != 4242 {
		t.Fatalf("readPID=%d err=%v", pid, err)
	}
	if pidAlive(-1) {
		t.Fatal("negative pid should be false")
	}
}

func TestServerStatusForMissing(t *testing.T) {
	s := ServerStatusFor("/no/such/file")
	if s.Running || s.PID != 0 {
		t.Fatalf("unexpected status: %+v", s)
	}
}

func TestDaemonStop(t *testing.T) {
	dir := t.TempDir()
	pidfile := filepath.Join(dir, "kotha.pid")
	if err := writePID(pidfile, 4242); err != nil {
		t.Fatal(err)
	}
	if err := StopServer(pidfile); err == nil {
		t.Fatal("expected error for non-existent pid")
	}
}

var _ = errors.Is