package server

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/kothagpt/kotha/internal/app"
	"github.com/kothagpt/kotha/internal/config"
	"github.com/kothagpt/kotha/internal/db"
	"github.com/kothagpt/kotha/internal/logging"
)

// ServerConfig holds runtime configuration for the background server.
type ServerConfig struct {
	Addr    string
	Cwd     string
	Debug   bool
	APIKey  string
	Timeout time.Duration
}

// DefaultServerConfig returns a sensible default configuration.
func DefaultServerConfig() ServerConfig {
	return ServerConfig{
		Addr:    "127.0.0.1:8080",
		Timeout: 30 * time.Second,
	}
}

// Server is a background HTTP server exposing the Kotha app over the
// OpenAPI surface (see openapi.yaml). It owns the app lifecycle.
type Server struct {
	cfg   ServerConfig
	app   *app.App
	srv   *http.Server
	mu    sync.Mutex
	start time.Time
}

// NewServer constructs a Server bound to the given config.
func NewServer(cfg ServerConfig) (*Server, error) {
	if cfg.Addr == "" {
		cfg.Addr = DefaultServerConfig().Addr
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = DefaultServerConfig().Timeout
	}
	provider, err := config.Load(cfg.Cwd, cfg.Debug)
	if err != nil {
		return nil, err
	}
	conn, err := db.Connect()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	application, err := app.New(ctx, conn, provider)
	if err != nil {
		return nil, err
	}
	srv := &Server{
		cfg:   cfg,
		app:   application,
		start: time.Now(),
	}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/sessions/{session_id}/ask", srv.handleAsk)
	mux.HandleFunc("POST /v1/sessions/{session_id}/cancel", srv.handleCancel)
	mux.HandleFunc("POST /v1/sessions/{session_id}/summarize", srv.handleSummarize)
	mux.HandleFunc("GET /v1/sessions", srv.handleListSessions)
	mux.HandleFunc("POST /v1/sessions", srv.handleCreateSession)
	mux.HandleFunc("GET /v1/sessions/{session_id}", srv.handleGetSession)
	mux.HandleFunc("DELETE /v1/sessions/{session_id}", srv.handleDeleteSession)
	mux.HandleFunc("GET /v1/sessions/{session_id}/messages", srv.handleListMessages)
	mux.HandleFunc("GET /v1/sessions/{session_id}/export", srv.handleExportSession)
	mux.HandleFunc("GET /v1/models", srv.handleCurrentModel)
	mux.HandleFunc("PUT /v1/models", srv.handleUpdateModel)
	mux.HandleFunc("GET /healthz", srv.handleHealth)
	mux.HandleFunc("POST /v1/stop", srv.handleStop)
	srv.srv = &http.Server{
		Addr:         cfg.Addr,
		Handler:      APIKeyAuth(cfg.APIKey, mux),
		IdleTimeout:  cfg.Timeout,
		ReadTimeout:  cfg.Timeout,
		WriteTimeout: cfg.Timeout,
	}
	return srv, nil
}

// Run blocks until the server is stopped (signal or Stop).
func (s *Server) Run() error {
	logging.Info("server starting", "addr", s.cfg.Addr)
	if err := s.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

// Stop gracefully shuts the server down.
func (s *Server) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.app != nil {
		s.app.Shutdown()
	}
	if s.srv == nil {
		return nil
	}
	return s.srv.Shutdown(ctx)
}

// Uptime returns the elapsed time since start.
func (s *Server) Uptime() time.Duration {
	return time.Since(s.start)
}

// Addr returns the listening address.
func (s *Server) Addr() string { return s.cfg.Addr }

// RunServer constructs and runs a server, handling SIGINT/SIGTERM for
// graceful shutdown. Returns an error only if startup fails.
func RunServer(cfg ServerConfig) error {
	srv, err := NewServer(cfg)
	if err != nil {
		return err
	}
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-quit
		logging.Info("server shutting down")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = srv.Stop(ctx)
	}()
	return srv.Run()
}
